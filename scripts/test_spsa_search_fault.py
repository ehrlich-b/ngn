"""Exercise search-error propagation through the real tuner and controller."""
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile
import unittest
from types import SimpleNamespace
from unittest.mock import patch

import spsa_chunks as s


class SearchFaultTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.build.cleanup)
        cls.tuner = Path(cls.build.name) / 'spsa'
        repo = Path(__file__).resolve().parents[1]
        subprocess.run(['go', 'build', '-p', '2', '-o', str(cls.tuner), './cmd/spsa'],
                       cwd=repo, check=True)

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.switch = self.root / 'mode'
        self.switch.write_text('healthy\n')
        fixture = Path(__file__).resolve().parent / 'fixtures/spsa_search_fault.py'
        self.engine = self.root / 'engine'
        self.engine.write_text('#!/bin/sh\nexec ' + shlex.join(
            [sys.executable, str(fixture), str(self.switch)]) + '\n')
        self.engine.chmod(0o700)
        self.book = self.root / 'openings.txt'
        self.book.write_text('7k/8/5KQ1/8/8/8/8/8 w - - 0 1\n')

    def command(self, root, history, count, resume):
        args = [str(self.tuner), '-engine', str(self.engine), '-openings', str(self.book),
                '-tc', '10+0.1', '-pairs', '2', '-iters', '1200', '-chunk-iters', str(count),
                '-report', '1', '-lowpower=false', '-out', str(history)]
        return args + (['-resume'] if resume else [])

    def campaign(self):
        campaign = s.Campaign({})
        campaign.root = self.root
        campaign.verify_inputs = lambda: None
        return campaign

    def test_controller_stops_on_search_error_without_checkpoint(self):
        self.switch.write_text('fault\n')
        campaign = self.campaign()
        with patch.object(s, 'argv', side_effect=self.command), \
                patch.object(s, 'clean_env', return_value=dict(os.environ)), \
                patch.object(s, 'DATA', self.root), \
                patch.object(s.socket, 'gethostname', return_value='NewPC'), \
                patch.object(s.os, 'getuid', return_value=1000), \
                patch.object(s.os, 'getpriority', return_value=0), \
                    patch.object(s.os, 'statvfs', return_value=SimpleNamespace(f_bavail=30 * 2**30, f_frsize=1)), \
                patch.object(campaign, 'locked', side_effect=lambda action: action()):
            self.assertEqual(campaign.execute(), 1)
        final = json.loads((self.root / 'final-vector.json').read_text())
        self.assertEqual(final['state'], 'FAULT')
        self.assertIsNone(final['checkpoint'])
        self.assertIn('search-error', final['reason'])
        events = [json.loads(line)['event'] for line in
                  (self.root / 'events.jsonl').read_text().splitlines()]
        self.assertIn('FAULT', events)
        self.assertNotIn('CHUNK_DONE', events)
        self.assertNotIn('RESUME_PROVED', events)
        self.assertEqual((self.root / 'resume-proof.jsonl').read_bytes(), b'')
        log = (self.root / 'proof-stop2.log').read_text()
        self.assertIn('info string error search session failed: fixture panic', log)
        self.assertIn('faulted game batch; theta and checkpoint unchanged', log)

    def test_resumed_search_error_preserves_exact_checkpoint(self):
        history = self.root / 'spsa.jsonl'
        subprocess.run(self.command(self.root, history, 2, False), check=True,
                       stdout=subprocess.DEVNULL)
        before = history.read_bytes()
        self.assertEqual([json.loads(line)['iter'] for line in before.splitlines()], [1, 2])
        self.switch.write_text('fault\n')
        campaign = self.campaign()
        events = []
        campaign.event = lambda event, **detail: events.append(event)
        with patch.object(s, 'argv', side_effect=self.command), \
                patch.object(s, 'clean_env', return_value=dict(os.environ)):
            with self.assertRaisesRegex(RuntimeError, '(?s)tuner exit 1.*search-error'):
                campaign.run_chunk(history, 100, True, 'resumed-fault')
        self.assertEqual(history.read_bytes(), before)
        self.assertNotIn('CHUNK_DONE', events)
        self.assertIn('resumed exact recipe from iter 2 ',
                      (self.root / 'resumed-fault.log').read_text())


if __name__ == '__main__':
    unittest.main()

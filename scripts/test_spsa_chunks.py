"""Small process/lock fixtures for the bounded SPSA controller."""
import fcntl
import json
import os
from pathlib import Path
import signal
import tempfile
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch

import spsa_chunks as s


class ControllerTests(unittest.TestCase):
    def test_contended_lock_obeys_budget_without_running_games(self):
        with tempfile.TemporaryDirectory() as directory:
            lock = Path(directory) / 'cpu.lock'
            lock.write_text('existing lock\n')
            with lock.open() as held, patch.object(s, 'LOCK', lock):
                fcntl.flock(held, fcntl.LOCK_EX | fcntl.LOCK_NB)
                campaign = s.Campaign({})
                campaign.event = lambda *a, **k: None
                campaign.deadline = time.monotonic() + .02
                with self.assertRaises(s.BudgetExpired):
                    campaign.locked(lambda: self.fail('games ran without lock'))
            self.assertEqual(lock.read_text(), 'existing lock\n')

    def test_fault_records_final_reason_without_starting_games(self):
        with tempfile.TemporaryDirectory() as directory:
            campaign = s.Campaign({})
            campaign.root = Path(directory)
            campaign.event = lambda *a, **k: None
            with patch.object(s.socket, 'gethostname', return_value='NewPC'), \
                    patch.object(s.os, 'getuid', return_value=1000), \
                    patch.object(s.os, 'getpriority', return_value=0), \
                    patch.object(s.os, 'statvfs', return_value=SimpleNamespace(f_bavail=30 * 2**30, f_frsize=1)), \
                    patch.object(campaign, 'verify_inputs', side_effect=RuntimeError('net hash changed')):
                self.assertEqual(campaign.execute(), 1)
            final = json.loads((campaign.root / 'final-vector.json').read_text())
            self.assertEqual(final['state'], 'FAULT')
            self.assertIn('net hash changed', final['reason'])
            self.assertIsNone(final['checkpoint'])

    def test_child_failure_is_not_retried(self):
        with tempfile.TemporaryDirectory() as directory:
            campaign = s.Campaign({})
            campaign.root = Path(directory)
            campaign.event = lambda *a, **k: None
            calls = []
            script = "import sys; print('fixture engine fault', flush=True); sys.exit(7)"
            def command(*args):
                calls.append(args)
                return [os.sys.executable, '-c', script]
            with patch.object(campaign, 'verify_inputs'), patch.object(s, 'argv', side_effect=command):
                with self.assertRaisesRegex(RuntimeError, 'tuner exit 7.*fixture engine fault'):
                    campaign.run_chunk(campaign.root / 'history', 100, False, 'fault')
            self.assertEqual(len(calls), 1)

    def test_cap_kills_running_child_and_preserves_checkpoint(self):
        with tempfile.TemporaryDirectory() as directory:
            campaign = s.Campaign({})
            campaign.root = Path(directory)
            campaign.event = lambda *a, **k: None
            campaign.deadline = time.monotonic() + .3
            pidfile = campaign.root / 'fixture.pid'
            history = campaign.root / 'history'
            row = dict(version=2, iter=1, recipe={'fixture': True})
            history.write_text(json.dumps(row) + '\n')
            before = history.read_bytes()
            script = ('import os,time,pathlib; '
                      f'pathlib.Path({str(pidfile)!r}).write_text(str(os.getpid())); time.sleep(30)')
            with patch.object(campaign, 'verify_inputs'), \
                    patch.object(s, 'argv', return_value=[os.sys.executable, '-c', script]):
                with self.assertRaises(s.BudgetExpired):
                    campaign.run_chunk(history, 100, False, 'cap')
            with self.assertRaises(ProcessLookupError):
                os.kill(int(pidfile.read_text()), 0)
            self.assertEqual(history.read_bytes(), before)

    def test_partial_history_is_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            history = Path(directory) / 'history'
            history.write_text('{"iter":1}')
            with self.assertRaisesRegex(RuntimeError, 'incomplete history'):
                s.read_rows(history)


if __name__ == '__main__':
    unittest.main()

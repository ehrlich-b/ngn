"""Offline recipe, ledger rejection, and compiled UCI default proofs."""
import copy
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from scripts import match_queue as mq
from scripts import option_verdict as ov

FIX = mq.REPO / 'experiments/option-verdict/fixtures'


class OptionVerdictTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def ledger(self, case):
        path = self.root / 'ledger.jsonl'
        # Only test fixtures contain this portable path placeholder.
        path.write_text((FIX / (case + '.jsonl')).read_text().replace('$REPO', str(mq.REPO)))
        return path

    def test_jobs_cover_registry_and_dry_run(self):
        jobs = ov.recipes()
        self.assertEqual(list(jobs.values())[0][1], 'T1104ContPlies')
        result = subprocess.run([os.sys.executable, str(mq.REPO / 'scripts/match_queue.py'),
                                 'run', '--jobs', str(ov.JOBS), '--dry-run'],
                                env=mq.clean_env(), capture_output=True, text=True, check=True)
        self.assertEqual(len(result.stdout.splitlines()), len(ov.options()))
        for line in result.stdout.splitlines():
            print(json.loads(line)['job'] + ': queue dry-run passed', flush=True)

    def test_empty_mixed_all_accepted_compiled_defaults(self):
        for case in ('empty', 'mixed', 'all-accepted'):
            with self.subTest(case=case):
                ledger = self.ledger(case)
                expected = {ov.recipes()[r['job_id']][1] for r in mq.rows(ledger) if r['verdict'] == 'H1'}
                generated = self.root / 'accepted.go'
                subprocess.run([os.sys.executable, '-m', 'scripts.option_verdict',
                                '--ledger', str(ledger), '--output', str(generated)],
                               cwd=mq.REPO, env=mq.clean_env(), check=True)
                self.assertEqual(set(ov.defaults(ledger)), expected)
                overlay = self.root / 'overlay.json'
                overlay.write_text(json.dumps({'Replace': {
                    str(mq.REPO / 'engine/accepted_option_defaults.go'): str(generated)}}))
                binary = self.root / 'ngn'
                subprocess.run(['go', 'build', '-p', '2', '-overlay', str(overlay), '-o', str(binary), '.'],
                               cwd=mq.REPO, env=mq.clean_env(), check=True)
                output = subprocess.run([str(binary)], input='uci\nquit\n', text=True,
                                        env=mq.clean_env(), capture_output=True, check=True).stdout
                advertised = dict(re.findall(r'option name (\S+) type spin default ([01]) min 0 max 1', output))
                self.assertEqual({n for n in ov.options() if advertised.get(n) == '1'}, expected)
                self.assertTrue(set(ov.options()) <= advertised.keys())
                subprocess.run(['go', 'test', '-p', '2', '-short', '-count=1', '-overlay', str(overlay),
                                './...'],
                               cwd=mq.REPO, env=mq.clean_env(), check=True)
                print(f'{case}: compiled default=1 options={sorted(expected)}', flush=True)

    def test_bad_rows_fail_closed_without_touching_output(self):
        good = mq.rows(self.ledger('all-accepted'))[0]
        bad_rows = []
        for key in ('job_hash', 'book_hash'):
            row = copy.deepcopy(good)
            row[key] = '0' * 64
            bad_rows.append(row)
        for key in ('source_hash', 'net_hash'):
            row = copy.deepcopy(good)
            row[key]['candidate'] = '0' * len(row[key]['candidate'])
            bad_rows.append(row)
        for key, value in (('argv', ['fake']), ('pairs', 0), ('faults', ['synthetic fault'])):
            row = copy.deepcopy(good)
            row[key] = value
            bad_rows.append(row)
        row = copy.deepcopy(good)
        row.update(games=2, pairs=1, pair_counts=[0, 0, 1, 0, 0], wdl=[0, 2, 0])
        bad_rows.append(row)
        for row in bad_rows:
            ledger = self.root / 'bad.jsonl'
            ledger.write_text(json.dumps(row) + '\n')
            with self.assertRaises(ValueError):
                ov.defaults(ledger)
        ledger.write_text(json.dumps(good))
        with self.assertRaises(ValueError):
            ov.defaults(ledger)
        ledger.write_text((json.dumps(good) + '\n') * 2)
        with self.assertRaises(ValueError):
            ov.defaults(ledger)
        output = self.root / 'defaults.go'
        output.write_text('preserve on validation failure')
        result = subprocess.run([os.sys.executable, '-m', 'scripts.option_verdict',
                                 '--ledger', str(ledger), '--output', str(output)],
                                env=mq.clean_env(), capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(output.read_text(), 'preserve on validation failure')

    def test_unrelated_accepted_job_cannot_enable_option(self):
        ledger = self.ledger('empty')
        row = mq.rows(mq.REPO / 'experiments/match-queue/fixtures/ledger.jsonl')[0]
        row['verdict'] = 'H1'
        ledger.write_text(json.dumps(row) + '\n')
        self.assertEqual(ov.defaults(ledger), {})

    def test_production_ledger_keeps_shipped_defaults_off(self):
        self.assertEqual(ov.defaults(mq.REPO / 'experiments/ledger.jsonl'), {})
        self.assertEqual((mq.REPO / 'engine/accepted_option_defaults.go').read_text(), ov.render({}))


if __name__ == '__main__':
    unittest.main()

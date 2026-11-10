"""Offline recipe, ledger rejection, and compiled UCI default proofs."""
import copy
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from unittest import mock

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
                                '--ledger', str(ledger), '--artifacts', str(FIX / 'artifacts.json'), '--output', str(generated)],
                               cwd=mq.REPO, env=mq.clean_env(), check=True)
                self.assertEqual(set(ov.defaults(ledger, artifacts=FIX / 'artifacts.json')), expected)
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
                ov.defaults(ledger, artifacts=FIX / 'artifacts.json')
        ledger.write_text(json.dumps(good))
        with self.assertRaises(ValueError):
            ov.defaults(ledger, artifacts=FIX / 'artifacts.json')
        ledger.write_text((json.dumps(good) + '\n') * 2)
        with self.assertRaises(ValueError):
            ov.defaults(ledger, artifacts=FIX / 'artifacts.json')
        output = self.root / 'defaults.go'
        output.write_text('preserve on validation failure')
        result = subprocess.run([os.sys.executable, '-m', 'scripts.option_verdict',
                                 '--ledger', str(ledger), '--artifacts', str(FIX / 'artifacts.json'), '--output', str(output)],
                                env=mq.clean_env(), capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(output.read_text(), 'preserve on validation failure')

    def test_moved_checkout_accepted_ledgers(self):
        for prefix in ('/another/checkout', '/another/checkout with spaces'):
            ledger = self.root / 'remote.jsonl'
            ledger.write_text((FIX / 'all-accepted.jsonl').read_text().replace('$REPO', prefix))
            self.assertEqual(set(ov.defaults(ledger, artifacts=FIX / 'artifacts.json')), set(ov.options()))
            # Relocation changes only argv paths, never recorded provenance.
            local = mq.rows(self.ledger('all-accepted'))
            remote = mq.rows(ledger)
            for original, moved in zip(local, remote):
                self.assertEqual({k: v for k, v in original.items() if k != 'argv'},
                                 {k: v for k, v in moved.items() if k != 'argv'})

    def test_moved_checkout_tampering_is_refused(self):
        ledger = self.root / 'remote.jsonl'
        ledger.write_text((FIX / 'all-accepted.jsonl').read_text().replace('$REPO', '/another/checkout'))
        good = mq.rows(ledger)[0]
        mutations = []
        for index, value in ((0, 'fake-taskset'), (3, '/another/checkout/build/fake-sprt'),
                             (5, '/another/checkout/build/fake-ngn'),
                             (7, '/elsewhere/build/ngn-7c2776ff'),
                             (9, '20+0.1'), (11, '/elsewhere/openings.txt'),
                             (13, '40000'), (25, '6')):
            row = copy.deepcopy(good)
            row['argv'][index] = value
            mutations.append(row)
        for old, new in (('T1104ContPlies=1', 'T1104ContPlies=0'),
                         ('EvalFile=/another/checkout/engine/nets/ngn-h768-2dcb6d93.nnue',
                          'EvalFile=/elsewhere/net.nnue'),
                         ('UseNNUE=true', 'UseNNUE=false')):
            row = copy.deepcopy(good)
            row['argv'] = [new if arg == old else arg for arg in row['argv']]
            mutations.append(row)
        row = copy.deepcopy(good)
        row['argv'].extend(['-newoption', 'T1104ContPlies=1'])
        mutations.append(row)
        for key in ('job_hash', 'book_hash', 'source_hash', 'net_hash', 'binary_hash', 'harness_hash', 'verdict'):
            row = copy.deepcopy(good)
            if key in ('source_hash', 'net_hash'):
                row[key]['candidate'] = '0' * len(row[key]['candidate'])
            elif key == 'binary_hash':
                row[key]['candidate'] = 'a' * 64
            else:
                row[key] = 'ACCEPTED' if key == 'verdict' else '0' * 64
            mutations.append(row)
        for index, row in enumerate(mutations):
            with self.subTest(mutation=index):
                ledger.write_text(json.dumps(row) + '\n')
                with self.assertRaises(ValueError):
                    ov.defaults(ledger, artifacts=FIX / 'artifacts.json')
        for verdict in ('H0', 'INCONCLUSIVE'):
            row = copy.deepcopy(good)
            row['verdict'] = verdict
            ledger.write_text(json.dumps(row) + '\n')
            self.assertEqual(ov.defaults(ledger, artifacts=FIX / 'artifacts.json'), {})

    def test_unrelated_accepted_job_cannot_enable_option(self):
        ledger = self.ledger('empty')
        row = mq.rows(mq.REPO / 'experiments/match-queue/fixtures/ledger.jsonl')[0]
        row['verdict'] = 'H1'
        ledger.write_text(json.dumps(row) + '\n')
        self.assertEqual(ov.defaults(ledger, artifacts=FIX / 'artifacts.json'), {})

    def test_valid_digest_tampering_preserves_output(self):
        ledger = self.ledger('all-accepted')
        good = mq.rows(ledger)[0]
        output = self.root / 'defaults.go'
        output.write_text('preserve on validation failure')
        for target in ('base', 'candidate', 'both', 'harness', 'both-and-harness'):
            with self.subTest(target=target):
                row = copy.deepcopy(good)
                if target in ('both', 'both-and-harness'):
                    row['binary_hash'] = dict(base='a' * 64, candidate='a' * 64)
                elif target in ('base', 'candidate'):
                    row['binary_hash'][target] = 'a' * 64
                if target in ('harness', 'both-and-harness'):
                    row['harness_hash'] = 'b' * 64
                ledger.write_text(json.dumps(row) + '\n')
                # Each altered row still passes the queue's schema validation.
                mq.rows(ledger)
                with self.assertRaisesRegex(ValueError, 'ledger provenance mismatch'):
                    ov.defaults(ledger, artifacts=FIX / 'artifacts.json')
                result = subprocess.run([os.sys.executable, '-m', 'scripts.option_verdict',
                                         '--ledger', str(ledger), '--artifacts', str(FIX / 'artifacts.json'),
                                         '--output', str(output)],
                                        env=mq.clean_env(), capture_output=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(output.read_text(), 'preserve on validation failure')

    def test_local_artifacts_supply_independent_hashes(self):
        jobs = ov.recipes()
        good = mq.rows(self.ledger('all-accepted'))[0]
        build = self.root / 'build'
        build.mkdir()
        binary = build / 'ngn-7c2776ff'
        harness = build / 'sprt'
        binary.write_bytes(b'synthetic engine')
        harness.write_bytes(b'synthetic harness')
        good['binary_hash'] = {r: mq.sha(binary) for r in ('base', 'candidate')}
        good['harness_hash'] = mq.sha(harness)
        ledger = self.root / 'local.jsonl'
        ledger.write_text(json.dumps(good) + '\n')
        schema = self.root / 'experiments/ledger.schema.json'
        schema.parent.mkdir()
        schema.write_bytes((mq.REPO / 'experiments/ledger.schema.json').read_bytes())
        with mock.patch.object(mq, 'REPO', self.root), mock.patch.object(ov, 'recipes', return_value=jobs):
            self.assertEqual(ov.defaults(ledger), {'T1104ContPlies': 1})
            binary.write_bytes(b'changed engine')
            with self.assertRaisesRegex(ValueError, 'binary_hash'):
                ov.defaults(ledger)
            binary.write_bytes(b'synthetic engine')
            harness.write_bytes(b'changed harness')
            with self.assertRaisesRegex(ValueError, 'harness_hash'):
                ov.defaults(ledger)
            harness.unlink()
            with self.assertRaises(FileNotFoundError):
                ov.defaults(ledger)

    def test_production_ledger_keeps_shipped_defaults_off(self):
        self.assertEqual(ov.defaults(mq.REPO / 'experiments/ledger.jsonl'), {})
        self.assertEqual((mq.REPO / 'engine/accepted_option_defaults.go').read_text(), ov.render({}))


if __name__ == '__main__':
    unittest.main()

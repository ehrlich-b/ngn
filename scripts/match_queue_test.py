"""Small synthetic proofs; no chess games, network access or service install."""
import copy
import datetime
import fcntl
import json
import math
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

from scripts import match_queue as mq

FIX = mq.REPO / 'experiments/match-queue/fixtures'
RUNNER = mq.REPO / 'scripts/match_queue.py'


ACCEPTED_CLOCKS = ('0.001', '0.001+0.1', '0.0019+0', '1+0.0001',
                   '60+0.6', '86400', '86400+3600', '0.001+3600',
                   '0.001000000000', '86399.999999999999',
                   '1+0', '1+0.000000000001', '1+3599.999999999999')
REJECTED_CLOCKS = ('0', '0.0001+0.1', '0.000999999+0', '0.0001',
                   '0.00099999999999999999999', '9223372036854776+0',
                   '2000000000+0', '1+2000000000', '86400.000000000001',
                   '86400.001', '1+3600.000000000001', '1+3600.001',
                   'NaN', 'Inf', 'NaN+0', 'Inf+0', '1+NaN', '1+Inf',
                   '-1', '-0.001', '1+-0.000000000000001', '1+-1',
                   '1\n', '1+0\n', 'oops', '1e3', '1+', '+1', ' 1',
                   '9' * 400 + '+0', '1+' + '9' * 400,
                   '0.00100000000000000000001', '1+0.000000000000001',
                   '1.' + '0' * 400, '1+0.' + '0' * 400)


def proof_directory(test):
    output = os.environ.get('MATCH_QUEUE_PROOF_OUTPUT')
    if output:
        parent = Path(output).resolve()
        if parent == mq.REPO or mq.REPO in parent.parents:
            raise ValueError('proof output must be outside checkout')
        parent.mkdir(parents=True, exist_ok=True)
        return Path(tempfile.mkdtemp(prefix=test._testMethodName + '-', dir=parent))
    temporary = tempfile.TemporaryDirectory()
    test.addCleanup(temporary.cleanup)
    return Path(temporary.name)


class LedgerTests(unittest.TestCase):
    def setUp(self):
        self.root = proof_directory(self)
        self.row = mq.rows(FIX / 'ledger.jsonl')[0]

    def test_schema_required_provenance_counts_and_verdict(self):
        mq.validate_row(self.row)
        # Validate every production row, including after the queue records results.
        mq.rows(mq.REPO / 'experiments/ledger.jsonl')
        for field in ('source_hash', 'binary_hash', 'net_hash', 'book_hash', 'pair_counts', 'verdict'):
            with self.subTest(missing=field):
                bad = copy.deepcopy(self.row)
                del bad[field]
                with self.assertRaises(ValueError):
                    mq.validate_row(bad)
        for field in ('source_hash', 'binary_hash', 'net_hash'):
            for role in ('base', 'candidate'):
                bad = copy.deepcopy(self.row)
                del bad[field][role]
                with self.assertRaises(ValueError):
                    mq.validate_row(bad)
        for field, value in [('pairs', 3), ('games', 3), ('book_hash', 'unknown'),
                             ('pair_counts', [0, 2]), ('verdict', 'PASS')]:
            bad = dict(self.row, **{field: value})
            with self.assertRaises(ValueError):
                mq.validate_row(bad)
        print('Missing source/binary/net/book hashes, pair counts and verdict all rejected.')

    def test_report_twice_byte_identical_and_ledger_only(self):
        # There are no jobs, binaries, net or book files beside this copied ledger.
        ledger = self.root / 'ledger.jsonl'
        shutil.copyfile(FIX / 'ledger.jsonl', ledger)
        outputs = [self.root / 'first.md', self.root / 'second.md']
        for output in outputs:
            subprocess.run([sys.executable, str(RUNNER), 'strength', '--ledger', str(ledger),
                            '--output', str(output)], check=True, cwd=self.root, env=mq.clean_env())
        subprocess.run(['diff', '-u', *map(str, outputs)], check=True, env=mq.clean_env())
        self.assertIn(b'synthetic-fixture', outputs[0].read_bytes())
        # The checked-in report is also exactly the empty production ledger view.
        mq.strength(mq.REPO / 'experiments/ledger.jsonl', outputs[0])
        self.assertEqual(outputs[0].read_bytes(), (mq.REPO / 'docs/STRENGTH.md').read_bytes())
        print('Both generator invocations diff byte-identically; committed report matches ledger.')

    def test_truncated_and_duplicate_ledger_fail_closed(self):
        path = self.root / 'bad.jsonl'
        path.write_text(json.dumps(self.row))
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            mq.rows(path)
        path.write_text((json.dumps(self.row) + '\n') * 2)
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            mq.rows(path)

    def test_populated_production_ledger(self):
        # Model a populated production path without putting synthetic results
        # in the real production ledger. Validation must not assume it is empty.
        repo = self.root / 'populated'
        (repo / 'experiments').mkdir(parents=True)
        shutil.copyfile(mq.REPO / 'experiments/ledger.schema.json',
                        repo / 'experiments/ledger.schema.json')
        ledger = repo / 'experiments/ledger.jsonl'
        ledger.write_text(json.dumps(self.row) + '\n')
        with mock.patch.object(mq, 'REPO', repo):
            self.assertEqual(mq.rows(mq.REPO / 'experiments/ledger.jsonl'), [self.row])
            mq.strength(ledger, repo / 'first.md')
            mq.strength(ledger, repo / 'second.md')
            self.assertEqual((repo / 'first.md').read_bytes(), (repo / 'second.md').read_bytes())
        print('Populated production-path ledger validates and renders deterministically in isolation.')


class RunnerTests(unittest.TestCase):
    def setUp(self):
        self.root = proof_directory(self)
        self.jobs = self.root / 'jobs'
        self.jobs.mkdir()
        for path in sorted(FIX.glob('0*.json')):
            shutil.copyfile(path, self.jobs / path.name)
        self.lock = self.root / 'cpu0-11.lock'
        self.lock.write_text('existing shared inode\n')
        self.ledger = self.root / 'ledger.jsonl'
        self.events = self.root / 'events.jsonl'
        self.env = mq.clean_env()
        self.env.update(MATCH_QUEUE_LOCK=str(self.lock), MATCH_QUEUE_EVENTS=str(self.events))
        self.argv = [sys.executable, str(RUNNER), 'run', '--repo', str(mq.REPO),
                     '--jobs', str(self.jobs), '--ledger', str(self.ledger),
                     '--strength', str(self.root / 'STRENGTH.md'), '--output', str(self.root / 'runs'),
                     '--sprt', str(FIX / 'fake_sprt.py'), '--lock', str(self.lock),
                     '--cpus', str(min(os.sched_getaffinity(0)))]

    def run_once(self, check=True):
        result = subprocess.run(self.argv + ['--once'], env=self.env, text=True,
                                capture_output=True, timeout=8)
        with (self.root / 'controller.log').open('a') as log:
            log.write(result.stdout + result.stderr)
        if check:
            self.assertEqual(result.returncode, 0, result.stderr)
        return result

    def test_standing_two_jobs_cpu_lock_gap_and_restart(self):
        inode = self.lock.stat().st_ino
        # Independent standing process; no owner/predecessor PID or process wait.
        with self.lock.open('r') as held, (self.root / 'controller.log').open('w') as log:
            fcntl.flock(held, fcntl.LOCK_EX)
            child = subprocess.Popen(self.argv, env=self.env, stdout=log, stderr=subprocess.STDOUT,
                                     start_new_session=True)
            try:
                time.sleep(.15)
                self.assertIsNone(child.poll())
                self.assertFalse(self.events.exists(), 'runner bypassed a competing CPU holder')
                fcntl.flock(held, fcntl.LOCK_UN)
                deadline = time.monotonic() + 5
                while time.monotonic() < deadline:
                    if self.ledger.exists() and len(self.ledger.read_text().splitlines()) == 2:
                        break
                    if child.poll() is not None:
                        self.fail((self.root / 'controller.log').read_text())
                    time.sleep(.02)
                result = mq.rows(self.ledger)
                self.assertEqual([r['job_id'] for r in result], ['first', 'second'])
                self.assertEqual([r['verdict'] for r in result], ['INCONCLUSIVE'] * 2)
                self.assertIsNone(child.poll(), 'standing runner exited after draining')
            finally:
                child.terminate()
                child.wait(timeout=5)
        events = [json.loads(line) for line in self.events.read_text().splitlines()]
        self.assertEqual([e['event'] for e in events], ['locked-start', 'locked-end'] * 2)
        gap = events[2]['monotonic'] - events[1]['monotonic']
        self.assertGreaterEqual(gap, 0)
        self.assertLess(gap, 5)
        terminal_gap = (datetime.datetime.fromisoformat(result[1]['started_utc']) -
                        datetime.datetime.fromisoformat(result[0]['finished_utc'])).total_seconds()
        self.assertGreaterEqual(terminal_gap, 0)
        self.assertLess(terminal_gap, 5)
        (self.root / 'proof-summary.json').write_text(json.dumps(dict(
            lock_probes=[e['event'] for e in events], event_gap_seconds=gap,
            finished_to_started_seconds=terminal_gap, job_ids=[r['job_id'] for r in result]), indent=2) + '\n')
        self.assertEqual(inode, self.lock.stat().st_ino)
        self.assertEqual(self.lock.read_text(), 'existing shared inode\n')
        before = self.ledger.read_bytes()
        self.run_once()
        self.assertEqual(before, self.ledger.read_bytes(), 'restart duplicated finished jobs')
        self.assertEqual(len(self.events.read_text().splitlines()), 4)
        print(f'Two independent fake SPRT jobs observed CPU lock at start/end; inter-job gap {terminal_gap:.6f}s; restart skipped both.')

    def test_completed_recipe_change_and_interrupted_job_rejected(self):
        self.run_once()
        path = self.jobs / '01-first.json'
        job = mq.read(path)
        job['bounds']['elo1'] = 4
        path.write_text(json.dumps(job))
        self.assertIn('completed job changed', self.run_once(check=False).stderr)
        # A new ID whose run directory already exists is an interrupted job.
        job['id'] = 'unfinished'
        path.write_text(json.dumps(job))
        (self.root / 'runs/unfinished').mkdir()
        self.assertNotEqual(self.run_once(check=False).returncode, 0)
        self.assertEqual(len(mq.rows(self.ledger)), 2)

    def test_new_jobs_arrive_while_idle(self):
        log = (self.root / 'controller.log').open('w')
        with log:
            # Begin with a directory whose entries are not JSON recipes.
            self.argv[self.argv.index('--jobs') + 1] = str(self.root / 'incoming')
            incoming = self.root / 'incoming'
            incoming.mkdir()
            child = subprocess.Popen(self.argv, env=self.env, stdout=log, stderr=subprocess.STDOUT)
            try:
                time.sleep(.1)
                self.assertIsNone(child.poll())
                shutil.copyfile(FIX / '01-first.json', incoming / '01-first.json')
                deadline = time.monotonic() + 4
                while time.monotonic() < deadline and not self.ledger.exists():
                    time.sleep(.02)
                self.assertEqual(len(mq.rows(self.ledger)), 1)
            finally:
                child.terminate()
                child.wait(timeout=5)

    def test_bad_summary_or_failure_never_scored(self):
        text = (FIX / 'fake_sprt.py').read_text().split("print('''", 1)[1].rsplit("''')", 1)[0]
        self.assertEqual(mq.summarize(text, 0)['verdict'], 'INCONCLUSIVE')
        for bad, rc in [(text, 1), ('no summary', 0), (text + '\ninfo string error: fixture', 0),
                        (text.replace('over 2 pairs', 'over 3 pairs'), 0)]:
            self.assertEqual(mq.summarize(bad, rc)['verdict'], 'ERROR')
        self.env['MATCH_QUEUE_FAKE_FAILURE'] = '1'
        self.assertNotEqual(self.run_once(check=False).returncode, 0)
        result = mq.rows(self.ledger)
        self.assertEqual(len(result), 1)
        self.assertEqual(result[0]['verdict'], 'ERROR')


class RecipeTests(unittest.TestCase):
    def test_committed_controls_dry_run(self):
        root = proof_directory(self)
        command = [sys.executable, str(RUNNER), 'run', '--dry-run', '--output', str(root / 'unused'),
                   '--lock', str(root / 'does-not-exist')]
        result = subprocess.run(command, check=True, text=True, capture_output=True, env=mq.clean_env())
        (root / 'dry-run.jsonl').write_text(result.stdout)
        commands = [json.loads(line) for line in result.stdout.splitlines()]
        self.assertEqual(len(commands), 2)
        aa, pc = [mq.load_job(path, mq.REPO) for path in sorted((mq.REPO / 'experiments/match-queue/jobs').glob('*.json'))]
        self.assertEqual((aa['min_games'], aa['max_games']), (2000, 2000))
        self.assertEqual(aa['base'], aa['candidate'])
        self.assertIn('h512', pc['base']['net']['path'])
        self.assertIn('h768', pc['candidate']['net']['path'])
        self.assertFalse((root / 'unused').exists(), 'dry-run created runtime outputs')
        for row in commands:
            self.assertIn('-noretries', row['argv'])
        print('Committed A/A 2000 and H768-v-H512 jobs validate without games or CPU lock.')

    def test_invalid_job_recipe_rejected(self):
        root = proof_directory(self)
        original = mq.read(FIX / '01-first.json')
        for change in [('time_control', '0'), ('time_control', 'oops'), ('max_games', 3),
                       ('min_games', 6), ('concurrency', 0)]:
            job = copy.deepcopy(original)
            job[change[0]] = change[1]
            path = root / 'bad.json'
            path.write_text(json.dumps(job))
            with self.assertRaises(ValueError):
                mq.load_job(path, mq.REPO)

    def test_time_control_base_survives_millisecond_conversion(self):
        root = proof_directory(self)
        job = mq.read(FIX / '01-first.json')
        path = root / 'job.json'
        # cmd/sprt truncates base seconds * 1000 to an integer. A zero
        # result selects fixed depth, even when the increment is nonzero.
        for tc in ('0.0001+0.1', '0.000999999+0', '0.0001'):
            with self.subTest(rejected=tc):
                job['time_control'] = tc
                path.write_text(json.dumps(job))
                with self.assertRaisesRegex(ValueError, 'at least 1 millisecond'):
                    mq.load_job(path, mq.REPO)
        for tc in ('0.001', '0.001+0.1', '0.0019+0', '1+0.0001'):
            with self.subTest(accepted=tc):
                job['time_control'] = tc
                path.write_text(json.dumps(job))
                loaded = mq.load_job(path, mq.REPO)
                self.assertEqual(loaded['time_control'], tc)
                self.assertGreaterEqual(int(float(tc.split('+')[0]) * 1000), 1)
        print('Sub-millisecond bases rejected; 1ms boundary and fractional increments accepted.')

    def test_time_control_milliseconds_fit_harness_integer(self):
        root = proof_directory(self)
        job = mq.read(FIX / '01-first.json')
        path = root / 'job.json'
        # Preserve the old overflow witnesses. Even the two formerly accepted
        # upper neighbors exceed the supported contract: conversion alone did
        # not protect the later watchdog multiplication/addition.
        limit = sys.maxsize + 1
        overflow_seconds = format(limit / 1000, '.20f')
        safe_seconds = format(math.nextafter(limit / 1000, 0), '.20f')
        huge_seconds = '1' + '0' * 400
        for tc in ('9223372036854776+0', overflow_seconds,
                   overflow_seconds + '+0', '1+' + overflow_seconds,
                   huge_seconds + '+0', '1+' + huge_seconds,
                   safe_seconds + '+0', '1+' + safe_seconds):
            with self.subTest(rejected=tc):
                job['time_control'] = tc
                path.write_text(json.dumps(job))
                with self.assertRaisesRegex(ValueError, 'time_control'):
                    mq.load_job(path, mq.REPO)
        print('Old native-int overflow and upper-neighbor witnesses rejected by supported decimal ranges.')

    def test_decimal_digit_cap(self):
        root = proof_directory(self)
        job = mq.read(FIX / '01-first.json')
        path = root / 'job.json'
        for digits in (13, 400):
            for tc, field in [('1.' + '0' * digits, 'base'),
                              ('1+0.' + '0' * digits, 'increment')]:
                with self.subTest(digits=digits, field=field):
                    job['time_control'] = tc
                    path.write_text(json.dumps(job))
                    with self.assertRaisesRegex(ValueError, f'time_control {field}:.*12 digits'):
                        mq.load_job(path, mq.REPO)
        job['time_control'] = '10+0.1'
        # Write raw JSON numbers: json.dumps(float(...)) would erase the witness.
        for field, value in [('elo0', '-3'), ('elo1', '3'),
                             ('alpha', '0.05'), ('beta', '0.05')]:
            for digits in (12, 13, 400):
                for exponent in ('', 'e+0', 'E-0'):
                    with self.subTest(field=field, digits=digits, exponent=exponent):
                        decimal = value + ('' if '.' in value else '.')
                        decimal += '0' * (digits - len(value.partition('.')[2]))
                        text = json.dumps(job).replace(f'"{field}": {job["bounds"][field]}',
                                                       f'"{field}": {decimal}{exponent}')
                        path.write_text(text)
                        if digits > 12:
                            with self.assertRaisesRegex(ValueError, 'numeric field:.*12 digits'):
                                mq.load_job(path, mq.REPO)
                        else:
                            self.assertEqual(mq.load_job(path, mq.REPO)['bounds'][field],
                                             job['bounds'][field])
        for field in ('version', 'min_games', 'max_games', 'concurrency'):
            with self.subTest(field=field):
                text = json.dumps(job).replace(f'"{field}": {job[field]}',
                                               f'"{field}": {job[field]}.' + '0' * 400)
                path.write_text(text)
                with self.assertRaisesRegex(ValueError, 'numeric field:.*12 digits'):
                    mq.load_job(path, mq.REPO)
        for tc in ('10+0.1', '1.' + '0' * 12 + '+0.' + '0' * 12):
            job['time_control'] = tc
            path.write_text(json.dumps(job))
            self.assertEqual(mq.load_job(path, mq.REPO)['time_control'], tc)
        print('12 fractional digits accepted; 13/400 rejected before conversion in both clock parts and all numeric bounds.')

    def test_exact_decimal_boundary_matrix(self):
        root = proof_directory(self)
        job = mq.read(FIX / '01-first.json')
        path = root / 'job.json'
        for tc in ACCEPTED_CLOCKS + REJECTED_CLOCKS:
            with self.subTest(tc=tc):
                job['time_control'] = tc
                path.write_text(json.dumps(job))
                if tc in REJECTED_CLOCKS:
                    with self.assertRaisesRegex(ValueError, 'time_control'):
                        mq.load_job(path, mq.REPO)
                    continue
                loaded = mq.load_job(path, mq.REPO)
                self.assertEqual(loaded['time_control'], tc)
                argv = mq.command(loaded, mq.REPO, Path('fake-sprt'), '0')
                self.assertEqual(argv[argv.index('-tc') + 1], tc)
                self.assertEqual(argv[argv.index('-maxmoves') + 1], '400')
        print(f'Exact decimal matrix: {len(ACCEPTED_CLOCKS)} accepted, {len(REJECTED_CLOCKS)} rejected; maxmoves remains 400.')

    def test_invalid_clocks_launch_nothing_in_dry_run_or_runtime(self):
        root = proof_directory(self)
        jobs = root / 'jobs'
        jobs.mkdir()
        lock = root / 'synthetic.lock'
        lock.write_text('existing shared inode\n')
        events, ledger, output = root / 'events.jsonl', root / 'ledger.jsonl', root / 'runs'
        env = mq.clean_env()
        env.update(MATCH_QUEUE_LOCK=str(lock), MATCH_QUEUE_EVENTS=str(events))
        argv = [sys.executable, str(RUNNER), 'run', '--once', '--jobs', str(jobs),
                '--output', str(output), '--ledger', str(ledger),
                '--strength', str(root / 'STRENGTH.md'), '--sprt', str(FIX / 'fake_sprt.py'),
                '--cpus', str(min(os.sched_getaffinity(0)))]
        job = mq.read(FIX / '01-first.json')
        with (root / 'rejections.log').open('w') as log:
            for tc in REJECTED_CLOCKS:
                job['time_control'] = tc
                (jobs / 'job.json').write_text(json.dumps(job))
                for dry in (True, False):
                    dry_output = root / 'dry-unused'
                    command = argv + (['--output', str(dry_output), '--dry-run', '--lock', str(root / 'absent.lock')] if dry
                                      else ['--lock', str(lock)])
                    result = subprocess.run(command, env=env, text=True, capture_output=True, timeout=3)
                    log.write(f'{tc!r} dry={dry} rc={result.returncode}\n' + result.stdout + result.stderr)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('time_control', result.stderr)
                    self.assertFalse(ledger.exists())
                    self.assertFalse(events.exists(), 'game subprocess launched')
                    self.assertFalse((root / 'STRENGTH.md').exists())
                    if dry:
                        self.assertFalse(dry_output.exists())
                    if output.exists():
                        self.assertEqual(list(output.iterdir()), [])
        self.assertFalse((root / 'absent.lock').exists())
        print('All invalid clocks reject in CLI dry-run/runtime, with no game events or ledger rows.')

    def test_user_unit_verify(self):
        root = proof_directory(self)
        unit = mq.REPO / 'experiments/match-queue/ngn-match-queue.service'
        if shutil.which('systemd-analyze'):
            result = subprocess.run(['systemd-analyze', '--user', 'verify', str(unit)],
                                    text=True, capture_output=True, env=mq.clean_env())
            (root / 'systemd-verify.txt').write_text(result.stdout + result.stderr)
            self.assertEqual(result.returncode, 0, result.stderr)
            print('systemd-analyze --user verify passed; unit not installed.')
        else:
            import configparser
            config = configparser.ConfigParser(interpolation=None)
            config.read(unit)
            self.assertEqual(config.sections(), ['Unit', 'Service', 'Install'])
            for key, expected in [('Type', 'simple'), ('KillMode', 'control-group'),
                                  ('Restart', 'no'), ('Nice', '10')]:
                self.assertEqual(config['Service'][key], expected)
            self.assertTrue(config['Service']['ExecStart'].startswith('/usr/bin/python3 '))
            self.assertIn('match_queue.py run', config['Service']['ExecStart'])
            (root / 'syntax-check.txt').write_text('INI sections and execution/shutdown directives passed.\n')


if __name__ == '__main__':
    unittest.main()

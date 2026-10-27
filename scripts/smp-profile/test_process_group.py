"""Real process trees, isolated in bounded Linux subreaper drivers."""
import concurrent.futures
import ctypes
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock


def driver(mode, root):
    # Linux subreaper adoption lets this driver reap helpers even on old code.
    assert ctypes.CDLL(None, use_errno=True).prctl(36, 1, 0, 0, 0) == 0
    sys.path.insert(0, str(Path(__file__).resolve().parent))
    import time_v
    def timeout(*_):
        raise TimeoutError('isolated driver exceeded eight seconds')
    signal.signal(signal.SIGALRM, timeout)
    signal.alarm(8)
    root = Path(root)
    pids = root / 'pids.json'
    children = []
    helper = None
    observations = {'mode': mode}
    real_popen = subprocess.Popen
    real_capture = time_v.capture_resource_evidence
    failure = KeyboardInterrupt('injected interrupt') if mode == 'keyboard' else RuntimeError('injected failure')
    # A TERM-resistant helper forces the escalation path; no shell/name killing.
    helper_code = 'import signal,time; signal.signal(signal.SIGTERM,signal.SIG_IGN); time.sleep(60)'
    code = ("import subprocess,sys,os,json,time; print(\"stdout marker\", flush=True); print(\"stderr marker\", file=sys.stderr, flush=True); "
            f"p=subprocess.Popen([sys.executable,'-c',{helper_code!r}]); "
            "time.sleep(.08); "
            f"open({str(pids)!r},'w').write(json.dumps([os.getpid(),p.pid])); "
            + ("sys.exit(7)" if mode == 'nonzero' else "sys.exit(0)" if mode in ('early', 'reaped', 'success') else "time.sleep(60)"))

    def spawn(*args, **kwargs):
        p = real_popen(*args, **kwargs)
        children.append(p)
        return p

    def ready(pid):
        nonlocal helper
        deadline = time.monotonic() + 3
        while time.monotonic() < deadline:
            try:
                leader, helper = json.loads(pids.read_text())
            except (FileNotFoundError, json.JSONDecodeError):
                time.sleep(.01)
            else:
                break
        else:
            raise TimeoutError('child did not publish complete tracked PIDs')
        assert leader == pid
        observations.update(leader=leader, helper=helper, pgid=os.getpgid(helper), driver_pgid=os.getpgrp())
        if mode in ('early', 'reaped', 'success', 'nonzero'):
            while time.monotonic() < deadline:
                if Path(f'/proc/{pid}/stat').read_text().split(') ')[1].startswith('Z'):
                    break
                time.sleep(.01)
            else:
                raise TimeoutError('direct child did not exit')
            if mode == 'reaped':
                os.waitpid(pid, 0)
        if mode in ('capture', 'early', 'reaped', 'keyboard'):
            raise failure
        if mode == 'sigint':
            os.kill(os.getpid(), signal.SIGINT)
            raise AssertionError('SIGINT did not interrupt')
        return real_capture(pid)

    sibling = real_popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
    try:
        started = time.monotonic()
        with mock.patch.object(time_v.subprocess, 'Popen', spawn), mock.patch.object(time_v, 'capture_resource_evidence', ready):
            if mode == 'telemetry':
                with mock.patch.object(time_v, 'telemetry', side_effect=failure):
                    try:
                        time_v.run([sys.executable, '-c', code], root/'execution')
                    except BaseException as exc:
                        observations.update(error=repr(exc), same_exception=exc is failure)
            else:
                try:
                    time_v.run([sys.executable, '-c', code], root/'execution',
                               timeout_seconds=0.5 if mode == 'timeout' else None)
                except BaseException as exc:
                    observations.update(error=repr(exc), same_exception=exc is failure)
        observations['elapsed'] = time.monotonic() - started
        assert helper is not None, observations
        state = Path(f'/proc/{helper}/stat')
        deadline = time.monotonic() + 2
        while True:
            stopped = not state.exists() or state.read_text().split(') ')[1].startswith('Z')
            if stopped or time.monotonic() >= deadline:
                break
            time.sleep(.01)
        try:
            os.waitpid(children[0].pid, os.WNOHANG)
            waitable = True
        except ChildProcessError:
            waitable = False
        observations.update(helper_stopped=stopped, direct_waitable=waitable, sibling_alive=sibling.poll() is None)
        print(json.dumps(observations), flush=True)
        assert stopped, 'owned helper survived'
        assert observations['pgid'] == observations['leader']
        assert observations['pgid'] != observations['driver_pgid']
        assert not waitable, 'direct child remains waitable'
        assert observations['sibling_alive'], 'unrelated group was signalled'
        if mode in ('success', 'nonzero'):
            receipt = json.loads((root/'execution'/'time.json').read_text())
            assert receipt['exit_code'] == (7 if mode == 'nonzero' else 0)
        elif mode == 'timeout':
            assert observations['error'].startswith('CommandTimeoutError'), observations
            assert observations['elapsed'] < 3, observations
            execution = root/'execution'
            assert not (execution/'time.json').exists()
            assert 'wall timeout' in json.loads((execution/'failure.json').read_text())['error']
            assert 'stdout marker' in (execution/'stdout.txt').read_text()
            assert 'stderr marker' in (execution/'stderr.txt').read_text()
        elif mode != 'sigint':
            assert observations['same_exception'], observations
        else:
            assert observations['error'].startswith('KeyboardInterrupt'), observations
    finally:
        signal.alarm(0)
        # Only explicitly tracked PIDs; independently recover baseline orphans.
        tracked = [p.pid for p in children] + [sibling.pid]
        if helper:
            tracked.append(helper)
        elif pids.exists():
            tracked.extend(json.loads(pids.read_text()))
        for pid in tracked:
            try:
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        for pid in tracked:
            try:
                os.waitpid(pid, 0)
            except ChildProcessError:
                pass
        print(json.dumps({'reaped_tracked_pids': tracked}), flush=True)


class ProcessGroupTests(unittest.TestCase):
    def scenario(self, mode):
        with tempfile.TemporaryDirectory() as root:
            env = {k: v for k, v in os.environ.items()
                   if not any(word in k.lower() for word in ('credential', 'token', 'secret', 'password'))}
            p = subprocess.run([sys.executable, '-B', __file__, '--driver', mode, root],
                               env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                               text=True, timeout=10)
            print(p.stdout, end='', flush=True)
            self.assertEqual(p.returncode, 0, p.stdout)

    def test_timeout_with_helper(self):
        self.scenario('timeout')

    def test_invalid_timeouts_before_spawn(self):
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import time_v
        with tempfile.TemporaryDirectory() as root, mock.patch.object(time_v.subprocess, 'Popen') as spawn:
            for value in (0, -1, float('inf'), float('-inf'), float('nan')):
                with self.subTest(value=value), self.assertRaises(ValueError):
                    time_v.run([sys.executable, '-c', 'pass'], Path(root)/'execution', timeout_seconds=value)
            spawn.assert_not_called()
            self.assertFalse((Path(root)/'execution').exists())

    def test_cli_timeout_and_validation(self):
        script = Path(__file__).resolve().parent/'time_v.py'
        with tempfile.TemporaryDirectory() as root:
            for value in ('0', '-1', 'nan', 'inf'):
                p = subprocess.run([sys.executable, str(script), '--output', root,
                                    '--timeout-seconds', value, '--', sys.executable, '-c', 'pass'],
                                   capture_output=True, text=True)
                self.assertNotEqual(p.returncode, 0)
                self.assertIn('finite and positive', p.stderr)
                self.assertEqual(list(Path(root).iterdir()), [])
            p = subprocess.run([sys.executable, str(script), '--output', root,
                                '--timeout-seconds', '0.3', '--', sys.executable, '-c',
                                'import time; time.sleep(60)'], capture_output=True, text=True, timeout=5)
            self.assertNotEqual(p.returncode, 0)
            self.assertIn('CommandTimeoutError', p.stderr)
            execution = next(Path(root).iterdir())/'execution'
            self.assertTrue((execution/'failure.json').exists())
            self.assertFalse((execution/'time.json').exists())

    def test_timeout_during_telemetry(self):
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import time_v
        real_telemetry = time_v.telemetry
        def slow_telemetry(pid):
            sample = real_telemetry(pid)
            time.sleep(0.3)
            return sample
        with tempfile.TemporaryDirectory() as root, mock.patch.object(time_v, 'telemetry', slow_telemetry):
            execution = Path(root)/'execution'
            with self.assertRaises(time_v.CommandTimeoutError):
                time_v.run([sys.executable, '-c', 'pass'], execution, timeout_seconds=0.2)
            self.assertFalse((execution/'time.json').exists())
            self.assertTrue((execution/'failure.json').exists())

    def test_success_with_timeout(self):
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import time_v
        with tempfile.TemporaryDirectory() as root:
            receipt = time_v.run([sys.executable, '-c', 'pass'], Path(root)/'execution', timeout_seconds=5)
            self.assertEqual(receipt['exit_code'], 0)

    def test_short_success_with_timeout(self):
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import time_v
        with tempfile.TemporaryDirectory() as root:
            execution = Path(root)/'execution'
            receipt = time_v.run([sys.executable, '-c',
                                  'import time; time.sleep(.03); print("completed")'],
                                 execution, timeout_seconds=0.2)
            self.assertEqual(receipt['exit_code'], 0)
            self.assertIn('completed', (execution/'stdout.txt').read_text())
            self.assertFalse((execution/'failure.json').exists())

    def test_completion_after_deadline_is_timeout(self):
        sys.path.insert(0, str(Path(__file__).resolve().parent))
        import time_v
        real_sleep = time.sleep
        def delayed_poll(delay):
            real_sleep(max(delay, 0.2))
        with tempfile.TemporaryDirectory() as root, mock.patch.object(time_v.time, 'sleep', delayed_poll):
            execution = Path(root)/'execution'
            with self.assertRaises(time_v.CommandTimeoutError):
                time_v.run([sys.executable, '-c', 'import time; time.sleep(.1)'],
                           execution, timeout_seconds=0.15)
            self.assertFalse((execution/'time.json').exists())
            self.assertTrue((execution/'failure.json').exists())

    def test_success_with_helper(self):
        self.scenario('success')

    def test_nonzero_with_helper(self):
        self.scenario('nonzero')

    def test_sigint(self):
        self.scenario('sigint')

    def test_capture(self):
        self.scenario('capture')

    def test_telemetry(self):
        self.scenario('telemetry')

    def test_keyboard(self):
        self.scenario('keyboard')

    def test_early_exit(self):
        self.scenario('early')

    def test_already_reaped(self):
        self.scenario('reaped')

    def test_concurrent_independent_runs(self):
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            jobs = [pool.submit(self.scenario, mode) for mode in ('sigint', 'capture')]
            for job in jobs:
                job.result()


if __name__ == '__main__':
    if sys.argv[1:2] == ['--driver']:
        driver(sys.argv[2], sys.argv[3])
    else:
        unittest.main()

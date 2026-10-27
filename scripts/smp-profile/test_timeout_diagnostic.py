"""Regression coverage for timeout diagnostics and owned process cleanup."""
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import time_v


class TimeoutDiagnosticTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.folder = self.root / 'run'
        self.env = time_v.environment()

    def test_failure_write_preserves_timeout_and_reaps_group(self):
        # The parent reaps its child on TERM before exiting; the runner reaps the parent.
        code = '''import os, signal, subprocess, sys, time
child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(30)'])
def stop(*args):
    child.wait(timeout=3)
    sys.exit(0)
signal.signal(signal.SIGTERM, stop)
print(os.getpid(), child.pid, flush=True)
print('child stderr', file=sys.stderr, flush=True)
time.sleep(30)
'''
        created = []
        original_type = time_v.CommandTimeoutError

        class TrackedTimeout(original_type):
            def __init__(self, *args):
                super().__init__(*args)
                created.append(self)

        real_write = Path.write_text
        failure = OSError('injected failure.json write error')

        def write(path, *args, **kwargs):
            if path.name == 'failure.json':
                raise failure
            return real_write(path, *args, **kwargs)

        with mock.patch.object(time_v, 'CommandTimeoutError', TrackedTimeout), \
             mock.patch.object(Path, 'write_text', write):
            with self.assertRaises(TrackedTimeout) as caught:
                time_v.run([sys.executable, '-c', code], self.folder,
                           env=self.env, timeout_seconds=0.5)
        self.assertIs(caught.exception, created[0])
        self.assertIn(str(failure), ' '.join(getattr(caught.exception, '__notes__', [])))
        parent, child = map(int, (self.folder / 'stdout.txt').read_text().split())
        self.assertEqual((self.folder / 'stderr.txt').read_text(), 'child stderr\n')
        for pid in (parent, child):
            with self.assertRaises(ProcessLookupError):
                os.kill(pid, 0)
        with self.assertRaises(ChildProcessError):
            os.waitpid(parent, os.WNOHANG)
        self.assertFalse((self.folder / 'time.json').exists())

    def test_ordinary_timeout(self):
        with self.assertRaises(time_v.CommandTimeoutError):
            time_v.run([sys.executable, '-c', 'import time; time.sleep(30)'],
                       self.folder, env=self.env, timeout_seconds=0.2)
        self.assertIn('wall timeout', json.loads((self.folder / 'failure.json').read_text())['error'])
        self.assertFalse((self.folder / 'time.json').exists())

    def test_success(self):
        receipt = time_v.run(
            [sys.executable, '-c', 'print("ok")'], self.folder, env=self.env)
        self.assertEqual(receipt['exit_code'], 0)
        self.assertEqual((self.folder / 'stdout.txt').read_text(), 'ok\n')
        self.assertTrue((self.folder / 'time.json').exists())

    def test_telemetry_error_is_not_hidden(self):
        failure = OSError('telemetry failure')
        with mock.patch.object(time_v, 'telemetry', side_effect=failure):
            with self.assertRaises(OSError) as caught:
                time_v.run([sys.executable, '-c', 'import time; time.sleep(30)'],
                           self.folder, env=self.env)
        self.assertIs(caught.exception, failure)
        self.assertFalse((self.folder / 'time.json').exists())

    def test_success_cleanup_error_is_not_hidden(self):
        failure = OSError('cleanup failure')
        real_stop = time_v.stop_owned_group
        def stop(proc):
            real_stop(proc)
            raise failure
        with mock.patch.object(time_v, 'stop_owned_group', side_effect=stop):
            with self.assertRaises(OSError) as caught:
                time_v.run([sys.executable, '-c', 'pass'], self.folder, env=self.env)
        self.assertIs(caught.exception, failure)
        self.assertFalse((self.folder / 'time.json').exists())


if __name__ == '__main__':
    unittest.main()

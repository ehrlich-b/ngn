import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class MeasurementDiagnosticTest(unittest.TestCase):
    def test_instrumentation_failures(self):
        for operation, fail_at, stage in [('capture_resource_evidence', 1, 'capture_resource_evidence'),
                                           ('telemetry', 1, 'telemetry'),
                                           ('telemetry', 2, 'telemetry')]:
            with self.subTest(operation=operation, fail_at=fail_at), tempfile.TemporaryDirectory() as temp:
                folder = Path(temp) / 'run'
                failure = RuntimeError('private command and environment payload')
                original = getattr(time_v, operation)
                calls = 0
                pids = []
                spawn = subprocess.Popen

                def tracked_spawn(*args, **kwargs):
                    proc = spawn(*args, **kwargs)
                    pids.append(proc.pid)
                    return proc

                def inject(*args):
                    nonlocal calls
                    calls += 1
                    if calls == fail_at:
                        raise failure
                    return original(*args)

                with mock.patch.object(time_v.subprocess, 'Popen', side_effect=tracked_spawn), \
                     mock.patch.object(time_v, operation, side_effect=inject):
                    with self.assertRaises(RuntimeError) as raised:
                        time_v.run([sys.executable, '-c', 'import time; time.sleep(.4)'], folder)
                self.assertIs(raised.exception, failure)
                with self.assertRaises(ProcessLookupError):
                    os.killpg(pids[0], 0)
                data = (folder / 'failure.json').read_text()
                self.assertLess(len(data), 512)
                self.assertEqual(json.loads(data), {'stage': stage, 'exception_class': 'RuntimeError'})
                self.assertNotIn('private', data)
                self.assertFalse((folder / 'time.json').exists())
                self.assertFalse((folder / 'telemetry.json').exists())

    def test_diagnostic_write_failure_is_secondary(self):
        failure = RuntimeError('primary')
        write_text = Path.write_text
        def write(path, *args, **kwargs):
            if path.name == 'failure.json':
                raise OSError('write failed')
            return write_text(path, *args, **kwargs)
        with tempfile.TemporaryDirectory() as temp, \
             mock.patch.object(time_v, 'capture_resource_evidence', side_effect=failure), \
             mock.patch.object(Path, 'write_text', write):
            with self.assertRaises(RuntimeError) as raised:
                time_v.run(
                    [sys.executable, '-c', 'import time; time.sleep(30)'], Path(temp) / 'run')
            self.assertIs(raised.exception, failure)
            self.assertTrue(any('Could not write failure.json' in note for note in failure.__notes__))

    def test_instrumentation_timeout_is_not_command_timeout(self):
        failure = time_v.CommandTimeoutError('private payload')
        with tempfile.TemporaryDirectory() as temp, \
             mock.patch.object(time_v, 'telemetry', side_effect=failure):
            folder = Path(temp) / 'run'
            with self.assertRaises(time_v.CommandTimeoutError) as raised:
                time_v.run([sys.executable, '-c', 'import time; time.sleep(30)'], folder)
            self.assertIs(raised.exception, failure)
            self.assertEqual(json.loads((folder / 'failure.json').read_text()),
                             {'stage': 'telemetry', 'exception_class': 'CommandTimeoutError'})

    def test_final_telemetry_failure(self):
        failure = RuntimeError('private payload')
        stopped = False
        real_stop = time_v.stop_owned_group
        real_telemetry = time_v.telemetry
        def stop(proc):
            nonlocal stopped
            real_stop(proc)
            stopped = True
        def sample(pid):
            if stopped:
                raise failure
            return real_telemetry(pid)
        with tempfile.TemporaryDirectory() as temp, \
             mock.patch.object(time_v, 'stop_owned_group', side_effect=stop), \
             mock.patch.object(time_v, 'telemetry', side_effect=sample):
            folder = Path(temp) / 'run'
            with self.assertRaises(RuntimeError) as raised:
                time_v.run([sys.executable, '-c', 'pass'], folder)
            self.assertIs(raised.exception, failure)
            self.assertEqual(json.loads((folder / 'failure.json').read_text()),
                             {'stage': 'telemetry', 'exception_class': 'RuntimeError'})
            self.assertFalse((folder / 'time.json').exists())

    def test_success_and_command_exit(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp) / 'success'
            receipt = time_v.run([sys.executable, '-c', 'pass'], folder)
            self.assertEqual(receipt['exit_code'], 0)
            self.assertFalse((folder / 'failure.json').exists())
            folder = Path(temp) / 'exit'
            with self.assertRaisesRegex(RuntimeError, 'exit 7'):
                time_v.run([sys.executable, '-c', 'raise SystemExit(7)'], folder)
            self.assertEqual(json.loads((folder / 'time.json').read_text())['exit_code'], 7)
            self.assertFalse((folder / 'failure.json').exists())

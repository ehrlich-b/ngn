"""Exercise host telemetry failures without replacing real child processes."""
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class UnavailableHostTelemetryTests(unittest.TestCase):
    def exercise(self, first, last, exit_code):
        cleaned = False
        cleanup = time_v.stop_owned_group

        def stop(proc):
            nonlocal cleaned
            cleanup(proc)
            cleaned = True

        def sample(_pid):
            return last if cleaned else first

        code = ("import sys; print('child stdout', flush=True); "
                "print('child stderr', file=sys.stderr, flush=True); "
                f"sys.exit({exit_code})")
        with tempfile.TemporaryDirectory() as root, \
                mock.patch.object(time_v, 'telemetry', side_effect=sample), \
                mock.patch.object(time_v, 'stop_owned_group', side_effect=stop):
            folder = Path(root) / 'execution'
            if exit_code:
                with self.assertRaisesRegex(RuntimeError, f'exit {exit_code}'):
                    time_v.run([sys.executable, '-c', code], folder)
            else:
                time_v.run([sys.executable, '-c', code], folder)
            self.assertTrue(cleaned)
            self.assertEqual((folder / 'stdout.txt').read_text(), 'child stdout\n')
            self.assertEqual((folder / 'stderr.txt').read_text(), 'child stderr\n')
            receipt = json.loads((folder / 'time.json').read_text())
            samples = json.loads((folder / 'telemetry.json').read_text())
            self.assertEqual((samples[0], samples[-1]), (first, last))
            self.assertEqual(receipt['exit_code'], exit_code)
            for key in ('user_seconds', 'system_seconds', 'peak_rss_kib',
                        'voluntary_context_switches', 'resource_evidence'):
                self.assertIsNotNone(receipt[key])
            return receipt

    def test_unavailable_and_malformed_at_each_endpoint(self):
        valid = {'/proc/stat': 'cpu 100 20 30 40 50 60 70 80 0 0\n'}
        cases = [{}, {'/proc/stat': {'unavailable': 'permission denied'}},
                 {'/proc/stat': 'cpu 1 2 3\n'},
                 {'/proc/stat': 'cpu 1 2 broken 4 5 6 7 8\n'},
                 {'/proc/stat': 'cpu 1 2 -3 4 5 6 7 8\n'}]
        for invalid in cases:
            for endpoint in ('start', 'end'):
                for code in (0, 7):
                    with self.subTest(invalid=invalid, endpoint=endpoint, code=code):
                        first, last = (invalid, valid) if endpoint == 'start' else (valid, invalid)
                        receipt = self.exercise(first, last, code)
                        self.assertIsNone(receipt['host_busy_cpu_seconds'])
                        self.assertIsNone(receipt['non_owned_busy_cpu_seconds_estimate'])
                        self.assertIn(endpoint, receipt['host_busy_cpu_seconds_unavailable_reason'])

    def test_valid_counters_and_decrease(self):
        first = {'/proc/stat': 'cpu 100 20 30 40 50 60 70 80 0 0\n'}
        last = {'/proc/stat': 'cpu 110 22 33 44 55 66 77 88 0 0\n'}
        receipt = self.exercise(first, last, 0)
        busy = 36 / os.sysconf('SC_CLK_TCK')
        self.assertAlmostEqual(receipt['host_busy_cpu_seconds'], busy)
        self.assertAlmostEqual(receipt['non_owned_busy_cpu_seconds_estimate'],
                               max(0, busy - receipt['user_seconds'] - receipt['system_seconds']))
        self.assertNotIn('host_busy_cpu_seconds_unavailable_reason', receipt)
        receipt = self.exercise(last, first, 7)
        self.assertIsNone(receipt['host_busy_cpu_seconds'])
        self.assertIsNone(receipt['non_owned_busy_cpu_seconds_estimate'])
        self.assertIn('decreased', receipt['host_busy_cpu_seconds_unavailable_reason'])


if __name__ == '__main__':
    unittest.main()

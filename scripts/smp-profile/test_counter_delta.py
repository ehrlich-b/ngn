import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

import time_v


def cpu_line(values):
    return {"/proc/stat": "cpu " + " ".join(map(str, values)) + "\n"}


class CounterDeltaTests(unittest.TestCase):
    def receipt(self, first, last):
        samples = [cpu_line(first), cpu_line(last)]
        real_wait4 = time_v.os.wait4

        def wait4(*args):
            pid, status, usage = real_wait4(*args)
            if pid:
                usage = SimpleNamespace(ru_utime=0.02, ru_stime=0.01,
                                        ru_maxrss=usage.ru_maxrss,
                                        ru_nvcsw=usage.ru_nvcsw,
                                        ru_nivcsw=usage.ru_nivcsw)
            return pid, status, usage

        with tempfile.TemporaryDirectory() as tmp, \
             mock.patch.object(time_v, "telemetry", side_effect=lambda pid: samples.pop(0) if len(samples) > 1 else samples[0]), \
             mock.patch.object(time_v.os, "wait4", side_effect=wait4), \
             mock.patch.object(time_v.os, "sysconf", return_value=100):
            return time_v.run(["/bin/true"], Path(tmp) / "run")

    def test_busy_deltas(self):
        big = 2**60
        cases = [
            ([big] * 10, [big + 1, big + 2, big + 3, big + 999, big + 500,
                           big + 4, big + 5, big + 6, big + 1000, big + 2000], 0.21, 0.18),
            ([big] * 10, [big] * 10, 0.0, 0.0),
            ([10, 20, 30, 40, 50, 60, 70, 80, 90, 100],
             [11, 22, 33, 1, 2, 64, 75, 86, 1, 2], 0.21, 0.18),
        ]
        for first, last, busy, external in cases:
            with self.subTest(first=first, last=last):
                receipt = self.receipt(first, last)
                self.assertEqual(receipt["host_busy_cpu_seconds"], busy)
                self.assertAlmostEqual(receipt["non_owned_busy_cpu_seconds_estimate"], external)
                self.assertNotIn("host_busy_cpu_seconds_unavailable_reason", receipt)

    def test_decreased_busy_counter_refused(self):
        for index in (0, 1, 2, 5, 6, 7):
            with self.subTest(index=index):
                first = [100] * 10
                last = first.copy()
                last[index] -= 1
                receipt = self.receipt(first, last)
                self.assertIsNone(receipt["host_busy_cpu_seconds"])
                self.assertIsNone(receipt["non_owned_busy_cpu_seconds_estimate"])
                self.assertEqual(receipt["host_busy_cpu_seconds_unavailable_reason"],
                                 "host CPU counters decreased between start and end")

    def test_unavailable_diagnostics(self):
        with tempfile.TemporaryDirectory() as tmp, \
             mock.patch.object(time_v, "telemetry", return_value={"/proc/stat": {"unavailable": "missing"}}):
            receipt = time_v.run(["/bin/true"], Path(tmp) / "run")
        self.assertIsNone(receipt["host_busy_cpu_seconds"])
        self.assertIsNone(receipt["non_owned_busy_cpu_seconds_estimate"])
        self.assertEqual(receipt["host_busy_cpu_seconds_unavailable_reason"],
                         "start /proc/stat unavailable: {'unavailable': 'missing'}")


if __name__ == "__main__":
    unittest.main()

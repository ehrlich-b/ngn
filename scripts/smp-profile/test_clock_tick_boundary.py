"""Exercise host accounting with deterministic counters and tiny children."""
import json
from itertools import chain, repeat
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import time_v


class ClockTickBoundaryTests(unittest.TestCase):
    def run_child(self, tick, exit_code=0):
        samples = chain(
            [{"/proc/stat": "cpu 100 0 100 100 0 0 0 0\n"}],
            repeat({"/proc/stat": "cpu 200 0 200 200 0 0 0 0\n"}),
        )
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp) / "execution"
            kwargs = {"side_effect": tick} if isinstance(tick, Exception) else {"return_value": tick}
            with patch.object(time_v, "telemetry", side_effect=lambda pid: next(samples)), \
                 patch.object(time_v.os, "sysconf", **kwargs), \
                 patch.object(time_v, "capture_resource_evidence", return_value={"fixture": True}):
                if exit_code:
                    with self.assertRaisesRegex(RuntimeError, f"exit {exit_code}"):
                        time_v.run([sys.executable, "-c", f"import sys; sys.exit({exit_code})"], folder)
                else:
                    time_v.run([sys.executable, "-c", "print('ok')"], folder)
            receipt = json.loads((folder / "time.json").read_text())
            self.assertEqual(receipt["exit_code"], exit_code)
            self.assertEqual(receipt["resource_evidence"]["fixture"], True)
            self.assertGreaterEqual(receipt["user_seconds"], 0)
            self.assertTrue((folder / "telemetry.json").is_file())
            return receipt

    def test_invalid_ticks_are_unavailable(self):
        for tick in (0, -1, True, False, float("nan"), float("inf"),
                     float("-inf"), "100", None, OSError("x" * 2000),
                     ValueError("x" * 2000)):
            with self.subTest(tick=tick):
                receipt = self.run_child(tick)
                self.assertIsNone(receipt["host_busy_cpu_seconds"])
                self.assertIsNone(receipt["non_owned_busy_cpu_seconds_estimate"])
                self.assertTrue(receipt["host_busy_cpu_seconds_unavailable_reason"])
                self.assertLessEqual(len(receipt["host_busy_cpu_seconds_unavailable_reason"]), 128)

    def test_valid_ticks(self):
        receipt = self.run_child(100)
        self.assertEqual(receipt["host_busy_cpu_seconds"], 2.0)
        self.assertIsNotNone(receipt["non_owned_busy_cpu_seconds_estimate"])
        self.assertNotIn("host_busy_cpu_seconds_unavailable_reason", receipt)

    def test_failed_command_still_writes_receipt(self):
        receipt = self.run_child(0, exit_code=7)
        self.assertIsNone(receipt["host_busy_cpu_seconds"])
        receipt = self.run_child(100, exit_code=7)
        self.assertEqual(receipt["host_busy_cpu_seconds"], 2.0)


if __name__ == "__main__":
    unittest.main()

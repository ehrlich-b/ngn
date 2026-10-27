import json
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

import time_v


class TimeoutValidationTests(unittest.TestCase):
    def test_invalid_values_have_no_filesystem_or_process_effects(self):
        invalid = [True, False, "1", b"1", [], {}, object(), 0, -1, 0.0,
                   -0.5, float("nan"), float("inf"), -float("inf")]
        with tempfile.TemporaryDirectory() as root:
            for index, value in enumerate(invalid):
                with self.subTest(value=value):
                    folder = Path(root) / str(index)
                    with mock.patch.object(subprocess, "Popen") as popen:
                        with self.assertRaisesRegex(ValueError, "timeout_seconds.*positive finite"):
                            time_v.run([sys.executable, "-c", "pass"], folder,
                                       timeout_seconds=value)
                        popen.assert_not_called()
                    self.assertFalse(folder.exists())

    def test_none_and_positive_int_float_run_successfully(self):
        with tempfile.TemporaryDirectory() as root:
            for index, value in enumerate((None, 2, 2.0, 10**400)):
                with self.subTest(value=value):
                    folder = Path(root) / str(index)
                    receipt = time_v.run([sys.executable, "-c", "print('ok')"], folder,
                                         timeout_seconds=value)
                    self.assertEqual(receipt["exit_code"], 0)
                    self.assertEqual((folder / "stdout.txt").read_text().strip(), "ok")
                    self.assertEqual(json.loads((folder / "time.json").read_text())["exit_code"], 0)

    def test_int_and_float_timeouts_stop_local_process(self):
        with tempfile.TemporaryDirectory() as root:
            for index, value in enumerate((1, 0.2)):
                with self.subTest(value=value):
                    folder = Path(root) / str(index)
                    start = time.monotonic()
                    with self.assertRaises(time_v.CommandTimeoutError):
                        time_v.run([sys.executable, "-c", "import time; time.sleep(30)"], folder,
                                   timeout_seconds=value)
                    self.assertLess(time.monotonic() - start, 5)
                    failure = json.loads((folder / "failure.json").read_text())
                    self.assertEqual(failure["timeout_seconds"], value)


if __name__ == "__main__":
    unittest.main()

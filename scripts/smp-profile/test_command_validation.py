import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class CommandValidationTests(unittest.TestCase):
    def test_malformed_commands_have_no_side_effects(self):
        cases = ["python", b"python", [], (), [""], [None], ["python", 1],
                 ["python", b"arg"], None, iter(["python"])]
        for command in cases:
            with self.subTest(command=command), tempfile.TemporaryDirectory() as tmp:
                folder = Path(tmp) / "output"
                with mock.patch.object(time_v.subprocess, "Popen") as popen:
                    with self.assertRaisesRegex(ValueError, "command"):
                        time_v.run(command, folder)
                    popen.assert_not_called()
                self.assertFalse(folder.exists())

    def test_valid_command_preserves_arguments(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp) / "output"
            command = [sys.executable, "-c",
                       "import json, sys; print(json.dumps(sys.argv[1:]))",
                       "", "two words", " leading and trailing "]
            receipt = time_v.run(command, folder)
            self.assertEqual(receipt["exit_code"], 0)
            self.assertEqual(receipt["command"], command)
            self.assertEqual(json.loads((folder / "stdout.txt").read_text()), command[3:])

    def test_cli_rejects_empty_executable_before_creating_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp) / "output"
            result = subprocess.run([sys.executable, str(Path(time_v.__file__)),
                                     "--output", str(folder), "--", ""],
                                    capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("executable", result.stderr)
            self.assertFalse(folder.exists())


if __name__ == "__main__":
    unittest.main()

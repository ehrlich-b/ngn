import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class CommandNulTests(unittest.TestCase):
    def test_nul_in_each_position_has_no_side_effects(self):
        for position in range(3):
            for value in ("\0", "before\0after"):
                with self.subTest(position=position, value=value), tempfile.TemporaryDirectory() as tmp:
                    command = [sys.executable, "middle", "last"]
                    command[position] = value
                    folder = Path(tmp) / "missing" / "output"
                    with mock.patch.object(time_v.subprocess, "Popen") as popen:
                        with self.assertRaisesRegex(ValueError, "NUL"):
                            time_v.run(command, folder)
                        popen.assert_not_called()
                    self.assertFalse(folder.parent.exists())

    def test_real_child_receives_valid_arguments(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp) / "output"
            arguments = ["", " two words ", "雪☃"]
            command = (sys.executable, "-c",
                       "import json, sys; print(json.dumps(sys.argv[1:]))", *arguments)
            receipt = time_v.run(command, folder)
            self.assertEqual(receipt["exit_code"], 0)
            self.assertEqual(receipt["command"], command)
            self.assertEqual(json.loads((folder / "stdout.txt").read_text()), arguments)


if __name__ == "__main__":
    unittest.main()

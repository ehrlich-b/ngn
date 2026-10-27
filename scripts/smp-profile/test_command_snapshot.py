from collections import UserList
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class CommandSnapshotTest(unittest.TestCase):
    def test_caller_mutation_after_spawn_does_not_change_receipt(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp) / "run"
            marker = Path(temp) / "executed"
            command = [sys.executable, "-c", "import pathlib, sys; pathlib.Path(sys.argv[1]).write_text('ran')", str(marker)]
            original = command.copy()
            original_telemetry = time_v.telemetry

            def mutate_after_spawn(pid=None):
                command[:] = ["caller", "changed"]
                return original_telemetry(pid)

            with mock.patch.object(time_v, "telemetry", side_effect=mutate_after_spawn):
                receipt = time_v.run(command, folder, env=os.environ.copy(), timeout_seconds=5)

            self.assertEqual(marker.read_text(), "ran")
            self.assertEqual(command, ["caller", "changed"])
            self.assertEqual(receipt["command"], original)
            self.assertEqual(json.loads((folder / "time.json").read_text())["command"], original)

    def test_mutable_sequence_is_copied_before_validation(self):
        with tempfile.TemporaryDirectory() as temp:
            command = UserList([sys.executable, "-c", "pass"])
            with mock.patch.object(time_v, "validate_command", wraps=time_v.validate_command) as validate:
                receipt = time_v.run(command, Path(temp) / "run", timeout_seconds=5)
            snapshot = validate.call_args.args[0]
            self.assertIs(type(snapshot), list)
            self.assertIsNot(snapshot, command)
            self.assertEqual(receipt["command"], list(command))

    def test_invalid_copied_commands_fail_before_folder_or_spawn(self):
        with tempfile.TemporaryDirectory() as temp:
            for index, command in enumerate(([], [""], ["ok", 1], ["ok", "bad\0arg"])):
                with self.subTest(command=command):
                    folder = Path(temp) / str(index)
                    with mock.patch.object(time_v.subprocess, "Popen") as spawn:
                        with self.assertRaises(ValueError):
                            time_v.run(command, folder)
                    self.assertFalse(folder.exists())
                    spawn.assert_not_called()

    def test_tuple_input_retains_return_type_and_json_snapshot(self):
        with tempfile.TemporaryDirectory() as temp:
            command = (sys.executable, "-c", "pass")
            folder = Path(temp) / "run"
            receipt = time_v.run(command, folder, timeout_seconds=5)
            self.assertEqual(receipt["command"], command)
            self.assertEqual(json.loads((folder / "time.json").read_text())["command"], list(command))


if __name__ == "__main__":
    unittest.main()

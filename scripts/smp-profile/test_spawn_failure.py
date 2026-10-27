import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class SpawnFailureTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.folder = self.root / "execution"

    def assert_spawn_artifacts(self):
        self.assertTrue((self.folder / "stdout.txt").is_file())
        self.assertTrue((self.folder / "stderr.txt").is_file())
        self.assertFalse((self.folder / "time.json").exists())

    def test_nonexistent_cwd(self):
        with mock.patch.object(time_v, "stop_owned_group") as stop:
            with self.assertRaises(FileNotFoundError) as caught:
                time_v.run([sys.executable, "-c", "pass"], self.folder,
                           cwd=self.root / "missing")
        stop.assert_not_called()
        self.assert_spawn_artifacts()
        failure = json.loads((self.folder / "failure.json").read_text())
        self.assertEqual(failure, {"stage": "spawn", "exception_class": "FileNotFoundError",
                                   "message": str(caught.exception)})

    def test_injected_spawn_error_preserves_identity_and_bounds_message(self):
        error = OSError("x" * 2000)
        with mock.patch.object(time_v.subprocess, "Popen", side_effect=error), \
             mock.patch.object(time_v, "stop_owned_group") as stop:
            with self.assertRaises(OSError) as caught:
                time_v.run(["unused"], self.folder)
        self.assertIs(caught.exception, error)
        stop.assert_not_called()
        self.assert_spawn_artifacts()
        failure = json.loads((self.folder / "failure.json").read_text())
        self.assertEqual(failure, {"stage": "spawn", "exception_class": "OSError",
                                   "message": str(error)[:1024]})

    def test_diagnostic_write_failure_preserves_spawn_error(self):
        error = PermissionError("launch denied")
        write_error = OSError("diagnostic unavailable")
        original_write = Path.write_text

        def write(path, *args, **kwargs):
            if path.name == "failure.json":
                raise write_error
            return original_write(path, *args, **kwargs)

        with mock.patch.object(time_v.subprocess, "Popen", side_effect=error), \
             mock.patch.object(Path, "write_text", write), \
             mock.patch.object(time_v, "stop_owned_group") as stop:
            with self.assertRaises(PermissionError) as caught:
                time_v.run(["unused"], self.folder)
        self.assertIs(caught.exception, error)
        self.assertTrue(any("diagnostic unavailable" in note for note in error.__notes__))
        stop.assert_not_called()
        self.assert_spawn_artifacts()
        self.assertFalse((self.folder / "failure.json").exists())

    def test_success_control(self):
        receipt = time_v.run([sys.executable, "-c", "print('ok')"], self.folder)
        self.assertEqual(receipt["exit_code"], 0)
        self.assertEqual((self.folder / "stdout.txt").read_text(), "ok\n")
        self.assertEqual(json.loads((self.folder / "time.json").read_text()), receipt)
        self.assertFalse((self.folder / "failure.json").exists())


if __name__ == "__main__":
    unittest.main()

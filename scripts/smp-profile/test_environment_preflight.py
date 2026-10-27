"""Explicit subprocess environments are checked before any side effects."""
from collections.abc import Mapping
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import time_v


class CountingMapping(Mapping):
    def __init__(self, data):
        self.data = data
        self.iterations = 0

    def __iter__(self):
        self.iterations += 1
        return iter(self.data)

    def __len__(self):
        return len(self.data)

    def __getitem__(self, key):
        return self.data[key]


class EnvironmentPreflightTests(unittest.TestCase):
    def test_invalid_environments_have_no_side_effects(self):
        cases = [[], [("A", "B")], "A=B", 42,
                 {1: "value"}, {"": "value"}, {"A=B": "value"},
                 {"A\0B": "value"}, {"A": 1}, {"A": None},
                 {"A": "value\0tail"}]
        with tempfile.TemporaryDirectory() as root:
            for index, env in enumerate(cases):
                with self.subTest(index=index):
                    folder = Path(root) / str(index) / "execution"
                    with mock.patch.object(subprocess, "Popen") as spawn:
                        with self.assertRaises(ValueError) as error:
                            time_v.run([sys.executable, "-c", "pass"], folder, env=env)
                    self.assertFalse(folder.parent.exists())
                    spawn.assert_not_called()
                    self.assertNotIn("value", str(error.exception).replace("key or value", ""))

    def test_explicit_mapping_copied_once_and_used_by_child(self):
        original = {"SYNTHETIC_UNICODE": "雪☃", "SYNTHETIC_EMPTY": "", "TMPDIR": "ignored"}
        env = CountingMapping(original.copy())
        code = ("import json, os; print(json.dumps([os.environ['SYNTHETIC_UNICODE'], "
                "os.environ['SYNTHETIC_EMPTY'], os.environ['TMPDIR']]))")
        with tempfile.TemporaryDirectory() as root:
            folder = Path(root) / "execution"
            time_v.run([sys.executable, "-c", code], folder, env=env)
            values = json.loads((folder / "stdout.txt").read_text())
            self.assertEqual(values, ["雪☃", "", str(folder.resolve() / "tmp")])
            self.assertTrue((folder / "tmp").is_dir())
        self.assertEqual(env.iterations, 1)
        self.assertEqual(env.data, original)

    def test_empty_mapping_does_not_inherit_ambient_environment(self):
        code = "import os; print('SYNTHETIC_AMBIENT' in os.environ)"
        with tempfile.TemporaryDirectory() as root, mock.patch.dict(
                'os.environ', {'SYNTHETIC_AMBIENT': 'synthetic'}, clear=False):
            folder = Path(root) / 'execution'
            time_v.run([sys.executable, '-c', code], folder, env={})
            self.assertEqual((folder / 'stdout.txt').read_text().strip(), 'False')


if __name__ == '__main__':
    unittest.main()

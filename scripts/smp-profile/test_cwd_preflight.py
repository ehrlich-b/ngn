import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import time_v


class CwdPreflightTests(unittest.TestCase):
    def test_invalid_cwd_does_not_create_evidence_or_spawn(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            regular_file = root / 'file'
            regular_file.write_text('file')
            for cwd in (42, 'bad\0path', b'bad\0path', root / 'missing', regular_file):
                with self.subTest(cwd=cwd):
                    folder = root / 'evidence'
                    with patch.object(time_v.subprocess, 'Popen') as popen:
                        with self.assertRaisesRegex(ValueError, 'invalid cwd') as caught:
                            time_v.run([sys.executable, '-c', 'pass'], folder, cwd=cwd)
                    self.assertIsNotNone(caught.exception.__cause__)
                    self.assertFalse(folder.exists())
                    popen.assert_not_called()

    def test_valid_cwd_types_and_inherited_cwd(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            directory = root / 'directory'
            directory.mkdir()
            link = root / 'link'
            link.symlink_to(directory, target_is_directory=True)
            for index, cwd in enumerate((str(directory), directory, os.fsencode(directory), link, None)):
                with self.subTest(cwd=cwd):
                    folder = root / f'evidence-{index}'
                    command = [sys.executable, '-c', 'import os; print(os.getcwd())']
                    original = command.copy()
                    env = dict(os.environ)
                    original_env = env.copy()
                    time_v.run(command, folder, env=env, cwd=cwd)
                    expected = os.getcwd() if cwd is None else str(directory.resolve())
                    self.assertEqual((folder / 'stdout.txt').read_text().strip(), expected)
                    self.assertEqual(command, original)
                    self.assertEqual(env, original_env)


if __name__ == '__main__':
    unittest.main()

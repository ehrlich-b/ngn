"""Exercise the replay dispatcher without running a suite on simulated hosts."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('qcapture_check', Path(__file__).with_name('check.py'))
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class HostPolicyTest(unittest.TestCase):
    def test_short_suite_blocks_only_exact_forge(self):
        args = ['go', 'test', '-p', '2', '-short', '-count=1', './...']
        for host in ['forge', 'forge2-vm', 'forge2', 'rig', 'forge-backup']:
            with self.subTest(host=host), patch.object(check.socket, 'gethostname', return_value=host), patch.object(check, 'execute') as execute:
                if host == 'forge':
                    with self.assertRaises(SystemExit) as stopped:
                        check.execute_on_host(args)
                    self.assertEqual(stopped.exception.code, 77)
                    execute.assert_not_called()
                else:
                    check.execute_on_host(args)
                    execute.assert_called_once_with(args)

    def test_vet_is_allowed_on_forge(self):
        args = ['go', 'vet', '-p', '2', './...']
        with patch.object(check.socket, 'gethostname', return_value='forge'), patch.object(check, 'execute') as execute:
            check.execute_on_host(args)
            execute.assert_called_once_with(args)

#!/usr/bin/env python3
"""Check exact-host short-gate dispatch; Go execution is covered by the real gate."""
import contextlib
import io
from pathlib import Path
import runpy
import sys
from types import SimpleNamespace
import unittest
from unittest import mock


class ShortGateHostTest(unittest.TestCase):
    def test_exact_host_policy(self):
        command = ['nice', '-n', '10', 'go', 'test', '-p', '2',
                   '-short', '-count=1', './...']
        for host, expected in [('forge', 75), ('forge2-vm', 0), ('rig', 0)]:
            with self.subTest(host=host):
                uname = SimpleNamespace(nodename=host)
                usage = SimpleNamespace(ru_utime=0, ru_stime=0, ru_maxrss=0)
                output = io.StringIO()
                with mock.patch.object(sys, 'argv', ['run.py', 'short']), \
                     mock.patch('os.uname', return_value=uname), \
                     mock.patch('os.sched_getaffinity', return_value={2, 3, 4}, create=True), \
                     mock.patch('os.sched_setaffinity', create=True) as affinity, \
                     mock.patch.dict('os.environ', {'RESIDUAL_TEST_SECRET': 'synthetic'}), \
                     mock.patch('subprocess.Popen') as popen, \
                     mock.patch('os.wait4', return_value=(123, 0, usage)), \
                     contextlib.redirect_stdout(output), \
                     self.assertRaises(SystemExit) as exit_result:
                    runpy.run_path(str(Path(__file__).with_name('run.py')), run_name='__main__')
                self.assertEqual(exit_result.exception.code, expected)
                affinity.assert_called_once_with(0, [2, 3])
                if expected == 75:
                    popen.assert_not_called()
                    self.assertIn('BLOCKED on forge:', output.getvalue())
                else:
                    popen.assert_called_once()
                    self.assertEqual(popen.call_args.args[0], command)
                    self.assertEqual(popen.call_args.kwargs['env']['GOMAXPROCS'], '2')
                    for key in ['OMP_NUM_THREADS', 'OPENBLAS_NUM_THREADS', 'MKL_NUM_THREADS',
                                'NUMEXPR_NUM_THREADS', 'VECLIB_MAXIMUM_THREADS']:
                        self.assertEqual(popen.call_args.kwargs['env'][key], '2')
                    self.assertNotIn('RESIDUAL_TEST_SECRET', popen.call_args.kwargs['env'])
                    self.assertNotIn('BLOCKED', output.getvalue())


if __name__ == '__main__':
    unittest.main(verbosity=2)

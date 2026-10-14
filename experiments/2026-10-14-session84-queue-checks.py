"""Native dependency admission for the next ordered search gate."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

spec=importlib.util.spec_from_file_location('queue',Path(__file__).parents[1]/'scripts/campaign_queue.py')
queue=importlib.util.module_from_spec(spec)
spec.loader.exec_module(queue)


class Dependency(unittest.TestCase):
    def test_verdict_and_faults(self):
        with tempfile.TemporaryDirectory() as temporary:
            path=Path(temporary)/'result.json'
            dependency={'pid':123,'result':str(path),'verdicts':['CERTIFIED']}
            for verdict,faults,passes in [('CERTIFIED',[],True),('BISECT',[],False),
                                          ('UNCERTIFIED_CAP',[],False),('CERTIFIED',['flag-out'],False)]:
                with self.subTest(verdict=verdict,faults=faults):
                    path.write_text(json.dumps({'verdict':verdict,'faults':faults}))
                    with mock.patch.object(queue.os,'pidfd_open',side_effect=ProcessLookupError):
                        if passes:
                            queue.wait_predecessor(dependency)
                        else:
                            with self.assertRaises(RuntimeError):
                                queue.wait_predecessor(dependency)

    def test_waits_and_closes_descriptor(self):
        with tempfile.TemporaryDirectory() as temporary:
            path=Path(temporary)/'result.json'
            path.write_text(json.dumps({'verdict':'CERTIFIED','faults':[]}))
            read,write=os.pipe()
            try:
                with mock.patch.object(queue.os,'pidfd_open',return_value=read), \
                     mock.patch.object(queue.select,'select',return_value=([read],[],[])) as wait:
                    queue.wait_predecessor({'pid':123,'result':str(path),'verdicts':['CERTIFIED']})
                wait.assert_called_once_with([read],[],[])
                with self.assertRaises(OSError):
                    os.fstat(read)
            finally:
                os.close(write)


if __name__=='__main__':
    unittest.main()

"""Admission checks for the detached warm restart; no engines or games."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location('warm', Path(__file__).with_name('2026-10-14-session84-spsa.py'))
warm = importlib.util.module_from_spec(spec)
spec.loader.exec_module(warm)


class Admission(unittest.TestCase):
    def test_net_adoption_holds_but_clean_shelf_admits_warm_restart(self):
        for verdict, faults, starts in [('SHELVED', [], True), ('KEEP', [], False),
                                        ('SHELVED', ['flag-out'], False)]:
            with self.subTest(verdict=verdict, faults=faults), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                directory = root / 'warm'
                directory.mkdir()
                net = root / 'net-result.json'
                net.write_text(json.dumps({'verdict':verdict,'faults':faults}))
                nomination = dict(predecessor_pid=123, artifacts={}, argv=['mock-tuner'],
                                  engine_source='9ec5d31', net_sha256='fixture',
                                  predecessor_results=[dict(path=str(net),verdicts=['SHELVED'])])
                (directory / 'nomination.json').write_text(json.dumps(nomination))
                endpoint = {'iter':2000,'recipe':{'pairs':6,'single':False,'iterations':2000}}
                def launch(*args, **kwargs):
                    (directory / 'spsa.jsonl').write_text(json.dumps(endpoint)+'\n')
                    return mock.Mock(pid=1234,wait=mock.Mock(return_value=0))
                with mock.patch.object(warm,'R',root), mock.patch.object(warm,'D',directory), \
                     mock.patch.object(warm.os,'pidfd_open',side_effect=ProcessLookupError), \
                     mock.patch.object(warm.subprocess,'Popen',side_effect=launch) as child:
                    warm.main()
                self.assertEqual(child.call_count,int(starts))

    def test_predecessor_verdict_controls_all_games(self):
        for verdict, faults, starts in [('CERTIFIED', [], True), ('BISECT', [], False),
                                        ('UNCERTIFIED_CAP', [], False), ('CERTIFIED', ['flag-out'], False)]:
            with self.subTest(verdict=verdict, faults=faults), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                directory = root / 'session84/spsa-warm'
                directory.mkdir(parents=True)
                for lane, result in [('net-transfer', {'verdict':'TRANSFER_PASS','faults':[]}),
                                     ('provenance', {'verdict':verdict,'faults':faults})]:
                    path = root / 'session83' / lane
                    path.mkdir(parents=True)
                    (path / 'result.json').write_text(json.dumps(result))
                nomination = {'predecessor_pid':123, 'artifacts':{}, 'argv':['mock-tuner'],
                              'engine_source':'9ec5d31', 'net_sha256':'fixture'}
                (directory / 'nomination.json').write_text(json.dumps(nomination))
                endpoint = {'iter':2000, 'recipe':{'pairs':6, 'single':False, 'iterations':2000}}
                def launch(*args, **kwargs):
                    self.assertTrue((directory / 'manifest.json').exists())
                    (directory / 'spsa.jsonl').write_text(json.dumps(endpoint) + '\n')
                    return mock.Mock(pid=1234, wait=mock.Mock(return_value=0))
                with mock.patch.object(warm, 'R', root), mock.patch.object(warm, 'D', directory), \
                     mock.patch.object(warm.os, 'pidfd_open', side_effect=ProcessLookupError), \
                     mock.patch.object(warm.subprocess, 'Popen', side_effect=launch) as child:
                    warm.main()
                self.assertEqual(child.call_count, int(starts))
                result = json.loads((directory / 'result.json').read_text())
                self.assertEqual(result['state'], 'READY_FOR_VECTOR_GATE' if starts else 'HELD_PREDECESSOR')


if __name__ == '__main__':
    unittest.main()

"""Dry-run the long-job branches; never launch an engine or trainer."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock

ROOT = Path('/home/ehrli/ngn-data/night-20261007/session83')


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ControllerChecks(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.r = Path(self.temp.name)
        self.d = self.r / 'session83'
        self.d.mkdir()
        for filename in ('provenance-template.json', 'net-transfer-template.json'):
            (self.d / filename).write_bytes((ROOT / filename).read_bytes())
        (self.r / 'session82/net-final115').mkdir(parents=True)
        (self.r / 'session80-training').mkdir()
        (self.r / 'session80-training/result.json').write_text(json.dumps({'net_sha256': 'fixture-net-sha'}))

    def tearDown(self):
        self.temp.cleanup()

    def setup_net(self, verdict, faults=None):
        (self.r / 'session82/net-final115/result.json').write_text(
            json.dumps({'verdict': verdict, 'faults': faults or []}))

    def followup(self):
        module = load('followup', 'followup.py')
        module.R, module.D = self.r, self.d
        module.wait_pid = Mock()
        module.run = Mock(return_value={'verdict': 'CERTIFIED'})
        return module

    def test_shelf_launches_same_old_net_provenance_only(self):
        self.setup_net('SHELVED')
        module = self.followup()
        module.main()
        self.assertEqual(module.run.call_count, 1)
        job = module.run.call_args.args[0]
        self.assertEqual(job['class'], 'provenance')
        self.assertEqual(job['candidate_options'], job['base_options'])
        self.assertEqual((job['elo0'], job['elo1'], job['min_games'], job['max_games']), (-8, 0, 1000, 4000))
        self.assertEqual(job['candidate_source'], '9ec5d31')
        self.assertEqual(job['base_source'], '5b882e2')

    def test_winner_transfers_then_certifies_on_identical_winning_net(self):
        self.setup_net('KEEP')
        module = self.followup()
        module.run.side_effect = [{'verdict': 'TRANSFER_PASS'}, {'verdict': 'CERTIFIED'}]
        module.main()
        transfer, provenance = [call.args[0] for call in module.run.call_args_list]
        self.assertEqual(transfer['class'], 'net-transfer')
        self.assertEqual(transfer['candidate'], transfer['base'])
        self.assertNotEqual(transfer['candidate_options'][0], transfer['base_options'][0])
        self.assertEqual(provenance['candidate_options'], provenance['base_options'])
        self.assertEqual(provenance['net_sha256'], 'fixture-net-sha')
        self.assertEqual(provenance['artifacts'][str(self.r / 'session82/net-final115/candidate.nnue')], 'fixture-net-sha')

    def test_failed_transfer_holds_and_net_fault_launches_nothing(self):
        self.setup_net('KEEP')
        module = self.followup()
        module.run.return_value = {'verdict': 'TRANSFER_FAIL'}
        with self.assertRaisesRegex(RuntimeError, 'hold package review'):
            module.main()
        self.assertEqual(module.run.call_count, 1)
        self.setup_net('UNSCORED', ['flag-out'])
        module = self.followup()
        with self.assertRaisesRegex(RuntimeError, 'operational failure'):
            module.main()
        module.run.assert_not_called()

    def test_winner_skips_pair_training_and_fault_does_not_retrain(self):
        module = load('pair', 'pair-training.py')
        module.R, module.D = self.r, self.d
        module.wait_pid, module.status, module.trial = Mock(), Mock(), Mock()
        self.setup_net('KEEP')
        module.main()
        module.trial.assert_not_called()
        self.assertEqual(json.loads((self.d / 'pair-result.json').read_text())['state'], 'SKIPPED_NET_KEEP')
        self.setup_net('UNSCORED', ['crash'])
        with self.assertRaises(AssertionError):
            module.main()
        module.trial.assert_not_called()


if __name__ == '__main__':
    unittest.main()

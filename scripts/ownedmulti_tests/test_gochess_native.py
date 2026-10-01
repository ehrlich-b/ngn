import copy
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import ownedmatch_gochess as wrapper

ROOT = Path('/home/ehrli/nnue-owned-top3-20260930/opponents/gochess')


def role():
    return {'name': 'GoChess', 'engine': {'path': str(ROOT / 'gochess-pinned'), 'sha256': wrapper.ENGINE_SHA},
            'network': {'path': str(ROOT / wrapper.NETWORK_NAME), 'sha256': wrapper.NETWORK_SHA},
            'options': [['Threads', 1], ['Hash', 128], ['Ponder', False], ['OwnBook', False],
                        ['BookFile', '<empty>'], ['NNUEFile', '$NETWORK'], ['UseNNUE', True], ['MoveOverhead', 100]]}


class NativeGoChessTraceTests(unittest.TestCase):
    def fixture(self):
        r = role()
        roles = [{'name': 'GoChess', 'engine': '/frozen/engine', 'sha256': wrapper.ENGINE_SHA, 'cwd': '/frozen'}]
        options = [(name, '/frozen/ngn.nnue' if value == '$NETWORK' else value) for name, value in r['options']]
        trace_roles = [{'display_name': 'GoChess', 'expected_processes': 1, 'expected_refreshes': 1,
                        'resolved_options': [{'name': n, 'value': wrapper.base.value_text(v)} for n, v in options]}]
        banner = f'NNUE v5 loaded from /frozen/{wrapper.NETWORK_NAME} (fingerprint {wrapper.FINGERPRINT})'
        def line(body): return '[Engine] [00:00:00] <1> ' + body
        lines = [line('<stderr> GoChess ---> ' + banner), line('GoChess ---> ' + wrapper.FINGERPRINT_LINE),
                 line('fastchess --- Refreshing engine GoChess')]
        for name, value in options:
            lines.append(line('fastchess --- Sending setoption to engine GoChess ' + name + ' ' + wrapper.base.value_text(value)))
            if name == 'NNUEFile': lines.append(line('GoChess ---> ' + wrapper.FINGERPRINT_LINE))
            if name == 'UseNNUE': lines.append(line('GoChess ---> ' + wrapper.USE_LINE))
        lines += [line('fastchess --- Engine GoChess refreshed.'),
                  line('<stderr> GoChess ---> Process exited normally with status 0')]
        return lines, trace_roles, roles, banner

    def test_exact_model_at_startup_and_refresh_is_audited(self):
        lines, trace_roles, roles, banner = self.fixture()
        with patch.object(wrapper, 'check_staged_network', return_value=banner):
            result = wrapper.operational_trace(lines, trace_roles, roles)
        self.assertTrue(result['pass'])
        self.assertEqual(len(result['verified_exact_gochess_v5_startup_banners']), 1)

    def test_fallback_wrong_binding_and_operational_failures_are_rejected(self):
        original, trace_roles, original_roles, banner = self.fixture()
        names = ('artifact', 'fingerprint', 'disabled', 'missing-banner', 'duplicate-banner',
                 'missing-fingerprint', 'missing-use', 'wrong-path', 'foreign-actor', 'cross-thread',
                 'missing-exit', 'extra-fingerprint', 'warning', 'uppercase-error', 'illegal-move')
        for name in names:
            lines, roles = list(original), copy.deepcopy(original_roles)
            if name == 'artifact': roles[0]['sha256'] = '0' * 64
            if name == 'fingerprint': lines[1] = lines[1].replace(wrapper.FINGERPRINT, '0' * 16)
            if name == 'disabled': lines = [x.replace(wrapper.USE_LINE, 'info string UseNNUE = false') for x in lines]
            if name == 'missing-banner': lines.pop(0)
            if name == 'duplicate-banner': lines.insert(0, lines[0])
            if name == 'missing-fingerprint': lines.pop(1)
            if name == 'missing-use': lines = [x for x in lines if wrapper.USE_LINE not in x]
            if name == 'wrong-path': lines[0] = lines[0].replace('/frozen/', '/foreign/')
            if name == 'foreign-actor': lines[0] = lines[0].replace('<stderr> GoChess', '<stderr> Other')
            if name == 'cross-thread': lines[1] = lines[1].replace('<1>', '<2>')
            if name == 'missing-exit': lines.pop()
            if name == 'extra-fingerprint': lines.insert(1, lines[1])
            if name == 'warning': lines.append('[WARN] [00:00:00] <1> fastchess --- Warning; connection stalled')
            if name == 'uppercase-error': lines.append('[Engine] [00:00:00] <1> GoChess ---> info string ERROR: model failed')
            if name == 'illegal-move': lines.append('[Engine] [00:00:00] <1> fastchess --- illegal move')
            with self.subTest(name=name), patch.object(wrapper, 'check_staged_network', return_value=banner), \
                    self.assertRaises(wrapper.base.CandidateMatchError):
                wrapper.operational_trace(lines, trace_roles, roles)
        print('GOCHESS_NEGATIVE_CONTROLS ' + json.dumps([{'name': n, 'rejected': True} for n in names]))

    def test_hce_or_changed_model_cannot_be_admitted_as_the_native_role(self):
        wrapper.validate_role(role())
        for name in ('network', 'missing-network', 'disabled', 'book', 'threads', 'duplicate', 'empty-model'):
            r = role()
            if name == 'network': r['network']['sha256'] = '0' * 64
            if name == 'missing-network': r['network'] = None
            if name == 'disabled': r['options'][6][1] = False
            if name == 'book': r['options'][3][1] = True
            if name == 'threads': r['options'][0][1] = 2
            if name == 'duplicate': r['options'].append(['UseNNUE', True])
            if name == 'empty-model': r['options'][5][1] = '<empty>'
            with self.subTest(name=name), self.assertRaises(wrapper.base.CandidateMatchError):
                wrapper.validate_role(r)


@unittest.skipUnless((ROOT / 'gochess-pinned').is_file() and (ROOT / wrapper.NETWORK_NAME).is_file()
                     and hasattr(os, 'sched_getaffinity'), 'pinned native GoChess on WSL required')
class NativeGoChessStartupTests(unittest.TestCase):
    def test_pinned_binary_and_model_activate_without_fallback(self):
        sys.path.insert(0, '/home/ehrli/nnue-owned-continuation-20260928/counterdraw-deps/chess-1.11.2')
        import chess
        with tempfile.TemporaryDirectory() as raw:
            cwd = Path(raw)
            binary, options = wrapper.stage_role_inputs(role(), cwd)
            launcher = cwd / 'role_exec.py'
            shutil.copy2(wrapper.base.HELPERS / 'role_exec.py', launcher)
            launcher.chmod(0o555)
            (cwd / 'role-config.json').write_text(json.dumps({
                'schema': 'ngn-candidate-role-exec-v1', 'engine': str(binary),
                'engine_sha256': wrapper.ENGINE_SHA, 'gomaxprocs': '1', 'expected_cwd': str(cwd)}))
            receipt = wrapper.preflight(launcher, cwd, options, role(), cwd / 'positive.jsonl', min(os.sched_getaffinity(0)))
            self.assertTrue(receipt['pass'])
            self.assertEqual(receipt['native_gochess']['fingerprint'], wrapper.FINGERPRINT)
            bestmoves = [line.split()[1] for line in receipt['search'] if line.startswith('bestmove ')]
            self.assertEqual(len(bestmoves), 1)
            self.assertIn(chess.Move.from_uci(bestmoves[0]), chess.Board().legal_moves)
            (cwd / 'net.txt').unlink()
            with self.assertRaises(OSError):
                wrapper.preflight(launcher, cwd, options, role(), cwd / 'negative.jsonl', min(os.sched_getaffinity(0)))
            print('GOCHESS_NATIVE_CONTROL ' + json.dumps({'positive_activation_count': 1,
                  'engine_sha256': wrapper.ENGINE_SHA, 'network_sha256': wrapper.NETWORK_SHA,
                  'fingerprint': wrapper.FINGERPRINT, 'legal_bestmove': bestmoves[0], 'missing_net_txt_rejected': True}))


if __name__ == '__main__':
    unittest.main()

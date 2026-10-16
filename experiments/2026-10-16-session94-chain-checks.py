"""WSL-only branch fixtures; every wait, game, service and engine call is mocked."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

SOURCE = Path('/home/ehrli/ngn-data/night-20261007/session94/claim-chain.py')
spec = importlib.util.spec_from_file_location('chain_target', SOURCE)
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value))


for control in ['KEEP', 'SHELVED']:
    for treatment in ['KEEP', 'SHELVED']:
        expected = 'moderated-t74' if treatment == 'KEEP' else 'volume-control' if control == 'KEEP' else 'accepted-final115'
        assert m.choose_net({'volume-control': {'verdict': control}, 'moderated-t74': {'verdict': treatment}}) == expected

template = json.loads((m.R / 'session83/net-transfer/manifest.json').read_text())
gate_template = json.loads((m.R / 'session87/net-final174/manifest.json').read_text())
original_Path = m.Path

for scenario in ['pass', 'small-gap', 'large-gap-good-width', 'large-gap-poor-width', 'confirmation-fail', 'net-win', 'bad-predecessor']:
    with tempfile.TemporaryDirectory(prefix='ngn-s94-fixture-') as temporary:
        root = Path(temporary)
        m.R = root / 'rig'; m.D = m.R / 'session94'; m.V = root / 'tool'
        m.D.mkdir(parents=True)
        write(m.D / 'chain-nomination.json', dict(net_chain_pid=1, authority='fixture', net_selection='moderated priority'))
        write(m.R / 'session93/net-chain-result.json', dict(state='HELD_REVIEW' if scenario == 'bad-predecessor' else 'REVIEW_NET_GATES'))
        for name in ['volume-control', 'moderated-t74']:
            write(m.R / ('session93/net-' + name) / 'result.json', dict(games=1000, pairs=500, faults=[], verdict='KEEP' if scenario == 'net-win' else 'SHELVED'))
            gate = dict(gate_template, net_sha256='net-' + name,
                        candidate_options=['EvalFile=/' + name, 'Threads=1'],
                        base_options=['EvalFile=/accepted', 'Threads=1'], artifacts={'/' + name: 'sha'})
            write(m.R / ('session93/net-' + name) / 'manifest.json', gate)
        write(m.R / 'session83/net-transfer/manifest.json', template)
        write(m.D / 'playing-build.json', dict(identity={'verdict': 'PASS'}))
        write(m.V / 'build-receipt.json', dict(state='READY_TOOLING'))
        events = []
        def redirect_path(path):
            if str(path) == '/home/ehrli/ngn-data/claim-v4-20261016/attempt-001':
                return root / 'attempt'
            return original_Path(path)
        def identity(name, attempt):
            events.append(('identity', name))
            return attempt / 'identity.json'
        def stage(mode, candidate, attempt):
            assert candidate == root / 'attempt/identity.json'
            events.append(mode)
            if mode == 'screen':
                return dict(ScreenEligible=scenario in ['pass', 'confirmation-fail', 'net-win'],
                            Primary={'Estimate': 3620 if scenario == 'small-gap' else 3550}, Anchors=[])
            assert events[-2] == 'screen'
            return dict(ClaimEligible=scenario != 'confirmation-fail', Primary={'Estimate': 3650})
        def run_gate(job, controls, status):
            assert job['class'] == 'net-transfer' and job['max_games'] == 400 and job['tc'] == '60+0.6'
            assert job['candidate_options'] == ['EvalFile=/moderated-t74', 'Threads=1']
            assert job['base_options'] == ['EvalFile=/accepted', 'Threads=1']
            assert 'argv' not in job and 'reused_control' not in job
            events.append('transfer')
            return dict(verdict='TRANSFER_PASS', faults=[])
        sidecar = SimpleNamespace(terminate=lambda: events.append('monitor-stop'), wait=lambda: None)
        with patch.object(m, 'Path', side_effect=redirect_path), patch.object(m, 'wait_pid'), \
             patch.object(m, 'identity', side_effect=identity), patch.object(m, 'claim_stage', side_effect=stage), \
             patch.object(m, 'load_module', return_value=SimpleNamespace(run=run_gate)), \
             patch.object(m, 'width_check', side_effect=lambda *args: events.append('width') or scenario != 'large-gap-poor-width'), \
             patch.object(m, 'resume_work', side_effect=lambda: events.append('resume')), \
             patch.object(m.subprocess, 'Popen', return_value=sidecar):
            if scenario == 'bad-predecessor':
                try: m.main()
                except AssertionError: pass
                else: raise AssertionError('bad predecessor admitted')
                assert events == []
                continue
            m.main()
        result = json.loads((m.D / 'claim-chain-result.json').read_text())
        if scenario in ['pass', 'net-win']:
            assert result['state'] == 'GOAL_MET' and 'resume' not in events
        elif scenario == 'large-gap-poor-width':
            assert result['state'] == 'NEEDS_SMP_PROFILE' and 'resume' not in events
        else:
            assert events[-1] == 'resume' and result['state'] == 'FAILED_CLAIM_WORK_RESUMED'
        assert ('width' in events) == scenario.startswith('large-gap')
        assert ('transfer' in events) == (scenario == 'net-win')
        if scenario == 'net-win':
            assert events[0] == 'transfer' and events[1] == ('identity', 'moderated-t74')
        assert 'monitor-stop' in events
print('PASS: deterministic net selection, fixed400 transfer, conditional confirmation, both failure branches, width before work, fault hold; no live actions')

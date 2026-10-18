"""Rig-only bounded checks for the next boundary's dispatch decisions."""
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session109'
spec = importlib.util.spec_from_file_location('boundary', D/'boundary.py')
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
real_load = m.load
config = real_load(D/'nomination.json')
final = dict(epochs=115, selection='final115', reexport_identical=True,
             parity_cases=20, net_sha256='prospective-test-only')
job = m.net_job(config, final)
assert job['base'] == job['candidate'] == config['base_binary']
assert job['base_source'] == job['candidate_source'] == 'fe92ced'
assert job['class'] == 'net' and job['max_games'] == job['min_games'] == 1000
assert job['artifacts'][job['base']] == config['base_binary_sha256']
assert len(job['artifacts']) == 5
assert len(job['candidate_options']) == len(job['base_options']) == 19
assert job['candidate_options'][1:] == job['base_options'][1:]
assert not {'argv','pid','reused_control'} & job.keys()
transfer = m.transfer_job(job)
assert transfer['class'] == 'net-transfer' and transfer['max_games'] == 400
assert transfer['tc'] == '60+0.6' and transfer['game_timeout'] == 900
controls = real_load(R/'session80/controls.json')
assert job['control_key'] in controls and transfer['control_key'] in controls
scenarios = []
for verdict, endpoint in [('SHELVED',False),('KEEP',False),('UNSCORED',False),('KEEP',True)]:
    calls, jobs, receipts = [], [], {}
    checkpoints = iter([dict(iter=2000 if endpoint else 1200), dict(iter=1225), dict(iter=1225)])
    def fake_load(path):
        if path.name == 'nomination.json':
            return config if path.parent.name.startswith('ngn-s109-check-') else {'argv':['native-spsa']}
        if path == m.CELL/'result.json': return final
        if path == m.S/'s4-timing/result.json': return {'verdict':'SHELVED'}
        return real_load(path)
    def run(job, *_):
        jobs.append(job)
        return {'verdict': verdict if len(jobs)==1 else 'TRANSFER_PASS',
                'faults': ['test fault'] if verdict=='UNSCORED' else []}
    def fake_module(name, _):
        if name == 's101_boundary': return SimpleNamespace(wait_pid=lambda pid: calls.append(['wait',pid]))
        if name == 's93_checkpoint': return SimpleNamespace(checkpoint=lambda: next(checkpoints))
        return SimpleNamespace(run=run)
    m.load, m.module = fake_load, fake_module
    m.write = lambda path, value: receipts.update({path.name:value})
    m.status = lambda *_: None
    m.time = SimpleNamespace(sleep=lambda _: None)
    m.subprocess = SimpleNamespace(run=lambda argv, **_: calls.append(argv), check_output=lambda *_, **__: '', STDOUT=-2)
    with tempfile.TemporaryDirectory(prefix='ngn-s109-check-') as tmp:
        m.D = Path(tmp)
        fault = False
        try: m.main()
        except AssertionError:
            fault = True
        stops = [c for c in calls if c[0]=='systemctl']
        timings = [c for c in calls if c[0]=='bash']
        resumes = [c for c in calls if c[0]=='systemd-run']
        if endpoint:
            assert not stops and not jobs and not timings and not resumes
            assert receipts['result.json']['state'] == 'HELD_FINAL_VECTOR_PRIORITY'
        elif verdict=='UNSCORED':
            assert fault and len(jobs)==1 and not timings and not resumes
        else:
            assert not fault and len(stops)==len(resumes)==1
            assert len(jobs)==(2 if verdict=='KEEP' else 1)
            assert len(timings)==(1 if verdict=='SHELVED' else 0)
        scenarios.append(dict(net_verdict=verdict, endpoint=endpoint, passed=True))
(D/'dispatch-checks.json').write_text(json.dumps(dict(state='PASS', scenarios=scenarios,
    templates='Immediate accepted source;fixed1000;conditional400;reused controls;matching final14'),indent=2)+'\n')
print((D/'dispatch-checks.json').read_text())

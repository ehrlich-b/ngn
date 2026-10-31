"""WSL-only fixture check: no live job, service, engine or checkpoint is touched."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

SOURCE=Path('/home/ehrli/ngn-data/night-20261007/session93/net-chain.py')
spec=importlib.util.spec_from_file_location('net_chain_checks_target',SOURCE)
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
template=json.loads((m.R/'session87/net-final174/manifest.json').read_text())

def write(path,value):
    path.parent.mkdir(parents=True,exist_ok=True)
    path.write_text(json.dumps(value))

for fault in [False,True]:
    with tempfile.TemporaryDirectory(prefix='ngn-s93-chain-') as temporary:
        root=Path(temporary);m.R=root;m.D=root/'session93';m.W=root/'session88/spsa-warm';m.V=root/'v4'
        m.D.mkdir()
        write(m.D/'net-chain-nomination.json',dict(authority='fixture',training_controller_pid=1,tuner_pid=2,tuning_controller_pid=3,tuning_service='test-never-real.service'))
        write(m.V/'build-receipt.json',dict(state='READY_TOOLING',dry_modes={'screen':{'status':'PASS'}}))
        write(m.R/'session87/net-final174/manifest.json',template)
        for name in ['volume-control','moderated-t74']:
            write(m.R/'session87/mixture'/name/'result.json',dict(epochs=115,selection='final115',reexport_identical=True,parity_cases=19 if fault and name=='moderated-t74' else 20,net_sha256=name))
        write(m.W/'nomination.json',dict(argv=['fixture-spsa','-iters','2000']))
        (m.W/'spsa.jsonl').write_text('fixture history\n')
        calls=[]
        def run_job(job,controls,status):
            assert (m.D/'spsa-planned-pause.json').exists()
            assert job['max_games']==1000 and job['class']=='net' and job['tc']=='10+0.1'
            assert job['candidate']==job['base'] and job['base_options']==template['base_options']
            assert len([x for x in job['candidate_options'] if x.startswith('EvalFile=')])==1
            net=job['candidate_options'][0].split('=',1)[1]
            assert job['artifacts'][net]==job['net_sha256']
            assert job['artifacts'][template['base_options'][0].split('=',1)[1]]==template['base_net_sha256']
            assert 'argv' not in job and 'pid' not in job and 'source_checks' not in job
            calls.append(job['name'])
            return dict(verdict='SHELVED',faults=[])
        fake=SimpleNamespace(run=run_job)
        loader=SimpleNamespace(exec_module=lambda target:None)
        with patch.object(m,'wait_pid') as waits, patch.object(m,'checkpoint',side_effect=[{'iter':50},{'iter':75},{'iter':75}]), patch.object(m.time,'sleep') as sleep, patch.object(m.subprocess,'run') as stop, patch.object(m.importlib.util,'spec_from_file_location',return_value=SimpleNamespace(loader=loader)), patch.object(m.importlib.util,'module_from_spec',return_value=fake):
            if fault:
                try: m.main()
                except AssertionError: pass
                else: raise AssertionError('training/parity fault admitted')
                stop.assert_not_called();sleep.assert_not_called();assert calls==[]
            else:
                m.main()
                stop.assert_called_once_with(['systemctl','--user','stop','test-never-real.service'],check=True)
                sleep.assert_called_once_with(300)
                assert calls==['s93-net-volume-control','s93-net-moderated-t74']
                pause=json.loads((m.D/'spsa-planned-pause.json').read_text())
                assert pause['checkpoint']['iter']==75 and pause['resume_argv'][-1]=='-resume'
                assert json.loads((m.D/'net-chain-result.json').read_text())['state']=='REVIEW_NET_GATES'
print('PASS: natural-terminal sequence, next-checkpoint pause/resume receipt, two fixed1000 nominations, no pause on training/parity fault; all service/game calls mocked')

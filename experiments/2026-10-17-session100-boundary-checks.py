"""Rig-only dry fixture for the new boundary sequence; all actions mocked."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

s=Path('/home/ehrli/ngn-data/night-20261007/session100-speed')
spec=importlib.util.spec_from_file_location('boundary',s/'boundary.py')
m=importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
template=json.loads((m.R/'session87/net-final174/manifest.json').read_text())


def write(path,value):
    path.parent.mkdir(parents=True,exist_ok=True)
    path.write_text(json.dumps(value))


for mode in ('net','bad-parity','panel-only'):
    fault=mode=='bad-parity'
    panel_only=mode=='panel-only'
    with tempfile.TemporaryDirectory(prefix='ngn-s100-fixture-') as directory:
        root=Path(directory)
        m.R=root
        m.S=root/'session100-speed'
        m.S.mkdir()
        write(m.S/'boundary-nomination.json',dict(gpu_controller_pid=1,tuner_pid=2,
            tuning_service='fixture-never-real.service',estimated_gpu_ready_edt='fixture',panel_only=panel_only))
        (root/'session94/gpu-next').mkdir(parents=True,exist_ok=True)
        (root/'session94/gpu-next/controller.log').write_text('C: reserve crossed')
        write(root/'session94/gpu-next/rebalanced-t74/result.json',dict(epochs=115,
            selection='final115',reexport_identical=True,parity_cases=19 if fault else 20,net_sha256='fixture-new-net'))
        (m.S/'checks.exit').write_text('0')
        (m.S/'panel-build.exit').write_text('0')
        write(root/'session88/spsa-warm/nomination.json',dict(argv=['fixture-spsa']))
        write(root/'session87/net-final174/manifest.json',template)
        calls=[]

        def run_job(job,controls,status):
            assert job['class']=='net' and job['max_games']==1000 and job['tc']=='10+0.1'
            assert job['base_options']==template['base_options']
            assert job['candidate_options'][0].endswith('/session94/gpu-next/rebalanced-t74/trained/latest.nnue')
            assert json.loads((m.S/'spsa-planned-pause.json').read_text())['checkpoint']['iter']==825
            calls.append('gate')
            return dict(verdict='SHELVED',faults=[])

        pause=SimpleNamespace(checkpoint=iter([dict(iter=800),dict(iter=825),dict(iter=825)]).__next__)
        runner=SimpleNamespace(run=run_job)

        def load_module(name,path):
            return pause if name=='s93_pause_helpers' else runner

        def command(argv,**kwargs):
            if argv[0]=='systemctl':
                assert argv==['systemctl','--user','stop','fixture-never-real.service']
                calls.append('pause')
            elif argv[0]=='taskset':
                calls.append('panel')
            else:
                assert argv[0]=='systemd-run' and argv[-1]=='-resume'
                calls.append('resume')

        with patch.object(m,'module',side_effect=load_module),patch.object(m,'wait_pid'),\
             patch.object(m.time,'sleep') as sleep,patch.object(m.subprocess,'run',side_effect=command),\
             patch.object(m.subprocess,'check_output',return_value=' PID ARGS\n 42 hopper'):
            if fault:
                try: m.main()
                except AssertionError: pass
                else: raise AssertionError('parity fault admitted')
                assert calls==[]
                sleep.assert_not_called()
            else:
                m.main()
                assert calls==(['pause','panel','resume'] if panel_only else ['pause','gate','panel','resume']),calls
                sleep.assert_called_once_with(300)
                result=m.S/('panel-boundary-result.json' if panel_only else 'boundary-result.json')
                assert json.loads(result.read_text())['state']==('PANEL_COMPLETE_TUNE_RESUMED' if panel_only else 'NET_GATE_PANEL_COMPLETE_TUNE_RESUMED')
print('PASS: next-checkpoint pause, unchanged-base fixed1000, exclusive panel, native resume once; parity failure leaves tuner alone; disk-stop panel-only path skips games; no live actions')

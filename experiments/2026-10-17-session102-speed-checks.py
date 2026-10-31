"""Mock only: test successor order, winner holds, checkpoint pause and resume."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
from types import SimpleNamespace

SCRIPT = Path(__file__).with_name('speed-continuation.py')


def case(predecessor, verdicts, fault=False, s3='SHELVED'):
    spec = importlib.util.spec_from_file_location('continuation', SCRIPT)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    with tempfile.TemporaryDirectory() as tmp:
        m.R = Path(tmp)
        m.D = m.R/'session102-speed'
        m.S = m.R/'session101-speed'
        m.D.mkdir()
        m.S.mkdir()
        m.write(m.D/'nomination.json', dict(boundary_pid=123, authority='fixture',
            gpu_after_shelves_argv=['launch-next-gpu'],accepted_package='fixture-base', base_binary='/fixture/base', rule='frozen',
            candidates=[dict(candidate=x, binary='/fixture/'+x) for x in ('s2','s1')]))
        state = 'NET_S3_COMPLETE_TUNE_RESUMED' if predecessor=='SHELVED' else 'REVIEW_NET_WINNER_TUNE_RESUMED'
        m.write(m.S/'boundary-result.json', dict(state=state, net_result=dict(verdict=predecessor),
            speed_result=dict(verdict=s3), resume_service='prior-tune.service'))
        m.write(m.S/'spsa-planned-pause.json', dict(resume_argv=['native-tune','-resume']))
        checkpoints = iter([dict(iter=1100),dict(iter=1125),dict(iter=1125)])
        m.module = lambda name,path: (SimpleNamespace(wait_pid=lambda pid:None) if name=='s101_boundary'
            else SimpleNamespace(checkpoint=lambda:next(checkpoints)))
        m.time = SimpleNamespace(sleep=lambda seconds:None)
        actions=[]
        def run(argv, **kwargs):
            actions.append(argv)
            if '--candidate' in argv:
                if fault:
                    raise subprocess.CalledProcessError(1,argv)
                name=argv[argv.index('--candidate')+1]
                out=Path(argv[argv.index('--out')+1])
                out.mkdir()
                m.write(out/'result.json',dict(verdict=verdicts[name]))
            return SimpleNamespace(returncode=0)
        m.subprocess = SimpleNamespace(STDOUT=subprocess.STDOUT,run=run,check_output=lambda argv,**kwargs:'999\n' if argv[0]=='systemctl' else '')
        try:
            m.main()
        except subprocess.CalledProcessError:
            assert fault
        timed=[a[a.index('--candidate')+1] for a in actions if '--candidate' in a]
        resumed=[a for a in actions if a[0]=='systemd-run']
        paused=[a for a in actions if a[:3]==['systemctl','--user','stop']]
        if predecessor!='SHELVED' or s3!='SHELVED':
            assert not actions
            assert m.load(m.D/'result.json')['state']=='HELD_PREDECESSOR_REVIEW'
        elif fault:
            assert timed==['s2'] and len(paused)==1 and not resumed
        else:
            expected=['s2'] if verdicts['s2']=='KEEP_ELIGIBLE' else ['s2','s1']
            assert timed==expected and len(paused)==len(resumed)==1
            assert resumed[0][-2:]==['native-tune','-resume']
            assert m.load(m.D/'spsa-planned-pause.json')['checkpoint']['iter']==1125
            assert m.load(m.D/'result.json')['state']=='SPEED_COMPLETE_TUNE_RESUMED'
            gpu=[a for a in actions if a==['launch-next-gpu']]
            assert bool(gpu)==all(x=='SHELVED' for x in verdicts.values())
        return dict(predecessor=predecessor, s3=s3, verdicts=verdicts, fault=fault, timing=timed, pass_fixture=True)


if __name__=='__main__':
    rows=[case('KEEP',{}),case('SHELVED',{},s3='KEEP_ELIGIBLE'),
        case('SHELVED',dict(s2='KEEP_ELIGIBLE')),
        case('SHELVED',dict(s2='SHELVED',s1='KEEP_ELIGIBLE')),
        case('SHELVED',dict(s2='SHELVED',s1='SHELVED')),
        case('SHELVED',{},fault=True)]
    print(json.dumps(dict(verdict='PASS',fixtures=rows),indent=2))

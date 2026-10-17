"""Dry fixtures for the new boundary's actual branches; no host actions."""
import importlib.util
from pathlib import Path
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('boundary', Path(__file__).with_name('boundary.py'))
b = importlib.util.module_from_spec(spec)
spec.loader.exec_module(b)


def case(net_verdict, invalid_final=False, endpoint=False):
    with tempfile.TemporaryDirectory(prefix='ngn-s101-dry-') as temporary:
        root = Path(temporary)
        out = root/'session101-speed'
        out.mkdir()
        (out/'checks.exit').write_text('0\n')
        config = dict(gpu_controller_pid=123, checks_pid=456, tuner_pid=789,
            tuning_service='old-tune.service', authority='dry', s3_topic_commit='dry')
        final = dict(epochs=114 if invalid_final else 115, selection='final115',
            reexport_identical=True, parity_cases=20, net_sha256='fake-net')
        oldnet = 'old.nnue'
        inputs = {'boundary-nomination.json':config, 'identity.json':dict(verdict='PASS'),
            'result.json':final,
            'nomination.json':dict(argv=['taskset','-c','0-11','fake-tuner']),
            'manifest.json':dict(candidate_options=['EvalFile='+oldnet,'Hash=64'],artifacts={oldnet:'fake'})}
        writes, commands, resumes, waits = {}, [], [], []
        iterations = iter([2000] if endpoint else [900,925,925])
        helpers = SimpleNamespace(checkpoint=lambda:dict(iter=next(iterations)))
        result = dict(verdict=net_verdict, faults=[])
        runner = SimpleNamespace(run=lambda *_:result)

        def load(path):
            if path == out/'s3-timing/result.json':
                return dict(verdict='SHELVED')
            return inputs[path.name]

        def run(command, **kwargs):
            commands.append(command)
            return SimpleNamespace(returncode=0)

        with patch.multiple(b, R=root, S=out, load=load,
                write=lambda path,value:writes.update({str(path.relative_to(root)):value}),
                status=lambda *_:None, wait_pid=lambda pid:waits.append(pid),
                module=lambda name,_:helpers if name=='s93_pause_helpers' else runner,
                resume_tune=lambda pause:resumes.append(pause)), \
                patch.object(b.time,'sleep'), patch.object(b.subprocess,'run',side_effect=run), \
                patch.object(b.subprocess,'check_output',return_value=''):
            if invalid_final:
                try:
                    b.main()
                except AssertionError:
                    pass
                else:
                    raise AssertionError('invalid final accepted')
                assert not commands and not resumes, (commands,resumes)
                return
            b.main()
        assert waits[:2] == [456,123]
        summary = writes['session101-speed/boundary-result.json']
        if endpoint:
            assert summary['state']=='TUNE_COMPLETE_REVIEW' and not commands and not resumes
            return
        assert len(resumes)==1 and resumes[0]['checkpoint']['iter']==925
        assert resumes[0]['resume_argv'][-1]=='-resume'
        assert commands[0]==['systemctl','--user','stop','old-tune.service']
        if net_verdict=='KEEP':
            assert len(commands)==1 and summary['speed_result'] is None
            assert summary['state']=='REVIEW_NET_WINNER_TUNE_RESUMED'
        else:
            assert len(commands)==2 and '--candidate' in commands[1] and 's3' in commands[1]
            assert summary['state']=='NET_S3_COMPLETE_TUNE_RESUMED'


case('KEEP')
case('SHELVED')
case('KEEP', invalid_final=True)
case('KEEP', endpoint=True)
print('PASS: net winner holds S3; net shelved runs S3; invalid final never pauses; final tune keeps priority; one native resume')

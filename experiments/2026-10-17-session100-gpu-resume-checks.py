"""Dry resume fixture: no process, checkpoint or service is touched."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
from unittest.mock import patch

source=Path('/home/ehrli/ngn-data/night-20261007/session100-speed/gpu-resume.py')
spec=importlib.util.spec_from_file_location('gpu_resume',source)
m=importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
with tempfile.TemporaryDirectory(prefix='ngn-s100-resume-fixture-') as directory:
    m.R=Path(directory)
    m.S=m.R/'session100-speed'
    m.CELL=m.R/'session94/gpu-next/rebalanced-t74'
    m.S.mkdir()
    (m.CELL/'trained').mkdir(parents=True)
    (m.S/'panel-boundary.pid').write_text('123')
    (m.S/'panel-boundary-result.json').write_text(json.dumps(dict(state='PANEL_COMPLETE_TUNE_RESUMED')))
    original=['fixture-python','-m','trainer.train','--epochs','115']
    (m.CELL/'manifest.json').write_text(json.dumps(dict(argv=original)))
    (m.CELL/'trained/history.json').write_text(json.dumps([dict(epoch=59)]))
    (m.CELL/'train.log').write_text('original disk stop')
    calls=[]
    fake=SimpleNamespace(subprocess=SimpleNamespace(Popen=lambda command,*a,**kw:calls.append(command)))

    def trial(name):
        assert fake.D==m.R/'session94/gpu-next' and name=='rebalanced-t74'
        fake.subprocess.Popen(original)
        fake.subprocess.Popen(['fixture-go-parity'])

    fake.trial=trial
    loader=SimpleNamespace(exec_module=lambda target:None)
    with patch.object(m.os,'pidfd_open',side_effect=ProcessLookupError),\
         patch.object(m.os,'statvfs',return_value=SimpleNamespace(f_bavail=28*1024**3,f_frsize=1)),\
         patch.object(m.importlib.util,'spec_from_file_location',return_value=SimpleNamespace(loader=loader)),\
         patch.object(m.importlib.util,'module_from_spec',return_value=fake):
        m.main()
    assert calls==[original+['--resume',str(m.CELL/'trained/latest.pt')],['fixture-go-parity']]
    assert (m.CELL/'train-disk-stop.log').read_text()=='original disk stop'
    assert json.loads((m.S/'gpu-resume-manifest.json').read_text())['epoch']==59
print('PASS: same115-epoch argv plus native latest.pt resume; original trial/finalization/parity reused; disk log preserved; no live actions')

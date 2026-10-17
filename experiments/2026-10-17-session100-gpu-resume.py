"""Resume the disk-stopped same training cell after the exclusive panel.

One native panel wait and a silent15-minute capacity wait. Reuse the original
trial's disk guard/finalization/parity; preserve failed logs and checkpoints.
"""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import select
import shutil
import time

R=Path('/home/ehrli/ngn-data/night-20261007')
S=R/'session100-speed'
CELL=R/'session94/gpu-next/rebalanced-t74'


def status(event,detail):
    with (R/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session100-gpu-resume',event=event,detail=detail))+'\n')


def main():
    pid=int((S/'panel-boundary.pid').read_text())
    status('waiting-panel',dict(pid=pid,required_host_c_gib=27.5,capacity_interval_seconds=900))
    try: fd=os.pidfd_open(pid)
    except ProcessLookupError: pass
    else:
        try: select.select([fd],[],[])
        finally: os.close(fd)
    panel=json.loads((S/'panel-boundary-result.json').read_text())
    assert panel['state']=='PANEL_COMPLETE_TUNE_RESUMED',panel
    while True:
        disk=os.statvfs('/mnt/c')
        if disk.f_bavail*disk.f_frsize>=27.5*1024**3: break
        time.sleep(900)
    manifest=json.loads((CELL/'manifest.json').read_text())
    argv=manifest['argv']+['--resume',str(CELL/'trained/latest.pt')]
    history=json.loads((CELL/'trained/history.json').read_text())
    resume=dict(epoch=history[-1]['epoch'],argv=argv,selection='same final115; no checkpoint shopping',
        reason='Original controller stopped at host C reserve; unchanged recipe/data/weights/schedule',
        utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    (S/'gpu-resume-manifest.json').write_text(json.dumps(resume,indent=2)+'\n')
    shutil.copyfile(CELL/'train.log',CELL/'train-disk-stop.log')
    path=R/'session94/source/experiments/2026-10-15-session87-train-mixture.py'
    spec=importlib.util.spec_from_file_location('original_trial',path)
    module=importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.D=R/'session94/gpu-next'
    original_popen=module.subprocess.Popen

    def launch(command,*args,**kwargs):
        if command==manifest['argv']: command=argv
        return original_popen(command,*args,**kwargs)

    module.subprocess.Popen=launch
    status('resuming',resume)
    module.trial('rebalanced-t74')
    status('complete',dict(result=str(CELL/'result.json'),next='Fixed1000 net gate at a planned checkpoint boundary; no automatic games/adoption'))


if __name__=='__main__':
    try: main()
    except BaseException as error:
        status('held-review',str(error))
        raise

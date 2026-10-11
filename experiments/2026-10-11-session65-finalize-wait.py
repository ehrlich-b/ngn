"""One native inotify wait for the recovered trainer, then selected reexport/parity."""
import ctypes
import datetime
import json
import os
from pathlib import Path
import select
import subprocess

r=Path('/home/ehrli/ngn-data/night-20261007')
p=r/'session57-data-step2'
control=r/'session65-finalizer'
def status(event,detail):
    with (control/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session65-data-finalizer',event=event,detail=detail))+'\n')
libc=ctypes.CDLL(None,use_errno=True)
fd=libc.inotify_init1(os.O_CLOEXEC|os.O_NONBLOCK)
assert fd>=0
watch=libc.inotify_add_watch(fd,os.fsencode(p),0x8|0x80) # close_write/moved_to
assert watch>=0
status('wait','native notification for recovered training-controller-session62.exit; no games')
deadline=datetime.datetime.now().timestamp()+6*3600
try:
    while not (p/'training-controller-session62.exit').exists():
        remaining=deadline-datetime.datetime.now().timestamp()
        assert remaining>0,'training terminal not ready; healthy trainer is left running'
        readable,_,_=select.select([fd],[],[],remaining)
        assert readable,'training terminal not ready; healthy trainer is left running'
        os.read(fd,65536)
finally:
    os.close(fd)
assert (p/'training-controller-session62.exit').read_text().strip()=='0'
assert json.loads((p/'training-terminal.json').read_text())['full58']
status('finalize-start','full58 terminal; rig13,15 selected identical reexport/20integer parity; actual engine final14')
subprocess.run([os.environ['NGN_TRAIN_PYTHON'],str(control/'finalize.py'),str(p),
                str(r/'session64-final14/candidate'),os.environ['NGN_PARITY_RECORDS'],
                'training-terminal.json','rig'],check=True,timeout=3600)
status('finalize-done',json.loads((p/'training-receipt.json').read_text()))

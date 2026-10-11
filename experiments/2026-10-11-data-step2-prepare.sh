#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-data-step2-20261011
export PYTHONPATH=/work/lanes/ngn-s47/blackbox-current PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 MKL_NUM_THREADS=2
cd "$S"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 prepare.py
python3 - <<'PY'
import ctypes,os,select
from pathlib import Path
libc=ctypes.CDLL(None,use_errno=True)
fd=libc.inotify_init1(os.O_CLOEXEC); assert fd>=0
assert libc.inotify_add_watch(fd,b'.',0x8|0x80)>=0
poll=select.poll();poll.register(fd,select.POLLIN)
while not Path('stream.done').exists():
 assert poll.poll(12*3600*1000),'old-corpus byte-stream sample wait expired'
 os.read(fd,65536)
os.close(fd)
PY
/work/lanes/ngn-s47/parity-venv/bin/python coverage.py > coverage.log 2>&1

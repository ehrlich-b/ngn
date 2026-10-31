#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session75
cd "$s"
finish() { rc=$?; echo "$rc" > scaling.exit; date -Is > scaling.done; }
trap finish EXIT
export PYTHONDONTWRITEBYTECODE=1
python3 - <<'PY'
import os,select
from pathlib import Path
p=Path.cwd()
for name,pid in [('panel',1315),('checks',3100)]:
 if not (p/(name+'.done')).exists():
  fd=os.pidfd_open(pid)
  try:
   poll=select.poll();poll.register(fd,select.POLLIN)
   assert poll.poll(3*3600*1000),name+' timeout'
  finally:os.close(fd)
 assert (p/(name+'.exit')).read_text().strip()=='0',name+' failed: no scaling'
PY
python3 smp.py

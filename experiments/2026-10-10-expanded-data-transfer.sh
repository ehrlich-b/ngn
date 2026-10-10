#!/usr/bin/env bash
# Lightweight Mac SSH relay; all network export/parity computation is on forge.
set -euo pipefail
S=/Users/ehrlich/repos/ngn-night/output/session48-transfer
mkdir -p "$S"
cd "$S"
finish() { local rc=$?; echo "$rc" > transfer.exit; date '+%Y-%m-%dT%H:%M:%S%z' > transfer.done; }
trap finish EXIT
date '+%Y-%m-%dT%H:%M:%S%z' > transfer.started
# One native completion wait on the existing training-only watcher; no polling.
ssh rig "wsl -e bash -s" > artifacts.tar <<'SH'
set -euo pipefail
python3 - <<'PY'
import json,os,select
from pathlib import Path
p=Path('/home/ehrli/ngn-data/night-20261007/session44-expanded-data/training-terminal-session47.json')
if not p.exists():
 fd=os.pidfd_open(15998)
 q=select.poll();q.register(fd,select.POLLIN)
 assert q.poll(6*3600*1000),'training terminal wait expired; never restart training'
 os.close(fd)
j=json.loads(p.read_text())
assert j['full58'] and j['postprocessing_machine']=='forge',j
PY
cd /home/ehrli/ngn-data/night-20261007/session44-expanded-data
tar cf - training-terminal-session47.json trained/history.json trained/config.json trained/best.pt trained/best.nnue -C /home/ehrli/ngn-data/t80-20261006 parity-records.ngnp
SH
ssh forge 'mkdir -p /work/lanes/ngn-s47/expanded-data; tar xf - -C /work/lanes/ngn-s47/expanded-data' < artifacts.tar
ssh forge 'cd /work/lanes/ngn-s47/expanded-data; setsid nohup systemd-run --user --scope --unit=ngn-s48-data-finalize -p AllowedCPUs=5-7 -p CPUQuota=300% -p MemoryMax=2G -p MemorySwapMax=0 nice -n 10 bash /work/lanes/ngn-s48/data-finalize.sh < /dev/null > finalizer.log 2>&1 & echo $! > finalizer.pid; cat finalizer.pid' > finalizer-launch.txt

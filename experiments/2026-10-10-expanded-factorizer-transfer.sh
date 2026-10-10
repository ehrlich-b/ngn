#!/usr/bin/env bash
# Lightweight SSH artifact relay; training/export/parity run on rig/forge.
set -euo pipefail
S=/Users/ehrlich/repos/ngn-night/output/session51-transfer
mkdir -p "$S"
cd "$S"
finish() { local rc=$?; echo "$rc" > transfer.exit; date '+%Y-%m-%dT%H:%M:%S%z' > transfer.done; }
trap finish EXIT
date '+%Y-%m-%dT%H:%M:%S%z' > transfer.started
ssh -o ServerAliveInterval=60 -o ServerAliveCountMax=20 rig "wsl -e bash -s" > artifacts.tar <<'SH'
set -euo pipefail
python3 - <<'PY'
import json,os,select
from pathlib import Path
p=Path('/home/ehrli/ngn-data/night-20261007/session51-expanded-factorizer')
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 q=select.poll();q.register(fd,select.POLLIN)
 assert q.poll(16*3600*1000),'training terminal wait expired; never restart training'
 os.close(fd)
assert (p/'controller.exit').read_text().strip()=='0'
j=json.loads((p/'training-terminal.json').read_text());assert j['full58'] and j['postprocessing_machine']=='forge'
PY
cd /home/ehrli/ngn-data/night-20261007/session51-expanded-factorizer
tar cf - training-terminal.json training-manifest.json baseline-config.json trained/history.json trained/config.json trained/best.pt trained/best.nnue -C /home/ehrli/ngn-data/t80-20261006 parity-records.ngnp
SH
ssh forge 'mkdir /work/lanes/ngn-s51/factorizer; tar xf - -C /work/lanes/ngn-s51/factorizer' < artifacts.tar
ssh forge 'cd /work/lanes/ngn-s51/factorizer; setsid nohup systemd-run --user --scope --unit=ngn-s51-factorizer-finalize -p AllowedCPUs=5-7 -p CPUQuota=300% -p MemoryMax=2G -p MemorySwapMax=0 nice -n10 bash /work/lanes/ngn-s51/factorizer-finalize.sh < /dev/null > finalizer.log 2>&1 & echo $! > finalizer.pid; cat finalizer.pid' > finalizer-launch.txt

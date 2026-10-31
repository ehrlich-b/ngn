#!/bin/bash
# Resume the existing disk-interrupted run once the host reserve is restored.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session84-training174
S=$R/session85
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
mkdir -p "$S"
echo $$ > "$S/recovery.pid"
finish() { local rc=$?; echo "$rc" > "$S/recovery.exit"; date -Is > "$S/recovery.done"; }
trap finish EXIT
python3 - "$R" <<'PY'
import datetime,json,os,sys,time
from pathlib import Path
r=Path(sys.argv[1])
def status(event,detail):
    with (r/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session85-training174-recovery',event=event,detail=detail))+'\n')
status('waiting-for-space',dict(required_bytes=int(25.5*1024**3),interval_seconds=300,continuation='same epoch92 checkpoint to final174; no new initialization or recipe change'))
while True:
    disk=os.statvfs('/mnt/c')
    free=disk.f_bavail*disk.f_frsize
    if free>=25.5*1024**3:
        break
    time.sleep(300)
status('space-restored',dict(host_c_free_bytes=free))
PY
export PYTHONPATH=$R/session80/source OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
# Same allocation and stop rules as the original run. Preserve its exit1/log.
set +e
systemd-run --user --wait --collect --unit=ngn-s85-training174-resume \
  -p AllowedCPUs=12-15 -p CPUQuota=200% -p MemoryMax=4G -p MemorySwapMax=0 \
  nice -n 19 taskset -c 12,14 "$PY" "$D/train174.py" --resume
rc=$?
set -e
echo "$rc" > "$D/resume-controller.exit"
date -Is > "$D/resume-controller.done"
exit "$rc"

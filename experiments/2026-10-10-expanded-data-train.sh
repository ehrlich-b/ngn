#!/usr/bin/env bash
# Launch only after reviewing corpus coverage and the architecture terminal.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session42-expanded-data
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache PYTHONPATH=$S/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session42-expanded-data',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; review training/parity before fresh actual-base games; no automatic adoption"
}
trap finish EXIT
date -Is > controller.started
free -h > memory-before.txt
sha256sum -c inputs.sha256 > input-check-before.txt
python3 - <<'PY'
import os
d=os.statvfs('/mnt/c'); assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
st training-start 'reviewed expanded corpus; matched58x256M/KB8H512/NB8/BF16/factorizer-off; rig12,14/4GiB'
# C: has little spare capacity. Watch only this training process group, with
# a small margin for a checkpoint write; never stop unrelated jobs.
taskset -c 12,14 nice -n 19 python3 - <<'PY' > trial.log 2>&1
import os,signal,subprocess,sys
p=subprocess.Popen(['/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python','trial.py'],start_new_session=True)
try:
 while True:
  d=os.statvfs('/mnt/c')
  if d.f_bavail*d.f_frsize < 25.25*1024**3:
   raise RuntimeError('C: checkpoint reserve crossed25.25GiB; preserve interrupted training, no automatic restart')
  try:
   rc=p.wait(timeout=60)
   break
  except subprocess.TimeoutExpired:
   pass
except BaseException:
 if p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=15)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGKILL);p.wait()
 raise
sys.exit(rc)
PY
sha256sum -c inputs.sha256 > input-check-after.txt
st training-done 'complete58/reexport/20 exact parity; fresh actual-base A/A40/fixed400 gate still required'

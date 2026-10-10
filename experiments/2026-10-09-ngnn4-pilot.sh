#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session35-ngnn4-pilot
export PYTHONPATH=$R/session34-architecture/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session35-ngnn4-pilot',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; cost pilot only, no net adoption"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
python3 - <<'PY'
import os
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
st pilot-start 'three complete16M epochs/H512L16/dense/BF16;12,14; then fixed10h budget receipt'
timeout --signal=TERM --kill-after=30s 45m taskset -c 12,14 nice -n 19 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python pilot.py > pilot.log 2>&1
sha256sum -c inputs.sha256 > input-check-after.txt
st pilot-done "$(cat receipt.json); cost only; factorizer verdict remains ahead of full training"

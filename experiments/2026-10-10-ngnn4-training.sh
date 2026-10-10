#!/usr/bin/env bash
# Resume the unstarted full training stage only; hold games for the key decision.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session38-ngnn4-training
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache PYTHONPATH=$R/session34-architecture/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session38-ngnn4-training',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; training/parity/cost only; key decision and architecture game manifest still required"; }
trap finish EXIT
date -Is > controller.started
python3 - <<'PY'
import json,os
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007')
assert (r/'session35-factorizer/controller.exit').read_text().strip()=='0'
assert json.loads((r/'session35-factorizer/gate-summary.json').read_text())['verdict']=='shelved'
assert json.loads((r/'session35-ngnn4-pilot/receipt.json').read_text())['budget_admitted']
d=os.statvfs('/mnt/c'); assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
free -h > memory-before.txt
sha256sum -c inputs.sha256 > input-check-before.txt
st training-start 'fresh58x256M/H512L16/KB8NB8/dense/BF16/factorizer-off; unchanged admitted recipe; games held for higher-priority key decision'
taskset -c 12,14 nice -n 19 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python trial.py > trial.log 2>&1
sha256sum -c inputs.sha256 > input-check-after.txt
st training-done 'complete58/reexport/20 exact parity/actual cost; no games or automatic adoption; freeze then-accepted source after key review'

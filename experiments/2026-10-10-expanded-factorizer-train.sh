#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session51-expanded-factorizer
export GOAMD64=v3 GOMAXPROCS=2 PYTHONDONTWRITEBYTECODE=1
export PYTHONPATH=$R/session44-expanded-data/source
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session51-expanded-factorizer',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; finalize on forge, then manual review/nomination; no automatic games"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
free -h > memory-before.txt
st training-start 'expanded data accepted; original58x256M recipe plus king-factorizer only;12,14/GPU/4GiB/nice19'
python3 trial.py "$S" > trial.log 2>&1
sha256sum -c inputs.sha256 > input-check-after.txt
st training-done 'full58/config matched; transfer to forge for reexport/20 integer parity; no automatic games or adoption'

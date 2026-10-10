#!/usr/bin/env bash
# Unchanged full58 selected network: export and Go parity only, never games.
set -euo pipefail
R=/work/lanes/ngn-s47
S=$R/expanded-data
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export PYTHONPATH=$R/blackbox-current PYTHONDONTWRITEBYTECODE=1 OPENBLAS_NUM_THREADS=2
cd "$S"
st() { python3 - /work/lanes/ngn-s48/status.jsonl "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session48-data-finalize',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > finalizer.exit; date -Is > finalizer.done; st end "rc $rc; no games/net adoption; review training-receipt.json before nomination"; }
trap finish EXIT
date -Is > finalizer.started
st start 'full58 copied from rig; unchanged selected checkpoint/reexport/20 integer Go parity on forge5-7/2GiB'
"$R/parity-venv/bin/python" "$R/finalize.py" "$S" "$R/blackbox-current" "$S/parity-records.ngnp"
st parity-done 'review receipt and source/phase diagnostic manually; data A/A40/fixed400 has priority over SEE diagnosis'

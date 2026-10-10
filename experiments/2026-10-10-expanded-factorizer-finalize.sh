#!/usr/bin/env bash
set -euo pipefail
R=/work/lanes/ngn-s51
S=$R/factorizer
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export PYTHONPATH=/work/lanes/ngn-s47/blackbox-current
export PYTHONDONTWRITEBYTECODE=1 OPENBLAS_NUM_THREADS=2
cd "$S"
finish() { local rc=$?; echo "$rc" > finalizer.exit; date -Is > finalizer.done; }
trap finish EXIT
date -Is > finalizer.started
/work/lanes/ngn-s47/parity-venv/bin/python "$R/factorizer-finalize.py" "$S" "$PYTHONPATH" "$S/parity-records.ngnp"

#!/usr/bin/env bash
set -euo pipefail
cd /work/lanes/ngn-data-step2-20261011
export PYTHONPATH=/work/lanes/ngn-s47/blackbox-current PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 MKL_NUM_THREADS=2
finish() { local rc=$?; echo "$rc" > recovery.exit; date -Is > recovery.done; }
trap finish EXIT
/work/lanes/ngn-s47/parity-venv/bin/python recovery.py
/work/lanes/ngn-s47/parity-venv/bin/python coverage.py > coverage.log 2>&1

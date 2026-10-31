#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session57-data-step2
export PYTHONPATH=$R/session44-expanded-data/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
finish() { local rc=$?; echo "$rc" > training-controller-session60.exit; date -Is > training-controller-session60.done; }
trap finish EXIT
date -Is > training-controller-session60.started
/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python train.py

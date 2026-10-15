#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
echo $$ > "$R/session87/mixture-controller.pid"
trap 'rc=$?; echo "$rc" > "$R/session87/mixture-controller.exit"; date -Is > "$R/session87/mixture-controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s87-mixture -p AllowedCPUs=12-15 -p CPUQuota=200% -p MemoryMax=4G -p MemorySwapMax=0 nice -n 19 taskset -c 12,14 env OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python "$R/session87/train-mixture.py"

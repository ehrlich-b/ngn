#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session111
setsid nohup systemd-run --user --scope --unit=ngn-s111-gpu-next \
  -p AllowedCPUs=12-15 -p CPUQuota=200% -p MemoryMax=6G -p MemorySwapMax=0 \
  env PYTHONPATH="$R/session87/training-source" \
  OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 \
  GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off \
  taskset -c 12-15 nice -n 19 \
  /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python "$D/gpu-next.py" \
  < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
sleep 2
cat "$D/launcher.pid"
cat "$D/controller.pid"

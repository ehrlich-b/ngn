#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session113-vector
setsid nohup systemd-run --user --scope --unit=ngn-s113-vector \
  -p AllowedCPUs=0-11 -p CPUQuota=1200% -p MemoryMax=3G -p MemorySwapMax=0 \
  taskset -c 0-11 nice -n 19 python3 "$D/vector-gate.py" \
  < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
cat "$D/launcher.pid"

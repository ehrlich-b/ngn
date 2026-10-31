#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session114-data
setsid nohup systemd-run --user --scope --unit=ngn-s114-data \
  -p AllowedCPUs=13,15 -p CPUQuota=200% -p MemoryMax=3G -p MemorySwapMax=0 \
  taskset -c 13,15 nice -n 19 ionice -c 3 python3 "$D/data.py" \
  < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
cat "$D/launcher.pid"

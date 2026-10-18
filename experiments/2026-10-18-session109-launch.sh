#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session109
setsid nohup systemd-run --user --scope --unit=ngn-s109-boundary \
  -p AllowedCPUs=0-15 -p CPUQuota=1600% -p MemoryMax=3G -p MemorySwapMax=0 \
  taskset -c 0-15 nice -n 19 python3 "$D/boundary.py" \
  < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
cat "$D/launcher.pid"

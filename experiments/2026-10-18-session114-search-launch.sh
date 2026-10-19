#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session114-search
setsid nohup systemd-run --user --scope --unit=ngn-s114-search \
  -p AllowedCPUs=0-15 -p CPUQuota=1600% -p MemoryMax=3G -p MemorySwapMax=0 \
  taskset -c 0-15 nice -n 19 python3 "$D/search-chain.py" \
  < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
cat "$D/launcher.pid"

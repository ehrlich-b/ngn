#!/usr/bin/env bash
# Detached, no CPU quota: claim occupancy and all games inherit full physical width.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session117
export PATH=/usr/local/go/bin:/usr/lib/wsl/lib:$PATH PYTHONDONTWRITEBYTECODE=1
setsid nohup systemd-run --user --scope --unit=ngn-s117-chain \
 -p AllowedCPUs=0-15 -p MemoryMax=4G -p MemorySwapMax=0 \
 taskset -c 0-15 nice -n 19 python3 "$D/claim-chain.py" \
 < /dev/null > "$D/controller.log" 2>&1 &
echo $! > "$D/launcher.pid"
cat "$D/launcher.pid"

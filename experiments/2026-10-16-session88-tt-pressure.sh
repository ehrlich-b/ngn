#!/bin/bash
# Diagnostic only, on the two non-game CPUs; never touches a game process.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session88/tt-pressure
echo $$ > "$D/controller.pid"
trap 'rc=$?; echo "$rc" > "$D/controller.exit"; date -Is > "$D/controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s88-tt-pressure \
  -p AllowedCPUs=13,15 -p CPUQuota=200% -p MemoryMax=2G -p MemorySwapMax=0 \
  nice -n 19 taskset -c 13,15 bash "$D/work.sh"

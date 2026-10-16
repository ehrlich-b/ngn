#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session89/tt-history
echo $$ > "$D/controller.pid"
trap 'rc=$?; echo "$rc" > "$D/controller.exit"; date -Is > "$D/controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s89-tt-history \
 -p AllowedCPUs=13,15 -p CPUQuota=200% -p MemoryMax=2G -p MemorySwapMax=0 \
 nice -n 19 taskset -c 13,15 bash "$D/work.sh"

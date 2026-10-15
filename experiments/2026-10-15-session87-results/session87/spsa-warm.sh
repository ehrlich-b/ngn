#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
echo $$ > "$R/session87/spsa-controller.pid"
trap 'rc=$?; echo "$rc" > "$R/session87/spsa-controller.exit"; date -Is > "$R/session87/spsa-controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s87-spsa-warm -p AllowedCPUs=0-11 -p CPUQuota=600% -p MemoryMax=3G -p MemorySwapMax=0 nice -n 19 taskset -c 0-11 python3 "$R/session87/2026-10-14-session84-spsa.py" "$R/session87/spsa-warm"

#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
echo $$ > "$R/session87/queue-controller.pid"
trap 'rc=$?; echo "$rc" > "$R/session87/queue-controller.exit"; date -Is > "$R/session87/queue-controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s87-queue -p AllowedCPUs=0-15 -p CPUQuota=800% -p MemoryMax=3G -p MemorySwapMax=0 nice -n 19 taskset -c 0-15 python3 "$R/session87/campaign_queue.py" "$R/session87/queue.json"

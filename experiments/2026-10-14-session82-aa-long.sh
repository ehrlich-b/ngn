#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session82
finish() { local rc=$?; echo "$rc" > "$D/aa-long.exit"; date -Is > "$D/aa-long.done"; }
trap finish EXIT
echo $$ > "$D/aa-long.pid"
export GOMAXPROCS=1 PYTHONDONTWRITEBYTECODE=1
python3 "$D/campaign_queue.py" "$D/aa-long-queue.json"

#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session81
finish() { local rc=$?; echo "$rc" > "$D/queue.exit"; date -Is > "$D/queue.done"; }
trap finish EXIT
export GOMAXPROCS=1 PYTHONDONTWRITEBYTECODE=1
profile_pid=$(cat "$D/profile/controller.pid")
tail --pid="$profile_pid" -f /dev/null
[ "$(cat "$D/profile/controller.exit")" = 0 ]
python3 "$D/campaign_queue.py" "$D/queue.json"

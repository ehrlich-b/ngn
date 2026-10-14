#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session82
finish() { local rc=$?; echo "$rc" > "$D/net.exit"; date -Is > "$D/net.done"; }
trap finish EXIT
echo $$ > "$D/net.pid"
export GOMAXPROCS=1 PYTHONDONTWRITEBYTECODE=1
python3 "$D/net.py"

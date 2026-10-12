#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session74-horizon-gate
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/controller.exit"; date -Is > "$s/controller.done"; }
trap finish EXIT
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
python3 games.py

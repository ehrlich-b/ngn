#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session72-see-rig
cd "$s"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
python3 ../session74-schedule/schedule.py see

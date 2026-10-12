#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session74-schedule
cd "$s"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
export GOMAXPROCS=1 PYTHONDONTWRITEBYTECODE=1
python3 schedule.py

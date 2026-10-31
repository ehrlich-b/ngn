#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-smp-panel
cd "$D"
finish(){ rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
export PYTHONDONTWRITEBYTECODE=1
python3 controller.py

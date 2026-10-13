#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session79-admission-see
cd "$D"
finish(){ rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
python3 games.py

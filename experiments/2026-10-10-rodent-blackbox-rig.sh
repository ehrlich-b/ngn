#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session50-rodent-blackbox
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 controller.py

#!/usr/bin/env bash
# Prepared only; manually nominate after TT terminal review and base selection.
set -euo pipefail
export GOMAXPROCS=1 GOAMD64=v3
cd /home/ehrli/ngn-data/night-20261007/session55-factorizer-games
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 games.py

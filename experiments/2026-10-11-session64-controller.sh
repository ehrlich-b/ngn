#!/usr/bin/env bash
set -euo pipefail
cd /work/lanes/ngn-s64/current-spsa
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 games.py

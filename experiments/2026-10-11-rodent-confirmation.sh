#!/usr/bin/env bash
set -euo pipefail
s=$1
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/controller.exit"; date -Is > "$s/controller.done"; }
trap finish EXIT
date -Is > controller.started
export GOMAXPROCS=1 GOAMD64=v3
/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python controller.py "$s"

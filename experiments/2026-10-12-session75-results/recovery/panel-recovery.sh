#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session75
cd "$s"
finish() { rc=$?; echo "$rc" > panel.exit; date -Is > panel.done; }
trap finish EXIT
export PYTHONDONTWRITEBYTECODE=1
python3 panel-recovery.py

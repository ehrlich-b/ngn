#!/usr/bin/env bash
set -euo pipefail
cd /work/lanes/ngn-s66-data-gate
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 gate.py

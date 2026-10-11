#!/usr/bin/env bash
set -euo pipefail
cd /home/ehrli/ngn-data/night-20261007/session65-spsa-transfer
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 games.py

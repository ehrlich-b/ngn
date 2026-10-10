#!/usr/bin/env bash
# Stage only. Launch after completed training review and actual source nomination.
set -euo pipefail
cd /home/ehrli/ngn-data/night-20261007/session46-expanded-data-games
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 - <<'PY'
import os
s=os.statvfs('/mnt/c');assert s.f_bavail*s.f_frsize>=25.5*1024**3
PY
python3 games.py

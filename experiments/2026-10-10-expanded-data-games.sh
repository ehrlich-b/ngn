#!/usr/bin/env bash
# Stage only. Launch after completed training review and actual source nomination.
set -euo pipefail
cd /work/lanes/ngn-s47/data-games
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 games.py

#!/usr/bin/env bash
# Prepared stage; manual actual-net nomination after factorizer terminal review.
set -euo pipefail
export GOMAXPROCS=1 GOAMD64=v3
cd /home/ehrli/ngn-data/night-20261007/session56-time-tt-games
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
python3 games.py

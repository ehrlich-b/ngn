#!/usr/bin/env bash
set -euo pipefail
cd /home/ehrli/ngn-data/night-20261007/session57-data-step2
finish() { local rc=$?; echo "$rc" > recovery-copy.exit; date -Is > recovery-copy.done; }
trap finish EXIT
python3 recovery-copy.py

#!/usr/bin/env bash
set -euo pipefail
cd /home/ehrli/ngn-data/night-20261007/session57-data-step2
finish() { local rc=$?; echo "$rc" > recovery-copy-session60.exit; date -Is > recovery-copy-session60.done; }
trap finish EXIT
date -Is > recovery-copy-session60.started
python3 recovery-copy-session59.py --resume-after-raw

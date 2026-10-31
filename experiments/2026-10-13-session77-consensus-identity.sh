#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-consensus
cd "$D"
finish(){ rc=$?; echo "$rc" > identity.exit; date -Is > identity.done; }
trap finish EXIT
export PYTHONDONTWRITEBYTECODE=1
python3 - <<'PY'
import os,select,pathlib
p=pathlib.Path.cwd()
try:
 fd=os.pidfd_open(6223);poll=select.poll();poll.register(fd,select.POLLIN);assert poll.poll(3600000);os.close(fd)
except ProcessLookupError:pass
assert (p/'checks.exit').read_text().strip()=='0','candidate checks failed'
PY
python3 identity.py > identity.log 2>&1

#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session41-remote-archive
cd "$S"
cd source
finish() { local rc=$?; echo "$rc" > "$S/checks.exit"; date -Is > "$S/checks.done"; }
trap finish EXIT
date -Is > "$S/checks.started"
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PYTHONPATH="$S/source" OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
taskset -c 12,14 nice -n 19 "$PY" -m unittest trainer.tests.test_remote_archive -v > "$S/remote-tests.log" 2>&1
taskset -c 12,14 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v > "$S/trainer-tests.log" 2>&1
"$PY" - <<'PY'
import hashlib,json
from pathlib import Path
s=Path.cwd().parent;files=['trainer/archive.py','trainer/remote_archive.py','trainer/archive_server.py','trainer/tests/test_remote_archive.py']
(s/'check-receipt.json').write_text(json.dumps(dict(tests_pass=True,cpus=[12,14],memory_max=2621440000,
 source_sha256={p:hashlib.sha256((s/'source'/p).read_bytes()).hexdigest() for p in files}),indent=2)+'\n')
PY

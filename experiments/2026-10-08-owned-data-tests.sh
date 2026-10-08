#!/usr/bin/env bash
# Training-tool checks only; never touches active sources or starts GPU work.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session7
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='owned-data-holdout',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; inspect logs; no automatic retry"; fi
}
trap finish EXIT
cd "$S"
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
mkdir source
tar -xf source.tar -C source
tar -xf overlay.tar -C source
cd source
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export PYTHONPATH="$S/source" CUDA_VISIBLE_DEVICES=''
st start "pid $$; NGNP optional content split; CPU 13,15; no GPU or games"
taskset -c 13,15 nice -n 19 "$PY" -m unittest trainer.tests.test_ngnp_split -v > "$S/split-tests.log" 2>&1
taskset -c 13,15 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v > "$S/trainer-tests.log" 2>&1
cd "$S"
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
paths=['trainer/train.py','trainer/ngnp_split.py','trainer/tests/test_ngnp_split.py','trainer/README.md']
receipt=dict(source_sha256={p:digest(r/'source'/p) for p in paths},
             archive_sha256=digest(r/'source.tar'),overlay_sha256=digest(r/'overlay.tar'),
             tests_pass=True,cpu=[13,15],nice=19,gpu_used=False,
             python=subprocess.check_output(['/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python','--version'],text=True).strip(),
             logs={p:dict(sha256=digest(r/p),tail=(r/p).read_text().splitlines()[-5:])
                   for p in ('split-tests.log','trainer-tests.log')})
(r/'test-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st done 'narrow and full trainer suites pass; see session7/test-receipt.json; no net trained or engine changed'

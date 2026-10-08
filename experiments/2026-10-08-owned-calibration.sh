#!/usr/bin/env bash
# Not queued: a committed reviewed teacher nomination must precede generation.
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session10
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
cd "$S"
export PYTHONPATH=$S/source PYTHONDONTWRITEBYTECODE=1 CUDA_VISIBLE_DEVICES=''
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
sha256sum -c calibration-inputs.sha256 > calibration-input-check-before.txt
if [[ ! -f nomination.json || ! -f activation-inputs.sha256 ]]; then
  printf '%s\n' 'Refused: commit a reviewed immediate-teacher activation after longer-net/admission/CUDA reviews.' >&2
  exit 2
fi
sha256sum -c activation-inputs.sha256 > activation-input-check-before.txt
if [[ ${1:-} == --dry-run && $# == 1 ]]; then
  taskset -c 13,15 nice -n 19 "$PY" calibration.py plan
  exit
fi
[[ $# == 0 ]]
[[ ! -e calibration.started && ! -e controller.done ]]
finish() {
  local rc=$?
  printf '%s\n' "$rc" > controller.exit
  date -Is > controller.done
  if [[ $rc != 0 ]]; then
    python3 - <<'PY'
import datetime,json
with open('/home/ehrli/ngn-data/night-20261007/status.jsonl','a') as stream:
    stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='owned-data-calibration',event='fail',detail='Preserve partial cells; no automatic retry or larger generation'))+'\n')
PY
  fi
}
trap finish EXIT
ulimit -f 16384
taskset -c 13,15 nice -n 19 "$PY" calibration.py run
sha256sum -c calibration-inputs.sha256 > calibration-input-check-after.txt
sha256sum -c activation-inputs.sha256 > activation-input-check-after.txt

#!/usr/bin/env bash
# CPU validation follows the affine sample; a short CUDA window follows factorizer.
set -euo pipefail
[[ ${1:-} == cpu || ${1:-} == cuda ]]
MODE=$1
R=/home/ehrli/ngn-data/night-20261007
S=$R/session17b
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PYTHONPATH=$S/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() {
  python3 - "$R/status.jsonl" "$MODE-$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='ngnn4-batched',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  echo "$rc" > "$S/$MODE-controller.exit"
  date -Is > "$S/$MODE-controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve, no retry"; fi
}
trap finish EXIT
wait_for() {
  python3 - "$1" "$2" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
    fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
    assert p.poll(12*60*60*1000),'native dependency wait timed out'
    os.close(fd)
assert done.is_file(),'dependency did not write a completion marker'
PY
}
date -Is > "$MODE-controller.started"
st wait "pid $$; native completion wait; 12h bound; no live jobs interrupted"
if [[ $MODE == cpu ]]; then
  wait_for "$R/session16a/controller.done" "$R/session16a/controller.pid"
  sha256sum -c frozen-inputs.sha256 > cpu-input-check.txt
  mkdir source
  tar -xf base.tar -C source
  (cd source && git apply ../candidate.patch)
  export CUDA_VISIBLE_DEVICES=''
  st start 'CPU13,15; gathered/GEMM forward and gradients, full trainer suite'
  (cd source && taskset -c 13,15 nice -n 19 "$PY" -m unittest trainer.tests.test_ngnn4_batched trainer.tests.test_ngnn4 -v) > cpu-narrow.log 2>&1
  (cd source && taskset -c 13,15 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v) > cpu-full.log 2>&1
  python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd()
x=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,
       cpu=[13,15],nice=19,cuda_excluded=True,
       logs={n:hashlib.sha256((r/n).read_bytes()).hexdigest() for n in ('cpu-narrow.log','cpu-full.log')},
       source={n:hashlib.sha256((r/'source'/n).read_bytes()).hexdigest() for n in
               ('trainer/model.py','trainer/tests/test_ngnn4_batched.py')})
(r/'cpu-receipt.json').write_text(json.dumps(x,indent=2)+'\n')
PY
  st done 'full CPU trainer checks pass; CUDA cost verdict remains pending'
else
  wait_for "$S/cpu-controller.done" "$S/cpu-controller.pid"
  [[ $(cat cpu-controller.exit) == 0 ]]
  wait_for "$R/session9b/controller.done" "$R/session9b/controller.pid"
  [[ $(cat "$R/session9b/controller.exit") == 0 ]]
  sha256sum -c frozen-inputs.sha256 > cuda-input-check.txt
  python3 - <<'PY'
import hashlib,json
from pathlib import Path
r=Path.cwd();x=json.loads((r/'cpu-receipt.json').read_text());assert x['passed']
for n,sha in x['source'].items():
    assert hashlib.sha256((r/'source'/n).read_bytes()).hexdigest()==sha
PY
  if pgrep -af 'trainer\.train|/datagen |/claimtool |/sprt |baseline-probe|candidate-probe' > cuda-blocked.txt; then exit 1; fi
  free_bytes=$(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ')
  [[ $free_bytes -ge 26843545600 ]]
  unset CUDA_VISIBLE_DEVICES
  st start 'actual CUDA tests then fixed full-batch GEMM/BMM benchmark; no real trainer'
  timeout --signal=TERM --kill-after=10s 3m taskset -c 13,15 nice -n 19 "$PY" -m unittest \
    trainer.tests.test_ngnn4_batched.NGNN4BatchedTests.test_cuda_fp32_bf16_forward_backward_all_bucket_counts \
    trainer.tests.test_ngnn4.NGNN4Tests.test_cuda_fp32_bf16_gradients_and_fused_step -v > cuda-tests.log 2>&1
  ! rg -q 'skipped' cuda-tests.log
  if timeout --signal=TERM --kill-after=10s 10m taskset -c 13,15 nice -n 19 "$PY" benchmark.py > cuda-benchmark.log 2>&1; then
    st done 'all four complete fixed samples pass throughput and memory rules; review before topic keep'
  else
    rc=$?
    if [[ $rc == 3 ]]; then st shelve 'complete fixed sample failed its rule; no extension'; else exit "$rc"; fi
  fi
fi

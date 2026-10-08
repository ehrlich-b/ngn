#!/usr/bin/env bash
# One tiny CUDA control after the existing exclusive admission window.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session8b
A=/home/ehrli/ngn-data/claim-v3-20261008/admission-session3
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='king-factorizer-cuda',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/cuda-controller.exit"
  date -Is > "$S/cuda-controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; no automatic retry or training candidate"; fi
}
trap finish EXIT
cd "$S"
sha256sum -c cuda-inputs.sha256 > cuda-input-check-before.txt
st wait "pid $$; native pidfd for admission 552631; 12h timeout; no GPU until normal admission completion"
python3 - "$A" "$S" <<'PY'
import json,os,select,sys
from pathlib import Path
a,s=map(Path,sys.argv[1:]); pid=552631
command=Path(f'/proc/{pid}/cmdline').read_bytes()
assert str(a/'controller.sh').encode() in command
fd=os.pidfd_open(pid)
with os.fdopen(fd):
    (s/'cuda-waiter-ready.json').write_text(json.dumps(dict(pid=pid,command=command.decode().replace('\0',' '),
        mechanism='pidfd',timeout_hours=12))+'\n')
    assert select.select([fd],[],[],12*60*60)[0],'dependency timeout; preserve all existing work'
assert (a/'controller.done').is_file()
admission=json.loads((a/'admission-receipt.json').read_text())
assert admission['admitted'] and admission['games']==12
receipt=json.loads((s/'test-receipt.json').read_text())
assert receipt['tests_pass'] and receipt['integer_parity_cases']==64
PY
if pgrep -af 'trainer\.train|/datagen |/claimtool |/sprt |cutechess|fastchess|baseline-probe|candidate-probe' > "$S/cuda-blocked-processes.txt"; then
  st fail 'another evidence job owns the window; no CUDA work'; exit 1
fi
free_bytes=$(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ')
[[ $free_bytes =~ ^[0-9]+$ && $free_bytes -ge 26843545600 ]]
sha256sum -c cuda-inputs.sha256 > cuda-input-check-pre-test.txt
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 PYTHONPATH="$S/source"
unset CUDA_VISIBLE_DEVICES
cd "$S/source"
taskset -c 13,15 nice -n 19 "$PY" -c 'import torch; assert torch.cuda.is_available(), "CUDA unavailable"' > "$S/cuda-available.log" 2>&1
st start 'one bounded CUDA FP32/BF16 forward/gradient/fused-AdamW/export control; no strength candidate'
timeout 180s taskset -c 13,15 nice -n 19 "$PY" -m unittest \
  trainer.tests.test_king_factorizer.KingFactorizerTests.test_cuda_forward_gradients_and_bf16_step -v > "$S/cuda-tests.log" 2>&1
cd "$S"
sha256sum -c cuda-inputs.sha256 > cuda-input-check-after.txt
taskset -c 13,15 nice -n 19 "$PY" - <<'PY'
import datetime,hashlib,json
from pathlib import Path
import torch
r=Path.cwd(); log=(r/'cuda-tests.log').read_text()
assert 'Ran 1 test' in log and 'OK' in log and 'skipped' not in log
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,
    torch=torch.__version__,cuda=torch.version.cuda,gpu=torch.cuda.get_device_name(0),
    cpu=[13,15],nice=19,timeout_seconds=180,
    log_sha256=hashlib.sha256((r/'cuda-tests.log').read_bytes()).hexdigest(),tail=log.splitlines()[-6:])
(r/'cuda-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st done 'CUDA FP32/BF16 and fused-AdamW/export control passes; no real training recipe or candidate queued'

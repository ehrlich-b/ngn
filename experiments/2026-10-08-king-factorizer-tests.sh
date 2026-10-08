#!/usr/bin/env bash
# Optional training-tool checks, CPU only; no active source or binary is changed.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=${NGN_FACTOR_TEST_ROOT:-$R/session8}
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='king-factorizer-preparation',event=sys.argv[2],detail=sys.argv[3]))+'\n')
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
st start "pid $$; optional king factorizer; CPU 13,15; GPU excluded; no games"
taskset -c 13,15 nice -n 19 "$PY" -m unittest trainer.tests.test_king_factorizer -v > "$S/factorizer-tests.log" 2>&1
taskset -c 13,15 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v > "$S/trainer-tests.log" 2>&1
taskset -c 13,15 nice -n 19 "$PY" - "$S" <<'PY' > "$S/fixture.log" 2>&1
import hashlib,json,sys
from pathlib import Path
import torch
from trainer.export import export_state
from trainer.king_buckets import DEFAULT_MAP
from trainer.model import NGNN
from trainer.ngnp import to_fen
from trainer.refeval import Network
from trainer.tests.test_ngnn3 import positions
root=Path(sys.argv[1])
torch.set_num_threads(2)
torch.manual_seed(20261008)
model=NGNN(16,8,DEFAULT_MAP,True)
with torch.no_grad():
    model.w1_shared.normal_(0,0.005)
    model.clip_quantized()
export_state(model.state_dict(),root/'factorized.nnue')
folded={k:v for k,v in model.state_dict().items() if k!='w1_shared'}
folded['w1']=model.effective_w1().detach()
export_state(folded,root/'folded.nnue')
assert (root/'factorized.nnue').read_bytes()==(root/'folded.nnue').read_bytes()
net=Network(root/'factorized.nnue')
cases=[dict(fen=to_fen(pos),eval_cp=net.evaluate(pos),bucket=net.bucket(pos)) for pos in positions()]
assert len(cases)==64
receipt=dict(format='NGNN3',hidden=16,buckets=8,king_buckets=8,
             network_sha256=hashlib.sha256((root/'factorized.nnue').read_bytes()).hexdigest(),cases=cases)
(root/'factorized-evals.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(dict(cases=len(cases),folded_byte_identical=True,network_sha256=receipt['network_sha256'])))
PY
sha256sum "$R/session6/baseline-engine.test" > "$S/parity-binary-before.sha256"
NGN_NGNN3_PARITY_NET="$S/factorized.nnue" NGN_NGNN3_PARITY_EVALS="$S/factorized-evals.json" \
  taskset -c 13,15 nice -n 19 "$R/session6/baseline-engine.test" -test.run '^TestNGNN3TrainerParity$' -test.v > "$S/parity.log" 2>&1
sha256sum -c "$S/parity-binary-before.sha256" > "$S/parity-binary-after.txt"
cd "$S"
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
inputs=sorted(p for directory in ('trainer','testdata/ngnn1','testdata/ngnn2','testdata/ngnn3')
              for p in (r/'source'/directory).rglob('*') if p.is_file() and p.suffix in ('.py','.nnue','.json')
              and '__pycache__' not in p.parts and not p.name.startswith('._'))
logs=('factorizer-tests.log','trainer-tests.log','fixture.log','parity.log')
receipt=dict(source_sha256={str(p.relative_to(r/'source')):digest(p) for p in inputs},
             archive_sha256=digest(r/'source.tar'),overlay_sha256=digest(r/'overlay.tar'),
             tests_pass=True,cpu=[13,15],nice=19,gpu_used=False,
             cuda_factorizer_check='pending; required before real training',
             integer_parity_cases=64,folded_byte_identical=True,
             parity_binary_sha256=digest(r.parent/'session6/baseline-engine.test'),
             python=subprocess.check_output(['/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python','--version'],text=True).strip(),
             logs={p:dict(sha256=digest(r/p),tail=(r/p).read_text().splitlines()[-6:]) for p in logs})
(r/'test-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st done 'CPU narrow/full trainer suites and 64 Go integer parity cases pass; CUDA trial smoke remains pending'

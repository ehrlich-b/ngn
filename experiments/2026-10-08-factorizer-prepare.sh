#!/usr/bin/env bash
# Freeze amended trial tooling around the already verified private engine.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session9b
PREVIOUS=$R/session9
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
export GOMAXPROCS=2 GOCACHE=$S/go-cache OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export CUDA_VISIBLE_DEVICES='' PYTHONDONTWRITEBYTECODE=1
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='factorizer-trial-preparation',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/prepare.exit"
  date -Is > "$S/prepare.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; no retry or candidate launch"; fi
}
trap finish EXIT
cd "$S"
sha256sum -c prepare-inputs.sha256 > prepare-input-check-before.txt
mkdir source frozen
tar -xf source.tar -C source
st start "pid $$; amended controller controls, CPU 13,15; reuse verified private builds; no GPU or games"
export PYTHONPATH=$S/source
taskset -c 13,15 nice -n 19 "$PY" trial.py selftest "$S" > controls.json
[[ $(cat "$PREVIOUS/prepare.exit") == 0 ]]
cp "$PREVIOUS/frozen/"* frozen/
cp "$PREVIOUS/engine-build.log" "$PREVIOUS/parity-build.log" "$PREVIOUS/parity-control.log" .
cp "$R/kb8h512/config.json" kb8h512.config.json
cp "$R/kb8h512-long/config.json" kb8h512-long.config.json
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); source=r/'source'; digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
previous=json.loads((r/'original-build-receipt.json').read_text())
assert previous['source_commit']==(r/'source-commit.txt').read_text().strip()
assert previous['source_archive_sha256']==digest(r/'source.tar')
for name,sha in previous['binary_sha256'].items():
    assert digest(r/'frozen'/name)==sha,name
for name in ('engine-build.log','parity-build.log','parity-control.log'):
    assert digest(r/name)==previous['logs'][name],name
engine=json.loads((r/'engine-source-evidence.json').read_text())['inputs']
trainer=json.loads((r/'trainer-source-evidence.json').read_text())['inputs']
assert len(engine)==230 and len(trainer)==25
for path, row in engine.items():
    assert digest(source/path)==row['candidate'],path
for path, sha in trainer.items():
    assert digest(source/path)==sha,path
assert digest(source/'default.pgo')=='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0'
assert digest(r/'frozen/openings.txt')=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
assert (r/'frozen/parity-records.ngnp').stat().st_size==20*32
assert json.loads((r/'controls.json').read_text())['passed']
assert 'PASS' in (r/'parity-control.log').read_text()
receipt=dict(source_commit=(r/'source-commit.txt').read_text().strip(),
    source_archive_sha256=digest(r/'source.tar'),source_reconciled=True,engine_inputs=len(engine),trainer_inputs=len(trainer),
    binary_sha256={name:digest(r/'frozen'/name) for name in ('ngn','engine.test','sprt')},
    profile_sha256=digest(source/'default.pgo'),go_version=subprocess.check_output(['go','version'],text=True).strip(),
    goamd64='v3',build_cpu=[13,15],nice=19,synthetic_integer_parity_cases=64,
    reused_build_receipt_sha256=digest(r/'original-build-receipt.json'),
    engine_build_argv=previous['engine_build_argv'],parity_build_argv=previous['parity_build_argv'],
    logs={name:digest(r/name) for name in ('engine-build.log','parity-build.log','parity-control.log','controls.json')})
(r/'build-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
paths=[r/p for p in ('trial.py','trial.sh','checkpoint-wait.py','manifest.md','source.tar','source-commit.txt',
                    'engine-source-evidence.json','trainer-source-evidence.json','build-receipt.json','original-build-receipt.json',
                    'kb8h512.config.json','kb8h512-long.config.json')]
paths+=sorted(p for p in (r/'frozen').rglob('*') if p.is_file())
paths+=sorted(p for p in source.rglob('*') if p.is_file() and not p.name.startswith('._')
              and '__pycache__' not in p.parts)
(r/'trial-inputs.sha256').write_text(''.join(f'{digest(p)}  {p}\n' for p in paths))
PY
sha256sum -c prepare-inputs.sha256 > prepare-input-check-after.txt
sha256sum -c trial-inputs.sha256 > trial-input-check.txt
st done 'accepted source/binaries and candidate controller frozen; nomination, final activation manifest and CUDA review required before real training'

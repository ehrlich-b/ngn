#!/usr/bin/env bash
# CPU-only tool preparation and existing-data analysis; no real datagen or GPU job.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session10
PRIOR=$R/session9b
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
export GOMAXPROCS=2 GOCACHE=$S/go-cache OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export CUDA_VISIBLE_DEVICES='' PYTHONDONTWRITEBYTECODE=1
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='owned-calibration-preparation',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/prepare.exit"
  date -Is > "$S/prepare.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve evidence; no automatic retry"; fi
}
trap finish EXIT
cd "$S"
sha256sum -c prepare-inputs.sha256 > prepare-input-check-before.txt
[[ $(cat "$PRIOR/prepare.exit") == 0 ]]
mkdir source frozen
tar -xf source.tar -C source
export PYTHONPATH=$S/source
st start "pid $$; CPU 13,15; isolated datagen builds/tests and frozen gen0b diagnostic; GPU and tuner untouched"
python3 - <<'PY'
import hashlib,json
from pathlib import Path
r=Path.cwd(); sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
engine=json.loads((r/'engine-source-evidence.json').read_text())['inputs']
trainer=json.loads((r/'trainer-source-evidence.json').read_text())['inputs']
assert len(engine)==230 and len(trainer)==25
for path,row in engine.items():
    assert sha(r/'source'/path)==row['candidate'],path
for path,value in trainer.items():
    assert sha(r/'source'/path)==value,path
build=json.loads((r/'build-receipt.json').read_text())
assert sha(r/'source.tar')==build['source_archive_sha256']
assert (r/'source-commit.txt').read_text().strip()==build['source_commit']
PY
cp "$PRIOR/frozen/"* frozen/
taskset -c 13,15 nice -n 19 "$PY" calibration.py selftest > controls.json
taskset -c 13,15 nice -n 19 "$PY" calibration.py baseline > baseline.log 2>&1
cd source
taskset -c 13,15 nice -n 19 go test -p 2 ./internal/datagen ./internal/ngnp ./cmd/datagen ./cmd/datastat -count=1 > "$S/datagen-tests.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./internal/datagen ./internal/ngnp ./cmd/datagen ./cmd/datastat -count=1 > "$S/datagen-race.log" 2>&1
COMMIT=$(cat "$S/source-commit.txt")
taskset -c 13,15 nice -n 19 go build -p 2 -trimpath -buildvcs=false -pgo="$S/source/default.pgo" -ldflags "-X main.engineCommit=$COMMIT" -o "$S/frozen/datagen" ./cmd/datagen > "$S/datagen-build.log" 2>&1
taskset -c 13,15 nice -n 19 go build -p 2 -trimpath -buildvcs=false -pgo="$S/source/default.pgo" -o "$S/frozen/datastat" ./cmd/datastat > "$S/datastat-build.log" 2>&1
cd "$S"
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
build=json.loads((r/'build-receipt.json').read_text())
for name,value in build['binary_sha256'].items():
    assert sha(r/'frozen'/name)==value,name
assert json.loads((r/'controls.json').read_text())['passed']
receipt=dict(source_commit=build['source_commit'],source_archive_sha256=sha(r/'source.tar'),
    source_reconciled=True,engine_inputs=230,trainer_inputs=25,tests_passed=True,
    binary_sha256={name:sha(r/'frozen'/name) for name in ('datagen','datastat')},
    profile_sha256=sha(r/'source/default.pgo'),cpu=[13,15],nice=19,gpu_used=False,
    go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',
    build_argv=[['go','build','-p','2','-trimpath','-buildvcs=false','-pgo='+str(r/'source/default.pgo'),
        *(['-ldflags','-X main.engineCommit='+build['source_commit']] if name=='datagen' else []),
        '-o',str(r/'frozen'/name),'./cmd/'+name] for name in ('datagen','datastat')],
    logs={name:dict(sha256=sha(r/name),tail=(r/name).read_text().splitlines()[-5:])
        for name in ('datagen-tests.log','datagen-race.log','datagen-build.log','datastat-build.log')})
(r/'calibration-build.json').write_text(json.dumps(receipt,indent=2)+'\n')
paths=[r/p for p in ('calibration.py','calibration.sh','base-review.py','manifest.md','source.tar',
    'source-commit.txt','engine-source-evidence.json','trainer-source-evidence.json','build-receipt.json',
    'calibration-build.json','kb8h512.config.json','kb8h512-long.config.json','baseline-input.json')]
paths+=sorted(p for p in (r/'frozen').rglob('*') if p.is_file())
paths+=sorted(p for p in (r/'source').rglob('*') if p.is_file() and not p.name.startswith('._')
    and '__pycache__' not in p.parts)
(r/'calibration-inputs.sha256').write_text(''.join(f'{sha(p)}  {p}\n' for p in paths))
PY
sha256sum -c prepare-inputs.sha256 > prepare-input-check-after.txt
sha256sum -c calibration-inputs.sha256 > calibration-input-check.txt
if bash calibration.sh --dry-run > dry-unreviewed.log 2>&1; then
  printf '%s\n' 'Unreviewed dry run unexpectedly passed' >&2
  exit 1
fi
[[ ! -e calibration.started && ! -e nodes-5000 && ! -e nodes-25000 ]]
st done 'datagen tests/builds and owned-data baseline complete; real 256-game calibration awaits reviewed teacher activation'

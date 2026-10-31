#!/usr/bin/env bash
# Uses spare CPU lanes; preserves all existing campaign processes and binaries.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session17f
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$S/go-cache
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='ngnn4-output-fusion',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
cleanup() {
  local rc=$?
  echo "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; no retry or acceptance"; fi
}
trap cleanup EXIT
date -Is > "$S/controller.started"
st start "pid $$; CPU 13,15 tests then CPU 13 identity/timing; live jobs untouched"
python3 - "$R/session17b/cpu-controller.pid" "$R/session17b/cpu-controller.done" <<'PYWAIT'
import os,select,sys
from pathlib import Path
pid,done=map(Path,sys.argv[1:])
if not done.exists():
    fd=os.pidfd_open(int(pid.read_text()));p=select.poll();p.register(fd,select.POLLIN)
    assert p.poll(20*60*1000),'CPU test dependency timeout'
    os.close(fd)
assert done.is_file(),'CPU dependency lacks completion marker'
PYWAIT
cd "$S"
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
cd candidate
git apply ../candidate.patch
for kind in baseline candidate; do
  cd "$S/$kind"
  taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe > "$S/$kind-build.log" 2>&1
done
cd "$S/candidate"
export CUDA_VISIBLE_DEVICES='' OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export NGN_NGNN4_PARITY_NET=$S/random-h512-l16.nnue NGN_NGNN4_PARITY_EVALS=$S/random-h512-l16.json
taskset -c 13,15 nice -n 19 go test -p 2 ./engine -run '^TestNGNN4' -count=1 -v > "$S/fusion-test.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -tags purego ./engine -run '^TestNGNN4' -count=1 -v > "$S/fusion-purego.log" 2>&1
st differential-pass 'NGNN4 guarded output/pack/affine/scalar and 320 H512 integer parity controls'
taskset -c 13,15 nice -n 19 go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
taskset -c 13,15 nice -n 19 go vet -p 2 ./engine > "$S/vet.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./... -count=1 > "$S/full-race.log" 2>&1
st tests-pass 'engine short/race, full short/race and engine vet'
for kind in baseline candidate; do
  cd "$S/$kind"
  taskset -c 13 nice -n 19 go test -p 1 ./engine -run '^$' -bench '^BenchmarkNGNN4Output$' -benchtime=500ms -count=3 > "$S/$kind-inference.txt" 2>&1
done
cd "$S"
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
receipt=dict(source_commit=(r/'base-commit.txt').read_text().strip(),
             source_archive_sha256=digest(r/'base.tar'), patch_sha256=digest(r/'candidate.patch'),
             binary_sha256={k:digest(r/f'{k}-probe') for k in ('baseline','candidate')},
             profile_sha256=digest(r/'baseline/default.pgo'),
             net_sha256=digest(r/'random-h512-l16.nnue'),
             old_net_sha256=digest(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),
             go_version=subprocess.check_output(['go','version'],text=True).strip(),
             build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','./cmd/speedprobe'],
             goamd64='v3', build_test_cpus=[13,15], tests_pass=True)
(r/'build-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st timing-start '48 identities then 20 AB/BA blocks; CPU 13; SPSA and GPU feed stay active'
if python3 "$S/check.py" > "$S/check.log" 2>&1; then
  st timing-pass 'fixed rule passed; receipt requires review before promotion'
else
  rc=$?
  if [[ $rc == 3 ]]; then st timing-shelve 'complete fixed sample did not pass; no extension'; else exit "$rc"; fi
fi
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
st done 'see session17f/timing/receipt.json; no engine or net installed'

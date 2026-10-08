#!/usr/bin/env bash
# Uses spare CPU lanes; preserves all existing campaign processes and binaries.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session15
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$S/go-cache
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='nnue-deferred',event=sys.argv[2],detail=sys.argv[3]))+'\n')
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
taskset -c 13,15 nice -n 19 go test -p 2 ./engine -run '^(TestNGNNDeferredSequences|TestNGNN[123].*|TestNGNN1KernelRowsDifferential)$' -count=1 -v > "$S/deferred-test.log" 2>&1
st differential-pass 'deferred sequences and existing NGNN1/2/3 oracle/row controls'
taskset -c 13,15 nice -n 19 go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
taskset -c 13,15 nice -n 19 go vet -p 2 ./engine > "$S/vet.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./... -count=1 > "$S/full-race.log" 2>&1
st tests-pass 'engine short/race, full short/race and engine vet'
cd "$S"
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd(); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
receipt=dict(source_commit=(r/'base-commit.txt').read_text().strip(),
             source_archive_sha256=digest(r/'base.tar'), patch_sha256=digest(r/'candidate.patch'),
             binary_sha256={k:digest(r/f'{k}-probe') for k in ('baseline','candidate')},
             profile_sha256=digest(r/'baseline/default.pgo'),
             old_net_sha256=digest(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),
             go_version=subprocess.check_output(['go','version'],text=True).strip(),
             build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','./cmd/speedprobe'],
             goamd64='v3', build_test_cpus=[13,15], tests_pass=True)
(r/'build-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st timing-start '36 identities then 40 AB/BA blocks; CPU 13; SPSA and GPU feed stay active'
if python3 "$S/check.py" > "$S/check.log" 2>&1; then
  st timing-pass 'fixed rule passed; receipt requires review before promotion'
else
  rc=$?
  if [[ $rc == 3 ]]; then st timing-shelve 'complete fixed sample did not pass; no extension'; else exit "$rc"; fi
fi
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
st done 'see session15/timing/receipt.json; no engine or net installed'

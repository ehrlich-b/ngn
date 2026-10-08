#!/usr/bin/env bash
# Isolated pure-speed experiment; never replaces a running campaign binary.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session4
SP=$R/spsa/kb8
PAUSED=0
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$S/go-cache
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='nnue-cache-pair',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
cleanup() {
  local rc=$?
  if [[ $PAUSED == 1 ]]; then
    cd "$SP"
    setsid nohup bash "$R/session1/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null &
    echo $! > "$SP/spsa.pid"
    st spsa-resume "pid $(cat "$SP/spsa.pid")"
  fi
  if [[ $rc != 0 ]]; then st fail "rc $rc; no retry or acceptance"; fi
  date -Is > "$S/controller.done"
}
trap cleanup EXIT
date -Is > "$S/controller.started"
st start "pid $$; isolated build/tests, then fixed identity/timing with tuner paused"
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
  taskset -c 13,15 nice -n 19 go test -p 2 -c -o "$S/$kind-engine.test" ./engine > "$S/$kind-test-build.log" 2>&1
done
cd "$S/candidate"
taskset -c 13,15 nice -n 19 go test -p 2 ./engine -run '^(TestNGNN3.*|TestNGNN1KernelRowsDifferential)$' -count=1 -v > "$S/rows-test.log" 2>&1
st rows-pass 'NGNN3 cache/oracle/transitions/symmetry/reload and guarded kernel aliasing'
taskset -c 13,15 nice -n 19 go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
taskset -c 13,15 nice -n 19 go vet -p 2 ./engine > "$S/vet.log" 2>&1
st tests-pass 'engine short/race, full short and engine vet'
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
# Training cannot reach its gate during this bounded experiment. Refuse a
# changed tuner identity instead of touching unrelated process groups.
pid=$(cat "$SP/spsa.pid")
pgid=$(ps -o pgid= -p "$pid" | tr -d ' ')
[[ $pgid == "$pid" ]]
ps -p "$pid" -o args= | rg -F "$R/session1/spsa-resume.cmd" > tuner-before.txt
cp "$SP/spsa.jsonl" tuner-before.jsonl
PAUSED=1
kill -TERM -- "-$pid"
for ((i=0; i<30; i++)); do
  if ! ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then break; fi
  sleep 1
done
if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then
  PAUSED=0
  st fail 'tuner did not drain; no timing or duplicate tuner'; exit 1
fi
st timing-start "tuner $pid drained; CPU 4; 36 identities and exactly 40 timing blocks"
if python3 "$S/check.py" > "$S/check.log" 2>&1; then
  st timing-pass 'fixed rule passed; receipt requires human review before commit'
else
  rc=$?
  if [[ $rc == 3 ]]; then st timing-shelve 'fixed complete sample did not pass; no extension'; else exit "$rc"; fi
fi
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
st done 'see session4/timing/receipt.json; no engine or net installed'

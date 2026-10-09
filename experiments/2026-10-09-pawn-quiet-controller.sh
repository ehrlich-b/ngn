#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session20-pawn-history
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='pawn-quiet-history',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; code/resource checks only, no games or adoption"; }
trap finish EXIT
date -Is > controller.started
st wait "pid $$; native SEE667047 completion, then CPU12,14 checks and fixed CPU12 resource sample"
python3 - "$R/session19-see" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(6*60*60*1000),'SEE dependency timeout';os.close(fd)
assert (p/'controller.done').exists(),'SEE ended without terminal receipt'
assert int((p/'controller.exit').read_text())==0,'SEE code checks failed'
PY
sha256sum -c inputs.sha256 > input-check-before.txt
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 12,14 nice -n 19 gofmt -w candidate/engine/pawn_quiet*.go candidate/engine/moveorder.go candidate/engine/search.go
cp candidate/engine/pawn_quiet_ordering_test.go baseline/engine/
st tests-start 'own pawn-context quiet ordering; no new pruning/LMR or eval term'
set +e
(cd baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestPawnQuietOrderingConsumesCompletedRoot$' -count=1 ./engine) > baseline-witness.log 2>&1
rc=$?
set -e
[[ $rc != 0 ]]
rg -F 'completed root did not train its pawn context' baseline-witness.log > baseline-witness-failure.txt
unlink baseline/engine/pawn_quiet_ordering_test.go
(cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestPawnQuiet' -count=1 ./engine) > candidate-narrow.log 2>&1
for mode in engine-short engine-race full-short full-race; do
 pkg=./engine; [[ $mode != full-* ]] || pkg=./...
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" "$pkg" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-probe-build.log" 2>&1
done
st tests-pass 'baseline context witness fails; candidate controls/full short/race/vet/PGO pass'
export GOMAXPROCS=1
st resource-start 'exactly12 balanced AB/BA six-position1s blocks onCPU12; diagnostic only, not Elo or acceptance'
python3 resource-sample.py > resource.log 2>&1
sha256sum -c inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
p=Path.cwd();sha=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()
files=['engine/moveorder.go','engine/search.go','engine/pawn_quiet_history.go','engine/pawn_quiet_history_test.go','engine/pawn_quiet_ordering_test.go']
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,cpus=[12,14],nice=19,
             source_sha256={s:sha(p/'candidate'/s) for s in files},binary_sha256={k:sha(p/(k+'-ngn')) for k in ['baseline','candidate']},
             log_sha256={x.name:sha(x) for x in p.glob('*.log') if x.name!='controller.log'},
             resource_sample=json.loads((p/'resource.json').read_text()),games_launched=False)
(p/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st resource-done 'fixed sample terminal; future powered games require actual immediate-base replay after higher priorities'

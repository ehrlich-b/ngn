#!/usr/bin/env bash
# Test the rebased SEE and pawn-history candidates, without playing games.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session21
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session21-checks',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/checks.exit"; date -Is > "$S/checks.done"; st end "rc $rc; code checks only, no games/adoption"; }
trap finish EXIT
date -Is > checks.started
st capacity-wait 'C: recovery to32GiB; silent five-minute checks; full candidate checks on12,14'
python3 - <<'PY'
import os,time
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
for lane in see pawn; do
 st "$lane-start" 'immediate39eb128/be34532c; full short/race/vet/PGO, no games'
 mkdir -p "$lane/baseline" "$lane/candidate"
 tar -xf base.tar -C "$lane/baseline"
 tar -xf base.tar -C "$lane/candidate"
 (cd "$lane/candidate" && git apply "../../$lane.patch")
 if [[ $lane == pawn ]]; then
  taskset -c 12,14 nice -n 19 gofmt -w pawn/candidate/engine/pawn_quiet*.go pawn/candidate/engine/moveorder.go pawn/candidate/engine/search.go
  cp pawn/candidate/engine/pawn_quiet_ordering_test.go pawn/baseline/engine/
  set +e
  (cd pawn/baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestPawnQuietOrderingConsumesCompletedRoot$' -count=1 ./engine) > pawn/baseline-witness.log 2>&1
  rc=$?
  set -e
  [[ $rc != 0 ]]
  rg -F 'completed root did not train its pawn context' pawn/baseline-witness.log > pawn/baseline-witness-failure.txt
  unlink pawn/baseline/engine/pawn_quiet_ordering_test.go
  (cd pawn/candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestPawnQuiet' -count=1 ./engine) > pawn/narrow.log 2>&1
 fi
 for mode in full-short full-race; do
  flags=(); [[ $mode != *race ]] || flags=(-race)
  (cd "$lane/candidate" && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$lane/$mode.log" 2>&1
 done
 (cd "$lane/candidate" && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > "$lane/vet.log" 2>&1
 for kind in baseline candidate; do
  (cd "$lane/$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/$lane/baseline/default.pgo" -o "$S/$lane/$kind-ngn" .) > "$lane/$kind-build.log" 2>&1
  if [[ $lane == pawn ]]; then
   (cd "$lane/$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/$lane/baseline/default.pgo" -o "$S/$lane/$kind-probe" ./cmd/speedprobe) > "$lane/$kind-probe-build.log" 2>&1
  fi
 done
 if [[ $lane == pawn ]]; then
  export GOMAXPROCS=1
  st pawn-resource 'exactly12 balanced AB/BA blocks; six1s positions on12; diagnostic only'
  python3 pawn-resource.py > pawn/resource.log 2>&1
 fi
 python3 - "$lane" <<'PY'
import datetime,hashlib,json,sys
from pathlib import Path
p=Path(sys.argv[1]);sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
files=[s.split(' b/',1)[1] for s in Path(str(p)+'.patch').read_text().splitlines() if s.startswith('+++ b/')]
j=dict(passed=True,completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='39eb128',cpus=[12,14],source_sha256={f:sha(p/'candidate'/f) for f in files},binaries={k:sha(p/(k+'-ngn')) for k in ['baseline','candidate']},games_launched=False)
if str(p)=='pawn':j['resource']=json.loads((p/'resource.json').read_text())
(p/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
 st "$lane-pass" 'full checks/builds terminal; actual-base review needed before any games'
done

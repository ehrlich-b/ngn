#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-time
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='time-admission',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/controller.exit"; date -Is > "$S/controller.done"; st end "rc $rc; tests/build only; no game verdict or promotion"; }
trap finish EXIT
cd "$S"
date -Is > controller.started
st wait "pid $$; native completion of cluster-TT tests, then CPU13,15 tests"
python3 - "$R/session19-tt/controller.done" "$R/session19-tt/controller.pid" <<'PY2'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
    fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
    assert p.poll(2*60*60*1000),'TT test dependency timeout';os.close(fd)
assert done.exists(),'TT tests ended without marker'
PY2
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
cp candidate/engine/time_admission_test.go baseline/engine/time_admission_test.go
cp "$R/session19-data/time-trace_test.go.txt" candidate/engine/campaign_time_trace_test.go
st tests-start 'admission regressions must fail baseline; full candidate short/race suites'
set +e
(cd baseline && taskset -c 13,15 nice -n 19 go test -p 2 -run '^TestIterationAdmissionChecks(Clock|HardDeadline)WithoutThrottle$' -count=1 ./engine) > baseline-regression.log 2>&1
rc=$?
set -e
[[ $rc == 1 ]]
rg -q 'expired soft projection admitted another depth' baseline-regression.log
rg -q 'missed hard admission deadline' baseline-regression.log
(cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -run 'TestIterationAdmission|TestTerminationObserver' -count=1 -v ./engine) > candidate-narrow.log 2>&1
for mode in engine-short engine-race full-short full-race; do
 pkg=./engine; [[ $mode != full-* ]] || pkg=./...
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" "$pkg" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
done
export NGN_CAMPAIGN_TRACE_NET=$R/kb8h512/best.nnue GOMAXPROCS=1
(cd candidate && timeout 10m taskset -c 13,15 nice -n 19 go test -p 1 -run '^TestCampaignLongClockTrace$' -count=1 -v ./engine) > time-trace.log 2>&1
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
files=['engine/time.go','engine/time_admission_test.go']
x=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,cpus=[13,15],nice=19,source_sha256={n:sha(r/'candidate'/n) for n in files},binary_sha256={k:sha(r/f'{k}-ngn') for k in ('baseline','candidate')},logs_sha256={n:sha(r/n) for n in ['baseline-regression.log','candidate-narrow.log','engine-short.log','engine-race.log','full-short.log','full-race.log','vet.log','time-trace.log']},games_launched=False)
(r/'receipt.json').write_text(json.dumps(x,indent=2)+'\n')
PY
st tests-pass 'full candidate suites and PGO builds pass; freeze actual accepted base/net before powered long-clock gate'

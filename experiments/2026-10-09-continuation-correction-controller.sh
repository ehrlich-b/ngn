#!/usr/bin/env bash
# Isolated code checks and a fixed cost diagnostic; no games or adoption.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session24-continuation
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='continuation-correction',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; checks/resource only, no strength verdict or adoption"
}
trap finish EXIT
date -Is > controller.started
st wait 'native qsearch-controller completion; then C:>=32GiB; tests12,14 / fixed cost sample14'
python3 - "$R/session23-qstatic" <<'PY'
import os,select,sys,time
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(40*3600*1000),'qsearch dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'qsearch controller ended without marker'
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
st tests-start 'isolated39eb128 temporal residual; no qsearch-TT/key/SEE/pawn/time overlay'
for kind in baseline witness candidate; do
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
done
(cd candidate && git apply ../candidate.patch)
# The witness needs the candidate table/helper types to compile, but retains
# pristine base search.go. This proves the missing consumer, not a compile error.
cp candidate/engine/{moveorder.go,continuation_correction.go,continuation_correction_test.go} witness/engine/
set +e
(cd witness && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestContinuationQuietHorizonWitness$' -count=1 -v ./engine) > baseline-witness.log 2>&1
rc=$?
set -e
[[ $rc == 1 ]]
rg -q 'quiet horizon ignored temporal residual' baseline-witness.log
(cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestContinuation' -count=1 -v ./engine) > candidate-narrow.log 2>&1
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-probe-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
files=[s.split(' b/',1)[1] for s in Path('candidate.patch').read_text().splitlines() if s.startswith('+++ b/')]
j=dict(code_checks_passed=True,base_source='39eb128',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
 go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',cpus=[12,14],nice=19,
 source_sha256={f:sha('candidate/'+f) for f in files},binaries={f:sha(f) for f in ['baseline-ngn','candidate-ngn','baseline-probe','candidate-probe']},
 inputs={f:sha(f) for f in ['base.tar','candidate.patch','baseline/default.pgo']},games_launched=False,strength_verdict=None)
Path('build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st tests-pass 'witness/narrow/full short/full race/vet/PGO passed; exactly12 balanced cost blocks next'
export GOMAXPROCS=1
python3 resource.py > resource.log 2>&1
st resource-done 'fixed cost diagnostic only; actual-base A/A and powered games remain required'

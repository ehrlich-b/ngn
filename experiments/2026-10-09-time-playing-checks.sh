#!/usr/bin/env bash
# Check two separate actual-base time candidates after the bounded GPU pilot.
# No trajectories, strength games, or automatic adoption.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session36-time-checks
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session36-time-checks',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; current-source checks only; games remain required"; }
trap finish EXIT
date -Is > controller.started
st wait 'native session35 pilot completion; then tests/builds12,14; no strength games'
python3 - "$R/session35-ngnn4-pilot" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(60*60*1000),'bounded pilot dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'pilot exited without completion marker'
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
sha256sum -c inputs.sha256 > input-check-before.txt
mkdir baseline
tar -xf base.tar -C baseline
(cd baseline && timeout 10m taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/baseline-ngn" .) > baseline-build.log 2>&1
for spec in candidate:candidate.patch:TestIterationAdmission:time_admission_test.go horizon:horizon.patch:TestTimeHorizon:time_horizon_test.go; do
 IFS=: read -r kind patch pattern testfile <<< "$spec"
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
 (cd "$kind" && git apply "../$patch")
 st checks-start "ac658e2; isolated $kind patch; narrow/full short/full race/vet; no stacked changes"
 (cd "$kind" && timeout 10m taskset -c 12,14 nice -n 19 go test -p 2 -run "^$pattern" -count=1 -v ./engine) > "$kind-narrow.log" 2>&1
 (cd "$kind" && timeout 20m taskset -c 12,14 nice -n 19 go test -p 2 -short ./... -count=1) > "$kind-full-short.log" 2>&1
 (cd "$kind" && timeout 20m taskset -c 12,14 nice -n 19 go test -p 2 -short -race ./... -count=1) > "$kind-full-race.log" 2>&1
 (cd "$kind" && timeout 10m taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > "$kind-vet.log" 2>&1
 (cd "$kind" && timeout 10m taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
 python3 - "$kind" "$patch" "$testfile" <<'PY'
import datetime,hashlib,json,platform,subprocess
import sys
from pathlib import Path
kind,patch,testfile=sys.argv[1:]
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',
 candidate_topic={'candidate':'night/time-playing','horizon':'night/time-horizon'}[kind],
 change={'candidate':'read clock at iteration admission only','horizon':'remaining-move estimate floor10 to25 only'}[kind],
 go_version=subprocess.check_output(['go','version'],text=True).strip(),
 goamd64='v3',machine='Ryzen9800X3D / WSL2',os=platform.platform(),cpus=[12,14],nice=19,
 checks=['four narrow tests','full-short ./...','full-race ./...','go vet ./...','baseline and candidate explicit-PGO build'],
 inputs={n:sha(n) for n in ['base.tar',patch,'controller.sh','manifest.md','baseline/default.pgo']},
 candidate_files={n:sha(kind+'/'+n) for n in ['engine/time.go','engine/'+testfile]},
 binaries={n:sha(n) for n in ['baseline-ngn',kind+'-ngn']},strength_games=0,automatic_adoption=False)
Path(kind+'-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
 st checks-done "$kind-receipt.json; separate actual-base code ready; long-clock strength gate still required"
done
sha256sum -c inputs.sha256 > input-check-after.txt

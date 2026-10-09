#!/usr/bin/env bash
# Revalidate the isolated time-admission patch and diagnose complete 120+1 banks.
# No strength gate or runtime adoption follows this diagnostic.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session23-time
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='time-current',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; diagnostic only; no strength verdict or adoption"
}
trap finish EXIT
date -Is > controller.started
st wait 'native row-affine completion; then C:>=32GiB; tests12,14 and trajectory14'
python3 - "$R/session22-row" <<'PY'
import os,select,sys,time
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(13*3600*1000),'row dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'row controller exited without marker'
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
st tests-start '39eb128/be34532c; isolated admission patch; no second time change'
for kind in baseline candidate; do
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
done
(cd candidate && git apply ../candidate.patch)
cp candidate/engine/time_admission_test.go baseline/engine/
set +e
(cd baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestIterationAdmissionChecks(Clock|HardDeadline)WithoutThrottle$' -count=1 ./engine) > baseline-regression.log 2>&1
rc=$?
set -e
[[ $rc == 1 ]]
rg -q 'expired soft projection admitted another depth' baseline-regression.log
rg -q 'missed hard admission deadline' baseline-regression.log
unlink baseline/engine/time_admission_test.go
for kind in baseline candidate; do
 cp clock-trajectory_test.go.txt "$kind/engine/campaign_clock_trajectory_test.go"
 taskset -c 12,14 nice -n 19 gofmt -w "$kind/engine/campaign_clock_trajectory_test.go"
 (cd "$kind" && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestCampaignTrajectoryBankAccounting$|^TestTerminationObserver' -count=1 ./engine) > "$kind-observer.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestIterationAdmission' -count=1 -v ./engine) > candidate-narrow.log 2>&1
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go test -p 2 -c -pgo="$S/baseline/default.pgo" -o "$S/$kind-trace" ./engine) > "$kind-trace-build.log" 2>&1
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
x=dict(base_source='39eb128',net='be34532c',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
 go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',cpus_build=[12,14],cpus_trace=[14],nice=19,
 source_sha256={f:sha('candidate/'+f) for f in ['engine/time.go','engine/time_admission_test.go','engine/campaign_clock_trajectory_test.go']},
 binaries={f:sha(f) for f in ['baseline-ngn','candidate-ngn','baseline-trace','candidate-trace']},
 inputs={f:sha(f) for f in ['base.tar','candidate.patch','clock-trajectory_test.go.txt','net.nnue','baseline/default.pgo']},
 schedule=[['baseline',0],['candidate',0],['candidate',1],['baseline',1]],
 tc='120+1',max_plies=160,strength_games=False,decision_rule='diagnostic only; games decide strength later on the then-current accepted base')
Path('trace-manifest.json').write_text(json.dumps(x,indent=2)+'\n')
PY
st trace-start 'exactly four complete120+1 self-play trajectories; base0/candidate0/candidate1/base1; no Elo verdict'
export GOMAXPROCS=1 NGN_CLOCK_TRAJECTORY_NET=$S/net.nnue
for spec in baseline:0 candidate:0 candidate:1 baseline:1; do
 kind=${spec%:*}; case_id=${spec#*:}
 export NGN_CLOCK_TRAJECTORY_VARIANT=$kind NGN_CLOCK_TRAJECTORY_CASE=$case_id
 timeout 12m taskset -c 14 nice -n 19 "$S/$kind-trace" -test.run '^TestCampaignClockTrajectory$' -test.timeout 11m -test.v > "trajectory-$kind-$case_id.log" 2>&1
done
python3 trajectory-review.py > trajectory-review.log 2>&1
st trace-done 'full banks, tail fractions and checkpoints in trajectory-summary.json; no strength verdict'

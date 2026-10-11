#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s56/admission
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$S"
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; }
trap finish EXIT
date -Is > checks.started
mkdir candidate
tar -xf base.tar -C candidate
(cd candidate && git apply ../tt-refresh.patch && git apply ../2026-10-10-time-quiet-admission.patch)
(cd candidate && timeout 10m go test -p 2 -run '^TestIterationAdmission' -count=1 -v ./engine) > narrow.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short ./engine -count=1) > engine-short.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short -race ./engine -count=1) > engine-race.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short ./... -count=1) > full-short.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short -race ./... -count=1) > full-race.log 2>&1
(cd candidate && timeout 10m go vet -p 2 ./...) > vet.log 2>&1
(cd candidate && timeout 10m go build -p 2 -pgo="$S/candidate/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
assert sha(p/'baseline-ngn')=='c14f03d8db4d00a3c43a1944e389b60cd5c52b8649c665e370cea1dcb3c0b848'
assert sha(p/'base.nnue')=='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
assert sha(p/'2026-10-10-time-quiet-admission.patch')=='76efa5266d49c5bf3f8106d2f9907a1e55066efa0637ac8c84b1cb0bcdc01dd7'
names=['tt-refresh.patch','base.tar','2026-10-10-time-quiet-admission.patch','baseline-ngn','candidate-ngn','base.nnue','checks.sh','manifest.md','candidate/default.pgo','candidate/engine/time.go','candidate/engine/time_admission_test.go','narrow.log','engine-short.log','engine-race.log','full-short.log','full-race.log','vet.log','build.log']
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source='ed34480',candidate='accepted TT refresh base plus unchanged admission patch76efa526 only',base_net='a4d90b41',machine='forge Debian13/Ryzen6900HX',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',cpus=[5,6,7],nice=10,memory_max_bytes=2*1024**3,host_pinning='on',checks=['four existing admission controls','engine/full short/race','vet','explicit accepted-PGO candidate build'],concurrent_workloads=['SPSA final-vector combination checks8-11/2GiB','reader117631 untouched','rig factorizer games0-11/3GiB; no training'],files_sha256={n:sha(p/n) for n in names},games_launched=False,decision='manual nomination after TT/Rodent; rig freshAA40 then60+0.6 SPRT0,+6/min400/cap4000; H1ANDprintedANDunroundedlower95>0/zero faults/intact inputs')
(p/'checks-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY

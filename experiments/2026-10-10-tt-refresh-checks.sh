#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s53/tt-refresh
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$S"
st() { python3 - status.jsonl "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session53-tt-refresh',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; st end "rc $rc; manual game nomination required"; }
trap finish EXIT
date -Is > checks.started
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch && gofmt -w engine/tt_refresh_test.go)
cp candidate/engine/tt_refresh_test.go baseline/engine/
set +e
(cd baseline && timeout 10m go test -p 2 -run '^TestTTMatchingRefreshWindow$' -count=1 -v ./engine) > baseline-witness.log 2>&1
rc=$?
set -e
test "$rc" = 1
rg -q -- '--- FAIL: TestTTMatchingRefreshWindow' baseline-witness.log
st baseline-witness 'new matching-window test fails on unmodified immediate source'
(cd candidate && timeout 10m go test -p 2 -run '^Test(TTMatchingRefresh|TTRefreshAllowance|ClusterTT|SynchronizedCacheReturnsOnlyCoherentRecords)' -count=1 -v ./engine) > narrow.log 2>&1
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
assert sha(p/'baseline-ngn')=='84431b35d3ae8ef92c6e9f1b967cc01ad35b400a1f812c347fe6f0dcbe55c2e1'
assert sha(p/'base.nnue')=='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
names=['base.tar','candidate.patch','baseline-ngn','candidate-ngn','base.nnue','checks.sh','manifest.md','candidate/default.pgo','candidate/engine/cache.go','candidate/engine/tt_refresh_test.go','baseline-witness.log','narrow.log','engine-short.log','engine-race.log','full-short.log','full-race.log','vet.log','build.log']
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source='9bfbae4',candidate='immediate source plus same-position retention allowance only',base_net='a4d90b41',machine='forge Debian13/Ryzen6900HX',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',cpus=[5,6,7],nice=10,memory_max_bytes=2*1024**3,host_pinning='on',checks=['baseline test failure witness','new matching policy/cluster/concurrency controls','engine/full short/race','vet','explicit accepted-PGO build'],concurrent_workloads=['SPSA8-11/2.5GiB/two games','reader untouched','rig Rodent/factorizer separate'],files_sha256={n:sha(p/n) for n in names},games_launched=False,decision='manual rig nomination after Rodent; ownAA40 then fixed400 at60+0.6; printedANDexact lower95>0/zero faults/intact inputs only')
(p/'checks-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st checks-pass 'candidate ready for manual nomination; no adoption'

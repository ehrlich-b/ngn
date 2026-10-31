#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session100-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" <<'PY'
import datetime,json,sys
with open(sys.argv[1], 'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session100-speed',event=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; st "checks-exit-$rc"; }
trap finish EXIT
st checks-start
for kind in baseline s1 s2; do
    mkdir "$kind"
    tar xf base.tar -C "$kind"
    if [ "$kind" != baseline ]; then tar xf "$kind.tar" -C "$kind"; fi
done
for kind in s1 s2; do
    cd "$S/$kind"
    if [ "$kind" = s1 ]; then
        gofmt -w engine/cache_alloc_linux.go engine/cache_alloc_other.go engine/cache_alloc_linux_test.go
        go test -p 2 -short -run 'Test(TTAllocation|ClusterTT|IndependentCache|TT)' ./engine -count=1 -v > "$S/$kind-focused.log" 2>&1
    else
        gofmt -w engine/moveorder.go engine/history_policy_experiment.go engine/history_storage_test.go
        go test -p 2 -short -run 'Test(CompactHistory|H1|History|CaptureHistory)' ./engine -count=1 -v > "$S/$kind-focused.log" 2>&1
    fi
    go test -p 2 -short ./engine -count=1 > "$S/$kind-engine-short.log" 2>&1
    go test -p 2 -short -race ./engine -count=1 > "$S/$kind-engine-race.log" 2>&1
    go test -p 2 -short ./... -count=1 > "$S/$kind-full-short.log" 2>&1
    go test -p 2 -short -race ./... -count=1 > "$S/$kind-full-race.log" 2>&1
    go vet -p 2 ./... > "$S/$kind-vet.log" 2>&1
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-windows.exe" . > "$S/$kind-windows-build.log" 2>&1
    st "$kind-suites-pass"
done
for kind in baseline s1 s2; do
    cd "$S/$kind"
    go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" . > "$S/$kind-build.log" 2>&1
    go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe > "$S/$kind-probe-build.log" 2>&1
done
cd "$S"
st identity-start
python3 identity.py > identity.log 2>&1
st identity-pass
python3 - <<'PY'
import datetime,hashlib,json,pathlib,platform,subprocess
s=pathlib.Path.cwd()
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
receipt=dict(base_source='a0ebe32 / accepted playing 9ec5d31',net='/home/ehrli/ngn-data/night-20261007/session82/net-final115/candidate.nnue',go=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),cpus=[13,15],nice=19,tests_pass=True,identity=json.loads((s/'identity.json').read_text()),binaries={k:sha(s/(k+'-ngn')) for k in ('baseline','s1','s2')},profile_sha256=sha(s/'baseline/default.pgo'),utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),verdict='PREPARED; exclusive profile panel and balanced speed gate pending; no adoption')
(s/'checks-result.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY

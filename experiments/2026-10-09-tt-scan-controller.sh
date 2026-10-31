#!/usr/bin/env bash
# Pure speed only; preserves all live campaign sources, binaries and jobs.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session25-tt-scan
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='tt-scan-speed',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; review fixed speed verdict before any adoption"
}
trap finish EXIT
date -Is > controller.started
st wait 'native continuation-controller completion; then C:>=32GiB; tests12,14 / exact identity and fixed20 speed blocks14'
python3 - "$R/session24-continuation" <<'PY'
import os,select,sys,time
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(48*3600*1000),'continuation dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'continuation ended without completion marker'
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
st tests-start 'original39eb128 TT scan only; no provenance key/SEE or search candidate overlay'
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 12,14 nice -n 19 gofmt -w candidate/engine/tt_scan_test.go
# Same new diagnostic fixture on the pristine baseline: no baseline runtime edit.
cp candidate/engine/tt_scan_test.go baseline/engine/
for kind in baseline candidate; do
 (cd "$kind" && NGN_PRIVATE_CACHE_DUMP="$S/$kind-cache.bin" NGN_TT_SCAN_DUMP="$S/$kind-collisions.bin" taskset -c 12,14 nice -n 19 go test -p 2 -run '^(TestTTScan.*|TestPrivateCacheCompatibilityDump|TestClusterTT.*|TestIndependentCacheSynchronizedCoherence|TestCache.*|TestSynchronizedCache.*)$' -count=1 -v ./engine) > "$kind-tt.log" 2>&1
done
cmp baseline-cache.bin candidate-cache.bin
cmp baseline-collisions.bin candidate-collisions.bin
for mode in engine-short engine-race full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 package=./engine; [[ $mode != full-* ]] || package=./...
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" "$package" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
assert (r/'baseline-cache.bin').read_bytes()==(r/'candidate-cache.bin').read_bytes()
assert (r/'baseline-collisions.bin').read_bytes()==(r/'candidate-collisions.bin').read_bytes()
j=dict(base_source='39eb128',source_commit=(r/'base-commit.txt').read_text().strip(),utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
 source_archive_sha256=sha(r/'base.tar'),patch_sha256=sha(r/'candidate.patch'),source_sha256={f:sha(r/'candidate'/f) for f in ['engine/cache.go','engine/tt_scan_test.go']},
 binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},profile_sha256=sha(r/'baseline/default.pgo'),old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),
 tt_dump_identity=True,collision_dump_identity=True,diagnostic_sha256={n:sha(r/n) for n in ['baseline-cache.bin','baseline-collisions.bin']},
 go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',build_test_cpus=[12,14],nice=19,tests_pass=True,
 build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','./cmd/speedprobe'])
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st tests-pass 'retention fields, exact TT traces, narrow/full short/race/vet; 36 search identities and20 fixed3s speed blocks next'
export GOMAXPROCS=1
if python3 check.py > check.log 2>&1; then
 st speed-keep 'complete fixed rule passed; review receipts/source before topic commit or adoption'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete fixed sample failed its rule; no extension'; else exit "$rc"; fi
fi

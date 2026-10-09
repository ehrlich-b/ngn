#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session33-null-frame
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='null-frame-speed',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review fixed evidence before retention"; }
trap finish EXIT
date -Is > controller.started
st wait 'native VNNI512-speed26077 completion; then full checks12,14 and fixed40-block speed14'
python3 - "$R/session32-vnni512/speed-controller.done" <<'PY'
import os,select,sys
from pathlib import Path
if not Path(sys.argv[1]).exists():
 fd=os.pidfd_open(26077);p=select.poll();p.register(fd,select.POLLIN)
 assert p.poll(2*3600*1000),'preceding fixed sample timeout'
 os.close(fd)
assert Path(sys.argv[1]).exists(),'preceding sample ended without receipt'
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=27*1024**3
PY
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
st tests-start 'f8686be accepted base; null-frame ownership only; no other candidate overlay'
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 12,14 nice -n 19 gofmt -w candidate/engine/ngnn1.go candidate/engine/ngnn1_null_frame_test.go
cp candidate/engine/ngnn1_null_frame_test.go baseline/engine/
if (cd baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestNGNNNullFrameOwnership$' -count=1 -v ./engine) > baseline-witness.log 2>&1; then
 exit 1
fi
grep -q 'copied a null accumulator' baseline-witness.log
for tag in avx2 purego; do
 flags=(); [[ $tag != purego ]] || flags=(-tags purego)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 "${flags[@]}" -run '^TestNGNN' -count=1 -v ./engine) > "differential-$tag.log" 2>&1
done
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
j=dict(base_source='f8686be',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source_archive_sha256=sha(r/'base.tar'),patch_sha256=sha(r/'candidate.patch'),
 source_sha256={f:sha(r/'candidate'/f) for f in ['engine/ngnn1.go','engine/ngnn1_null_frame_test.go']},binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},
 profile_sha256=sha(r/'baseline/default.pgo'),old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),
 go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',build_test_cpus=[12,14],nice=19,tests_pass=True,
 build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','./cmd/speedprobe'])
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st tests-pass 'old-fail/new-pass; AVX2/purego/full short/race/vet; then36 exact identities and40 balanced speed blocks'
if python3 check.py > check.log 2>&1; then
 st speed-keep 'fixed rule passes; raw/source review required before commit and integration'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete fixed sample failed rule; no extension'; else exit "$rc"; fi
fi
sha256sum -c frozen-inputs.sha256 > input-check-after.txt

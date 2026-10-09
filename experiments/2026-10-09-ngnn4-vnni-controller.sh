#!/usr/bin/env bash
# One original affine-kernel candidate, held behind capacity and existing CPU work.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session31-vnni
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='ngnn4-vnni',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > "$S/controller.exit"
 date -Is > "$S/controller.done"
 st end "rc $rc; speed/cost evidence only, no training or adoption"
}
trap finish EXIT
cd "$S"
date -Is > controller.started
st wait 'C:>=32GiB; CPU12,14 tests / CPU14 fixed sample; no match overlap on these CPUs'
python3 - <<'PY'
import os,time
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
st test-start 'verified aafa7fd NGNN4 four-row base; only runtime-selected VNNI dot changes candidate'
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 12,14 nice -n 19 gofmt -w candidate/engine/ngnn4_vnni_test.go candidate/engine/ngnn4_kernels_amd64.go
export NGN_NGNN4_PARITY_NET=$S/random-h512-l16.nnue NGN_NGNN4_PARITY_EVALS=$S/random-h512-l16.json
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-build.log" 2>&1
done
(cd baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestNGNN4' -count=1 -v ./engine) > baseline-ngnn4-tests.log 2>&1
for tag in avx2 purego; do
 flags=(); [[ $tag != purego ]] || flags=(-tags purego)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 "${flags[@]}" -run '^TestNGNN4' -count=1 -v ./engine) > "differential-$tag.log" 2>&1
done
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 package=./engine; [[ $mode != full-* ]] || package=./...
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" "$package" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
grep -q 'VNNI hardware and OS state admitted' differential-avx2.log
! grep -q '\-\-\- SKIP: TestNGNN4VNNIDirectAndFallback' differential-avx2.log
st tests-pass 'VNNI direct and forced AVX2 fallback; all widths guarded rows/packed/scalar;320 integer cases; full short/race/vet'
for kind in baseline candidate; do
 (cd "$kind" && GOMAXPROCS=1 taskset -c 14 nice -n 19 go test -p 1 -run '^$' -bench '^BenchmarkNGNN4Output$' -benchtime=500ms -count=3 ./engine) > "$kind-inference.txt" 2>&1
done
python3 - <<'PY'
import hashlib,json,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
files=['engine/ngnn4.go','engine/ngnn4_kernels_amd64.go','engine/ngnn4_kernels_amd64.s','engine/ngnn4_kernels_generic.go','engine/ngnn4_rows_test.go','engine/ngnn4_vnni_test.go']
j=dict(runtime_base='39eb128',architecture_base='aafa7fd',source_archive_sha256=sha(r/'base.tar'),patch_sha256=sha(r/'candidate.patch'),
 binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},net_sha256=sha(r/'random-h512-l16.nnue'),
 old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),ngnn3_net_sha256=sha(r/'accepted-kb8.nnue'),profile_sha256=sha(r/'baseline/default.pgo'),
 source_sha256={f:sha(r/'candidate'/f) for f in files},go_version=subprocess.check_output(['go','version'],text=True).strip(),
 build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','./cmd/speedprobe'],goamd64='v3',build_test_cpus=[12,14],tests_pass=True)
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st timing-start '48 exact identities, exactly20 balanced3s AB/BA blocks CPU14; if keep,12 architecture-cost blocks'
if python3 check.py > check.log 2>&1; then
 st speed-keep 'fixed speed rule passed; review raw results/source before topic commit; architecture-cost.json is budget evidence'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete fixed sample failed rule; no extension'; else exit "$rc"; fi
fi
sha256sum -c frozen-inputs.sha256 > input-check-after.txt

#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session19-data
A=/home/ehrli/nnue-owned-k4-20260920/archive-a1
OLD=/home/ehrli/nnue-owned-k4-20260920/data-farseer-20260928/build/k4sample
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
X=$D/tool/target/release/ngnk4archive
CONFLICTS=/home/ehrli/nnue-owned-k4-20260920/runs/k4-production-pilot-20260921/selection/conflicts.bin
T80=/home/ehrli/nnue-public-toolchain-20260906/data/t80-2022-08/test80-2022-08-aug-16tb7p.v6-dd.min.binpack
export PATH=/usr/local/go/bin:/home/ehrli/nnue-public-toolchain-20260906/install/rust-1.88.0/bin:$PATH
export CARGO_HOME=$A/cargo-home GOMAXPROCS=1 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='data-volume',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$D/controller.exit"; date -Is > "$D/controller.done"; st end "rc $rc; corpus preparation only, no training/promotion"; }
trap finish EXIT
wait_for() {
  python3 - "$1" "$2" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
    try: fd=os.pidfd_open(int(pidfile.read_text()))
    except ProcessLookupError: assert done.exists(),'dependency ended without completion marker'
    else:
        p=select.poll();p.register(fd,select.POLLIN)
        assert p.poll(12*60*60*1000),'dependency timeout'
        os.close(fd)
assert done.is_file(),'dependency missing completion marker'
PY
}
sc() {
  local name=$1 input=$2 keep=$3 seed=$4
  st scatter-start "$name; keep-per-mille $keep; CPU13,15 / nice19 / workers2"
  timeout --signal=TERM --kill-after=20s 3h taskset -c 13,15 nice -n 19 "$X" scatter --input "$input" --conflicts "$CONFLICTS" --probe-keys "$A/calibration-probe.keys" --output "$D/scatter-$name" --buckets 256 --workers 2 --seed "$seed" --keep-per-mille "$keep" > "$D/scatter-$name.stdout" 2> "$D/scatter-$name.log"
  st scatter-done "$name; see scatter-$name/scatter.json"
}
cd "$D"
date -Is > controller.started
st wait "pid $$; scout terminal -> accepted 120+1 trace -> tool checks/T80 statistics; full corpus waits CUDA windows and downloads"
wait_for "$R/session18d/controller.done" "$R/session18d/controller.pid"
mkdir trace-source
tar -xf accepted-source.tar -C trace-source
cp time-trace_test.go.txt trace-source/engine/campaign_time_trace_test.go
export NGN_CAMPAIGN_TRACE_NET=$R/kb8h512/best.nnue
export GOCACHE=$R/session17f/go-cache
st trace-start 'accepted NGNN3 KB8 H512; twelve 120+1 observer searches; no policy change'
(cd trace-source && timeout 10m taskset -c 13,15 nice -n 19 go test -p 1 -run '^TestCampaignLongClockTrace$' -count=1 -v ./engine) > time-trace.log 2>&1
st trace-done 'accepted 120+1 spending trace ready; no game verdict'
# Reuse the already owned external archive converter; only the small consume
# patch changes, and no helper/vendor source enters the NGN engine repository.
[[ $(sha256sum "$OLD/src/bin/archive.rs" | cut -d ' ' -f 1) == 4e298ee53850a3f8f25e01c84e50a1152f5302cc260bbe5a7b067929e23acb7f ]]
mkdir tool
cp -a "$OLD/src" "$OLD/Cargo.toml" "$OLD/Cargo.lock" tool/
(cd tool && patch -p1 < ../gather-consume.patch)
(cd tool && taskset -c 13,15 nice -n 19 cargo test --offline --release --bin ngnk4archive -j 2) > tool-tests.log 2>&1
(cd tool && taskset -c 13,15 nice -n 19 cargo build --offline --release --bin ngnk4archive -j 2) > tool-build.log 2>&1
taskset -c 13,15 nice -n 19 "$PY" gather-check.py "$D/gather-fixture" "$X" > gather-check.json 2> gather-check.log
sha256sum "$X" > tool.sha256
st tool-pass 'ordinary and consuming gathers byte-identical; only newly created source buckets may be consumed'
# Statistical reference needs no output positions and uses the same pinned
# converter source as the additional archives.
sc t80stats "$T80" 0 2026100900
date -Is > preparation.done
st preparation-done 'T80 score/result reference ready; preserving batched then sparse CUDA windows'
wait_for "$R/session17s/cuda-controller.done" "$R/session17s/cuda-controller.pid"
wait_for "$D/download.done" "$D/download.pid"
[[ $(cat download.exit) == 0 ]]
sc t76 "$D/raw/farseerT76.binpack" 1000 2026100901
"$PY" fit-source-scale.py --reference "$D/scatter-t80stats/scatter.json" --source t76="$D/scatter-t76/scatter.json" --out "$D/scales-t76.json" > "$D/scales-t76.log"
scale=$("$PY" -c 'import json,sys; print(json.load(open(sys.argv[1]))["sources"]["t76"]["raw_scale"])' "$D/scales-t76.json")
taskset -c 13,15 nice -n 19 "$X" gather --scatter "$D/scatter-t76@$scale" --map "$A/score-map.json" --output "$D/t76.bf" --manifest "$D/t76-manifest.json" --seed 2026100911 --consume-buckets true > gather-t76.stdout 2> gather-t76.log
# This downloaded copy was created by this campaign; keep its byte/hash receipt.
unlink "$D/raw/farseerT76.binpack"
st gather-done 'T76 retained; own raw/buckets removed after durable conversion'
sc t74stats "$D/raw/farseerT74.binpack" 0 2026100902
keep=$("$PY" - <<'PY'
import json
from pathlib import Path
d=Path.cwd();s=json.loads((d/'scatter-t74stats/scatter.json').read_text())['counts']
available=85*1024**3-(d/'t76.bf').stat().st_size
n=s['eligible_train']-s['excluded_conflict_key']
keep=min(1000,int(available/(32*n)*1000))
assert keep>0,'no room for additional source'
(d/'sample-plan.json').write_text(json.dumps({'extra_scatter_budget_bytes':85*1024**3,'t76_bytes':(d/'t76.bf').stat().st_size,'t74_eligible_nonconflict':n,'t74_keep_per_mille':keep,'rule':'size from source counts before any training; preserve C: floor and source diversity'},indent=2)+'\n')
print(keep)
PY
)
sc t74 "$D/raw/farseerT74.binpack" "$keep" 2026100903
"$PY" fit-source-scale.py --reference "$D/scatter-t80stats/scatter.json" --source t76="$D/scatter-t76/scatter.json" --source t74="$D/scatter-t74/scatter.json" --out "$D/scales.json" > "$D/scales.log"
scale=$("$PY" -c 'import json,sys; print(json.load(open(sys.argv[1]))["sources"]["t74"]["raw_scale"])' "$D/scales.json")
taskset -c 13,15 nice -n 19 "$X" gather --scatter "$D/scatter-t74@$scale" --map "$A/score-map.json" --output "$D/t74.bf" --manifest "$D/t74-manifest.json" --seed 2026100912 --consume-buckets true > gather-t74.stdout 2> gather-t74.log
unlink "$D/raw/farseerT74.binpack"
st corpus-done 'T80 + T76 + sampled T74; per-source scales and original T80 map; coverage review required before unchanged 58-epoch training recipe'

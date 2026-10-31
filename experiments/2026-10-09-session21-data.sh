#!/usr/bin/env bash
# Continue the incomplete T74 conversion; preserve the failed attempt.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session21
D=$R/session19-data
A=/home/ehrli/nnue-owned-k4-20260920/archive-a1
X=$D/tool/target/release/ngnk4archive
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PYTHONPATH=$R/session9b/source OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
mkdir -p "$S/data"
cd "$S/data"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session21-data',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/data/controller.exit"; date -Is > "$S/data/controller.done"; st end "rc $rc; corpus/coverage only, training requires review"; }
trap finish EXIT
date -Is > controller.started
st wait 'native provenance-controller completion; then C:70GiB capacity; unchanged194-per-mille T74 sample'
python3 - "$S" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(16*3600*1000),'provenance completion wait expired';os.close(fd)
assert (p/'controller.done').exists() and (p/'controller.exit').read_text().strip()=='0'
PY
st capacity-wait 'silent five-minute checks for70GiB;12h bound; T74 conversion on13,15'
python3 - <<'PY'
import os,time
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=70*1024**3:break
 assert time.monotonic()<deadline,'C: corpus-capacity wait expired'
 time.sleep(300)
PY
python3 "$D/space-guard.py" "$$" "$R/status.jsonl" > space-guard.log 2>&1 &
echo $! > space-guard.pid
st scatter-start 'T74 retry after discarded partial buckets; same seed2026100903/keep194; no training'
timeout --signal=TERM --kill-after=20s 3h taskset -c 13,15 nice -n 19 "$X" scatter --input "$D/raw/farseerT74.binpack" --conflicts /home/ehrli/nnue-owned-k4-20260920/runs/k4-production-pilot-20260921/selection/conflicts.bin --probe-keys "$A/calibration-probe.keys" --output "$S/data/scatter-t74" --buckets 256 --workers 2 --seed 2026100903 --keep-per-mille 194 > scatter.stdout 2> scatter.log
"$PY" "$D/fit-source-scale.py" --reference "$D/scatter-t80stats/scatter.json" --source "t76=$D/scatter-t76/scatter.json" --source "t74=$S/data/scatter-t74/scatter.json" --out scales.json > scales.log
scale=$("$PY" -c 'import json; print(json.load(open("scales.json"))["sources"]["t74"]["raw_scale"])')
taskset -c 13,15 nice -n 19 "$X" gather --scatter "$S/data/scatter-t74@$scale" --map "$A/score-map.json" --output "$S/data/t74.bf" --manifest "$S/data/t74-manifest.json" --seed 2026100912 --consume-buckets true > gather.stdout 2> gather.log
unlink "$D/raw/farseerT74.binpack"
st corpus-done 'T74 durable; only own compressed source removed; T76/T80 preserved'
taskset -c 13,15 nice -n 19 "$PY" - <<'PY' > coverage.log 2>&1
import importlib.util,json,time
from pathlib import Path
import numpy as np
from trainer.archive import ARCHIVE_DTYPE,board_hash
r=Path('/home/ehrli/ngn-data/night-20261007');s=r/'session21/data';d=r/'session19-data'
spec=importlib.util.spec_from_file_location('coverage',d/'coverage.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
sources=[('t80',Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf')),('t76',d/'t76.bf'),('t74',s/'t74.bf')]
results={};union=np.empty(0,dtype=np.uint64);started=time.monotonic()
for name,path in sources:
 rows=np.memmap(path,dtype=ARCHIVE_DTYPE,mode='r');sample=np.empty(0,dtype=np.uint64)
 for start in range(0,len(rows),262144):sample=m.update(sample,board_hash(rows[start:start+262144],20261006))
 union=m.update(union,sample);results[name]={'path':str(path),'bytes':path.stat().st_size,'records':len(rows),**m.estimate(sample)}
before=results['t80']['distinct'];after=m.estimate(union)
j=dict(method='bottom-65536 distinct occupied-board hashes; ngn-board-mix64-v1/20261006',sources=results,combined=after,estimated_unique_ratio=after['distinct']/before,record_ratio=sum(v['records'] for v in results.values())/results['t80']['records'],seconds=time.monotonic()-started,limits='occupied boards, not game groups or mirrored inputs; approximate correlated source/union errors',training_launched=False)
(s/'coverage.json').write_text(json.dumps(j,indent=2)+'\n');print(json.dumps(j))
PY
st coverage-done 'review distinct expansion, source scale fits and actual accepted base before unchanged58-epoch data-only training'

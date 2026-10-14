set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session83/sample
A=/home/ehrli/nnue-owned-k4-20260920/archive-a1
X=$R/session19-data/tool/target/release/ngnk4archive
mkdir -p "$D"
cd "$D"
echo $$ > controller.pid
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session83-sampledT74',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; no automatic training"; }
trap finish EXIT
python3 - "$R/session83" <<'PY'
import os,select,sys
from pathlib import Path
d=Path(sys.argv[1])
if not (d/'fullchecks.done').exists():
 try:fd=os.pidfd_open(int((d/'fullchecks-launch.pid').read_text()))
 except ProcessLookupError:pass
 else:select.select([fd],[],[]);os.close(fd)
assert (d/'fullchecks.exit').read_text().strip()=='0'
v=os.statvfs('/mnt/c')
assert v.f_bavail*v.f_frsize >= 25.5*1024**3 + 35902850688*1.02,'insufficient sample reconstruction capacity'
PY
# Reuse the existing campaign guard on this data job only, at27GiB/15s.
sed -e 's/poll.poll(60000)/poll.poll(15000)/' -e 's/free>=32\*1024\*\*3/free>=27*1024**3/' -e 's/owned-{parent}:/owned-{parent,os.getpid()}:/' "$R/session19-data/space-guard.py" > guard.py
python3 guard.py "$$" "$R/status.jsonl" > guard.log 2>&1 &
st start "controller$$; exact194-per-mille sample;13,15; preserve raw; no forge/training"
"$X" scatter --input "$R/session19-data/raw/farseerT74.binpack" --conflicts /home/ehrli/nnue-owned-k4-20260920/runs/k4-production-pilot-20260921/selection/conflicts.bin --probe-keys "$A/calibration-probe.keys" --output "$D/scatter-t74" --buckets 256 --workers 2 --seed 2026100903 --keep-per-mille 194 > scatter.stdout 2> scatter.log
st scatter-done 'gather uses original scale0.9827691877392202/map/seed2026100912;consume only own sample buckets'
"$X" gather --scatter "$D/scatter-t74@0.9827691877392202" --map "$A/score-map.json" --output "$D/t74.bf" --manifest "$D/t74-manifest.json" --seed 2026100912 --consume-buckets true > gather.stdout 2> gather.log
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
p=Path('t74.bf');h=hashlib.sha256()
with p.open('rb') as f:
 for block in iter(lambda:f.read(8*1024**2),b''):h.update(block)
receipt=dict(bytes=p.stat().st_size,sha256=h.hexdigest(),completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
receipt['matches_original']=receipt['bytes']==35902850688 and receipt['sha256']=='2024df427c42c2409ec815e93a98672f1863b58d6e8d50c49e15a7b8541add37'
Path('result.json').write_text(json.dumps(receipt,indent=2)+'\n')
assert receipt['matches_original'],'regenerated original corpus differs; preserve result, do not train'
PY
st ready 'exact original sampled corpus reconstructed; result.json; no automatic training'

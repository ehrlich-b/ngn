#!/usr/bin/env bash
# Same predeclared T74 sample/conversion, relocated to the authorized forge lane.
set -euo pipefail
S=/work/lanes/ngn-data-volume-20261010
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session41-t74-forge',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review corpus before unchanged data-only training"; }
trap finish EXIT
date -Is > controller.started
st transfer-wait 'native file notification; no rig input deletion; original194-per-mille/seed2026100903'
python3 - <<'PY'
import ctypes,os,select
from pathlib import Path
libc=ctypes.CDLL(None,use_errno=True)
fd=libc.inotify_init1(os.O_CLOEXEC);assert fd>=0
assert libc.inotify_add_watch(fd,b'.',0x8|0x80)>=0
poll=select.poll();poll.register(fd,select.POLLIN)
while not Path('transfer.done').exists():
 assert poll.poll(3*3600*1000),'transfer completion wait expired'
 os.read(fd,65536)
os.close(fd)
PY
chmod +x converter
st scatter-start 'same source byte/hash pins enforced by existing decoder; CPUs8-11/nice19/workers2/2GiB'
timeout --signal=TERM --kill-after=20s 12h ./converter scatter --input "$S/farseerT74.binpack" --conflicts "$S/conflicts.bin" --probe-keys "$S/calibration-probe.keys" --output "$S/scatter-t74" --buckets 256 --workers 2 --seed 2026100903 --keep-per-mille 194 > scatter.stdout 2> scatter.log
python3 fit-source-scale.py --reference "$S/t80-statistics.json" --source "t76=$S/t76-statistics.json" --source "t74=$S/scatter-t74/scatter.json" --out scales.json > scales.log
scale=$(python3 -c 'import json; print(json.load(open("scales.json"))["sources"]["t74"]["raw_scale"])')
st gather-start "same source scale $scale/calibrated map/seed2026100912; consume only own forge buckets"
timeout --signal=TERM --kill-after=20s 6h ./converter gather --scatter "$S/scatter-t74@$scale" --map "$S/score-map.json" --output "$S/t74.bf" --manifest "$S/t74-manifest.json" --seed 2026100912 --consume-buckets true > gather.stdout 2> gather.log
python3 - <<'PY'
import hashlib,json
from pathlib import Path
p=Path('t74.bf');h=hashlib.sha256()
with p.open('rb') as f:
 for b in iter(lambda:f.read(8*1024**2),b''):h.update(b)
pin=dict(bytes=p.stat().st_size,sha256=h.hexdigest(),mtime_ns=p.stat().st_mtime_ns)
Path('files.json').write_text(json.dumps({'t74.bf':pin},indent=2)+'\n')
Path('t74-public-descriptor.json').write_text(json.dumps(dict(bytes=pin['bytes'],sha256=pin['sha256'],host='192.168.4.65',port=8767,file='t74.bf'),indent=2)+'\n')
Path('t74.archive.json.tmp').write_text(json.dumps(dict(bytes=pin['bytes'],sha256=pin['sha256'],url='http://192.168.4.65:8767/'+Path('token').read_text().strip()+'/t74.bf'),indent=2)+'\n')
Path('t74.archive.json.tmp').replace('t74.archive.json')
PY
st corpus-done 'mappedT74/scale/pin ready; retain both raw inputs; no automatic training'

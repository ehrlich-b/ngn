#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session41-remote-archive
D=$R/session19-data
cd "$S"
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PYTHONPATH="$S/source" OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
finish() { local rc=$?; echo "$rc" > coverage.exit; date -Is > coverage.done; }
trap finish EXIT
date -Is > coverage.started
# Watch already-started checks without repeated assistant polls.
python3 - <<'PY'
import os,select
from pathlib import Path
if not Path('checks.done').exists():
 fd=os.pidfd_open(int(Path('checks.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(3600*1000),'reader check wait expired'
 os.close(fd)
assert Path('checks.exit').read_text().strip()=='0','reader checks did not pass'
PY
token=$(cat token)
curl --fail --silent --show-error --max-time 43200 "http://192.168.4.65:8767/$token/descriptor.json" > t74.archive.json.tmp
mv t74.archive.json.tmp t74.archive.json
taskset -c 12,14 nice -n 19 "$PY" - <<'PY' > coverage.log 2>&1
import datetime,importlib.util,json,time
from pathlib import Path
import numpy as np
from trainer.archive import ARCHIVE_DTYPE,board_hash
from trainer.remote_archive import RemoteArray
r=Path('/home/ehrli/ngn-data/night-20261007');s=r/'session41-remote-archive';d=r/'session19-data'
spec=importlib.util.spec_from_file_location('coverage',d/'coverage.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
sources=[('t80',Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf')),('t76',d/'t76.bf'),('t74',s/'t74.archive.json')]
results={};union=np.empty(0,dtype=np.uint64);started=time.monotonic()
for name,path in sources:
 if name=='t74':
  rows=RemoteArray(path,ARCHIVE_DTYPE);count=rows.count;size=rows.size;pin=rows.descriptor['sha256']
 else:
  rows=np.memmap(path,dtype=ARCHIVE_DTYPE,mode='r');count=len(rows);size=path.stat().st_size;pin=None
 sample=np.empty(0,dtype=np.uint64)
 for start in range(0,count,262144):sample=m.update(sample,board_hash(rows[start:min(start+262144,count)],20261006))
 union=m.update(union,sample);results[name]={'path':str(path),'bytes':size,'records':count,'remote_sha256':pin,**m.estimate(sample)}
 print(json.dumps(dict(source=name,**results[name])),flush=True)
before=results['t80']['distinct'];after=m.estimate(union)
j=dict(method='bottom-65536 distinct occupied-board hashes; ngn-board-mix64-v1/20261006',sources=results,combined=after,estimated_unique_ratio=after['distinct']/before,record_ratio=sum(v['records'] for v in results.values())/results['t80']['records'],seconds=time.monotonic()-started,limits='occupied boards, not game groups or mirrored inputs; approximate correlated source/union errors',training_launched=False)
(s/'coverage.json').write_text(json.dumps(j,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session41-remote-data',event='coverage-done',detail='review unique expansion/source scale/actual accepted base before data-only training',estimated_unique_ratio=j['estimated_unique_ratio']))+'\n')
PY

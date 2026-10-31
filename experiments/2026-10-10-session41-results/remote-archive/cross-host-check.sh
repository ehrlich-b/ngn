set -eu
S=/home/ehrli/ngn-data/night-20261007/session41-remote-archive
cd "$S"
cp /mnt/c/Users/ehrli/ngn-session41-fixture-pin.json range-fixture-pin.json
PYTHONPATH="$S/source" taskset -c 12,14 nice -n 19 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python - <<'PY'
import hashlib,json,time
from pathlib import Path
from trainer.archive import ARCHIVE_DTYPE
from trainer.remote_archive import RemoteArray
p=Path('range-fixture-pin.json');pin=json.loads(p.read_text());descriptor=Path('range-fixture.archive.json');descriptor.write_text(json.dumps(dict(bytes=pin['bytes'],sha256=pin['sha256'],url='http://192.168.4.65:8767/'+Path('token').read_text().strip()+'/range-fixture.bf')))
rows=RemoteArray(descriptor,ARCHIVE_DTYPE);times=[]
for _ in range(4):
 start=time.monotonic();data=rows[0:rows.count];times.append(time.monotonic()-start)
 assert hashlib.sha256(data.tobytes()).hexdigest()==pin['sha256']
local=Path('range-fixture.bf').read_bytes();import numpy as np
indexes=np.array([261111,50,8192,50,200000]);assert rows[indexes].tobytes()==np.frombuffer(local,dtype=ARCHIVE_DTYPE)[indexes].tobytes()
mbps=[pin['bytes']/seconds/1024**2 for seconds in times]
j=dict(sha256=pin['sha256'],bytes=pin['bytes'],exact_full_reads=4,exact_irregular_indexes=True,mib_per_second=mbps,admission_min_mib_per_second=25,passed=min(mbps)>=25,rig_cpus=[12,14],forge_cpus=[8,9,10,11],forge_server_cpu_quota=50)
Path('cross-host-receipt.json').write_text(json.dumps(j,indent=2)+'\n');print(json.dumps(j));assert j['passed']
PY

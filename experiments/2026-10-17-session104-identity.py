"""Rig-only successor identity against accepted S3; reuse the established panel."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session104-speed'
spec = importlib.util.spec_from_file_location('original_identity', R/'session100-speed/identity.py')
old = importlib.util.module_from_spec(spec)
spec.loader.exec_module(old)


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False)+'\n')


def main():
    binaries = dict(base=R/'session101-speed/s3-ngn', s2=D/'s2-ngn', s1=D/'s1-ngn')
    rows = []
    for hash_mib in (64, 256):
        engines = [old.UCI(str(path)[:-4], hash_mib) for path in binaries.values()]
        try:
            for frame in old.FRAMES:
                values = [engine.search(frame) for engine in engines]
                assert values[0] == values[1] == values[2], (hash_mib, frame['id'], values)
                rows.append(dict(hash_mib=hash_mib, id=frame['id'], identity=values[0]))
        finally:
            for engine in engines:
                engine.close()
    for name, net, depth in [('ngnn3', old.NET, 14),
            ('ngnn2', R/'session100-speed/baseline/engine/nets/ngn-h512-c6c12796.nnue', 14),
            ('hce', '', 11)]:
        cells = []
        for kind, binary in binaries.items():
            probe = binary.with_name(binary.name[:-4]+'-probe')
            raw = subprocess.check_output(['taskset','-c','15','nice','-n','19',str(probe),
                '-net',str(net),'-mode','identity','-depth',str(depth)],
                env=dict(os.environ,GOMAXPROCS='1'),text=True,timeout=900)
            (D/f'identity-{name}-{kind}.jsonl').write_text(raw)
            data = [json.loads(line) for line in raw.splitlines()]
            assert len(data) == 12 and all(row['nodes'] > 0 for row in data)
            cells.append([{k:v for k,v in row.items() if k not in ('seconds','nps')} for row in data])
        assert cells[0] == cells[1] == cells[2], name
    engine = old.UCI(str(D/'s1'),256)
    try:
        engine.send('position startpos moves '+' '.join(old.FRAMES[0]['moves']))
        engine.send('go movetime 5000')
        time.sleep(4)
        regions = []
        for line in Path(f'/proc/{engine.child.pid}/smaps').read_text().splitlines():
            if re.match(r'^[0-9a-f]+-[0-9a-f]+ ',line):
                regions.append(dict(mapping=line))
            elif line.startswith(('Size:','AnonHugePages:','VmFlags:')):
                key,value=line.split(':',1)
                regions[-1][key]=value.strip()
        advised = [r for r in regions if 'hg' in r.get('VmFlags','').split()]
        huge_kib = sum(int(r.get('AnonHugePages','0 kB').split()[0]) for r in advised)
        write(D/'s1-hugepage-witness.json',dict(pid=engine.child.pid,
            anon_huge_kib=huge_kib,advised_regions=advised))
        engine.until('bestmove ')
        assert huge_kib >= 256*1024, huge_kib
    finally:
        engine.close()
    sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    for kind, topic in [('s2','f527ab3'),('s1','fcc1432')]:
        write(D/(kind+'-identity.json'),dict(verdict='PASS',candidate=kind,
            immediate_base='4958bc4/81215229/compiled final14',topic=topic,
            uci_cases=24,probe_cases=36,identity_rows=rows,tests_pass=True,
            hugepage_witness_kib=huge_kib if kind=='s1' else None,
            binary_sha256=dict(base=sha(binaries['base']),candidate=sha(binaries[kind])),
            net_sha256=sha(old.NET),goamd64='v3',check_cpus=[13,15],
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            automatic_adoption=False))


if __name__=='__main__':
    main()

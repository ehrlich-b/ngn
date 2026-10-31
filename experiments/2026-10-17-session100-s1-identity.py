"""Recheck revised S1 against the already verified unchanged-base identities."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time

from identity import FRAMES, NET, S, UCI

KIND='s1-prefault'
original=json.loads((S/'identity.json').read_text())
for hash_mib in (64,256):
    e=UCI(KIND,hash_mib)
    try:
        for frame in FRAMES:
            want=next(r['identity'] for r in original['uci_records'] if r['hash_mib']==hash_mib and r['id']==frame['id'])
            assert e.search(frame)==want,(hash_mib,frame['id'])
    finally: e.close()
for name,net,depth in [('ngnn3',NET,14),('ngnn2',S/'baseline/engine/nets/ngn-h512-c6c12796.nnue',14),('hce','',11)]:
    argv=['taskset','-c','15','nice','-n','19',str(S/(KIND+'-probe')),'-net',str(net),'-mode','identity','-depth',str(depth)]
    raw=subprocess.check_output(argv,env=dict(os.environ,GOMAXPROCS='1'),text=True,timeout=900)
    (S/f'identity-{name}-{KIND}.jsonl').write_text(raw)
    strip=lambda rows:[{k:v for k,v in r.items() if k not in ('seconds','nps')} for r in rows]
    want=strip([json.loads(line) for line in (S/f'identity-{name}-baseline.jsonl').read_text().splitlines()])
    assert strip([json.loads(line) for line in raw.splitlines()])==want,name
e=UCI(KIND,256)
try:
    e.send('position startpos moves '+' '.join(FRAMES[0]['moves']))
    e.send('go movetime 5000')
    time.sleep(4)
    regions=[]
    for line in Path(f'/proc/{e.child.pid}/smaps').read_text().splitlines():
        if re.match(r'^[0-9a-f]+-[0-9a-f]+ ',line): regions.append(dict(mapping=line))
        elif line.startswith(('Size:','AnonHugePages:','VmFlags:')):
            key,value=line.split(':',1);regions[-1][key]=value.strip()
    advised=[r for r in regions if 'hg' in r.get('VmFlags','').split()]
    huge_kib=sum(int(r.get('AnonHugePages','0 kB').split()[0]) for r in advised)
    witness=dict(pid=e.child.pid,advised_regions=advised,anon_huge_kib=huge_kib)
    (S/(KIND+'-hugepage-witness.json')).write_text(json.dumps(witness,indent=2)+'\n')
    e.until('bestmove ')
    assert huge_kib>=256*1024,witness
finally: e.close()
(S/(KIND+'-identity.json')).write_text(json.dumps(dict(verdict='PASS',uci_cases=24,probe_cases=36,
    baseline_reference=str(S/'identity.json'),hugepage_witness_kib=huge_kib,
    binary_sha256=hashlib.sha256((S/(KIND+'-ngn')).read_bytes()).hexdigest(),automatic_adoption=False),indent=2)+'\n')

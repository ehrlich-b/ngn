#!/usr/bin/env bash
set -euo pipefail
taskset -c 13,15 nice -n 19 python3 - <<'PY'
from pathlib import Path
import datetime,hashlib,json,re
s=Path('/home/ehrli/ngn-data/night-20261007/session11')
read=lambda p:json.loads(p.read_text())
digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
r=read(s/'audit-receipt.json'); params=read(s/'registry.json')
assert (s/'controller.exit').read_text().strip()=='0'
assert r['parameters']==14 and r['configurations']==30 and r['searches']==180
assert set(r['witnessed'])=={p['name'] for p in params} and not r['unwitnessed']
assert read(s/'source-pins.json')==r['source_pins']
for path,h in r['source_pins'].items(): assert digest(Path(path))==h,path
expected={'defaults-before','defaults-after'}|{p['name']+'-'+side for p in params for side in ('minimum','maximum')}
results={}; raw_hashes={}; search_count=0
for name in sorted(expected):
    p=s/(name+'.json'); data=read(p); raw_hashes[p.name]=digest(p)
    assert data['returncode']==0 and (s/(name+'.stderr')).read_bytes()==b''
    assert data['argv'][-1]=='/home/ehrli/ngn-data/night-20261007/kb8bin/ngn-kb8'
    assert data['argv'][:2]==['taskset','-c'] and data['argv'][2] in ('13','15')
    assert data['argv'][3:6]==['nice','-n','19']
    assert data['commands'].count('go depth 14 nodes 1000000')==6
    assert data['commands'].count('ucinewgame')==6
    selected=[c for c in data['commands'] if c.startswith('setoption ')]
    if name.startswith('defaults-'): assert not selected and data['override'] is None
    else:
        assert selected==[f"setoption name {data['override'][0]} value {data['override'][1]}"]
        param=next(p for p in params if p['name']==data['override'][0])
        side=name.rsplit('-',1)[1]
        assert data['override'][1]==param[side]
        assert data['argv'][2]==('13' if side=='minimum' else '15')
    parsed=[]; last_info=None
    for line in data['stdout']:
        assert not re.search(r'error|failed|unknown option',line,re.I)
        if line.startswith('info depth '): last_info=line.split()
        if line.startswith('bestmove '):
            t=last_info; assert t is not None
            score_index=t.index('score'); pv=t[t.index('pv')+1:]
            record=dict(depth=int(t[t.index('depth')+1]),score_kind=t[score_index+1],
                score=int(t[score_index+2]),nodes=int(t[t.index('nodes')+1]),pv=pv,best=line.split()[1])
            assert 0<record['depth']<=14 and 0<record['nodes']<=1005000
            assert pv and pv[0]==record['best'] and record['best'] not in ('0000','(none)')
            parsed.append(record); last_info=None
    assert parsed==data['results'] and len(parsed)==6
    search_count+=len(parsed); results[name]=parsed
assert search_count==180
assert results['defaults-before']==results['defaults-after']
witnesses={}
for p in params:
    name=p['name']; expected_w=[]
    for side in ('minimum','maximum'):
        for i,rec in enumerate(results[name+'-'+side]):
            if rec!=results['defaults-before'][i]:
                expected_w.append(dict(side=side,position=i,baseline=results['defaults-before'][i],changed=rec))
    assert expected_w==r['witnesses'][name] and expected_w
    witnesses[name]=dict(differing_records=len(expected_w),first=expected_w[0])
dependency_checks=read(s/'dependency-check.json')
for path,info in dependency_checks['checks'].items():
    p=Path(path); assert digest(p)==info['sha256']
    lines=p.read_text().splitlines()
    assert len(lines)==info['entries'] and lines and all(l.endswith(': OK') for l in lines)
for p in (s/'input-check-before.txt',s/'input-check-after.txt'):
    assert all(line.endswith(': OK') for line in p.read_text().splitlines())
for p in (s/'audit-inputs.sha256',s/'audit-receipt.json',s/'registry.json',s/'source-pins.json',
          s/'launch-observed.json',s/'dependency-check.json',s/'controller.done',s/'controller.exit'):
    raw_hashes[p.name]=digest(p)
review=dict(reviewed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    independent_raw_parser=True,searches=180,configurations=30,default_identity=True,
    live_parameters=14,unwitnessed=[],witnesses=witnesses,raw_hashes=raw_hashes,
    dependency_checks=dependency_checks,engine_sha256=r['source_pins'][str(Path('/home/ehrli/ngn-data/night-20261007/kb8bin/ngn-kb8'))],
    tuner_sha256=r['source_pins'][str(Path('/home/ehrli/ngn-data/night-20261007/spsa/spsa'))],
    scope='Only option/source wiring verified; no games, tune convergence or strength result.')
(s/'final-review.json').write_text(json.dumps(review,indent=2)+'\n')
print(json.dumps({k:v for k,v in review.items() if k not in ('witnesses','raw_hashes','dependency_checks')}))
print(json.dumps({k:v['differing_records'] for k,v in witnesses.items()}))
PY

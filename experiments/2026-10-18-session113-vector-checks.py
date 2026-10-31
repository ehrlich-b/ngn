"""Bounded manifest/endpoint checks; no engines, games, wait or live dispatch."""
import copy
import importlib.util
import json
from pathlib import Path

R=Path('/home/ehrli/ngn-data/night-20261007')
D=R/'session113-vector'
spec=importlib.util.spec_from_file_location('vector',D/'vector-gate.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
# Derive a synthetic endpoint from the actual schema, not the live endpoint.
row=json.loads((R/'session88/spsa-warm/spsa.jsonl').read_text().splitlines()[-1])
row['iter']=2000
text='[iter 2000] theta+ 50.0% over last 300 games (0 forfeits)\n=== FINAL VECTOR (test) ===\n'
text+='\n'.join('setoption name '+k+' value '+str(v) for k,v in row['rounded'].items())
assert m.final_endpoint(json.dumps(row),text)==row
for altered,log in [(dict(row,iter=1975),text),(row,text.replace('=== FINAL VECTOR','=== INCOMPLETE')),(row,text.replace('(0 forfeits)','(1 forfeits)'))]:
    try:m.final_endpoint(json.dumps(altered),log)
    except AssertionError:pass
    else:raise AssertionError('invalid terminal admitted')
template=json.loads((D/'template.json').read_text())
job=m.vector_job(template,row)
assert job['candidate']==job['base']==template['base']
assert job['max_games']==job['min_games']==1000 and job['class']=='net'
assert job['base_options']==template['base_options']
a={x.split('=',1)[0]:x.split('=',1)[1] for x in job['candidate_options']}
b={x.split('=',1)[0]:x.split('=',1)[1] for x in job['base_options']}
assert {k:v for k,v in a.items() if k not in row['rounded']}=={k:v for k,v in b.items() if k not in row['rounded']}
assert all(int(a[k])==v for k,v in row['rounded'].items())
assert len(job['artifacts'])==4 and job['net_sha256']==job['base_net_sha256']
(D/'dispatch-checks.json').write_text(json.dumps(dict(verdict='PASS',scenarios=['final2000 accepted','1975 refused','missing terminal refused','forfeit refused','same source/net only final vector differs'],automatic_games=False),indent=2)+'\n')
print('Vector dispatch checks PASS')

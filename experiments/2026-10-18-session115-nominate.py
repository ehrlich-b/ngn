"""Nominate a prechecked isolated search candidate on the accepted warm vector."""
import copy
import datetime
import hashlib
import json
from pathlib import Path
import sys

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session115-search'
name = sys.argv[1]
c = D/'candidates'/name
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
base_source = (R/'session115/accepted-source.txt').read_text().strip()
checks = dict(base=base_source, vector='compiled warm2000', candidate=name,
    passed=['focused including singular witness','engine short/race','full short/race','vet','PGO build'],
    binary_sha256=sha(c/'candidate-ngn'), patch_sha256=sha(R/'session107/candidates'/(name+'.patch')),
    completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(c/'checks-result.json').write_text(json.dumps(checks,indent=2)+'\n')
queue = json.loads((R/'session114-search/queue.json').read_text())
job = copy.deepcopy(next(j for j in queue['jobs'] if j['name'] == 's114-'+name))
for path in [job['base'],job['candidate']]:
    job['artifacts'].pop(path)
job.update(base_source=base_source, base=str(R/'session115/accepted-ngn'),
    candidate_source=base_source+' + '+name+' isolated patch', candidate=str(c/'candidate-ngn'),
    name='s115-'+name, output=str(D/name), candidate_checks=str(c/'checks-result.json'),
    source_checks_reused=False, candidate_patch_sha256=checks['patch_sha256'], vector='compiled warm2000')
vector = json.loads((R/'session115/endpoint.json').read_text())['rounded']
for role in ('base_options','candidate_options'):
    job[role] = [option.split('=',1)[0]+'='+str(vector[option.split('=',1)[0]])
                 if option.split('=',1)[0] in vector else option for option in job[role]]
job['artifacts'][job['base']] = json.loads((R/'session115/default-parity.json').read_text())['binary_sha256']
job['artifacts'][job['candidate']] = checks['binary_sha256']
(D/(name+'-nomination.json')).write_text(json.dumps(job,indent=2)+'\n')
print(name, 'checks PASS, nomination written', flush=True)
# The first gate starts as soon as its own checks pass; subsequent checks use13,15.
if name == 'qsearch':
    import subprocess
    subprocess.run(['bash',str(D/'launch.sh')],check=True)

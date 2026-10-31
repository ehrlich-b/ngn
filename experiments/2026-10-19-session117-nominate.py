"""Reuse the declared search gate, changing only its immediate accepted base."""
import copy,datetime,hashlib,json,sys
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session117';Q=R/'session117-search'
name=sys.argv[1];c=Q/'candidates'/name
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
source=(D/'accepted-source.txt').read_text().strip()
checks=dict(base=source,vector='compiled warm2000',candidate=name,
 passed=['focused including singular witness','engine/full short/race','vet','PGO build'],
 binary_sha256=sha(c/'candidate-ngn'),patch_sha256=sha(c/'candidate.patch'),
 completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(c/'checks-result.json').write_text(json.dumps(checks,indent=2)+'\n')
job=copy.deepcopy(json.loads((R/'session115-search'/(name+'-nomination.json')).read_text()))
for path in [job['base'],job['candidate']]:job['artifacts'].pop(path)
job.update(base_source=source,base=str(D/'accepted-ngn'),candidate_source=source+' + '+name+' isolated patch',
 candidate=str(c/'candidate-ngn'),name='s117-'+name,output=str(Q/name),candidate_checks=str(c/'checks-result.json'),
 source_checks_reused=False,candidate_patch_sha256=checks['patch_sha256'])
job['artifacts'][job['base']]=sha(D/'accepted-ngn');job['artifacts'][job['candidate']]=checks['binary_sha256']
(Q/(name+'-nomination.json')).write_text(json.dumps(job,indent=2)+'\n')
print(name,'PASS; nomination ready, games unstarted',flush=True)

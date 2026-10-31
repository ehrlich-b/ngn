import hashlib,json,os,subprocess
from pathlib import Path
D=Path('/home/ehrli/ngn-data/night-20261007/session94');P=D/'playing';S=P/'source'
argv=['/usr/local/go/bin/go','build','-buildvcs=true','-p','2','-o',str(P/'ngn'),'.']
(P/'build-argv.json').write_text(json.dumps(argv)+'\n')
with (P/'build.log').open('w') as log: subprocess.run(argv,cwd=S,stdout=log,stderr=subprocess.STDOUT,check=True)
info=subprocess.check_output(['/usr/local/go/bin/go','version','-m',str(P/'ngn')],text=True)
(P/'buildinfo.txt').write_text(info)
assert 'vcs.modified=false' in info and 'vcs.revision=9ec5d31' in info
subprocess.run(['python3',str(D/'playing-identity.py')],check=True)
identity=json.loads((P/'identity.json').read_text());assert identity['verdict']=='PASS' and identity['pairs']==36
receipt=dict(engine=dict(Path=str(P/'ngn'),SHA256=hashlib.sha256((P/'ngn').read_bytes()).hexdigest()),source_root=str(S),source_commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=S,text=True).strip(),argv=argv,identity=identity,change='Source and behavior unchanged;clean metadata build for prospective claim')
(D/'playing-build.json').write_text(json.dumps(receipt,indent=2)+'\n')

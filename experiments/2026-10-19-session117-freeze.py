"""Freeze one clean-source metadata rebuild and the tested 3555 claim rule."""
import hashlib,json,subprocess
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session117';S=D/'source'
T=Path('/home/ehrli/ngn-data/claim-v4-20261016/tooling-s117');F=T/'frozen'
A=Path('/home/ehrli/ngn-data/claim-v4-20261016/attempt-002')
F.mkdir(parents=True,exist_ok=True);A.mkdir(exist_ok=True)
read=lambda p:json.loads(p.read_text())
write=lambda p,v:p.write_text(json.dumps(v,indent=2)+'\n')
bind=lambda p:dict(Path=str(p),SHA256=hashlib.sha256(p.read_bytes()).hexdigest())
source=(D/'accepted-source.txt').read_text().strip()
assert read(D/'identity.json')['verdict']=='PASS'
assert not subprocess.check_output(['git','status','--porcelain'],cwd=S)
engine_argv=['/usr/local/go/bin/go','build','-buildvcs=true','-p','2','-pgo='+str(R/'session100-speed/baseline/default.pgo'),'-o',str(D/'accepted-ngn'),'.']
info=subprocess.check_output(['/usr/local/go/bin/go','version','-m',str(D/'accepted-ngn')],text=True)
(D/'buildinfo.txt').write_text(info)
assert 'vcs.modified=false' in info and 'vcs.revision='+source in info
identity=read(Path('/home/ehrli/ngn-data/claim-v4-20261016/attempt-001/candidate-identity.json'))
identity.update(Engine=bind(D/'accepted-ngn'),SourceRoot=str(S),SourceCommit=source,BuildArgv=engine_argv)
# Existing accepted net/provenance bindings are reused; prepare verifies them at use.
write(A/'candidate-identity.json',identity)
argv=['/usr/local/go/bin/go','build','-buildvcs=true','-p','2','-o',str(F/'claimtool'),'./cmd/claimtool']
write(F/'claimtool-build.json',argv)
with (T/'build.log').open('w') as log:subprocess.run(argv,cwd=S,stdout=log,stderr=subprocess.STDOUT,check=True)
info=subprocess.check_output(['/usr/local/go/bin/go','version','-m',str(F/'claimtool')],text=True)
(F/'buildinfo.txt').write_text(info)
assert 'vcs.modified=false' in info and 'vcs.revision='+source in info
modes={}
for mode in ['screen','confirmation','tooling-check','stability-screen','stability-confirmation']:
 command=[str(F/'claimtool'),'prepare','--mode',mode,'--candidate',str(A/'candidate-identity.json'),
  '--source',str(S),'--tool-build-argv',str(F/'claimtool-build.json'),'--out',str(T/('dry-'+mode)),'--dry-run']
 with (T/('dry-'+mode+'.log')).open('w') as log:subprocess.run(command,stdout=log,stderr=subprocess.STDOUT,check=True)
 modes[mode]=dict(status='PASS',argv=command)
write(T/'build-receipt.json',dict(state='READY_TOOLING',source_commit=source,source_clean=True,runner=bind(F/'claimtool'),
 screen_point_minimum=3555,confirmation_lower95_strict=3600,dry_modes=modes,
 reused_full_checks=str(R/'session115-search/candidates/qsearch/checks-result.json'),
 reused_rule_checks=str(R/'session100-speed/claim-checks.exit'),focused_tests=str(D/'claim-focused.log')))
write(D/'package.json',dict(source_commit=source,engine=bind(D/'accepted-ngn'),identity=str(A/'candidate-identity.json'),
 metadata_identity=str(D/'identity.json'),tooling=str(T),attempt=str(A),automatic_adoption=False))

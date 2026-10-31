"""One current-final14 forge control, no candidate games or adoption."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import subprocess

p=Path.cwd()
sha=lambda n:hashlib.sha256((p/n).read_bytes()).hexdigest()
def status(event,detail):
    with (p/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session65-data-aa',event=event,detail=detail))+'\n')
assert sha('ngn')=='4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d'
assert sha('base.nnue')=='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
assert sha('sprt')=='9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert sha('openings.txt')=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
argv=['taskset','-c','4-11','nice','-n','10',str(p/'sprt'),'-new',str(p/'ngn'),'-base',str(p/'ngn'),'-tc','10+0.1','-concurrency','2','-maxmoves','400','-gametimeout','180','-openings',str(p/'openings.txt'),'-lowpower=false','-mingames','1000000','-maxgames','40']
for side in ['new','base']:
    for key,value in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:argv+=['-'+side+'option',key+'='+value]
names=['ngn','base.nnue','sprt','openings.txt','paired-parser.py','aa.py','controller.sh','manifest.md']
hashes={n:sha(n) for n in names}
manifest=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source='f1d0a03/topic5023b1d exact engine',stage='base admission only; candidate net not nominated',machine='forge Debian13/Ryzen6900HX',os=platform.platform(),host_pinning='on',cpus='4-11',concurrency=2,nice=10,memory_max_bytes=3*1024**3,memory_swap_max=0,tc='10+0.1',environment=dict(GOMAXPROCS='1',GOAMD64='v3'),go_version='go version go1.25.5 linux/amd64',profile_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',aa_argv=argv,files_sha256=hashes,decision='complete40/20, both CI contain0, zero faults/intact pins; no retry, gate or adoption')
(p/'game-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
hashes['game-manifest.json']=sha('game-manifest.json')
(p/'game-inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
processes=subprocess.check_output(['ps','-eo','pid,ppid,psr,pcpu,rss,args'],text=True).splitlines()
(p/'background-before-aa.txt').write_text('\n'.join([processes[0],*[x for x in processes[1:] if '/work/lanes/ngn-' in x]])+'\n')
subprocess.run(['sha256sum','-c','game-inputs.sha256'],check=True,stdout=(p/'input-check-before-aa.txt').open('w'))
status('aa-start','promoted final14 package; no competing NGN CPU work; training/reader load recorded')
with (p/'aa.log').open('w') as log:
    subprocess.run(['timeout','--signal=TERM','--kill-after=15s','1800',*argv],check=True,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GOMAXPROCS='1',GOAMD64='v3'))
spec=importlib.util.spec_from_file_location('parser',p/'paired-parser.py');parser=importlib.util.module_from_spec(spec);spec.loader.exec_module(parser)
result=parser.gate((p/'aa.log').read_text(),'aa')
crashes=p/'ngn_crashes.log'
if crashes.exists():assert all('Crash Handler Initialized' in x or 'Crash log:' in x for x in crashes.read_text().splitlines())
(p/'aa-summary.json').write_text(json.dumps(result,indent=2)+'\n')
subprocess.run(['sha256sum','-c','game-inputs.sha256'],check=True,stdout=(p/'input-check-after-aa.txt').open('w'))
status('aa-done',result)

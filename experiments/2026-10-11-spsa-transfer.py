"""Manually nominated current-source final14 transfer, own A/A40/fixed400."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import subprocess

p = Path.cwd()
def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')
def status(event, detail):
    with (p / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session65-spsa-transfer', event=event, detail=detail)) + '\n')
sha = lambda name: hashlib.sha256((p / name).read_bytes()).hexdigest()
expected = {'baseline-ngn':'4a83038adb4de354d67faafa9e9bd1884d1717a80bc91b18108cba3b811293bc',
            'candidate-ngn':'4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d',
            'base.nnue':'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86',
            'sprt':'9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a',
            'openings.txt':'974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'}
for name, pin in expected.items():
    assert sha(name) == pin, name
common = ['taskset','-c','0-11','nice','-n','19',str(p/'sprt'),
          '-new',str(p/'baseline-ngn'),'-base',str(p/'baseline-ngn'),
          '-tc','60+0.6','-concurrency','6','-maxmoves','400','-gametimeout','900',
          '-openings',str(p/'openings.txt'),'-lowpower=false','-mingames','1000000']
for side in ['new','base']:
    for key,value in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),
                      ('Hash','64'),('Threads','1'),('OwnBook','false')]:
        common += ['-'+side+'option',key+'='+value]
aa = common + ['-maxgames','40']
gate = list(common)
gate[gate.index('-new')+1] = str(p/'candidate-ngn')
gate += ['-maxgames','400']
names = [*expected,'vector.json','default-parity.json','checks.exit','build.sha256',
         'candidate.patch','games.py','controller.sh','paired-parser.py','manifest.md']
hashes = {name:sha(name) for name in names}
write('game-manifest.json',dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
      base_source='7bbb2bf current pre-SPSA defaults',candidate_source='5023b1d integrated f1d0a03',
      change='frozen endpoint2000/24k final14 only',machine='rig WSL2/Ryzen9800X3D',
      os=platform.platform(),tc='60+0.6',cpus='0-11',concurrency=6,nice=19,
      memory_max_bytes=3*1024**3,memory_swap_max=0,environment=dict(GOMAXPROCS='1',GOAMD64='v3'),
      concurrent_workloads=['training12,14/GPU/4GiB','checks13,15 as needed','hopper untouched'],
      go_version='go version go1.25.5 linux/amd64',
      pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
      aa_argv=aa,gate_argv=gate,files_sha256=hashes,
      decision='AA40 intervals contain0; fixed400 positive printed/unrounded lower95 confirms transfer; overlap0 unconfirmed/promotion stands; upper95<0 revert; faults/incomplete unscored',
      automatic_promotion=False,automatic_revert=False))
hashes['game-manifest.json'] = sha('game-manifest.json')
(p/'game-inputs.sha256').write_text(''.join(pin+'  '+name+'\n' for name,pin in hashes.items()))
spec=importlib.util.spec_from_file_location('parser',p/'paired-parser.py')
parser=importlib.util.module_from_spec(spec);spec.loader.exec_module(parser)
for mode,argv,limit in [('aa',aa,14400),('gate',gate,57600)]:
    subprocess.run(['sha256sum','-c','game-inputs.sha256'],check=True,
                   stdout=(p/('input-check-before-'+mode+'.txt')).open('w'))
    status(mode+'-start','rig0-11/c6/3GiB/nice19/swap0; data training retains priority')
    with (p/(mode+'.log')).open('w') as log:
        subprocess.run(['timeout','--signal=TERM','--kill-after=15s',str(limit),*argv],check=True,
                       stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GOMAXPROCS='1',GOAMD64='v3'))
    result=parser.gate((p/(mode+'.log')).read_text(),mode)
    if mode=='gate':
        elo,lo,hi=result['exact_elo_ci95']
        result['transfer_verdict']=('confirmed' if lo>0 and result['ci95'][0]>0 else
                                    'revert-promotion' if hi<0 else 'unconfirmed-promotion-stands')
    crashes=p/'ngn_crashes.log'
    if crashes.exists():
        assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in crashes.read_text().splitlines())
    write(mode+'-summary.json',result)
    subprocess.run(['sha256sum','-c','game-inputs.sha256'],check=True,
                   stdout=(p/('input-check-after-'+mode+'.txt')).open('w'))
    status(mode+'-done',result)

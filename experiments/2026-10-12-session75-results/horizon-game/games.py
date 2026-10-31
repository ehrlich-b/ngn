"""Prospective fixed120+1 horizon gate, conditional on original clean predecessors."""
import ctypes
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import select
import subprocess
import sys
import time

p=Path.cwd()
panel=p.parent/'session74-clock-panel'
see=p.parent/'session72-see-rig'

def write(name,data):
    (p/name).write_text(json.dumps(data,indent=2,allow_nan=False)+'\n')

def status(event,detail):
    with (p/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),event=event,detail=detail))+'\n')

def digest(name):return hashlib.sha256((p/name).read_bytes()).hexdigest()

def prepare():
    common=['taskset','-c','0-11',str(p/'sprt'),'-new',str(p/'baseline-ngn'),'-base',str(p/'baseline-ngn'),
            '-tc','120+1','-concurrency','6','-maxmoves','400','-gametimeout','1800','-openings',str(p/'openings.txt'),
            '-lowpower=false','-mingames','1000000']
    for side in ['new','base']:
        for key,value in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false'),('Move Overhead','100')]:
            common+=['-'+side+'option',key+'='+value]
    aa=common+['-maxgames','40'];gate=list(common)
    gate[gate.index('-new')+1]=str(p/'candidate-ngn');gate+=['-maxgames','400']
    files=['baseline-ngn','candidate-ngn','base.nnue','sprt','openings.txt','paired-parser.py','games.py','controller.sh','manifest.md','checks-review.json','manual-review.json']
    hashes={name:digest(name) for name in files}
    assert hashes['baseline-ngn']=='4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d'
    assert hashes['candidate-ngn']==json.loads((p/'checks-review.json').read_text())['files_sha256']['horizon-ngn']
    assert hashes['base.nnue']=='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
    assert hashes['sprt']=='9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
    assert hashes['openings.txt']=='f2fed55a1645dba85a9ef3e047eb5183b5b6342878c6dc8483e6768973470acc'
    write('game-manifest.json',dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        base_source='f1d0a035f8b4f0b7883fed0a859e5769d076089f',candidate_source='bc59c3f',
        change='remaining-move floor10 to25 only',machine='Ryzen9800X3D / WSL2',os=platform.platform(),
        tc='120+1',cpus='0-11',concurrency=6,nice=19,memory_max_bytes=3*1024**3,memory_swap_max=0,
        GOMAXPROCS=1,GOAMD64='v3',go_version='go1.25.5 linux/amd64',
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        aa_argv=aa,gate_argv=gate,aa_timeout_seconds=10800,gate_timeout_seconds=43200,
        files_sha256=hashes,predecessors='session75 manual actual-base/integrity review, complete panel admission, SMP checks/scaling terminal0; no old AA/game sample reuse',
        decision='ownAA40 printed/exact contains0; fixed400/200 printed/exact lower95>0 and zero faults/intact inputs; else shelf/reject; faults unscored',
        automatic_adoption=False))
    hashes['game-manifest.json']=digest('game-manifest.json')
    (p/'game-inputs.sha256').write_text(''.join(pin+'  '+name+'\n' for name,pin in hashes.items()))

def wait():
    completion=p.parent/'session75'
    libc=ctypes.CDLL(None);fd=libc.inotify_init1(os.O_NONBLOCK|os.O_CLOEXEC)
    assert fd>=0 and libc.inotify_add_watch(fd,os.fsencode(completion),0x8|0x80)>=0
    try:
        status('scaling-wait','native terminal notification; fixed no-games scaling first; manual unchanged-base nomination')
        end=time.monotonic()+3*3600
        while not (completion/'scaling.done').exists():
            seconds=end-time.monotonic();assert seconds>0 and select.select([fd],[],[],seconds)[0]
            os.read(fd,65536)
    finally:os.close(fd)
    assert (completion/'scaling.exit').read_text().strip()=='0','scaling failed: manual hold'
    assert (completion/'checks.exit').read_text().strip()=='0','SMP correctness failed: manual hold'
    review=json.loads((p/'manual-review.json').read_text())
    summary=json.loads((panel/'panel-summary.json').read_text())
    allowed=review['actual_base_unchanged'] and review['zero_prior_horizon_games'] and summary['horizon_admission']=='ELIGIBLE FOR PROSPECTIVE GATE'
    status('predecessor-review',dict(manual_actual_base_review=review,panel=summary['horizon_admission'],games_allowed=allowed))
    return allowed

def capacity():
    processes=subprocess.check_output(['ps','-eo','pid,ppid,rss,args'],text=True)
    busy=[line for line in processes.splitlines() if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)',line)]
    memory={k:int(v)*1024 for k,v in re.findall(r'^(MemTotal|MemAvailable):\s+(\d+) kB',Path('/proc/meminfo').read_text(),re.M)}
    d=os.statvfs('/mnt/c');used=memory['MemTotal']-memory['MemAvailable']
    receipt=dict(nonavailable_memory_bytes=used,new_memory_budget_bytes=3*1024**3,c_free_bytes=d.f_bavail*d.f_frsize,competing_chess_processes=busy)
    write('capacity-before-aa.json',receipt)
    if busy or receipt['c_free_bytes']<25.5*1024**3 or used+3*1024**3>12*1024**3:status('capacity-hold',receipt);return False
    return True

def run():
    if not wait() or not capacity():return
    spec=importlib.util.spec_from_file_location('existing_parser',p/'paired-parser.py');parser=importlib.util.module_from_spec(spec);spec.loader.exec_module(parser)
    manifest=json.loads((p/'game-manifest.json').read_text())
    for mode in ['aa','gate']:
        with (p/('input-check-before-'+mode+'.txt')).open('w') as out:subprocess.run(['sha256sum','-c','game-inputs.sha256'],stdout=out,check=True)
        status(mode+'-start',dict(tc='120+1',cpus='0-11',concurrency=6,nice=19))
        with (p/(mode+'.log')).open('w') as out:
            result=subprocess.run(['timeout','--signal=TERM','--kill-after=15s',str(manifest[mode+'_timeout_seconds']),*manifest[mode+'_argv']],stdout=out,stderr=subprocess.STDOUT)
        with (p/('input-check-after-'+mode+'.txt')).open('w') as out:subprocess.run(['sha256sum','-c','game-inputs.sha256'],stdout=out,check=True)
        assert result.returncode==0,(mode,'operationally unscored',result.returncode)
        crashes=p/'ngn_crashes.log'
        if crashes.exists():assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in crashes.read_text().splitlines())
        summary=parser.gate((p/(mode+'.log')).read_text(),mode)
        write(mode+'-summary.json',summary);status(mode+'-done',summary)

if __name__=='__main__':
    if sys.argv[1:]==['prepare']:prepare()
    else:run()

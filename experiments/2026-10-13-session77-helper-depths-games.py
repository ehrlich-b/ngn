"""Prospective helper-depths S gate after a clean unchanged-base predecessor."""
import ctypes
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import select
import subprocess
import sys
import time

lane = Path.cwd()
horizon = lane.parent / 'session75-horizon-gate'


def write(name, value):
    (lane / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (lane / 'status.jsonl').open('a') as out:
        out.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), event=event, detail=detail)) + '\n')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def prepare():
    options = [('EvalFile', str(lane/'base.nnue')), ('UseNNUE', 'true'), ('Hash', '256'),
               ('OwnBook', 'false'), ('Move Overhead', '100')]
    common = ['taskset', '-c', '0,2,4,6,8,10,12,14', str(lane/'sprt'), '-new', str(lane/'new.sh'), '-base', str(lane/'base.sh'),
              '-tc', '10+0.1', '-concurrency', '1', '-maxmoves', '400', '-gametimeout', '180',
              '-openings', str(lane/'openings.txt'), '-lowpower=false', '-mingames', '1000000']
    for side in ['new', 'base']:
        for key, value in options:
            common += ['-'+side+'option', key+'='+value]
    aa = common + ['-newoption', 'Threads=4', '-baseoption', 'Threads=4', '-maxgames', '40']
    gate = common + ['-newoption', 'Threads=4', '-baseoption', 'Threads=4', '-maxgames', '400']
    files = ['baseline-ngn', 'candidate-ngn', 'manifest.md', 'candidate-source.txt', 'base.nnue', 'sprt', 'openings.txt', 'paired-parser.py', 'role.py', 'new.sh', 'base.sh', 'games.py', 'controller.sh', 'smp.py']
    pins = {name: digest(lane/name) for name in files}
    assert pins['baseline-ngn'] == '4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d'
    assert pins['base.nnue'] == 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
    write('game-manifest.json', dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        base_source='f1d0a03', candidate_source=(lane/'candidate-source.txt').read_text().strip(), net='a4d90b41', final14='compiled defaults',
        machine='rig Ryzen9800X3D WSL2', go_version='go1.25.5 linux/amd64', GOAMD64='v3',
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        tc='10+0.1', concurrency=1, cpus={'new': [0,2,4,6], 'base': [8,10,12,14]}, nice=19, Hash=256,
        GOMAXPROCS='harness1; engines unset environment, production scheduler follows Threads4/4',
        memory_max_bytes=3*1024**3, swap_max=0, minimum_full_width_fraction=.8,
        aa_argv=aa, gate_argv=gate, aa_environment={'NGN_S77_MODE':'aa'}, gate_environment={'NGN_S77_MODE':'gate'}, aa_timeout_seconds=7200, gate_timeout_seconds=43200,
        files_sha256=pins, predecessor='session77-root-ties-gate clean full shelf (consensus only: valid panel opportunity HOLD may skip); winner/fault holds; actual base remainsf1d0a03',
        decision='ownAA40 contains0 in printed/exact pentaCI, zero faults; fixed400 policy kept iff printed/exact lower95>0, zero faults/intact inputs, effective-width receipts; otherwise unresolved/loss; faults unscored',
        automatic_adoption=False, promotion=False, claim=False, limits='one SMP policy change; no automatic adoption, extension, rescue or pooling; long-clock integrated evidence owed'))


def wait():
    previous=lane.parent/"session77-root-ties-gate"
    if not (previous/'controller.done').exists():
        libc=ctypes.CDLL(None);fd=libc.inotify_init1(os.O_NONBLOCK|os.O_CLOEXEC)
        assert fd>=0 and libc.inotify_add_watch(fd,os.fsencode(previous),0x8|0x80)>=0
        try:
            status('predecessor-wait',str(previous/'controller.done')+' native completion')
            end=time.monotonic()+24*3600
            while not (previous/'controller.done').exists():
                seconds=end-time.monotonic();assert seconds>0 and select.select([fd],[],[],seconds)[0];os.read(fd,65536)
        finally:os.close(fd)
    gate=json.loads((previous/'gate-summary.json').read_text()) if (previous/'gate-summary.json').exists() else {}
    allowed=(previous/'controller.exit').read_text().strip()=='0'

    # Consensus may be skipped only for the panel's prospectively frozen
    # opportunity rule; correctness failure is never a skip permission.
    if previous.name == 'session77-consensus-gate' and not gate:
        panel=lane.parent/'session77-smp-panel'
        summary=json.loads((panel/'panel-summary.json').read_text()) if (panel/'panel-summary.json').exists() else {}
        allowed=allowed and (panel/'controller.exit').read_text().strip()=='0' and summary.get('consensus_admission')=='HOLD' and not summary.get('mate_misses') and summary.get('rows')==288 and summary.get('sample_rows')==48 and summary.get('solve_rows')==288
    else:
        allowed=allowed and gate.get('games')==400 and gate.get('pairs')==200 and gate.get('verdict')=='shelved' and gate.get('flags')==[0,0]

    checks=lane.parent/'session77-next-smp'/"helper-depths"
    allowed=allowed and (checks/'checks.exit').read_text().strip()=='0' and json.loads((checks/'identity.json').read_text()).get('verdict')=='PASS'
    status('predecessor-review',dict(allowed=allowed,verdict=gate.get('verdict'),action='unchangedf1d0a03 single-policy S gate' if allowed else 'manual base/instrument hold; no games'))
    return allowed


def widths(paths, base_width, mode):
    counts={'new':{},'base':{}}
    processes=0
    for path in paths:
        rows=[json.loads(line) for line in path.read_text().splitlines()]
        process=rows[0]['payload']; role=process['role']; width={'new':4,'base':base_width}[role]
        assert process['cpus']=={'new':[0,2,4,6],'base':[8,10,12,14]}[role]
        assert process['binary']==('candidate-ngn' if mode=='gate' and role=='new' else 'baseline-ngn')
        assert rows[-1]==dict(kind='exit',payload=0)
        pending=False; effective=None; acknowledgement=False
        for row in rows[1:-1]:
            payload=row['payload']
            if row['kind']=='input' and payload.startswith('setoption name Threads '):
                assert not pending
                assert int(payload.rsplit(' ',1)[1])==width
                acknowledgement=True
            elif row['kind']=='input' and payload.startswith('go '):
                assert not pending and not acknowledgement
                pending=True;effective=None
            elif row['kind']=='output' and payload.startswith('info string threads '):
                match=re.fullmatch(r'info string threads configured (\d+) effective (\d+)',payload)
                assert match and int(match[1])==width
                if acknowledgement:
                    assert int(match[2])==width
                    acknowledgement=False
                else:
                    assert pending and effective is None and 1<=int(match[2])<=width
                    effective=int(match[2])
            elif row['kind']=='output' and payload.startswith('bestmove '):
                assert pending and (width==1 or effective is not None)
                actual=1 if width==1 else effective
                counts[role][actual]=counts[role].get(actual,0)+1
                pending=False
            elif row['kind']=='output' and payload.startswith('info string error '):
                raise AssertionError(payload)
            elif row['kind']=='stderr':
                if 'Crash Handler Initialized' not in payload and 'Crash log:' not in payload:
                    assert not re.search(r'panic|fatal|illegal|crash|error',payload,re.I),payload
        assert not pending and not acknowledgement
        processes+=1
    fractions={role:count.get({'new':4,'base':base_width}[role],0)/sum(count.values()) for role,count in counts.items()}
    assert processes>0 and all(value>=.8 for value in fractions.values())
    return dict(processes=processes,effective_width_counts=counts,full_width_fraction=fractions)


def run():
    if not wait():
        return
    used={key:int(value)*1024 for key,value in re.findall(r'^(MemTotal|MemAvailable):\s+(\d+) kB',Path('/proc/meminfo').read_text(),re.M)}
    disk=os.statvfs('/mnt/c')
    busy=[line for line in subprocess.check_output(['ps','-eo','pid,args'],text=True).splitlines() if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)',line)]
    capacity=dict(used_bytes=used['MemTotal']-used['MemAvailable'],new_budget_bytes=3*1024**3,c_free_bytes=disk.f_bavail*disk.f_frsize,busy=busy)
    write('capacity-before-aa.json',capacity)
    assert not busy and capacity['c_free_bytes']>=25.5*1024**3 and capacity['used_bytes']+3*1024**3<=12*1024**3
    prepare()
    manifest=json.loads((lane/'game-manifest.json').read_text())
    spec=importlib.util.spec_from_file_location('paired_parser',lane/'paired-parser.py')
    parser=importlib.util.module_from_spec(spec);spec.loader.exec_module(parser)
    # Exercise the transparent launchers and per-go receipt parser before AA.
    spec=importlib.util.spec_from_file_location('smp_diagnostic',lane/'smp.py')
    diagnostic=importlib.util.module_from_spec(spec);spec.loader.exec_module(diagnostic)
    diagnostic.LANE=lane
    for base_width in [4]:
        os.environ["NGN_S77_MODE"]="gate"
        before=set(lane.glob('new-*.jsonl'))|set(lane.glob('base-*.jsonl'))
        for role,width in [('new',4),('base',base_width)]:
            engine=diagnostic.UCI(lane/(role+'.sh'),width,f'preflight-{base_width}-{role}.log')
            engine.send('uci');engine.until('uciok')
            for option in [f'EvalFile value {lane/"base.nnue"}', 'UseNNUE value true', 'Hash value 256',f'Threads value {width}', 'OwnBook value false']:
                engine.send('setoption name '+option)
            engine.send('isready');engine.until('readyok')
            for _ in range(2):
                engine.send('ucinewgame');engine.send('position startpos');engine.send('go depth 6')
                engine.until('bestmove ')
            engine.close()
        after=set(lane.glob('new-*.jsonl'))|set(lane.glob('base-*.jsonl'))
        write(f'preflight-{base_width}-widths.json',widths(after-before,base_width,"gate"))
    status('launcher-preflight-pass','two natural bounded searches per role/configuration; CPU masks, exits and effective-width receipts pass')
    for mode in ['aa','gate']:
        for name,pin in manifest['files_sha256'].items():
            assert digest(lane/name)==pin
        before=set(lane.glob('new-*.jsonl'))|set(lane.glob('base-*.jsonl'))
        status(mode+'-start',dict(tc='10+0.1',concurrency=1,threads=[4,4]))
        with (lane/(mode+'.log')).open('w') as out:
            result=subprocess.run(['timeout','--signal=TERM','--kill-after=15s',str(manifest[mode+'_timeout_seconds']),*manifest[mode+'_argv']],stdout=out,stderr=subprocess.STDOUT,env=dict(os.environ,NGN_S77_MODE=mode))
        assert result.returncode==0,('operationally unscored',mode,result.returncode)
        for name,pin in manifest['files_sha256'].items():
            assert digest(lane/name)==pin
        after=set(lane.glob('new-*.jsonl'))|set(lane.glob('base-*.jsonl'))
        receipt=widths(after-before,4,mode)
        write(mode+'-widths.json',receipt)
        crashes=lane/'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in crashes.read_text().splitlines())
        summary=parser.gate((lane/(mode+'.log')).read_text(),mode)
        summary['configuration_only']=False
        write(mode+'-summary.json',summary)
        status(mode+'-done',summary)


if __name__=='__main__':
    prepare() if sys.argv[1:]==['prepare'] else run()

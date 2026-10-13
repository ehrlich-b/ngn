"""One native SEE completion wait, then the reserved CPU/GPU diagnostic block."""
import ctypes
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import select
import subprocess
import time

root = Path('/home/ehrli/ngn-data/night-20261007')
p = root / 'session74-clock-panel'
see = root / 'session72-see-rig'


def status(event, detail):
    with (p/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),event=event,detail=detail))+'\n')

def write(name,data):
    (p/name).write_text(json.dumps(data,indent=2,allow_nan=False)+'\n')

def prepare():
    selection=json.loads((p/'panel-selection.json').read_text())
    rows=json.loads((p/'panel.json').read_text())
    for shard in range(6):
        write(f'panel-shard-{shard}.json',rows[shard::6])
    write('diagnostic-manifest.json',dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        plan='astra-48h-2026-10-12.md section4 reserved post-SEE diagnostic block',
        accepted_source='f1d0a035f8b4f0b7883fed0a859e5769d076089f', source_snapshot='3ab873e documentation only after f1d0a03',
        net_sha256='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86',
        treatment='existing floor10 to25 only, all other engine policy unchanged',
        panel_selection=selection, machine='Ryzen9800X3D / WSL2', cpus='0-11', nice=19,
        concurrency=6, GOMAXPROCS=1, Hash=64, Threads=1, GOAMD64='v3',
        binaries='PGO engine test executables; diagnostic-only helper and observations',
        worker_order='even shards base then horizon; odd shards horizon then base',
        cpu_memory_max_bytes=3*1024**3,gpu_cpus='12,14',gpu_memory_max_bytes=4*1024**3,swap_max=0,
        background='GPU outcome/archive diagnostic declared and launched before CPU samples; no scored games overlap',
        budgets_ms=[1000,4000,16000], played_child_budget_ms=16000,
        clock_policy='replay full history and prior-own-turn allocator phase/move count; actual 120+1 banks; preserve allocation-before-update',
        state='cold TT/history per frame and budget; does not reconstruct warm in-game search state',
        horizon_admission='all traces legal/complete, unchanged static+SEE/fixed source behavior; soft ratio<1 for all54, median<0.75, mean actual tournament elapsed ratio<0.90; no Elo inference',
        gpu_admission='diagnosis only; no capacity/sampling treatment without discriminating signal and adequate split provenance',
        games=0,automatic_adoption=False,automatic_game_nomination=False,
        horizon_next='new immediate-base A/A40 plus fixed400 at120+1, positive printed/exact lower95, zero faults; new manifest before games',
        files_sha256={name:hashlib.sha256((p/name).read_bytes()).hexdigest() for name in
                     ['horizon.patch','panel_test.go','panel_see_test.go','panel.json','panel-selection.json','future-openings.txt','gpu-panel.py','panel-controller.py']}))


def wait_pid(pid):
    fd=os.pidfd_open(pid)
    try:
        poll=select.poll();poll.register(fd,select.POLLIN)
        assert poll.poll(12*3600*1000),'SEE still live; no interruption'
    finally:os.close(fd)


def run():
    status('manual-recovery-review', 'session75: SEE control interrupted by WSL exit; no candidate games or source adoption; unchanged actual base; diagnostic only, no automatic scored successor')
    assert (p/'panel-checks.done').exists() and (p/'panel-checks.exit').read_text().strip()=='0'
    processes=subprocess.check_output(['ps','-eo','pid,ppid,rss,args'],text=True)
    busy=[line for line in processes.splitlines() if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)',line)]
    memory={k:int(v)*1024 for k,v in re.findall(r'^(MemTotal|MemAvailable):\s+(\d+) kB',Path('/proc/meminfo').read_text(),re.M)}
    disk=os.statvfs('/mnt/c')
    receipt=dict(nonavailable_memory_bytes=memory['MemTotal']-memory['MemAvailable'],new_memory_budget_bytes=7*1024**3,
                 c_free_bytes=disk.f_bavail*disk.f_frsize,competing_chess_processes=busy)
    write('capacity-before-panel.json',receipt)
    if busy or receipt['c_free_bytes']<25.5*1024**3 or receipt['nonavailable_memory_bytes']+7*1024**3>12*1024**3:
        status('capacity-hold',receipt);return
    with (p/'gpu-launcher.log').open('w') as log:
        gpu=subprocess.Popen(['systemd-run','--user','--wait','--unit=ngn-s74-gpu-panel',
            '-p','AllowedCPUs=12,14','-p','CPUQuota=200%','-p','MemoryMax=4G','-p','MemorySwapMax=0',
            'nice','-n','19','taskset','-c','12,14','env','PYTHONPATH='+str(p/'base'),
            '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python',str(p/'gpu-panel.py')],stdout=log,stderr=subprocess.STDOUT)
        write('gpu-launch.json',dict(pid=gpu.pid))
        status('panel-start','six CPU workers plus declared GPU evaluator block; fixed54 rows; no games')
        jobs=[]
        for shard in range(6):
            directory=p/'workers'/str(shard);directory.mkdir(parents=True)
            log_file=(p/f'worker-{shard}.log').open('w')
            order=['base','horizon'] if shard%2==0 else ['horizon','base']
            argv=['bash','-c','set -e; for variant in "$@"; do NGN_PANEL_VARIANT="$variant" "$NGN_PANEL_ROOT/$variant-panel.test" -test.run "^TestCampaignPanelDiagnostic$" -test.v -test.timeout=90m; done','panel-worker',*order]
            env=dict(os.environ,GOMAXPROCS='1',NGN_PANEL_ROOT=str(p),NGN_PANEL_FILE=str(p/f'panel-shard-{shard}.json'),
                     NGN_PANEL_NET=str(root/'session72-see-rig/base.nnue'))
            process=subprocess.Popen(['taskset','-c','0-11',*argv],cwd=directory,env=env,stdout=log_file,stderr=subprocess.STDOUT)
            jobs.append((process,log_file))
        write('cpu-launch.json',dict(pids=[j[0].pid for j in jobs],concurrency=6))
        codes=[]
        for process,log_file in jobs:
            codes.append(process.wait());log_file.close()
        gpu_code=gpu.wait()
    write('execution-exits.json',dict(cpu=codes,gpu=gpu_code))
    assert all(code==0 for code in codes) and gpu_code==0,'diagnostic failure; no admission'
    records=[]
    for shard in range(6):
        records.extend(json.loads(line.removeprefix('CAMPAIGNPANEL ')) for line in (p/f'worker-{shard}.log').read_text().splitlines() if line.startswith('CAMPAIGNPANEL '))
    write('cpu-panel-values.json',records)
    assert len(records)==54*8,(len(records),'incomplete fixed diagnostic')
    base={r['id']:r for r in records if r['kind']=='search' and r['variant']=='base' and r['mode']=='tournament'}
    horizon={r['id']:r for r in records if r['kind']=='search' and r['variant']=='horizon'}
    assert len(base)==len(horizon)==54
    ratio=sorted(horizon[key]['soft_ns']/base[key]['soft_ns'] for key in base)
    elapsed_ratio=sum(r['elapsed_ns'] for r in horizon.values())/sum(r['elapsed_ns'] for r in base.values())
    evaluations={(r['id'],r['variant']):r for r in records if r['kind']=='evaluation'}
    static_see_identical=all(all(evaluations[(key,'base')][field]==evaluations[(key,'horizon')][field]
                               for field in ['static_root_stm','q_root_stm','see_disagreements']) for key in base)
    admitted=all(value<1 for value in ratio) and ratio[len(ratio)//2]<.75 and elapsed_ratio<.90 and static_see_identical
    summary=dict(fixed_rows=54,recorded_rows=len(records),cpu_exits=codes,gpu_exit=gpu_code,
        source_static_q_see_identity=static_see_identical,median_soft_ratio=ratio[len(ratio)//2],aggregate_elapsed_ratio=elapsed_ratio,
        baseline_stop_reasons={reason:sum(row['reason']==reason for row in base.values()) for reason in {row['reason'] for row in base.values()}},
        baseline_discarded_tail_ns=sum(row['tail_ns'] for row in base.values()),
        horizon_admission='ELIGIBLE FOR PROSPECTIVE GATE' if admitted else 'HOLD FOR DIAGNOSIS',
        limits='cold cache/history frame replays, observed banks and prior allocator state; no closed-loop bank trajectory or Elo; deep NGN is not ground truth',
        automatic_games=False,automatic_adoption=False)
    write('panel-summary.json',summary);status('panel-done',summary)

if __name__=='__main__':
    import sys
    if sys.argv[1:]==['prepare']:prepare()
    else:run()

"""At GPU's natural terminal, checkpoint-pause SPSA, net gate, speed panel.

Reuses the existing native runner/control and pause helpers. Does not adopt
anything, run a claim, stop a gate, or begin another training cell.
"""
import copy
import datetime
import importlib.util
import json
import os
from pathlib import Path
import re
import select
import subprocess
import sys
import time

R=Path('/home/ehrli/ngn-data/night-20261007')
S=R/'session100-speed'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value,indent=2,allow_nan=False)+'\n')


def module(name, path):
    spec=importlib.util.spec_from_file_location(name,path)
    result=importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def status(event, detail):
    with (R/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session100-boundary',event=event,detail=detail))+'\n')


def wait_pid(pid):
    try: fd=os.pidfd_open(pid)
    except ProcessLookupError: return
    try: select.select([fd],[],[])
    finally: os.close(fd)


def main():
    config=load(Path(sys.argv[1]) if len(sys.argv)>1 else S/'boundary-nomination.json')
    status('waiting-gpu',dict(pid=config['gpu_controller_pid'],expected_ready_edt=config['estimated_gpu_ready_edt']))
    wait_pid(config['gpu_controller_pid'])
    cell=R/'session94/gpu-next/rebalanced-t74'
    panel_only=config.get('panel_only',False)
    if panel_only:
        assert 'C: reserve crossed' in (R/'session94/gpu-next/controller.log').read_text()
    else:
        final=load(cell/'result.json')
        assert final['epochs']==115 and final['selection']=='final115'
        assert final['reexport_identical'] and final['parity_cases']==20
    assert (S/'checks.exit').read_text().strip()=='0'
    assert (S/'panel-build.exit').read_text().strip()=='0'
    pause_helpers=module('s93_pause_helpers',R/'session93/net-chain.py')
    start=pause_helpers.checkpoint()
    assert start is not None
    if start['iter']>=2000:
        # Preserve the final-vector gate's place if the tune finishes early.
        status('tune-complete-review',dict(iter=start['iter'],next='Final-vector gate before panel; no service stopped'))
        write(S/'boundary-result.json',dict(state='TUNE_COMPLETE_REVIEW',automatic_adoption=False))
        return
    status('waiting-checkpoint',dict(after_iter=start['iter'],interval_seconds=300))
    while True:
        time.sleep(300)
        row=pause_helpers.checkpoint()
        if row is not None and row['iter']>start['iter']:
            break
        os.kill(config['tuner_pid'],0)
    original=load(R/'session88/spsa-warm/nomination.json')
    if row['iter']>=2000:
        write(S/'boundary-result.json',dict(state='TUNE_COMPLETE_REVIEW',automatic_adoption=False))
        status('tune-complete-review','Endpoint arrived during checkpoint wait; run final-vector gate first')
        return
    resume_argv=original['argv']+['-resume']
    pause=dict(state='PLANNED_PAUSE',checkpoint=row,resume_argv=resume_argv,
        authority='October17 09:45 item2 and item3; planned checkpoint pause for net gate/exclusive speed panel',
        tuning_service=config['tuning_service'])
    write(S/'spsa-planned-pause.json',pause)
    subprocess.run(['systemctl','--user','stop',config['tuning_service']],check=True)
    wait_pid(config['tuner_pid'])
    row=pause_helpers.checkpoint()
    assert row is not None and row['iter']>=pause['checkpoint']['iter']
    pause.update(state='PAUSED_AT_VERIFIED_CHECKPOINT',checkpoint=row)
    write(S/'spsa-planned-pause.json',pause)
    status('spsa-paused',dict(iter=row['iter'],games=12*row['iter']))
    if panel_only:
        result=dict(verdict='NOT_RUN_DISK_STOP',faults=[])
    else:
        job=copy.deepcopy(load(R/'session87/net-final174/manifest.json'))
        oldnet=next(x.split('=',1)[1] for x in job['candidate_options'] if x.startswith('EvalFile='))
        for key in ['argv','environment','pid','started_utc','reused_control','source_checks','finalization']:
            job.pop(key,None)
        job['artifacts'].pop(oldnet)
        net=str(cell/'trained/latest.nnue')
        job.update(name='s100-net-rebalanced-t74',output=str(S/'net-rebalanced-t74'),
            net_sha256=final['net_sha256'],finalization=final,
            candidate_options=['EvalFile='+net]+[x for x in job['candidate_options'] if not x.startswith('EvalFile=')],
            treatment='Rebalanced T74 final115 vs accepted81215229; same engine/final14',review_after=False)
        job['artifacts'][net]=final['net_sha256']
        write(S/'net-nomination.json',job)
        runner=module('campaign_queue',R/'session93/campaign_queue.py')
        result=runner.run(job,R/'session80/controls.json',R/'status.jsonl')
        assert result['verdict'] in ('KEEP','SHELVED') and not result['faults'],result
    processes=subprocess.check_output(['ps','-eo','pid,args'],text=True).splitlines()
    busy=[line for line in processes if re.search(r'trainer\.train|(?:^|/)(?:sprt|spsa|datagen)(?:\s|$)',line)]
    assert not busy,busy
    status('exclusive-panel-start',dict(net_verdict=result['verdict'],expected_minutes=110,
        accepted_package='9ec5d31/81215229/final14; no adoption; review profile before speed timings'))
    with (S/'panel.log').open('x') as log:
        subprocess.run(['taskset','-c','0-15','nice','-n','19','python3',str(S/'panel.py')],
            check=True,stdout=log,stderr=subprocess.STDOUT)
    # Any fault above intentionally holds for review; a clean panel resumes the
    # saved tune once, on the unchanged net and the exact native RNG/theta state.
    argv=['systemd-run','--user','--unit=ngn-s100-spsa-resume','-p','Type=exec',
        '-p','AllowedCPUs=0-11','-p','CPUQuota=600%','-p','MemoryMax=3G','-p','MemorySwapMax=0',
        '-p','Environment=GOMAXPROCS=1','-p','StandardOutput=append:'+str(R/'session88/spsa-warm/tune.log'),
        '-p','StandardError=inherit','nice','-n','19','taskset','-c','0-11']+resume_argv
    subprocess.run(argv,check=True)
    write(S/('panel-boundary-result.json' if panel_only else 'boundary-result.json'),dict(state='PANEL_COMPLETE_TUNE_RESUMED' if panel_only else 'NET_GATE_PANEL_COMPLETE_TUNE_RESUMED',
        net_result=result,panel=str(S/'panel/result.json'),resume_service='ngn-s100-spsa-resume.service',
        automatic_adoption=False,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()))
    status('complete',dict(net_verdict=result['verdict'],panel=str(S/'panel/result.json'),
        resume_service='ngn-s100-spsa-resume.service',next='Review hotspot evidence; balance S1 then S2 speed gates separately at an exclusive boundary'))


if __name__=='__main__':
    try: main()
    except BaseException as error:
        write(S/('panel-boundary-result.json' if len(sys.argv)>1 else 'boundary-result.json'),dict(state='HELD_REVIEW',error=str(error),automatic_adoption=False))
        status('held-review',str(error))
        raise

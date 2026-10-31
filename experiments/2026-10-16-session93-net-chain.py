"""Wait for natural training completion, checkpoint-pause SPSA, gate both nets.

WSL only. Uses the existing campaign runner and reusable controls. No gate is
interrupted, no adoption occurs, and a training fault leaves the tune running.
"""
import copy
import datetime
import importlib.util
import json
import os
from pathlib import Path
import select
import subprocess
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session93'
W = R / 'session88/spsa-warm'
V = Path('/home/ehrli/ngn-data/claim-v4-20261016/tooling-koivisto')


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session93-net-chain', event=event, detail=detail)) + '\n')


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def checkpoint():
    # The native writer appends and closes each report. A partial concurrent
    # write is retried at the next five-minute check, never used for pausing.
    try:
        row = json.loads((W / 'spsa.jsonl').read_text().splitlines()[-1])
    except (IndexError, json.JSONDecodeError):
        return None
    recipe = row['recipe']
    assert row['version'] == 2 and recipe['iterations'] == 2000 and recipe['pairs'] == 6
    assert not recipe['single'] and row['iter'] % recipe['report'] == 0
    assert recipe['engine_options'][0].endswith('/session82/net-final115/candidate.nnue')
    return row


def main():
    config = json.loads((D / 'net-chain-nomination.json').read_text())
    ready = json.loads((V / 'build-receipt.json').read_text())
    assert ready['state'] == 'READY_TOOLING' and all(x['status'] == 'PASS' for x in ready['dry_modes'].values())
    status('waiting-training', dict(pid=config['training_controller_pid'],
                                   result=str(R / 'session87/mixture/moderated-t74/result.json')))
    wait_pid(config['training_controller_pid'])
    cells = []
    for name in ['volume-control', 'moderated-t74']:
        cell = R / 'session87/mixture' / name
        result = json.loads((cell / 'result.json').read_text())
        assert result['epochs'] == 115 and result['selection'] == 'final115'
        assert result['reexport_identical'] and result['parity_cases'] == 20
        cells.append((name, cell, result))
    start = checkpoint()
    assert start is not None, 'checkpoint not complete; review without pausing'
    status('waiting-checkpoint', dict(after_iter=start['iter'], interval_seconds=300,
                                      tuner_pid=config['tuner_pid']))
    while start['iter'] < 2000:
        time.sleep(300)
        row = checkpoint()
        if row is not None and row['iter'] > start['iter']:
            break
        # A natural completed tune may proceed; a failed tune needs review.
        try:
            os.kill(config['tuner_pid'], 0)
        except ProcessLookupError:
            raise RuntimeError('tuner stopped before the next checkpoint; review')
    else:
        row = start
    original = json.loads((W / 'nomination.json').read_text())
    pause = dict(state='PLANNED_PAUSE', authority=config['authority'],
                 checkpoint=row, checkpoint_path=str(W/'spsa.jsonl'),
                 resume_argv=original['argv'] + ['-resume'], environment={'GOMAXPROCS':'1'},
                 scope='AllowedCPUs0-11/CPUQuota600%/MemoryMax3G/MemorySwapMax0/nice19',
                 controller_note='Original fresh-start controller is stopped; do not rerun it on populated history.',
                 recorded_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(D / 'spsa-planned-pause.json', pause)
    subprocess.run(['systemctl','--user','stop',config['tuning_service']], check=True)
    wait_pid(config['tuning_controller_pid'])
    final = checkpoint()
    assert final is not None and final['iter'] >= row['iter']
    pause.update(checkpoint=final, state='PAUSED_AT_VERIFIED_CHECKPOINT',
                 stopped_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    (D / 'spsa-paused-history.jsonl').write_bytes((W / 'spsa.jsonl').read_bytes())
    write(D / 'spsa-planned-pause.json', pause)
    status('spsa-paused', dict(iter=final['iter'],games=12*final['iter'], disposition=str(D/'spsa-planned-pause.json')))
    template = json.loads((R / 'session87/net-final174/manifest.json').read_text())
    jobs = []
    for name, cell, result in cells:
        job = copy.deepcopy(template)
        oldnet = next(x.split('=',1)[1] for x in job['candidate_options'] if x.startswith('EvalFile='))
        for key in ['argv','environment','pid','started_utc','reused_control','source_checks','finalization']:
            job.pop(key,None)
        job['artifacts'].pop(oldnet)
        net = str(cell/'trained/latest.nnue')
        job.update(name='s93-net-'+name, output=str(D/('net-'+name)),
                   net_sha256=result['net_sha256'], finalization=result,
                   candidate_options=['EvalFile='+net]+[x for x in job['candidate_options'] if not x.startswith('EvalFile=')],
                   treatment=name+' final115 only vs accepted81215229; identical engine/final14/Threads1/Hash64',
                   review_after=False)
        job['artifacts'][net] = result['net_sha256']
        jobs.append(job)
    queue = dict(controls=str(R/'session80/controls.json'),status=str(R/'status.jsonl'),jobs=jobs)
    write(D/'net-queue.json',queue)
    spec=importlib.util.spec_from_file_location('campaign_queue',D/'campaign_queue.py')
    runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
    results=[]
    for job in jobs:
        result=runner.run(job,Path(queue['controls']),Path(queue['status']))
        assert result['verdict'] in ('KEEP','SHELVED') and not result['faults'], result
        results.append(result)
    result=dict(state='REVIEW_NET_GATES',results=results,automatic_adoption=False,
                next='Review both fixed1000 verdicts; any claim candidate needs fixed400/60+0.6 point>0 before prospective v4 identity/receipt/screen.',
                finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(D/'net-chain-result.json',result)
    status('complete',dict(result=str(D/'net-chain-result.json'),verdicts=[x['verdict'] for x in results]))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(D/'net-chain-result.json',dict(state='HELD_REVIEW',error=str(error),automatic_adoption=False))
        status('held-review',str(error))
        raise

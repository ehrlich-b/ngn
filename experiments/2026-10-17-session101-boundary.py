"""At the GPU terminal, gate rebalanced T74, then S3 if the net is shelved.

Native waits for existing jobs; checkpoint-only planned SPSA pause; resume the
same tune once after clean terminals. No adoption, claim, or gate interruption.
"""
import copy
import datetime
import errno
import importlib.util
import json
import os
from pathlib import Path
import re
import select
import subprocess
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
S = R / 'session101-speed'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session101-boundary', event=event, detail=detail))+'\n')


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except OSError as error:
        if error.errno in (errno.ESRCH, errno.EINVAL) and not Path(f'/proc/{pid}').exists():
            return
        raise
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def resume_tune(pause):
    command = ['systemd-run', '--user', '--unit=ngn-s101-spsa-resume', '-p', 'Type=exec',
        '-p', 'AllowedCPUs=0-11', '-p', 'CPUQuota=600%', '-p', 'MemoryMax=3G', '-p', 'MemorySwapMax=0',
        '-p', 'Environment=GOMAXPROCS=1', '-p', 'StandardOutput=append:'+str(R/'session88/spsa-warm/tune.log'),
        '-p', 'StandardError=inherit', 'nice', '-n', '19'] + pause['resume_argv']
    subprocess.run(command, check=True)


def main():
    config = load(S/'boundary-nomination.json')
    status('waiting-training', dict(pid=config['gpu_controller_pid'],
        completion='natural final115 plus identical reexport and20 parity; then next checkpoint',
        checks_pid=config['checks_pid']))
    wait_pid(config['checks_pid'])
    assert (S/'checks.exit').read_text().strip() == '0'
    assert load(S/'identity.json')['verdict'] == 'PASS'
    wait_pid(config['gpu_controller_pid'])
    cell = R/'session94/gpu-next/rebalanced-t74'
    final = load(cell/'result.json')
    assert final['epochs'] == 115 and final['selection'] == 'final115'
    assert final['reexport_identical'] and final['parity_cases'] == 20
    helpers = module('s93_pause_helpers', R/'session93/net-chain.py')
    start = helpers.checkpoint()
    assert start is not None
    if start['iter'] >= 2000:
        write(S/'boundary-result.json', dict(state='TUNE_COMPLETE_REVIEW', automatic_adoption=False,
            next='Final-vector gate takes priority; net and S3 remain unstarted'))
        status('tune-complete-review', dict(iter=start['iter']))
        return
    status('waiting-checkpoint', dict(after_iter=start['iter'], interval_seconds=300))
    while True:
        time.sleep(300)
        checkpoint = helpers.checkpoint()
        if checkpoint is not None and checkpoint['iter'] > start['iter']:
            break
        os.kill(config['tuner_pid'], 0)
    if checkpoint['iter'] >= 2000:
        write(S/'boundary-result.json', dict(state='TUNE_COMPLETE_REVIEW', automatic_adoption=False,
            next='Final-vector gate before net/S3; no service stopped'))
        return
    pause = dict(state='PLANNED_PAUSE', checkpoint=checkpoint,
        resume_argv=load(R/'session88/spsa-warm/nomination.json')['argv']+['-resume'],
        tuning_service=config['tuning_service'], authority=config['authority'])
    write(S/'spsa-planned-pause.json', pause)
    subprocess.run(['systemctl','--user','stop',config['tuning_service']], check=True)
    wait_pid(config['tuner_pid'])
    checkpoint = helpers.checkpoint()
    assert checkpoint is not None and checkpoint['iter'] >= pause['checkpoint']['iter']
    pause.update(state='PAUSED_AT_VERIFIED_CHECKPOINT', checkpoint=checkpoint)
    write(S/'spsa-planned-pause.json', pause)
    status('spsa-paused', dict(iter=checkpoint['iter'], games=12*checkpoint['iter']))
    job = copy.deepcopy(load(R/'session87/net-final174/manifest.json'))
    oldnet = next(x.split('=',1)[1] for x in job['candidate_options'] if x.startswith('EvalFile='))
    for key in ['argv','environment','pid','started_utc','reused_control','source_checks','finalization']:
        job.pop(key,None)
    job['artifacts'].pop(oldnet)
    net = str(cell/'trained/latest.nnue')
    job.update(name='s101-net-rebalanced-t74', output=str(S/'net-rebalanced-t74'),
        net_sha256=final['net_sha256'], finalization=final,
        candidate_options=['EvalFile='+net]+[x for x in job['candidate_options'] if not x.startswith('EvalFile=')],
        treatment='Rebalanced T74 final115 vs accepted81215229; same accepted engine/final14', review_after=False)
    job['artifacts'][net] = final['net_sha256']
    write(S/'net-nomination.json', job)
    runner = module('campaign_queue', R/'session93/campaign_queue.py')
    net_result = runner.run(job, R/'session80/controls.json', R/'status.jsonl')
    assert net_result['verdict'] in ('KEEP','SHELVED') and not net_result['faults'], net_result
    speed_result = None
    if net_result['verdict'] == 'SHELVED':
        processes = subprocess.check_output(['ps','-eo','pid,args'], text=True).splitlines()
        busy = [line for line in processes if re.search(r'trainer\.train|(?:^|/)(?:sprt|spsa|datagen)(?:\s|$)',line)]
        assert not busy, busy
        status('s3-timing-start', dict(expected_minutes=80, source=config['s3_topic_commit'],
            base='accepted9ec5d31/81215229/final14', next='Fixed40 paired blocks per topology; no extensions'))
        command = ['taskset','-c','0-15','nice','-n','19','python3',str(S/'speed-gate.py'),
            '--candidate','s3','--profile-admission',str(S/'s3-profile-admission.json'),
            '--out',str(S/'s3-timing'),'--base-binary',str(R/'session100-speed/baseline-ngn'),
            '--candidate-binary',str(S/'s3-ngn'),'--identity',str(S/'identity.json')]
        with (S/'s3-timing.log').open('x') as log:
            subprocess.run(command, check=True, stdout=log, stderr=subprocess.STDOUT)
        speed_result = load(S/'s3-timing/result.json')
        assert speed_result['verdict'] in ('KEEP_ELIGIBLE','SHELVED'), speed_result
    # A winning net changes the next base after review. Do not time S3 on a
    # superseded net or integrate either candidate automatically.
    resume_tune(pause)
    state = 'REVIEW_NET_WINNER_TUNE_RESUMED' if net_result['verdict']=='KEEP' else 'NET_S3_COMPLETE_TUNE_RESUMED'
    write(S/'boundary-result.json', dict(state=state, net_result=net_result, speed_result=speed_result,
        resume_service='ngn-s101-spsa-resume.service', automatic_adoption=False,
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        next='Review net winner then transfer400, or S3 timing then immediate-base S2/S1; no claim armed'))
    status('complete', dict(state=state, net_verdict=net_result['verdict'],
        speed_verdict=speed_result['verdict'] if speed_result else 'UNSTARTED_NET_WINNER',
        resume_service='ngn-s101-spsa-resume.service'))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(S/'boundary-result.json', dict(state='HELD_REVIEW', error=str(error), automatic_adoption=False))
        status('held-review',str(error))
        raise

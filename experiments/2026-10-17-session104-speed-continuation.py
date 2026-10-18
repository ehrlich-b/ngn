"""Continue S2/S1 against accepted S3, after their rebuilt checks.

Use existing identities, panel, timing runner and native SPSA resume. A winner
holds successors for adoption/rebuild; no game gate is interrupted or extended.
"""
import datetime
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session104-speed'
S = R / 'session101-speed'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session104-speed', event=event, detail=detail))+'\n')


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def eligible(previous):
    return (previous['state'] == 'NET_S3_COMPLETE_TUNE_RESUMED'
            and previous['net_result']['verdict'] == 'SHELVED'
            and previous['speed_result']['verdict'] == 'KEEP_ELIGIBLE')


def main():
    config = load(D/'nomination.json')
    previous_helpers = module('s101_boundary', S/'boundary.py')
    status('waiting-successor-checks', dict(pid=config['checks_pid'],
        next='Rebuilt S2 then S1 against accepted S3; winner holds successor'))
    previous_helpers.wait_pid(config['checks_pid'])
    assert (D/'checks.exit').read_text().strip() == '0'
    for candidate in config['candidates']:
        assert load(D/(candidate['candidate']+'-identity.json'))['verdict'] == 'PASS'
    previous_helpers.wait_pid(config['boundary_pid'])
    previous = load(S/'boundary-result.json')
    if not eligible(previous):
        write(D/'result.json', dict(state='HELD_PREDECESSOR_REVIEW', previous=previous,
            automatic_adoption=False, next='Review net/S3 winner or predecessor fault; tune untouched'))
        status('held-predecessor-review', dict(state=previous['state']))
        return
    helpers = module('s93_checkpoint', R/'session93/net-chain.py')
    start = helpers.checkpoint()
    assert start is not None
    if start['iter'] >= 2000:
        write(D/'result.json', dict(state='HELD_FINAL_VECTOR_PRIORITY', automatic_adoption=False))
        status('held-final-vector-priority', dict(iter=start['iter']))
        return
    service = previous['resume_service']
    tuner_pid = int(subprocess.check_output(['systemctl', '--user', 'show', service,
        '--property=MainPID', '--value'], text=True).strip())
    assert tuner_pid > 0
    status('waiting-checkpoint', dict(after_iter=start['iter'], pid=tuner_pid, interval_seconds=300))
    while True:
        time.sleep(300)
        checkpoint = helpers.checkpoint()
        if checkpoint is not None and checkpoint['iter'] > start['iter']:
            break
        # A failed tune needs review rather than an automatic restart.
        assert Path(f'/proc/{tuner_pid}').exists(), 'tuner exited before next checkpoint'
    if checkpoint['iter'] >= 2000:
        write(D/'result.json', dict(state='HELD_FINAL_VECTOR_PRIORITY', automatic_adoption=False))
        status('held-final-vector-priority', dict(iter=checkpoint['iter']))
        return
    pause = dict(state='PLANNED_PAUSE', checkpoint=checkpoint, tuning_service=service,
        resume_argv=load(S/'spsa-planned-pause.json')['resume_argv'], authority=config['authority'])
    write(D/'spsa-planned-pause.json', pause)
    subprocess.run(['systemctl','--user','stop',service], check=True)
    previous_helpers.wait_pid(tuner_pid)
    final = helpers.checkpoint()
    assert final is not None and final['iter'] >= checkpoint['iter']
    pause.update(state='PAUSED_AT_VERIFIED_CHECKPOINT', checkpoint=final)
    write(D/'spsa-planned-pause.json', pause)
    status('spsa-paused', dict(iter=final['iter'], games=12*final['iter']))
    processes = subprocess.check_output(['ps','-eo','pid,args'], text=True).splitlines()
    busy = [line for line in processes if re.search(r'trainer\.train|(?:^|/)(?:sprt|spsa|datagen)(?:\s|$)',line)]
    assert not busy, busy
    results = []
    for candidate in config['candidates']:
        name = candidate['candidate']
        status('timing-start', dict(candidate=name, expected_minutes=80,
            base=config['accepted_package'], rule=config['rule']))
        command = ['taskset','-c','0-15','nice','-n','19','python3',str(S/'speed-gate.py'),
            '--candidate',name,'--profile-admission',str(D/(name+'-profile-admission.json')),
            '--out',str(D/(name+'-timing')),'--base-binary',config['base_binary'],
            '--candidate-binary',candidate['binary'],'--identity',str(D/(name+'-identity.json'))]
        with (D/(name+'-timing.log')).open('x') as stream:
            subprocess.run(command, check=True, stdout=stream, stderr=subprocess.STDOUT)
        result = load(D/(name+'-timing/result.json'))
        assert result['verdict'] in ('SHELVED','KEEP_ELIGIBLE'), result
        results.append(dict(candidate=name, result=result))
        status('timing-complete', dict(candidate=name, verdict=result['verdict']))
        if result['verdict'] == 'KEEP_ELIGIBLE':
            break
    resume_service = 'ngn-s104-spsa-resume.service'
    subprocess.run(['systemd-run','--user','--unit='+resume_service,'-p','Type=exec',
        '-p','AllowedCPUs=0-11','-p','CPUQuota=600%','-p','MemoryMax=3G','-p','MemorySwapMax=0',
        '-p','Environment=GOMAXPROCS=1',
        '-p','StandardOutput=append:'+str(R/'session88/spsa-warm/tune.log'),
        '-p','StandardError=inherit','nice','-n','19'] + pause['resume_argv'], check=True)
    # S3 is accepted; its code does not alter training. The GPU recipe can
    # start once S1 has finished and no successor needs another exclusive gate.
    gpu_started = len(results) == len(config['candidates'])
    if gpu_started:
        subprocess.run(config['gpu_after_speed_argv'], check=True)
        status('gpu-recipe-launched', dict(service='ngn-s104-wdl050.service',
            cell='wdl050', next='Final115/export/parity;no automatic games or adoption'))
    write(D/'result.json', dict(state='SPEED_COMPLETE_TUNE_RESUMED', results=results,
        resume_service=resume_service, gpu_recipe_launched=gpu_started, automatic_adoption=False,
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        next='Review speed winner and rebuild successors, or continue SPSA and GPU recipe work'))
    status('complete', dict(verdicts=[(x['candidate'],x['result']['verdict']) for x in results],
        resume_service=resume_service))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(D/'result.json', dict(state='HELD_REVIEW', error=str(error), automatic_adoption=False))
        status('held-review',str(error))
        raise

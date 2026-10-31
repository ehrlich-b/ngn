"""Gate WDL.50 at its natural terminal, then transfer a winner or time S4.

Reuse campaign_queue, controls, checkpoint state and the fixed speed runner.
No adoption, new training, claim, gate interruption or sample extension.
"""
import copy
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session109'
S = R/'session108-speed'
CELL = R/'session104-speed/recipe/wdl050'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False)+'\n')


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session109-boundary', event=event, detail=detail))+'\n')


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def net_job(config, final):
    job = copy.deepcopy(load(R/'session101-speed/net-rebalanced-t74/manifest.json'))
    for key in ['argv','environment','pid','started_utc','reused_control','source_checks','finalization']:
        job.pop(key, None)
    net = str(CELL/'trained/latest.nnue')
    binary = config['base_binary']
    retained = [job['harness'], job['book'], config['base_net']]
    job['artifacts'] = {path: job['artifacts'][path] for path in retained}
    job['artifacts'].update({binary: config['base_binary_sha256'], net: final['net_sha256']})
    job.update(name='s109-net-wdl050', output=str(D/'net-wdl050'),
        base_source='fe92ced', candidate_source='fe92ced', candidate=binary, base=binary,
        net_sha256=final['net_sha256'], finalization=final, review_after=False,
        treatment='WDL.50 final115 vs accepted81215229; same accepted fe92ced/final14',
        candidate_options=['EvalFile='+net]+[o for o in job['candidate_options'] if not o.startswith('EvalFile=')])
    return job


def transfer_job(job):
    result = copy.deepcopy(job)
    result.update(name='s109-net-wdl050-transfer', output=str(D/'net-transfer'),
        tc='60+0.6', max_games=400, min_games=400, game_timeout=900,
        decision='Fixed400/200 at60+0.6;exact point>0,zero faults;no extension or automatic adoption')
    result['class'] = 'net-transfer'
    result['control_key'] = job['control_key'].replace('/10+0.1/', '/60+0.6/')
    return result


def main():
    config = load(D/'nomination.json')
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    previous = module('s101_boundary', R/'session101-speed/boundary.py')
    helpers = module('s93_checkpoint', R/'session93/net-chain.py')
    status('waiting-training', dict(controller_pid=os.getpid(), pid=config['gpu_controller_pid'],
        completion='Final115/export/20parity;next durable tune checkpoint;net gate first'))
    previous.wait_pid(config['gpu_controller_pid'])
    final = load(CELL/'result.json')
    assert final['epochs'] == 115 and final['selection'] == 'final115'
    assert final['reexport_identical'] and final['parity_cases'] == 20
    start = helpers.checkpoint()
    assert start is not None
    if start['iter'] >= 2000:
        write(D/'result.json', dict(state='HELD_FINAL_VECTOR_PRIORITY', automatic_adoption=False))
        status('held-final-vector-priority', dict(iter=start['iter']))
        return
    status('waiting-checkpoint', dict(after_iter=start['iter'], interval_seconds=300))
    while True:
        time.sleep(300)
        checkpoint = helpers.checkpoint()
        if checkpoint is not None and checkpoint['iter'] > start['iter']:
            break
        assert Path('/proc/'+str(config['tuner_pid'])).exists(), 'tuner exited before next checkpoint'
    if checkpoint['iter'] >= 2000:
        write(D/'result.json', dict(state='HELD_FINAL_VECTOR_PRIORITY', automatic_adoption=False))
        status('held-final-vector-priority', dict(iter=checkpoint['iter']))
        return
    pause = dict(state='PLANNED_PAUSE', checkpoint=checkpoint,
        resume_argv=load(R/'session88/spsa-warm/nomination.json')['argv']+['-resume'],
        tuning_service=config['tuning_service'], authority=config['authority'])
    write(D/'spsa-planned-pause.json', pause)
    subprocess.run(['systemctl','--user','stop',config['tuning_service']], check=True)
    previous.wait_pid(config['tuner_pid'])
    checkpoint = helpers.checkpoint()
    assert checkpoint is not None and checkpoint['iter'] >= pause['checkpoint']['iter']
    pause.update(state='PAUSED_AT_VERIFIED_CHECKPOINT', checkpoint=checkpoint)
    write(D/'spsa-planned-pause.json', pause)
    status('spsa-paused', dict(iter=checkpoint['iter'], games=12*checkpoint['iter']))
    runner = module('campaign_queue', R/'session93/campaign_queue.py')
    job = net_job(config, final)
    write(D/'net-nomination.json', job)
    net_result = runner.run(job, R/'session80/controls.json', R/'status.jsonl')
    assert net_result['verdict'] in ('KEEP','SHELVED') and not net_result['faults'], net_result
    transfer_result = speed_result = None
    if net_result['verdict'] == 'KEEP':
        transfer = transfer_job(job)
        write(D/'transfer-nomination.json', transfer)
        transfer_result = runner.run(transfer, R/'session80/controls.json', R/'status.jsonl')
        assert transfer_result['verdict'] in ('TRANSFER_PASS','TRANSFER_FAIL') and not transfer_result['faults'], transfer_result
    else:
        # Reuse the prior exclusive-process check; no new timing instrument.
        import re
        processes = subprocess.check_output(['ps','-eo','pid,args'], text=True).splitlines()
        busy = [line for line in processes if re.search(r'trainer\.train|(?:^|/)(?:sprt|spsa|datagen)(?:\s|$)', line)]
        assert not busy, busy
        status('s4-timing-start', dict(expected_minutes=80, base=config['accepted_package'],
            rule='Fixed40 paired20s blocks/topology;both order strata lower95>1 at all three topologies'))
        with (D/'s4-timing.log').open('x') as log:
            subprocess.run(['bash',str(S/'timing.sh')], check=True, stdout=log, stderr=subprocess.STDOUT)
        speed_result = load(S/'s4-timing/result.json')
        assert speed_result['verdict'] in ('KEEP_ELIGIBLE','SHELVED'), speed_result
    service = 'ngn-s109-spsa-resume.service'
    subprocess.run(['systemd-run','--user','--unit='+service,'-p','Type=exec',
        '-p','AllowedCPUs=0-11','-p','CPUQuota=600%','-p','MemoryMax=3G','-p','MemorySwapMax=0',
        '-p','Environment=GOMAXPROCS=1','-p','StandardOutput=append:'+str(R/'session88/spsa-warm/tune.log'),
        '-p','StandardError=inherit','nice','-n','19','taskset','-c','0-11']+pause['resume_argv'], check=True)
    state = 'REVIEW_NET_WINNER_TUNE_RESUMED' if net_result['verdict']=='KEEP' else 'NET_S4_COMPLETE_TUNE_RESUMED'
    write(D/'result.json', dict(state=state, net_result=net_result, transfer_result=transfer_result,
        speed_result=speed_result, resume_service=service, automatic_adoption=False,
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()))
    status('complete', dict(state=state, net_verdict=net_result['verdict'],
        transfer_verdict=transfer_result['verdict'] if transfer_result else None,
        speed_verdict=speed_result['verdict'] if speed_result else None, resume_service=service))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(D/'result.json', dict(state='HELD_REVIEW', error=str(error), automatic_adoption=False))
        status('held-review', str(error))
        raise

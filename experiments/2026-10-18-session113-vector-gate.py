"""Gate the native warm tune's final2000 vector at its natural terminal.

One fixed1000 gate using the existing runner/control. No tune pause, training,
claim, engine adoption or changes to any running gate.
"""
import copy
import datetime
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session113-vector'
W = R/'session88/spsa-warm'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False)+'\n')


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session113-vector', event=event, detail=detail))+'\n')


def final_endpoint(history, log):
    endpoint = json.loads(history.splitlines()[-1])
    recipe = endpoint['recipe']
    assert endpoint['version'] == 2 and endpoint['iter'] == recipe['iterations'] == 2000
    assert recipe['pairs'] == 6 and not recipe['single']
    assert '=== FINAL VECTOR ' in log, 'no successful native terminal marker'
    reports = [line for line in log.splitlines() if 'forfeits)' in line]
    assert reports and all('(0 forfeits)' in line for line in reports), 'tune forfeits'
    final_text = log.rsplit('=== FINAL VECTOR ', 1)[1]
    options = {name:int(value) for name,value in re.findall(r'^setoption name (\w+) value (-?\d+)', final_text, re.M)}
    assert options == endpoint['rounded'], 'native final options differ from checkpoint'
    assert set(options) == {p['name'] for p in recipe['params']}
    assert recipe['engine_options'][0].endswith('/session82/net-final115/candidate.nnue')
    return endpoint


def vector_job(template, endpoint):
    job = copy.deepcopy(template)
    for key in ['argv','environment','pid','started_utc','reused_control','source_checks','finalization']:
        job.pop(key, None)
    binary = job['base']
    old_candidate_net = next(x.split('=',1)[1] for x in job['candidate_options'] if x.startswith('EvalFile='))
    job['artifacts'].pop(old_candidate_net, None)
    names = set(endpoint['rounded'])
    candidate = [x for x in job['base_options'] if x.split('=',1)[0] not in names]
    candidate += [name+'='+str(value) for name,value in sorted(endpoint['rounded'].items())]
    job.update(name='s113-spsa-final2000', output=str(D/'gate'), candidate=binary,
        candidate_source='fe92ced', candidate_options=candidate,
        net_sha256=job['base_net_sha256'], candidate_vector=endpoint['rounded'],
        class_note='SPSA vector, fixed-count net/vector rule',
        treatment='Final2000 warm24k rounded vector versus compiled final14; identical accepted fe92ced/81215229 binary and net',
        decision='Fixed1000/500 at10+0.1/c6; printed AND exact lower95>0; zero faults; no extension or automatic adoption',
        tuning_endpoint=endpoint, review_after=True)
    return job


def main():
    config = load(D/'nomination.json')
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    status('waiting-tune', dict(pid=config['tuner_pid'], endpoint='2000/24000; natural successful terminal'))
    module('s101_wait', R/'session101-speed/boundary.py').wait_pid(config['tuner_pid'])
    endpoint = final_endpoint((W/'spsa.jsonl').read_text(), (W/'tune.log').read_text())
    processes = subprocess.check_output(['ps','-eo','pid,args'], text=True).splitlines()
    busy = [line for line in processes if re.search(r'trainer\.train|(?:^|/)(?:sprt|spsa|datagen)(?:\s|$)', line)]
    assert not busy, busy
    template = load(D/'template.json')
    assert template['base_source'] == template['candidate_source'] == config['accepted_source']
    job = vector_job(template, endpoint)
    write(D/'gate-nomination.json', job)
    runner = module('campaign_queue', R/'session93/campaign_queue.py')
    result = runner.run(job, R/'session80/controls.json', R/'status.jsonl')
    assert result['verdict'] in ('KEEP','SHELVED') and not result['faults'], result
    write(D/'result.json', dict(state='REVIEW_VECTOR_GATE', gate_result=result,
        automatic_adoption=False, finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        next='KEEP: review/adopt vector then retarget unstarted work; SHELVED: dispatch checked search SPRTs; S5 timing after due gates'))
    status('complete', dict(verdict=result['verdict'], result=str(D/'result.json')))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(D/'result.json', dict(state='HELD_REVIEW', error=str(error), automatic_adoption=False))
        status('held-review', str(error))
        raise

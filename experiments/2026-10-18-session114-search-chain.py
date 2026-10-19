"""Continue checked candidates only while the immediate accepted base is unchanged."""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session114-search'


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def report(state, **detail):
    result = dict(state=state, utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  automatic_adoption=False, **detail)
    (D/'result.json').write_text(json.dumps(result, indent=2)+'\n')
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=result['utc'], stage='session114-search',
                                    event=state, detail=detail))+'\n')


def main():
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    queue = json.loads((D/'queue.json').read_text())
    wait = load_module('s101_wait', R/'session101-speed/boundary.py').wait_pid
    report('WAITING_VECTOR', pid=queue['after']['pid'])
    wait(queue['after']['pid'])
    predecessor = json.loads(Path(queue['after']['result']).read_text())
    if predecessor['verdict'] != 'SHELVED' or predecessor['faults']:
        report('REVIEW_VECTOR_GATE', verdict=predecessor['verdict'],
               next='Adopt only a clean KEEP, then retarget and recheck all unstarted candidates')
        return
    runner = load_module('campaign_queue', R/'session93/campaign_queue.py')
    for job in queue['jobs']:
        report('RUNNING', candidate=job['name'], output=job['output'])
        result = runner.run(job, Path(queue['controls']), Path(queue['status']))
        if result['verdict'] != 'SHELVED' or result['faults']:
            report('REVIEW_SEARCH_GATE', candidate=job['name'], verdict=result['verdict'],
                   next='Review KEEP/PROVISIONAL before adoption and retargeting; faults remain unscored')
            return
    # S5 already passed all required checks on this exact package. Its timing
    # needs the exclusive rig, so wait for the only admitted data owner too.
    report('WAITING_DATA_FOR_S5', pid=queue['data_pid'])
    wait(queue['data_pid'])
    report('S5_TIMING', rule='Unchanged fixed40 paired20s, both strata lower95>1 at all three topologies')
    subprocess.run(['bash', str(R/'session113-speed/timing.sh')], check=True)
    speed = json.loads((R/'session113-speed/s5-timing/result.json').read_text())
    report('REVIEW_S5', result=speed, next='KEEP requires manual integration; shelf closes alpha-beta and moves speed lane to SEE')


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        report('HELD_REVIEW', error=str(error))
        raise

"""Run the next controlled target blend after Session109's exclusive boundary.

The original training/export/parity controller retains its storage admission.
This successor owns only the GPU cell, never the tune, gates, or speed timing.
"""
import datetime
import importlib.util
import json
import os
from pathlib import Path

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session111'
CELL = D/'recipe/wdl025'


def load(path):
    return json.loads(path.read_text())


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False)+'\n')


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session111-gpu', event=event, detail=detail))+'\n')


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def main():
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    nomination = load(D/'nomination.json')
    previous = module('s101_wait', R/'session101-speed/boundary.py')
    status('waiting-boundary', dict(pid=nomination['boundary_pid'],
        completion='Existing net gate and conditional transfer or S4 timing; native wait'))
    previous.wait_pid(nomination['boundary_pid'])
    boundary = load(R/'session109/result.json')
    if boundary['state'] not in ('NET_S4_COMPLETE_TUNE_RESUMED', 'REVIEW_NET_WINNER_TUNE_RESUMED'):
        write(D/'result.json', dict(state='HELD_BOUNDARY_REVIEW', predecessor=boundary))
        status('held-boundary-review', dict(state=boundary['state']))
        return
    trial = module('owned_recipe', R/'session87/train-mixture.py')
    trial.D = D/'recipe'
    manifest = load(CELL/'manifest.json')
    trial.SOURCE = Path(manifest['trainer_source'])
    available = trial.free_bytes()
    required = manifest['storage']['required_free_bytes']
    if available < required:
        result = dict(state='HELD_STORAGE', host_c_free_bytes=available, required_free_bytes=required,
            next='Free host C: space to the existing37GiB admission; rerun this unstarted cell only',
            automatic_games=False, automatic_adoption=False)
        write(D/'result.json', result)
        status('held-storage', result)
        return
    status('training-dispatch', dict(cell='wdl025', host_c_free_bytes=available))
    trial.trial('wdl025')
    result = dict(state='READY_FOR_GATE', cell='wdl025', finalization=load(CELL/'result.json'),
        automatic_games=False, automatic_adoption=False,
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(D/'result.json', result)
    status('ready-for-gate', result)


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write(D/'result.json', dict(state='HELD_REVIEW', error=str(error), automatic_adoption=False))
        status('held-review', str(error))
        raise

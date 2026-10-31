"""Dispatch the predeclared final115 net gate after native predecessor waits."""
import json
import os
from pathlib import Path
import select
import shutil
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session82'


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def main():
    # Only dependencies that reserve the game lane or produce net readiness.
    for pid in (32061, 28820):
        wait_pid(pid)
    ready = json.loads((R / 'session80-training/result.json').read_text())
    assert ready['epochs'] == 115 and ready['selection'] == 'final115'
    assert ready['reexport_identical'] and ready['parity_cases'] == 20
    net = D / 'net-final115' / 'candidate.nnue'
    net.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(R / 'session80-training/trained/latest.nnue', net)
    job = json.loads((D / 'net-gate-template.json').read_text())
    job['candidate_options'][0] = 'EvalFile=' + str(net)
    job['net_sha256'] = ready['net_sha256']
    job['artifacts'][str(net)] = ready['net_sha256']
    job['training_ready'] = ready
    queue = dict(controls=str(R / 'session80/controls.json'),
                 status=str(R / 'status.jsonl'), jobs=[job])
    path = D / 'net-queue.json'
    path.write_text(json.dumps(queue, indent=2) + '\n')
    subprocess.run(['python3', str(D / 'campaign_queue.py'), str(path)], check=True)


if __name__ == '__main__':
    main()

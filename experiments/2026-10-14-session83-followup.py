"""Native net-terminal wait, then approved transfer/certification nominations."""
import json
import os
from pathlib import Path
import select
import sys

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session83'
sys.path.insert(0, str(D / 'source/scripts'))
from campaign_queue import run, write


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def retarget(job, path, sha, both):
    old = job['base_options'][0].split('=', 1)[1]
    for role in ('candidate_options', 'base_options') if both else ('candidate_options',):
        job[role][0] = 'EvalFile=' + str(path)
    if both:
        job['artifacts'].pop(old)
    job['artifacts'][str(path)] = sha
    job['net_sha256'] = sha


def dispatch(job, name):
    queue = dict(controls=str(R / 'session80/controls.json'), status=str(R / 'status.jsonl'), jobs=[job])
    write(D / (name + '-queue.json'), queue)
    return run(job, Path(queue['controls']), Path(queue['status']))


def main():
    wait_pid(32307)
    net = json.loads((R / 'session82/net-final115/result.json').read_text())
    if net['verdict'] not in ('KEEP', 'SHELVED') or net['faults']:
        raise RuntimeError('net gate operational failure; no successor games')
    provenance = json.loads((D / 'provenance-template.json').read_text())
    if net['verdict'] == 'KEEP':
        candidate = R / 'session82/net-final115/candidate.nnue'
        sha = json.loads((R / 'session80-training/result.json').read_text())['net_sha256']
        transfer = json.loads((D / 'net-transfer-template.json').read_text())
        retarget(transfer, candidate, sha, False)
        if dispatch(transfer, 'transfer')['verdict'] != 'TRANSFER_PASS':
            raise RuntimeError('net transfer did not pass; hold package review')
        retarget(provenance, candidate, sha, True)
    result = dispatch(provenance, 'provenance')
    if result['verdict'] == 'UNSCORED':
        raise RuntimeError('provenance operational failure; no retry')
    # Certification terminal is reviewed by the next session before search.
    # This controller changes neither engine code nor the accepted log pointer.


if __name__ == '__main__':
    main()

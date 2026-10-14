"""Warm restart final14 on accepted final115 after natural certification."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import select
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session84/spsa-warm'


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session84-spsa-warm', event=event, detail=detail)) + '\n')


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    nomination = json.loads((D / 'nomination.json').read_text())
    status('waiting', {'predecessor_pid': nomination['predecessor_pid'],
                       'requires': 'TRANSFER_PASS and CERTIFIED with zero faults'})
    try:
        fd = os.pidfd_open(nomination['predecessor_pid'])
    except ProcessLookupError:
        pass
    else:
        select.select([fd], [], [])
        os.close(fd)
    transfer = json.loads((R / 'session83/net-transfer/result.json').read_text())
    certification = json.loads((R / 'session83/provenance/result.json').read_text())
    if (transfer['verdict'] != 'TRANSFER_PASS' or transfer['faults'] or
            certification['verdict'] != 'CERTIFIED' or certification['faults']):
        write(D / 'result.json', dict(state='HELD_PREDECESSOR', transfer=transfer['verdict'],
                                     certification=certification['verdict'], games_started=False))
        status('held', 'predecessor needs manual review; no tuning games')
        return
    for path, pin in nomination['artifacts'].items():
        assert digest(path) == pin, path
    env = os.environ.copy()
    env['GOMAXPROCS'] = '1'
    manifest = dict(nomination, environment={'GOMAXPROCS': '1'},
                    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(D / 'manifest.json', manifest)
    status('start', {'manifest': str(D / 'manifest.json'), 'games': 24000})
    with (D / 'tune.log').open('w') as log:
        child = subprocess.Popen(nomination['argv'], env=env, stdout=log, stderr=subprocess.STDOUT)
        (D / 'tuner.pid').write_text(str(child.pid) + '\n')
        rc = child.wait()
    assert rc == 0, f'SPSA exit{rc}'
    rows = [json.loads(line) for line in (D / 'spsa.jsonl').read_text().splitlines()]
    endpoint = rows[-1]
    assert endpoint['iter'] == 2000
    assert endpoint['recipe']['pairs'] == 6 and not endpoint['recipe']['single']
    assert endpoint['recipe']['iterations'] == 2000
    faults = 'forfeits' in (D / 'tune.log').read_text() and any(
        '(0 forfeits)' not in line for line in (D / 'tune.log').read_text().splitlines() if 'forfeits)' in line)
    assert not faults, 'tuning reported forfeits; no converged-vector gate'
    write(D / 'result.json', dict(state='READY_FOR_VECTOR_GATE', games=24000, endpoint=endpoint,
                                 source=nomination['engine_source'], net_sha256=nomination['net_sha256'],
                                 automatic_adoption=False, automatic_games=False,
                                 verdict='No Elo verdict; nominate fixed1000 against compiled final14 on the same accepted net'))
    status('ready-for-vector-gate', {'result': str(D / 'result.json'), 'games': 24000})


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        status('failed', str(error))
        raise

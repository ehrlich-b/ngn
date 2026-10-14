"""Fix the known fixture-metadata omission after the unchanged live controller.

The live controller has already loaded its old Python body. Preserve its
metadata-mismatch failure and run only the previously unexecuted parity cases;
never repeat training/export or repair an actual evaluation mismatch.
"""
import hashlib
import datetime
import json
import math
import os
from pathlib import Path
import select
import subprocess

from trainer.refeval import Network

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session80-training'


def metadata(path, fixture):
    network = Network(path)
    return dict(fixture, format=f'NGNN{network.version}', hidden=network.hidden,
                buckets=network.buckets, king_buckets=network.king_buckets,
                network_sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    pid = int(subprocess.check_output(['systemctl', '--user', 'show', 'ngn-s80-training.service',
                                      '-p', 'MainPID', '--value'], text=True).strip())
    if pid:
        try:
            fd = os.pidfd_open(pid)
        except ProcessLookupError:
            pass
        else:
            select.select([fd], [], [])
            os.close(fd)
    if (D / 'result.json').exists():
        return
    out = D / 'trained'
    history = json.loads((out / 'history.json').read_text())
    assert [row['epoch'] for row in history] == list(range(1, 116))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    exposure = history[-1]['source_presentations_cumulative']
    assert sum(exposure.values()) == 115 * 256000000
    assert 'NGNN3 fixture metadata mismatch' in (out / 'parity.log').read_text()
    # Export comparison already passed before the failed metadata guard. Keep
    # both original byte-identical files and the original failure receipt.
    fixture = json.loads((out / 'parity.json').read_text())
    assert len(fixture['cases']) == 20
    (out / 'parity-missing-metadata.json').write_text(json.dumps(fixture, indent=2) + '\n')
    (out / 'parity-metadata-failure.log').write_text((out / 'parity.log').read_text())
    fixed = out / 'parity-complete.json'
    fixed.write_text(json.dumps(metadata(out / 'latest.nnue', fixture), indent=2) + '\n')
    env = os.environ.copy()
    env.update(NGN_NGNN3_PARITY_NET=str(out / 'latest.nnue'), NGN_NGNN3_PARITY_EVALS=str(fixed),
               GOAMD64='v3', GOTOOLCHAIN='local', GOPROXY='off', GOMAXPROCS='2',
               GOCACHE=str(R / 'session17f/go-cache'))
    with (out / 'parity-complete.log').open('w') as log:
        subprocess.run(['taskset', '-c', '13,15', '/usr/local/go/bin/go', 'test', '-p', '1', '-count=1',
                        './engine', '-run', '^TestNGNN3TrainerParity$', '-v'], cwd=R / 'session80/source',
                       env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
    terminal = dict(epochs=115, total_positions=115 * 256000000, source_presentations=exposure,
                    selection='final115', reexport_identical=True, parity_cases=20,
                    net_sha256=hashlib.sha256((out / 'latest.nnue').read_bytes()).hexdigest(),
                    checkpoint_sha256=hashlib.sha256((out / 'latest.pt').read_bytes()).hexdigest(),
                    metadata_repair='missing shape/hash only; original failure preserved')
    (D / 'result.json').write_text(json.dumps(terminal, indent=2) + '\n')
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session80-training', event='ready-for-gate', detail=terminal)) + '\n')


if __name__ == '__main__':
    main()

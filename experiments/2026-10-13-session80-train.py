"""Approved 115-epoch local T80/T76/full-T74 trial; final epoch only."""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import signal
import subprocess
import sys

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session80-training'
SOURCE = R / 'session80/source'


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session80-training', event=event, detail=detail)) + '\n')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    D.mkdir(exist_ok=True)
    old = json.loads((R / 'session57-data-step2/training-manifest.json').read_text())
    paths = [old['data_paths'][0], old['data_paths'][1], str(R / 'session57-data-step2/t74full.bf')]
    corpus = {name: old['corpus_pins'][name] for name in ('t80', 't76', 't74full')}
    for path, pin in zip(paths, corpus.values()):
        assert Path(path).stat().st_size == pin['bytes'], path
    argv = old['argv'].copy()
    argv[argv.index('--data') + 1:argv.index('--data-format')] = paths
    argv[argv.index('--out') + 1] = str(D / 'trained')
    argv[argv.index('--epochs') + 1] = '115'
    assert '--resume' not in argv and '--king-factorizer' not in argv
    assert not (D / 'trained').exists()
    total = sum(pin['bytes'] // 32 for pin in corpus.values())
    manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    approval='plan root note2026-10-13 21:00 item2',
                    accepted_engine_source='f203e2f', base_net_sha256=old['base_net_sha256'],
                    treatment='115 epochs on T80+T76+fullT74; T75 unavailable locally; mixture also changes',
                    omitted='mapped T75 descriptor points at closed forge; no forge access',
                    corpus_pins=corpus, data_paths=paths, records=total,
                    raw_source_shares={name: pin['bytes'] / (total * 32) for name, pin in corpus.items()},
                    presentations=115 * 256000000, raw_passes=115 * 256000000 / total,
                    argv=['taskset', '-c', '12,14', *argv], trainer_source=str(SOURCE),
                    trainer_commit='7ef9821', source_archive_sha256=digest(R / 'session80/source.tar'),
                    recipe='4636de46/KB8 H512 NB8; same seed20261006/AdamW/WDL/cosine/BF16; fresh initialization',
                    selection='latest.pt/latest.nnue after epoch115 only; ignore best artifacts',
                    exposure='exact source_presentations_cumulative in every history row',
                    cpus=[12, 14], nice=19, memory_max_bytes=4 * 1024**3, swap=0,
                    environment={key: os.environ.get(key) for key in ('PYTHONPATH', 'OMP_NUM_THREADS', 'MKL_NUM_THREADS', 'OPENBLAS_NUM_THREADS')},
                    stop='nonfinite loss/export/parity failure or C: below25.25GiB; no performance cutoff',
                    gate='fixed1000/500 at10+0.1 c6 vs immediate accepted a4d90b41, printed/exact lower95>0 and zero faults; winning net transfer400 at60+0.6 before foreign games',
                    automatic_games=False)
    write(D / 'manifest.json', manifest)
    disk = os.statvfs('/mnt/c')
    assert disk.f_bavail * disk.f_frsize >= 25.5 * 1024**3
    status('start', {'manifest': str(D / 'manifest.json'), 'epochs': 115, 'cpus': [12, 14]})
    with (D / 'train.log').open('w') as log:
        child = subprocess.Popen(manifest['argv'], cwd=SOURCE, stdout=log,
                                 stderr=subprocess.STDOUT, start_new_session=True)
        (D / 'trainer.pid').write_text(str(child.pid) + '\n')
        try:
            while True:
                disk = os.statvfs('/mnt/c')
                if disk.f_bavail * disk.f_frsize < 25.25 * 1024**3:
                    raise RuntimeError('C: reserve crossed; preserve partial training')
                try:
                    rc = child.wait(timeout=300)
                    break
                except subprocess.TimeoutExpired:
                    pass
        except BaseException:
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=30)
            raise
    assert rc == 0, f'trainer exited{rc}'
    out = D / 'trained'
    history = json.loads((out / 'history.json').read_text())
    assert [row['epoch'] for row in history] == list(range(1, 116))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    exposure = history[-1]['source_presentations_cumulative']
    assert sum(exposure.values()) == 115 * 256000000
    status('training-complete', {'epochs': 115, 'source_presentations': exposure})
    import numpy as np
    import torch
    from trainer.export import export_state
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network
    checkpoint = torch.load(out / 'latest.pt', map_location='cpu', weights_only=True)
    assert checkpoint['epoch'] == 115
    export_state(checkpoint['model'], out / 'final-reexport.nnue', 3)
    assert (out / 'latest.nnue').read_bytes() == (out / 'final-reexport.nnue').read_bytes()
    network = Network(out / 'latest.nnue')
    records = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
    assert len(records) == 20
    cases = []
    for record in records:
        position = decode_record(record)
        cases.append(dict(fen=to_fen(position), eval_cp=network.evaluate(position), bucket=network.bucket(position)))
    write(out / 'parity.json', dict(format='NGNN3', hidden=network.hidden, buckets=network.buckets,
                                   king_buckets=network.king_buckets,
                                   network_sha256=digest(out / 'latest.nnue'), cases=cases))
    env = os.environ.copy()
    env.update(NGN_NGNN3_PARITY_NET=str(out / 'latest.nnue'), NGN_NGNN3_PARITY_EVALS=str(out / 'parity.json'),
               GOAMD64='v3', GOTOOLCHAIN='local', GOPROXY='off', GOMAXPROCS='2',
               GOCACHE=str(R / 'session17f/go-cache'))
    with (out / 'parity.log').open('w') as log:
        subprocess.run(['taskset', '-c', '13,15', '/usr/local/go/bin/go', 'test', '-p', '1', '-count=1',
                        './engine', '-run', '^TestNGNN3TrainerParity$', '-v'], cwd=SOURCE,
                       env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
    terminal = dict(epochs=115, total_positions=115 * 256000000, source_presentations=exposure,
                    selection='final115', reexport_identical=True, parity_cases=20,
                    net_sha256=digest(out / 'latest.nnue'), checkpoint_sha256=digest(out / 'latest.pt'))
    write(D / 'result.json', terminal)
    status('ready-for-gate', terminal)


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        status('failed', str(error))
        raise

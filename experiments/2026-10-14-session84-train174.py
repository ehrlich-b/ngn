"""Predeclared 174-epoch follow-up after final115 KEEP; final epoch only."""
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
D = R / 'session84-training174'
SOURCE = R / 'session80/source'


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session84-training174', event=event, detail=detail)) + '\n')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    D.mkdir(exist_ok=True)
    if sys.argv[1:] == ['--finalize-only']:
        finalize()
        return
    resume = sys.argv[1:] == ['--resume']
    assert not sys.argv[1:] or resume, 'expected no arguments, --resume or --finalize-only'
    if resume:
        # Continue this run's own complete atomic checkpoint, with its original
        # optimizer/RNG states, epoch seeds and 174-epoch cosine horizon.
        manifest = json.loads((D / 'manifest.json').read_text())
        history = json.loads((D / 'trained/history.json').read_text())
        checkpoint = D / 'trained/latest.pt'
        assert 0 < history[-1]['epoch'] < 174
        assert manifest['argv'][manifest['argv'].index('--epochs') + 1] == '174'
        manifest = dict(manifest, argv=[*manifest['argv'], '--resume', str(checkpoint)])
        write(D / 'resume-manifest.json', dict(
            created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            original_manifest=str(D / 'manifest.json'),
            resume_checkpoint=str(checkpoint), resume_checkpoint_sha256=digest(checkpoint),
            completed_epoch=history[-1]['epoch'], argv=manifest['argv'],
            reason='Continue original final174 after its disk guard stopped it; same recipe/corpus/seed/horizon; final174 only',
            original_failure_preserved=True))
    else:
        old = json.loads((R / 'session80-training/manifest.json').read_text())
        gate = json.loads((R / 'session82/net-final115/result.json').read_text())
        assert gate['verdict'] == 'KEEP' and not gate['faults']
        paths, corpus = old['data_paths'], old['corpus_pins']
        argv = old['argv'].copy()[3:]
        argv[argv.index('--out') + 1] = str(D / 'trained')
        argv[argv.index('--epochs') + 1] = '174'
        assert '--resume' not in argv and '--king-factorizer' not in argv
        assert not (D / 'trained').exists()
        total = sum(pin['bytes'] // 32 for pin in corpus.values())
        manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                        approval='Session scope: keep GPU producing evidence;174 win-path cell proposed in required Fable cross-review; existing net gate unchanged',
                        accepted_engine_source='9ec5d31', base_net_sha256='812152293dad04184599b2f9ab82779168e6e134b221de136d8f8daae86d9f22',
                        treatment='174 vs115 training horizon: fresh initialization, same T80/T76/fullT74 corpus and recipe as accepted final115; only training duration and its cosine horizon change',
                        omitted='mapped T75 descriptor points at closed forge; no forge access',
                        corpus_pins=corpus, data_paths=paths, records=total,
                        raw_source_shares={name: pin['bytes'] / (total * 32) for name, pin in corpus.items()},
                        presentations=174 * 256000000, raw_passes=174 * 256000000 / total,
                        argv=['taskset', '-c', '12,14', *argv], trainer_source=str(SOURCE),
                        trainer_commit='7ef9821', source_archive_sha256=old['source_archive_sha256'],
                        recipe='final115/KB8 H512 NB8; same seed20261006/AdamW/WDL/cosine/BF16; fresh initialization; no174 resume from115',
                        selection='latest.pt/latest.nnue after epoch174 only; ignore best artifacts; no checkpoint shopping',
                        exposure='exact source_presentations_cumulative in every history row',
                        cpus=[12, 14], nice=19, memory_max_bytes=4 * 1024**3, swap=0,
                        environment={key: os.environ.get(key) for key in ('PYTHONPATH', 'OMP_NUM_THREADS', 'MKL_NUM_THREADS', 'OPENBLAS_NUM_THREADS')},
                        stop='nonfinite loss/export/parity failure or C: below25.25GiB; no performance cutoff',
                        gate='fixed1000/500 at10+0.1 c6 vs immediate accepted81215229, printed/exact lower95>0 and zero faults; winning net transfer400 at60+0.6 before foreign games',
                        automatic_games=False)
        write(D / 'manifest.json', manifest)
    disk = os.statvfs('/mnt/c')
    assert disk.f_bavail * disk.f_frsize >= 25.5 * 1024**3
    status('resume' if resume else 'start', {'manifest': str(D / ('resume-manifest.json' if resume else 'manifest.json')), 'epochs': 174, 'cpus': [12, 14]})
    with (D / ('resume-train.log' if resume else 'train.log')).open('w') as log:
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
    finalize()


def finalize():
    # systemd does not inherit the shell's PYTHONPATH. Use the immutable
    # training source directly, including when finalizing an already complete run.
    sys.path.insert(0, str(SOURCE))
    manifest = json.loads((D / 'manifest.json').read_text())
    epochs = int(manifest['argv'][manifest['argv'].index('--epochs') + 1])
    assert epochs == 174
    out = D / 'trained'
    history = json.loads((out / 'history.json').read_text())
    assert [row['epoch'] for row in history] == list(range(1, epochs + 1))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    exposure = history[-1]['source_presentations_cumulative']
    assert sum(exposure.values()) == 174 * 256000000
    status('training-complete', {'epochs': 174, 'source_presentations': exposure})
    import numpy as np
    import torch
    from trainer.export import export_state
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network
    checkpoint = torch.load(out / 'latest.pt', map_location='cpu', weights_only=True)
    assert checkpoint['epoch'] == 174
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
    terminal = dict(epochs=174, total_positions=174 * 256000000, source_presentations=exposure,
                    selection='final174', reexport_identical=True, parity_cases=20,
                    net_sha256=digest(out / 'latest.nnue'), checkpoint_sha256=digest(out / 'latest.pt'))
    write(D / 'result.json', terminal)
    status('ready-for-gate', terminal)


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        status('failed', str(error))
        raise

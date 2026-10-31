"""Approved matched mixture pair, conditional on a clean final115 shelf."""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import select
import signal
import subprocess
import sys

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session83'
SOURCE = D / 'source'
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session83-mixture-pair', event=event, detail=detail)) + '\n')


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def free_bytes():
    stat = os.statvfs('/mnt/c')
    return stat.f_bavail * stat.f_frsize


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def trial(name, t74, pin):
    directory = D / name
    directory.mkdir()
    old = json.loads((R / 'session80-training/manifest.json').read_text())
    t80, t76 = old['data_paths'][:2]
    weights = {t80: .4535, t76: .3321, str(t74): .2144}
    weights_path = directory / 'source-weights.json'
    write(weights_path, weights)
    argv = old['argv'].copy()
    argv[argv.index('--data') + 1:argv.index('--data-format')] = [t80, t76, str(t74)]
    argv[argv.index('--epochs') + 1] = '58'
    argv[argv.index('--out') + 1] = str(directory / 'trained')
    argv += ['--archive-source-weights', str(weights_path)]
    manifest = dict(approval='October13 21:00 item8 after final115 null; Astra matched pair',
                    trainer_commit='c411391', trainer_source=str(SOURCE),
                    argv=argv, source_weights=weights, total_positions=14848000000,
                    corpus_pins={'t80': old['corpus_pins']['t80'], 't76': old['corpus_pins']['t76'], 't74': pin},
                    selection='final58 only; latest.pt/latest.nnue; ignore best artifacts',
                    recipe='KB8/H512/NB8; seed20261006; AdamW/lr.001/WD.01/WDL.75/K400/cosine/BF16; fresh initialization',
                    sampler='opt-in seeded multinomial batch quotas and independent per-source block cycles; both cells use identical sampler',
                    matched='only T74 corpus changes; T80/T76 rows and batch quotas match exactly; both corpora exceed256M rows/epoch',
                    cpu_affinity='12,14; parity13,15; nice19/4GiB/swap0',
                    gate='No automatic games; compare paired scientific contrast, then any net adoption by fixed1000 versus immediate accepted base/lower95>0 and zero faults',
                    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(directory / 'manifest.json', manifest)
    assert free_bytes() >= 25.5 * 1024**3
    status('start', {'trial': name, 'manifest': str(directory / 'manifest.json')})
    with (directory / 'train.log').open('w') as log:
        child = subprocess.Popen(argv, cwd=SOURCE, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
        (directory / 'trainer.pid').write_text(str(child.pid) + '\n')
        try:
            while True:
                if free_bytes() < 25.25 * 1024**3:
                    raise RuntimeError('C: reserve crossed; preserve partial trial')
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
    out = directory / 'trained'
    history = json.loads((out / 'history.json').read_text())
    assert [row['epoch'] for row in history] == list(range(1, 59))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    exposure = history[-1]['source_presentations_cumulative']
    assert sum(exposure.values()) == 14848000000
    import numpy as np
    import torch
    from trainer.export import export_state
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network
    checkpoint = torch.load(out / 'latest.pt', map_location='cpu', weights_only=True)
    assert checkpoint['epoch'] == 58
    export_state(checkpoint['model'], out / 'final-reexport.nnue', 3)
    assert (out / 'latest.nnue').read_bytes() == (out / 'final-reexport.nnue').read_bytes()
    network = Network(out / 'latest.nnue')
    records = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
    assert len(records) == 20
    cases = []
    for record in records:
        position = decode_record(record)
        cases.append(dict(fen=to_fen(position), eval_cp=network.evaluate(position), bucket=network.bucket(position)))
    fixture = dict(format='NGNN3', hidden=network.hidden, buckets=network.buckets,
                   king_buckets=network.king_buckets, network_sha256=digest(out / 'latest.nnue'), cases=cases)
    write(out / 'parity.json', fixture)
    env = os.environ.copy()
    env.update(NGN_NGNN3_PARITY_NET=str(out / 'latest.nnue'), NGN_NGNN3_PARITY_EVALS=str(out / 'parity.json'),
               GOAMD64='v3', GOTOOLCHAIN='local', GOPROXY='off', GOMAXPROCS='2',
               GOCACHE=str(R / 'session17f/go-cache'))
    with (out / 'parity.log').open('w') as log:
        subprocess.run(['taskset', '-c', '13,15', '/usr/local/go/bin/go', 'test', '-p', '1', '-count=1',
                        './engine', '-run', '^TestNGNN3TrainerParity$', '-v'], cwd=SOURCE,
                       env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
    terminal = dict(epochs=58, total_positions=14848000000, source_presentations=exposure,
                    selection='final58', reexport_identical=True, parity_cases=20,
                    net_sha256=digest(out / 'latest.nnue'), checkpoint_sha256=digest(out / 'latest.pt'))
    write(directory / 'result.json', terminal)
    status('ready-for-gate', dict(trial=name, **terminal))
    return terminal


def main():
    wait_pid(32307)
    net = json.loads((R / 'session82/net-final115/result.json').read_text())
    if net['verdict'] == 'KEEP':
        write(D / 'pair-result.json', {'state': 'SKIPPED_NET_KEEP', 'no_training': True})
        status('skipped', 'final115 won; pair is conditional on null')
        return
    assert net['verdict'] == 'SHELVED' and not net['faults'], 'net operational failure; no training'
    wait_pid(39648)
    sample = json.loads((D / 'sample/result.json').read_text())
    assert sample['matches_original'] and (D / 'sample/controller.exit').read_text().strip() == '0'
    assert (D / 'fullchecks.exit').read_text().strip() == '0'
    old = json.loads((R / 'session80-training/manifest.json').read_text())
    control = trial('mixture-control', D / 'sample/t74.bf', sample)
    treatment = trial('mixture-full', R / 'session57-data-step2/t74full.bf', old['corpus_pins']['t74full'])
    assert control['source_presentations'][old['data_paths'][0]] == treatment['source_presentations'][old['data_paths'][0]]
    assert control['source_presentations'][old['data_paths'][1]] == treatment['source_presentations'][old['data_paths'][1]]
    write(D / 'pair-result.json', dict(state='READY_FOR_GATES', control=control, treatment=treatment,
                                     automatic_games=False, automatic_adoption=False))
    status('pair-ready', str(D / 'pair-result.json'))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        status('failed', str(error))
        raise

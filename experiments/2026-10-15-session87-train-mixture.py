"""Two controlled source-weight cells; final epochs only, no automatic games."""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session87/mixture'
SOURCE = R / 'session87/training-source'
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'
GIB = 1024 ** 3


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     stage='session87-mixture', event=event, detail=detail)) + '\n')


def free_bytes():
    disk = os.statvfs('/mnt/c')
    return disk.f_bavail * disk.f_frsize


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def prepare():
    old = json.loads((R / 'session84-training174/manifest.json').read_text())
    paths = old['data_paths']
    shares = old['raw_source_shares']
    cells = [('volume-control', [shares['t80'], shares['t76'], shares['t74full']]),
             ('moderated-t74', [.35, .25, .40])]
    for name, weights in cells:
        directory = D / name
        directory.mkdir(parents=True)
        weight_path = directory / 'source-weights.json'
        write(weight_path, dict(zip(paths, weights)))
        argv = old['argv'].copy()
        argv[argv.index('--epochs') + 1] = '115'
        argv[argv.index('--out') + 1] = str(directory / 'trained')
        # This panel has never selected a campaign recipe. Both cells exclude
        # exactly the same occupied boards and use the same fresh validation.
        argv[argv.index('--archive-split-seed') + 1] = '20261015'
        argv += ['--archive-source-weights', str(weight_path)]
        manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                        authority='October15 13:05 merged plan: controlled source weights and fresh validation; at most two ungated nets',
                        name=name, argv=argv, trainer_source=str(SOURCE), trainer_commit='c411391',
                        corpus_pins=old['corpus_pins'], source_weights=dict(zip(paths, weights)),
                        common_recipe='KB8/H512/NB8,115x256M,seed20261006,AdamW/lr.001/WD.01/WDL.75/K400/cosine/BF16; fresh initialization',
                        scientific_contrast='Only source weights differ between these two cells; control uses explicit raw-volume weights; treatment reduces fullT74 from58.22% to40%',
                        accepted_base_caveat='Versus Final115 this family also changes to the tested weighted sampler and a fresh board-hash split; do not attribute that contrast solely to weights',
                        validation=dict(seed=20261015, modulus=1024, selection=False,
                                        contract='Fresh occupied-board holdout, excluded from both cells; separate from old seed20261006 selection panel; metrics descriptive only'),
                        selection='latest epoch115 only; ignore best checkpoint; no checkpoint shopping',
                        total_positions=115 * 256000000,
                        gate='Fixed1000/500 at10+0.1/c6 vs immediate accepted package, printed and exact lower95>0,zero faults; winning net transfer400 at60+0.6,point>0 before foreign games',
                        storage=dict(floor_bytes=25*GIB, projected_peak_growth_bytes=512*1024**2,
                                     margin_bytes=2*GIB, required_free_bytes=int(27.5*GIB),
                                     explanation='Existing immutable corpora read in place; two latest/best checkpoints plus atomic writes, nets, logs and parity; no reconstruction'),
                        cpus=[12,14], nice=19, memory_max='4GiB', swap=0,
                        outstanding_rule='Final174 plus volume-control allowed; moderated cell waits for a clean completed Final174 gate before starting',
                        automatic_games=False, automatic_adoption=False)
        write(directory / 'manifest.json', manifest)


def trial(name):
    directory = D / name
    manifest = json.loads((directory / 'manifest.json').read_text())
    assert free_bytes() >= manifest['storage']['required_free_bytes'], 'Peak-space admission failed'
    status('start', dict(cell=name, manifest=str(directory / 'manifest.json'), host_c_free_bytes=free_bytes()))
    with (directory / 'train.log').open('w') as log:
        child = subprocess.Popen(manifest['argv'], cwd=SOURCE, stdout=log,
                                 stderr=subprocess.STDOUT, start_new_session=True)
        (directory / 'trainer.pid').write_text(str(child.pid) + '\n')
        try:
            while True:
                if free_bytes() < 25.25 * GIB:
                    raise RuntimeError('C: reserve crossed; preserve partial cell')
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
    assert [row['epoch'] for row in history] == list(range(1,116))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    exposure = history[-1]['source_presentations_cumulative']
    assert sum(exposure.values()) == manifest['total_positions']
    sys.path.insert(0, str(SOURCE))
    import numpy as np
    import torch
    from trainer.export import export_state
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network
    checkpoint = torch.load(out / 'latest.pt', map_location='cpu', weights_only=True)
    assert checkpoint['epoch'] == 115
    export_state(checkpoint['model'], out / 'final-reexport.nnue',3)
    assert (out / 'latest.nnue').read_bytes() == (out / 'final-reexport.nnue').read_bytes()
    network = Network(out / 'latest.nnue')
    records = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp',dtype=RECORD_DTYPE)
    assert len(records) == 20
    cases = []
    for record in records:
        position = decode_record(record)
        cases.append(dict(fen=to_fen(position),eval_cp=network.evaluate(position),bucket=network.bucket(position)))
    write(out / 'parity.json',dict(format='NGNN3',hidden=network.hidden,buckets=network.buckets,
                                  king_buckets=network.king_buckets,network_sha256=digest(out/'latest.nnue'),cases=cases))
    env = dict(os.environ,NGN_NGNN3_PARITY_NET=str(out/'latest.nnue'),NGN_NGNN3_PARITY_EVALS=str(out/'parity.json'),
               GOAMD64='v3',GOTOOLCHAIN='local',GOPROXY='off',GOMAXPROCS='2',GOCACHE=str(R/'session17f/go-cache'))
    with (out/'parity.log').open('w') as log:
        subprocess.run(['taskset','-c','13,15','/usr/local/go/bin/go','test','-p','1','-count=1',
                        './engine','-run','^TestNGNN3TrainerParity$','-v'],cwd=SOURCE,env=env,
                       stdout=log,stderr=subprocess.STDOUT,check=True)
    result = dict(epochs=115,total_positions=manifest['total_positions'],source_presentations=exposure,
                  selection='final115',reexport_identical=True,parity_cases=20,
                  net_sha256=digest(out/'latest.nnue'),checkpoint_sha256=digest(out/'latest.pt'))
    write(directory/'result.json',result)
    status('ready-for-gate',dict(cell=name,**result))


def main():
    prepare()
    trial('volume-control')
    gate = R/'session87/net-final174/result.json'
    status('waiting',dict(for_result=str(gate),reason='at most two ungated nets',interval_seconds=300))
    while not gate.exists():
        time.sleep(300)
    result = json.loads(gate.read_text())
    assert result['verdict'] in ('KEEP','SHELVED') and not result['faults'], 'Final174 gate needs fault review'
    trial('moderated-t74')


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        status('failed',str(error))
        raise

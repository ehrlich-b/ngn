"""Manual continuation after the proved absolute holdout-index overflow."""
import datetime
import json
import math
import os
from pathlib import Path
import signal
import subprocess
from urllib.request import urlopen

r = Path('/home/ehrli/ngn-data/night-20261007')
s = r / 'session57-data-step2'


def status(event, detail):
    with (r / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session57-data-step2', event=event, detail=detail)) + '\n')


def main():
    prior = json.loads((s / 'interrupted-session62-loader/training-manifest.json').read_text())
    payload = dict(coverage=prior['coverage'], descriptors={
        name: json.loads((s / (name + '.archive.json')).read_text())
        for name in ('t74full', 't75')})
    assert payload['coverage']['training_eligible']
    assert 1.9 <= payload['coverage']['estimated_unique_ratio'] <= 2.3
    baseline = json.loads((r / 'session51-expanded-factorizer/baseline-training-manifest.json').read_text())
    argv = baseline['argv'].copy()
    begin, end = argv.index('--data') + 1, argv.index('--data-format')
    data = [str(Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf')),
            str(r / 'session19-data/t76.bf'), str(s / 't74full.archive.json'), str(s / 't75.archive.json')]
    argv[begin:end] = data
    argv[argv.index('--out') + 1] = str(s / 'trained')
    assert '--king-factorizer' not in argv
    corpus = dict(t80=baseline['corpus_pins']['t80'], t76=baseline['corpus_pins']['t76'])
    corpus.update({name: {key: descriptor[key] for key in ['bytes', 'sha256']}
                   for name, descriptor in payload['descriptors'].items()})
    manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    record='experiments/2026-10-11-data-step2.md', change='data only',
                    accepted_engine_source='ed34480',
                    base_net_sha256='a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86',
                    trainer_source=str(r / 'session62-loader-index/source'),
                    trainer_io_commit='09d53c7', source_archive_sha256=baseline['source_archive_sha256'],
                    trainer_io_fix='start + selected.astype(np.intp); absolute holdout indexes only',
                    trainer_io_receipt=str(r / 'session62-loader-index/repair-receipt.json'),
                    data_paths=data, corpus_pins=corpus, coverage=payload['coverage'], argv=argv,
                    cpus=[12, 14], nice=19, memory_max_bytes=4 * 1024**3,
                    environment={key: os.environ.get(key) for key in ['PYTHONPATH', 'OMP_NUM_THREADS', 'MKL_NUM_THREADS', 'OPENBLAS_NUM_THREADS']},
                    concurrent_workloads=['rig immediate-admission games0-11/c6/3GiB',
                                          'forge corpus reader8768; old reader117631 untouched', 'hopper1170 untouched'],
                    initialization='fresh seeded random initialization; no warm start',
                    selection='minimum full holdout loss across58 complete epochs',
                    postprocessing_machine='forge', automatic_games=False)
    (s / 'training-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    assert not (s / 'trained').exists()
    disk = os.statvfs('/mnt/c')
    assert disk.f_bavail * disk.f_frsize >= 25.5 * 1024**3
    status('training-recovery-start', 'manually reviewed failed validation overflow; no durable epoch/checkpoint; identical data/seed/58x256M/recipe with widened absolute holdout indexes; rig12,14/4GiB')
    with (s / 'train.log').open('w') as log:
        child = subprocess.Popen(argv, cwd=manifest['trainer_source'], stdout=log,
                                 stderr=subprocess.STDOUT, start_new_session=True)
        (s / 'trainer.pid').write_text(str(child.pid) + '\n')
        try:
            while True:
                disk = os.statvfs('/mnt/c')
                if disk.f_bavail * disk.f_frsize < 25.25 * 1024**3:
                    raise RuntimeError('C: checkpoint reserve crossed25.25GiB; preserve interruption')
                try:
                    rc = child.wait(timeout=60)
                    break
                except subprocess.TimeoutExpired:
                    pass
        except BaseException:
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=20)
            raise
    assert rc == 0, f'trainer exit{rc}; no automatic restart'
    history = json.loads((s / 'trained/history.json').read_text())
    assert [row['epoch'] for row in history] == list(range(1, 59))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    config = json.loads((s / 'trained/config.json').read_text())
    expected = json.loads((r / 'session51-expanded-factorizer/baseline-config.json').read_text())
    expected.update(data=data, out=str(s / 'trained'))
    assert {key: val for key, val in config.items() if key != 'archive_split'} == {key: val for key, val in expected.items() if key != 'archive_split'}
    terminal = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), full58=True,
                    epochs=58, total_positions=sum(row['positions'] for row in history),
                    king_factorizer=False, postprocessing_machine='forge', automatic_games=False)
    (s / 'training-terminal.json').write_text(json.dumps(terminal, indent=2) + '\n')
    status('training-done', 'full58/config matched; manual forge reexport/20parity and actual-base nomination next')


if __name__ == '__main__':
    main()

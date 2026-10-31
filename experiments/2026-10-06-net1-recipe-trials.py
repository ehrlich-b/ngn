"""Run sequential owned GPU recipe trials against an existing frozen snapshot."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

import numpy as np
import torch

from trainer.export import export_state
from trainer.ngnp import RECORD_DTYPE, Shards, decode_record, to_fen
from trainer.refeval import Network


def write_json(path, payload):
    temporary = path.with_suffix('.json.tmp')
    temporary.write_text(json.dumps(payload, indent=2, allow_nan=False) + '\n')
    temporary.replace(path)


def digest_prefix(path, byte_count):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        while byte_count:
            block = stream.read(min(byte_count, 1024 * 1024))
            if not block:
                raise RuntimeError(f'{path}: snapshot prefix was truncated')
            digest.update(block)
            byte_count -= len(block)
    return digest.hexdigest()


def parity(out, selected):
    checkpoint = torch.load(out / 'best.pt', map_location='cpu', weights_only=True)
    export_state(checkpoint['model'], out / 'best-reexport.nnue', checkpoint['config']['format'])
    assert (out / 'best.nnue').read_bytes() == (out / 'best-reexport.nnue').read_bytes()
    (out / 'best-reexport.nnue').unlink()
    network = Network(out / 'best.nnue')
    cases = []
    for record in selected:
        position = decode_record(record)
        cases.append(dict(fen=to_fen(position), eval_cp=network.evaluate(position), bucket=network.bucket(position)))
    net_hash = hashlib.sha256((out / 'best.nnue').read_bytes()).hexdigest()
    payload = dict(format=f'NGNN{network.version}', hidden=network.hidden, buckets=network.buckets,
                   king_buckets=network.king_buckets, network_sha256=net_hash, cases=cases)
    write_json(out / 'parity.json', payload)
    env = os.environ.copy()
    env.update({f'NGN_NGNN{network.version}_PARITY_NET': str(out / 'best.nnue'),
                f'NGN_NGNN{network.version}_PARITY_EVALS': str(out / 'parity.json')})
    with (out / 'parity.log').open('w') as log:
        subprocess.run(['/usr/local/go/bin/go', 'test', '-p', '1', '-count=1', './engine',
                        '-run', f'^TestNGNN{network.version}TrainerParity$', '-v'], env=env,
                       check=True, stdout=log, stderr=subprocess.STDOUT, timeout=120)
    return dict(network_sha256=net_hash,
                checkpoint_sha256=hashlib.sha256((out / 'best.pt').read_bytes()).hexdigest(),
                parity_cases=len(cases), reexport_identical=True, best_epoch=checkpoint['epoch'])


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument('--root', type=Path, required=True)
    cli.add_argument('--epochs', type=int, default=32)
    cli.add_argument('--budget-seconds', type=float, default=9000)
    args = cli.parse_args()
    if args.epochs < 1 or args.budget_seconds <= 0:
        cli.error('Epochs and wall-time budget must be positive')
    root = args.root.resolve()
    snapshot_path = root / 'snapshot.json'
    snapshot = json.loads(snapshot_path.read_text())
    data = [row['path'] for row in snapshot['shards'] if row['split'] == 'train']
    holdout = snapshot['holdout']
    torch.set_num_threads(2)
    for row in snapshot['shards']:
        row['prefix_sha256'] = digest_prefix(row['path'], row['records'] * 32)
    write_json(root / 'snapshot-hashes.json', snapshot)
    validation = Shards(holdout, require_kings=True)
    # Fix the same 20 positions for all four exports, spread across the whole holdout.
    targets = np.linspace(0, validation.count - 1, 20, dtype=np.int64)
    selected, seen = [], 0
    for raw in validation.batches(16384, 0, shuffle=False):
        records = raw.reshape(-1).view(RECORD_DTYPE)
        for target in targets[(targets >= seen) & (targets < seen + len(raw))]:
            selected.append(records[int(target - seen)].copy())
        seen += len(raw)
    assert len(selected) == 20
    write_json(root / 'parity-positions.json', dict(holdout=holdout, filtered_indices=targets.tolist(),
                                                  fens=[to_fen(decode_record(record)) for record in selected]))
    plan = [('a-ngnn2-h256', 2, 256, 0.75), ('b-ngnn2-h512', 2, 512, 0.75),
            ('c-ngnn3-h256', 3, 256, 0.75)]
    results = []
    trial_wall_seconds = 0.0
    common = ['--data', *data, '--validation-data', *holdout, '--record-counts', str(root / 'record-counts.json'),
              '--buckets', '8', '--batch', '16384', '--lr', '0.001', '--epochs', str(args.epochs),
              '--decay', 'cosine', '--min-lr-ratio', '0.1', '--threads', '2', '--precision', 'bf16',
              '--seed', '20261006', '--keep-best-only']
    for index in range(4):
        if index == 3:
            winner = min(results, key=lambda trial: trial['best']['validation_loss_lambda_075'])
            plan.append(('d-best-lambda050', winner['format'], winner['hidden'], 0.5))
        name, version, hidden, lam = plan[index]
        out = root / name
        out.mkdir(exist_ok=False)
        command = [sys.executable, '-m', 'trainer.train', *common, '--out', str(out),
                   '--format', str(version), '--hidden', str(hidden), '--wdl', str(lam)]
        print(json.dumps(dict(event='trial_started', name=name, format=version, hidden=hidden, wdl=lam)), flush=True)
        started = time.perf_counter()
        remaining = args.budget_seconds - trial_wall_seconds
        if remaining <= 0:
            raise RuntimeError('GPU trial wall-time budget exhausted')
        with (out / 'train.log').open('w') as log:
            subprocess.run(command, check=True, stdout=log, stderr=subprocess.STDOUT, timeout=remaining)
        wall = time.perf_counter() - started
        trial_wall_seconds += wall
        history = json.loads((out / 'history.json').read_text())
        best = min(history, key=lambda row: row['validation_loss'])
        metadata = json.loads((out / 'config.json').read_text())
        with (out / 'train.log').open() as log:
            counts = json.loads(next(log))
        receipt = dict(name=name, format=version, hidden=hidden, wdl=lam, epochs=len(history),
                       wall_seconds=wall, training_seconds=sum(row['seconds'] for row in history),
                       raw_records=counts['raw_records'], filtered_records=counts['filtered_records'],
                       validation_records=counts['validation_records'], best=best, history=history,
                       config=metadata, **parity(out, selected))
        results.append(receipt)
        write_json(root / 'results.json', dict(snapshot_sha256=hashlib.sha256(snapshot_path.read_bytes()).hexdigest(),
                                               trial_wall_seconds=trial_wall_seconds, trials=results))
        print(json.dumps(dict(event='trial_completed', name=name, wall_seconds=wall,
                              best_epoch=best['epoch'], best=best)), flush=True)
        # Preserve only the best final artifacts and curves once parity passes.
        (out / 'latest.pt').unlink()
        (out / 'latest.nnue').unlink()
        size = sum(path.stat().st_size for path in root.rglob('*') if path.is_file())
        if size > 300_000_000:
            raise RuntimeError(f'Trial outputs exceed 300 MB: {size}')
    # Confirm every frozen prefix remained identical, including both closed holdout shards.
    for row in snapshot['shards']:
        assert row['prefix_sha256'] == digest_prefix(row['path'], row['records'] * 32), row['path']
    print(json.dumps(dict(event='completed', trial_wall_seconds=trial_wall_seconds,
                          prefix_hashes_unchanged=True)), flush=True)
    write_json(root / 'completed.json', dict(trial_wall_seconds=trial_wall_seconds,
                                            prefix_hashes_unchanged=True, trials=4))


if __name__ == '__main__':
    main()

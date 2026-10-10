"""Run the matched expanded-data recipe after a session nominates its base."""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import subprocess
import time

import numpy as np
import torch

from trainer.export import export_state
from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
from trainer.refeval import Network

R = Path('/home/ehrli/ngn-data/night-20261007')
S = R / 'session42-expanded-data'
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'
nomination = json.loads((S / 'nomination.json').read_text())
source = S / 'source'
out = S / 'trained'
sha = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()
assert not out.exists()
fmt = nomination['format']
assert fmt in (3, 4)
data = nomination['data_paths']
argv = [PYTHON, '-m', 'trainer.train', '--data', *data,
        '--data-format', 'archive', '--archive-holdout-modulus', '1024',
        '--archive-split-seed', '20261006', '--out', str(out),
        '--format', str(fmt), '--kb-map', 'default', '--hidden', '512',
        '--buckets', '8', '--batch', '16384', '--lr', '.001',
        '--epochs', '58', '--positions-per-epoch', '256000000',
        '--weight-decay', '.01', '--wdl', '.75', '--k', '400',
        '--decay', 'cosine', '--min-lr-ratio', '.1', '--threads', '2',
        '--precision', 'bf16', '--seed', '20261006', '--keep-best-only']
if fmt == 4:
    argv += ['--hidden2', '16']
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                **nomination, argv=argv, source_paths_sorted=sorted(data),
                cpus=[12, 14], nice=19, memory_max_bytes=4 * 1024**3,
                selection='minimum full content-holdout validation_loss over58 complete epochs',
                initialization='fresh seeded random initialization; no warm start',
                change='data only; accepted architecture and original recipe retained',
                automatic_games=False)
(S / 'training-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
started = time.monotonic()
with (S / 'train.log').open('w') as log:
    subprocess.run(argv, cwd=source, stdout=log, stderr=subprocess.STDOUT, check=True)
wall = time.monotonic() - started
history = json.loads((out / 'history.json').read_text())
assert len(history) == 58 and [r['epoch'] for r in history] == list(range(1, 59))
assert all(r['positions'] == 256000000 and math.isfinite(r['validation_loss']) for r in history)
checkpoint = torch.load(out / 'best.pt', map_location='cpu', weights_only=True)
best = min(history, key=lambda r: r['validation_loss'])
assert checkpoint['epoch'] == best['epoch']
config = checkpoint['config']
assert (config['hidden'], config['buckets'], config['format'], config['king_factorizer']) == (512, 8, fmt, False)
if fmt == 4:
    assert config['hidden2'] == 16
export_state(checkpoint['model'], out / 'best-reexport.nnue', fmt)
assert (out / 'best-reexport.nnue').read_bytes() == (out / 'best.nnue').read_bytes()
network = Network(out / 'best.nnue')
records = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
assert len(records) == 20
cases = []
for record in records:
    position = decode_record(record)
    cases.append(dict(fen=to_fen(position), eval_cp=network.evaluate(position), bucket=network.bucket(position)))
fixture = dict(format=f'NGNN{fmt}', hidden=network.hidden, buckets=network.buckets,
               king_buckets=network.king_buckets, network_sha256=sha(out / 'best.nnue'), cases=cases)
if fmt == 4:
    fixture['hidden2'] = network.hidden2
(out / 'parity.json').write_text(json.dumps(fixture, indent=2) + '\n')
env = dict(os.environ, **{f'NGN_NGNN{fmt}_PARITY_NET': str(out / 'best.nnue'),
                         f'NGN_NGNN{fmt}_PARITY_EVALS': str(out / 'parity.json')})
with (out / 'parity.log').open('w') as log:
    subprocess.run(['/usr/local/go/bin/go', 'test', '-p', '2', '-count=1', './engine',
                    '-run', f'^TestNGNN{fmt}TrainerParity$', '-v'], cwd=source,
                   env=env, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=600)
receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               training_wall_seconds=wall, epochs=58,
               total_positions=sum(r['positions'] for r in history), best_epoch=best['epoch'],
               best=best, reexport_identical=True, parity_cases=20,
               net_sha256=sha(out / 'best.nnue'), checkpoint_sha256=sha(out / 'best.pt'),
               archive_split=config['archive_split'], automatic_games=False)
(S / 'training-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps(receipt))

"""Measure one declared architecture's actual pipeline before a full trial."""
import datetime
import json
import math
from pathlib import Path
import subprocess
import time

ROOT = Path('/home/ehrli/ngn-data/night-20261007/session35-ngnn4-pilot')
SOURCE = Path('/home/ehrli/ngn-data/night-20261007/session34-architecture/source')
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'
DATA = '/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf'
argv = [PYTHON, '-m', 'trainer.train', '--data', DATA, '--data-format', 'archive',
        '--archive-holdout-modulus', '1024', '--archive-split-seed', '20261006',
        '--out', str(ROOT / 'scratch'), '--format', '4', '--hidden2', '16',
        '--kb-map', 'default', '--hidden', '512', '--buckets', '8', '--batch', '16384',
        '--lr', '.001', '--epochs', '3', '--positions-per-epoch', '16000000',
        '--wdl', '.75', '--k', '400', '--decay', 'cosine', '--min-lr-ratio', '.1',
        '--threads', '2', '--precision', 'bf16', '--seed', '20261006', '--keep-best-only']
assert not (ROOT / 'scratch').exists()
started = time.monotonic()
epochs = []
with (ROOT / 'train.log').open('w') as log, (ROOT / 'timed.jsonl').open('w') as journal:
    child = subprocess.Popen(argv, cwd=SOURCE, stdout=subprocess.PIPE,
                             stderr=log, text=True)
    config_elapsed = None
    for line in child.stdout:
        elapsed = time.monotonic() - started
        log.write(line)
        log.flush()
        row = json.loads(line)
        journal.write(json.dumps(dict(elapsed=elapsed, row=row)) + '\n')
        journal.flush()
        if 'config' in row:
            config_elapsed = elapsed
            assert row['config']['hidden2'] == 16
            assert not row['config']['king_factorizer']
            assert row['validation_records'] == 2319644
        else:
            assert row['epoch'] == len(epochs) + 1
            assert row['positions'] == 16000000
            assert math.isfinite(row['validation_loss'])
            epochs.append(dict(elapsed=elapsed, **row))
    assert child.wait() == 0
elapsed = time.monotonic() - started
assert len(epochs) == 3 and config_elapsed is not None
rates = [row['steady_positions_per_second'] for row in epochs]
assert all(math.isfinite(rate) and rate > 0 for rate in rates)
overheads = [epochs[i]['elapsed'] - epochs[i-1]['elapsed'] - epochs[i]['seconds']
             for i in [1, 2]]
recurring = max(0, *overheads)
first_overhead = max(0, epochs[0]['elapsed'] - config_elapsed - epochs[0]['seconds'])
projected = config_elapsed + first_overhead + 58 * (256000000 / min(rates) + recurring)
receipt = dict(source_commit='2acf834945d48378209853464b86cefe72587ccc',
               architecture='NGNN4 KB8 H512 L16 NB8', factorizer=False,
               argv=argv, epochs=epochs, total_seconds=elapsed,
               startup_seconds=config_elapsed, first_validation_checkpoint_seconds=first_overhead,
               recurring_validation_checkpoint_seconds=recurring,
               minimum_steady_positions_per_second=min(rates),
               full_recipe_epochs=58, full_recipe_positions_per_epoch=256000000,
               projected_full_hours=projected/3600, budget_hours=10,
               budget_admitted=projected <= 10*3600, pilot_checkpoint_adoption=False,
               completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(ROOT / 'receipt.json').write_text(json.dumps(receipt, indent=2, allow_nan=False)+'\n')
print(json.dumps({k: v for k, v in receipt.items() if k not in ['epochs', 'argv']}))

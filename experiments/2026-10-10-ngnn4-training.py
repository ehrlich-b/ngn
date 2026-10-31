"""Complete one prospectively frozen architecture recipe, parity and cost report."""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess
import time

import numpy as np
import torch

from trainer.export import export_state
from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
from trainer.refeval import Network

R = Path('/home/ehrli/ngn-data/night-20261007')
S = R / 'session38-ngnn4-training'
SOURCE = R / 'session34-architecture/source'
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'
DATA = Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf')
sha = lambda p: hashlib.sha256(Path(p).read_bytes()).hexdigest()
out = S / 'trained'
assert not out.exists()
assert DATA.stat().st_size == 75977281056
argv = [PYTHON, '-m', 'trainer.train', '--data', str(DATA), '--data-format', 'archive',
        '--archive-holdout-modulus', '1024', '--archive-split-seed', '20261006',
        '--out', str(out), '--format', '4', '--hidden2', '16', '--kb-map', 'default',
        '--hidden', '512', '--buckets', '8', '--batch', '16384', '--lr', '.001',
        '--epochs', '58', '--positions-per-epoch', '256000000', '--weight-decay', '.01',
        '--wdl', '.75', '--k', '400', '--decay', 'cosine', '--min-lr-ratio', '.1',
        '--threads', '2', '--precision', 'bf16', '--seed', '20261006', '--keep-best-only']
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                architecture_source='2acf834945d48378209853464b86cefe72587ccc',
                base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',
                later_game_base='pending higher-priority key confirmation; no automatic games',
                base_net='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a',
                corpus_sha256_reused='270f65bdf9f5c948160cae5330030a60f99e94383de2a290b565ae7aad7cde26',
                argv=argv, cpus=[12, 14], nice=19, pilot_weights_used=False,
                selection='minimum full-holdout validation_loss over58 complete epochs')
(S / 'training-manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
started = time.monotonic()
with (S / 'train.log').open('w') as log:
    subprocess.run(argv, cwd=SOURCE, stdout=log, stderr=subprocess.STDOUT, check=True)
wall = time.monotonic() - started
history = json.loads((out / 'history.json').read_text())
assert len(history) == 58
assert [row['epoch'] for row in history] == list(range(1, 59))
assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
checkpoint = torch.load(out / 'best.pt', map_location='cpu', weights_only=True)
best = min(history, key=lambda row: row['validation_loss'])
assert checkpoint['epoch'] == best['epoch']
config = checkpoint['config']
assert (config['hidden'], config['hidden2'], config['buckets'], config['format'], config['king_factorizer']) == (512,16,8,4,False)
export_state(checkpoint['model'], out / 'best-reexport.nnue', 4)
assert (out / 'best-reexport.nnue').read_bytes() == (out / 'best.nnue').read_bytes()
network = Network(out / 'best.nnue')
selected = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
assert len(selected) == 20
cases = []
for record in selected:
    position = decode_record(record)
    cases.append(dict(fen=to_fen(position), eval_cp=network.evaluate(position), bucket=network.bucket(position)))
fixture = dict(format='NGNN4', hidden=network.hidden, hidden2=network.hidden2,
               buckets=network.buckets, king_buckets=network.king_buckets,
               network_sha256=sha(out / 'best.nnue'), cases=cases)
(out / 'parity.json').write_text(json.dumps(fixture, indent=2)+'\n')
env = dict(os.environ, NGN_NGNN4_PARITY_NET=str(out / 'best.nnue'),
           NGN_NGNN4_PARITY_EVALS=str(out / 'parity.json'))
with (out / 'parity.log').open('w') as log:
    subprocess.run(['/usr/local/go/bin/go','test','-p','2','-count=1','./engine',
                    '-run','^TestNGNN4TrainerParity$','-v'], cwd=SOURCE, env=env,
                   stdout=log, stderr=subprocess.STDOUT, check=True, timeout=600)
receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               training_wall_seconds=wall, epochs=58, total_positions=sum(r['positions'] for r in history),
               best_epoch=best['epoch'], best=best, reexport_identical=True, parity_cases=20,
               net_sha256=sha(out / 'best.nnue'), checkpoint_sha256=sha(out / 'best.pt'))
(S / 'training-receipt.json').write_text(json.dumps(receipt, indent=2)+'\n')
rounds = []
env = dict(os.environ, GOMAXPROCS='1')
for i in range(4):
    order = ['base','candidate'] if i % 2 == 0 else ['candidate','base']
    rates = {}
    for kind in order:
        probe = SOURCE.parent / ('baseline-probe' if kind == 'base' else 'candidate-probe')
        net = R / 'kb8h512/best.nnue' if kind == 'base' else out / 'best.nnue'
        command = ['taskset','-c','14','nice','-n','19',str(probe),'-mode','timed','-net',str(net),'-duration','2s']
        proc = subprocess.run(command, env=env, capture_output=True, text=True, check=True, timeout=120)
        assert not proc.stderr
        rows = [json.loads(line) for line in proc.stdout.splitlines()]
        assert len(rows) == 6 and all(r['nodes'] > 0 for r in rows)
        (S / f'cost-{i}-{kind}.jsonl').write_text(proc.stdout)
        rates[kind] = sum(r['nodes'] for r in rows)/sum(r['seconds'] for r in rows)
    rounds.append(dict(index=i, order=order, nps=rates, ratio=rates['candidate']/rates['base']))
cost = dict(rounds=rounds, geometric_nps_ratio=math.exp(statistics.mean(math.log(r['ratio']) for r in rounds)),
            decision='cost report only; fresh400-game gate decides architecture', trained_net=receipt['net_sha256'])
(S / 'actual-cost.json').write_text(json.dumps(cost, indent=2)+'\n')
print(json.dumps(dict(training=receipt, actual_cost=cost)))

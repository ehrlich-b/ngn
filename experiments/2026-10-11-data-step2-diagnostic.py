"""Frozen, bounded source-stratified diagnostic; never train or select a net."""
import datetime
import hashlib
import json
import os
from pathlib import Path

import numpy as np
import torch

from trainer.archive import ARCHIVE_DTYPE, board_hash, canonical_records, piece_counts
from trainer.model import NGNN, decode_gpu
from trainer.ngnp import RECORD_DTYPE
from trainer.refeval import Network
from trainer.remote_archive import RemoteArray

r = Path('/home/ehrli/ngn-data/night-20261007')
p = r / 'session66-data-diagnostic'
t = r / 'session57-data-step2/trained'
config = json.loads((t / 'config.json').read_text())
sources = config['archive_split']['sources']
seed, block_records, blocks_per_source, rows_per_source = 2026101106, 262144, 16, 2048
sha = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session66-data-diagnostic', event=event, detail=detail)) + '\n')


assert sha(t / 'best.nnue') == '4636de46c8afb33227cff72831d3fa80b86b1c84d9394661f70b29f10ddc945f'
base_path = r / 'session65-spsa-transfer/base.nnue'
assert sha(base_path) == 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
rng = np.random.default_rng(seed)
plans = []
for source in sources:
    count = source['records']
    ids = rng.choice((count + block_records - 1) // block_records,
                     blocks_per_source, replace=False).tolist()
    plans.append(dict(path=source['path'], records=count, block_ids=ids))
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                seed=seed, block_records=block_records, blocks_per_source=blocks_per_source,
                rows_per_source=rows_per_source, sources=plans,
                split_seed=20261006, split_modulus=1024, checkpoint_epoch=58,
                checkpoint_sha256=sha(t / 'best.pt'), net_sha256=sha(t / 'best.nnue'),
                baseline_sha256=sha(base_path), diagnostic_sha256=sha(__file__),
                cpus=[12, 14], gpu=True, memory_max_bytes=4 * 1024**3,
                instruction='experiments/2026-10-11-data-step2-diagnostic.md',
                selection='2048 holdout rows/source, frozen-seed uniform without replacement',
                purpose='source/material/king-bucket prevalence and float/integer residuals; no training or games')
(p / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
status('sample-start', '16 frozen blocks/source; original holdout partition')
mapping = np.asarray(config['king_map'])
panel, metadata, exposures = [], [], []
for source_id, plan in enumerate(plans):
    path, count = plan['path'], plan['records']
    data = (RemoteArray(path, ARCHIVE_DTYPE) if path.endswith('.archive.json') else
            np.memmap(path, dtype=ARCHIVE_DTYPE, mode='r', shape=(count,)))
    held, indexes = [], []
    train_hist = np.zeros((8, 8, 8), dtype=np.int64)
    hold_hist = np.zeros((8, 8, 8), dtype=np.int64)
    for block_id in plan['block_ids']:
        start = block_id * block_records
        rows = data[start:min(start + block_records, count)]
        hold = (board_hash(rows, 20261006) & np.uint64(1023)) == 0
        material = np.clip((piece_counts(rows['occupancy']).astype(np.int16) - 2) * 8 // 32, 0, 7)
        kings = [rows[field].astype(np.int64) for field in ['us_king', 'them_king']]
        buckets = [mapping[king ^ np.where((king & 7) >= 4, 7, 0)] for king in kings]
        cell = (material * 8 + buckets[0]) * 8 + buckets[1]
        train_hist += np.bincount(cell[~hold], minlength=512).reshape(8, 8, 8)
        hold_hist += np.bincount(cell[hold], minlength=512).reshape(8, 8, 8)
        held.append(rows[hold].copy())
        indexes.append(start + np.flatnonzero(hold))
    held, indexes = np.concatenate(held), np.concatenate(indexes)
    assert len(held) >= rows_per_source
    selected = rng.choice(len(held), rows_per_source, replace=False)
    selected_rows = held[selected]
    canonical = canonical_records(selected_rows)
    panel.append(canonical)
    metadata.extend(dict(source=source_id, absolute_row=int(indexes[index]),
                         material_bucket=int(np.clip((int(selected_rows[j]['occupancy']).bit_count() - 2) * 8 // 32, 0, 7)),
                         us_king_bucket=int(mapping[int(selected_rows[j]['us_king']) ^ (7 if int(selected_rows[j]['us_king']) & 7 >= 4 else 0)]),
                         them_king_bucket=int(mapping[int(selected_rows[j]['them_king']) ^ (7 if int(selected_rows[j]['them_king']) & 7 >= 4 else 0)]))
                    for j, index in enumerate(selected))
    raw_share = count / config['archive_split']['raw_records']
    exposures.append(dict(source=source_id, path=path, raw_records=count,
                          raw_corpus_share=raw_share,
                          volume_expected_presentations_58=raw_share * 14848000000,
                          sampled_train=int(train_hist.sum()), sampled_holdout=int(hold_hist.sum()),
                          sampled_train_material=train_hist.sum(axis=(1, 2)).tolist(),
                          sampled_train_us_king=train_hist.sum(axis=(0, 2)).tolist(),
                          sampled_train_them_king=train_hist.sum(axis=(0, 1)).tolist(),
                          sampled_joint_train=train_hist.tolist(),
                          sampled_joint_holdout=hold_hist.tolist()))
    status('source-sampled', dict(source=source_id, panel=rows_per_source))
records = np.concatenate(panel)
assert len(records) == 8192
records.tofile(p / 'panel.ngnp')
(p / 'panel-metadata.json').write_text(json.dumps(metadata) + '\n')
(p / 'sample-exposures.json').write_text(json.dumps(exposures, indent=2) + '\n')
status('panel-frozen', dict(rows=len(records), panel_sha256=sha(p / 'panel.ngnp')))
torch.set_num_threads(2)
checkpoint = torch.load(t / 'best.pt', map_location='cpu', weights_only=True)
assert checkpoint['epoch'] == 58
model = NGNN(512, 8, config['king_map']).cuda().eval()
model.load_state_dict(checkpoint['model'])
del checkpoint
network, baseline = Network(t / 'best.nnue'), Network(base_path)
fp32, bf16, integer, old_integer = [], [], [], []
with torch.inference_mode():
    for start in range(0, len(records), 256):
        batch = records[start:start + 256]
        raw = torch.from_numpy(batch.view(np.uint8).reshape(-1, 32).copy()).cuda()
        features, stm, _, _, bucket = decode_gpu(raw, 8, model.king_map, 8)
        fp32.extend(model(features, stm, bucket).cpu().tolist())
        with torch.autocast('cuda', dtype=torch.bfloat16):
            bf16.extend(model(features, stm, bucket).float().cpu().tolist())
        integer.extend(network.evaluate_record(row) for row in batch)
        old_integer.extend(baseline.evaluate_record(row) for row in batch)
values = {name: np.asarray(value, dtype=np.float64) for name, value in
          [('fp32', fp32), ('bf16', bf16), ('integer', integer), ('accepted_integer', old_integer)]}
assert all(np.isfinite(value).all() for value in values.values())
sigmoid = lambda value: 1 / (1 + np.exp(-np.clip(value / 400, -700, 700)))
result = records['result'].astype(np.float64) / 2
target = .75 * sigmoid(records['score'].astype(np.float64)) + .25 * result


def metrics(mask):
    residual = values['integer'][mask] - values['fp32'][mask]
    absolute = np.abs(residual)
    return dict(n=int(mask.sum()), mean_integer_minus_fp32_cp=float(residual.mean()),
                mae_cp=float(absolute.mean()), p95_absolute_cp=float(np.quantile(absolute, .95)),
                max_absolute_cp=float(absolute.max()),
                loss={name: float(np.mean((sigmoid(value[mask]) - target[mask]) ** 2))
                      for name, value in values.items()},
                result_loss={name: float(np.mean((sigmoid(value[mask]) - result[mask]) ** 2))
                             for name, value in values.items()})


grouped = {}
for field in ['source', 'material_bucket', 'us_king_bucket', 'them_king_bucket']:
    labels = np.asarray([row[field] for row in metadata])
    grouped[field] = {str(label): metrics(labels == label) for label in np.unique(labels)}
summary = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               rows=len(records), panel_sha256=sha(p / 'panel.ngnp'),
               aggregate=metrics(np.ones(len(records), dtype=bool)), groups=grouped,
               sources=[{k: v for k, v in exposure.items() if not k.startswith('sampled_joint')}
                        for exposure in exposures],
               limits='source-balanced frozen sampled holdout; exposure estimates, not actual training counters; no Elo inference',
               automatic_training=False, automatic_games=False)
(p / 'evaluations.json').write_text(json.dumps(dict(metadata=metadata, values={name: value.tolist() for name, value in values.items()})) + '\n')
(p / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
status('diagnostic-done', summary['aggregate'])

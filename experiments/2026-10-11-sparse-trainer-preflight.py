"""Original sparse gather/sum diagnostic; never save a model or export."""
import datetime
import hashlib
import json
import math
from pathlib import Path
import time

import numpy as np
import torch
from torch.nn import functional as F

from trainer.model import NGNN, blended_loss, decode_gpu
from trainer.ngnp import RECORD_DTYPE

r = Path('/home/ehrli/ngn-data/night-20261007')
p = r / 'session68-sparse-preflight'
t = r / 'session57-data-step2/trained'
panel = r / 'session66-data-diagnostic/panel.ngnp'
sha = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                script_sha256=sha(__file__), panel_sha256=sha(panel), panel_rows=8192,
                instruction='experiments/2026-10-11-sparse-trainer-preflight.md',
                batch=16384, warmup_steps=8, blocks=12, steps_per_block=64,
                cpus=[12, 14], memory_max_bytes=4 * 1024**3, swap_max=0, nice=19,
                mechanism='original sparse feature decode + embedding_bag sum; unchanged output heads',
                proposal_rule='point throughput gain>=20%; paired log-ratio lower95>0',
                automatic_training=False, automatic_games=False, save_checkpoint=False)
(p / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                    stage='sparse-trainer-preflight', event=event, detail=detail)) + '\n')


def sparse_decode(raw, mapping):
    batch = len(raw)
    occupancy = raw[:, :8].view(torch.int64).reshape(batch, 1)
    squares = torch.arange(64, device=raw.device).reshape(1, 64)
    occupied = ((occupancy >> squares) & 1).bool()
    ordinal = (occupied.long().cumsum(1) - 1).clamp_min(0)
    nibble = torch.arange(32, device=raw.device).reshape(1, 32)
    codes = (raw[:, 8:24][:, nibble[0] // 2].long() >> ((nibble % 2) * 4)) & 15
    board = torch.where(occupied, codes.gather(1, ordinal), 0)
    wk = ((board == 5) & occupied).long().argmax(1)
    bk = ((board == 11) & occupied).long().argmax(1) ^ 56
    wm, bm = torch.where((wk & 7) >= 4, 7, 0), torch.where((bk & 7) >= 4, 7, 0)
    white = ((board // 6) * 384 + (board % 6) * 64 + squares) ^ wm[:, None]
    black = ((1 - board // 6) * 384 + (board % 6) * 64 + (squares ^ 56)) ^ bm[:, None]
    white += mapping[wk ^ wm][:, None] * 768
    black += mapping[bk ^ bm][:, None] * 768
    indexes = torch.cat([torch.where(occupied, white, 6144),
                         torch.where(occupied, black, 6144)])
    stm = raw[:, 27].long()
    score = raw[:, 24:26].view(torch.int16).reshape(batch).float() * (1 - 2 * stm)
    result = raw[:, 26].float() * .5
    result = torch.where(stm == 0, result, 1 - result)
    bucket = ((occupied.sum(1) - 2) * 8 // 32).clamp(0, 7)
    return indexes, stm, score, result, bucket


def sparse_forward(model, indexes, stm, bucket, amp):
    weights = model.effective_w1()
    weights = weights.to(torch.bfloat16) if amp else weights
    weights = torch.cat([weights, weights.new_zeros((1, model.hidden))])
    acc = F.embedding_bag(indexes, weights, mode='sum', padding_idx=6144) + model.b1
    white, black = acc.clamp(0, 1).square().chunk(2)
    heads = model.o[bucket]
    us = torch.where((stm == 0)[:, None], white, black)
    them = torch.where((stm == 0)[:, None], black, white)
    return ((us * heads[:, :model.hidden] + them * heads[:, model.hidden:]).sum(1)
            + model.ob[bucket]) * 400


torch.set_num_threads(2)
torch.set_num_interop_threads(1)
torch.set_float32_matmul_precision('high')
records = np.fromfile(panel, dtype=RECORD_DTYPE)
assert len(records) == 8192
raw = torch.from_numpy(np.tile(records.view(np.uint8).reshape(-1, 32), (2, 1))).cuda()
checkpoint = torch.load(t / 'best.pt', map_location='cpu', weights_only=True)
assert checkpoint['epoch'] == 58
config, state = checkpoint['config'], checkpoint['model']
models = {name: NGNN(512, 8, config['king_map']).cuda() for name in ['dense', 'sparse']}
for model in models.values():
    model.load_state_dict(state)
del checkpoint
mapping = models['dense'].king_map
status('numerical-start', 'all8192 frozen rows; no training artifact writes')
indexes, stm, score, result, bucket = sparse_decode(raw, mapping)
features, dense_stm, dense_score, dense_result, dense_bucket = decode_gpu(raw, 8, mapping, 8)
reconstructed = torch.zeros((len(indexes), 6145), device='cuda')
reconstructed.scatter_add_(1, indexes, torch.ones_like(indexes, dtype=torch.float32))
assert torch.equal(reconstructed[:, :6144], features)
assert all(torch.equal(a, b) for a, b in zip([stm, score, result, bucket],
                                           [dense_stm, dense_score, dense_result, dense_bucket]))
del reconstructed, features
numerical = {}
for amp in [False, True]:
    values, grads = {}, {}
    for name, model in models.items():
        model.zero_grad(set_to_none=True)
        with torch.autocast('cuda', dtype=torch.bfloat16, enabled=amp):
            if name == 'dense':
                f, s, sc, re, b = decode_gpu(raw, 8, model.king_map, 8)
                output = model(f, s, b)
            else:
                ix, s, sc, re, b = sparse_decode(raw, model.king_map)
                output = sparse_forward(model, ix, s, b, amp)
            loss = blended_loss(output, sc, re, .75, 400.)
        loss.backward()
        assert torch.isfinite(output).all().item() and torch.isfinite(loss).item()
        values[name] = (output.detach().cpu(), float(loss))
        grads[name] = {key: parameter.grad.detach().cpu().clone()
                       for key, parameter in model.named_parameters()}
    residual = (values['sparse'][0] - values['dense'][0]).abs()
    numerical['bf16' if amp else 'fp32'] = dict(
        output_mae_cp=float(residual.mean()), output_max_cp=float(residual.max()),
        loss_dense=values['dense'][1], loss_sparse=values['sparse'][1],
        max_absolute_gradient_residual={key: float((grads['sparse'][key] - grads['dense'][key]).abs().max())
                                       for key in grads['dense']})
del indexes, grads, values, output, loss, f, ix
status('numerical-done', numerical)

optimizers = {name: torch.optim.AdamW(model.parameters(), lr=.0001, weight_decay=.01)
              for name, model in models.items()}


def update(name):
    model, optimizer = models[name], optimizers[name]
    optimizer.zero_grad(set_to_none=True)
    with torch.autocast('cuda', dtype=torch.bfloat16):
        if name == 'dense':
            features, stm, score, result, bucket = decode_gpu(raw, 8, model.king_map, 8)
            evaluation = model(features, stm, bucket)
        else:
            indexes, stm, score, result, bucket = sparse_decode(raw, model.king_map)
            evaluation = sparse_forward(model, indexes, stm, bucket, True)
        loss = blended_loss(evaluation, score, result, .75, 400.)
    loss.backward()
    optimizer.step()
    model.clip_quantized()


for name in ['dense', 'sparse']:
    models[name].load_state_dict(state)
    for _ in range(8):
        update(name)
torch.cuda.synchronize()
timings = []
status('timing-start', '12 fixed alternating64-update blocks')
for block in range(12):
    row = dict(block=block + 1)
    order = ['dense', 'sparse'] if block % 2 == 0 else ['sparse', 'dense']
    for name in order:
        torch.cuda.reset_peak_memory_stats()
        torch.cuda.synchronize()
        started = time.perf_counter()
        for _ in range(64):
            update(name)
        torch.cuda.synchronize()
        row[name + '_seconds'] = time.perf_counter() - started
        row[name + '_peak_allocated_bytes'] = torch.cuda.max_memory_allocated()
    row['throughput_ratio'] = row['dense_seconds'] / row['sparse_seconds']
    timings.append(row)
    with (p / 'blocks.jsonl').open('a') as stream:
        stream.write(json.dumps(row) + '\n')
ratios = np.log([row['throughput_ratio'] for row in timings])
mean, error = float(ratios.mean()), 1.96 * float(ratios.std(ddof=1)) / math.sqrt(12)
interval = [math.exp(value) for value in [mean, mean - error, mean + error]]
summary = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               numerical=numerical, feature_geometry_exact=True, timed_blocks=12,
               steps_per_block=64, throughput_ratio_ci95=interval,
               proposal_verdict='eligible-for-full-pipeline-study' if interval[0] >= 1.2 and interval[1] > 1 else 'shelved',
               limits='fixed repeated panel, no I/O; numerical reduction/gradient differences remain; no quality/Elo claim',
               automatic_training=False, automatic_games=False)
(p / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
status('preflight-done', summary)

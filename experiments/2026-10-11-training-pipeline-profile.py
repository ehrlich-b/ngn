"""Bounded profile of the unchanged dense trainer; discard every update."""
import datetime
import hashlib
import json
from pathlib import Path
import time

import numpy as np
import torch

from trainer.archive import ARCHIVE_DTYPE, Archive, board_hash
from trainer.model import NGNN, blended_loss, decode_gpu
from trainer.remote_archive import RemoteArray
from trainer.train import prefetched

r = Path('/home/ehrli/ngn-data/night-20261007')
p = r / 'session68-training-profile'
t = r / 'session57-data-step2/trained'
config = json.loads((t / 'config.json').read_text())
seed, block_size, warmup, timed, profiled = 2026101108, 262144, 16, 128, 8
sha = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                    stage='training-pipeline-profile', event=event, detail=detail)) + '\n')


sources = [source for source in config['archive_split']['sources']
           if not source['path'].endswith('/t75.archive.json')]
all_blocks = [(i, start, min(start + block_size, source['records']))
              for i, source in enumerate(sources)
              for start in range(0, source['records'], block_size)]
ids = np.random.default_rng(seed).permutation(len(all_blocks))[:16]
blocks = [all_blocks[int(i)] for i in ids]
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                instruction='experiments/2026-10-11-training-pipeline-profile.md',
                source='/home/ehrli/ngn-data/night-20261007/session62-loader-index/source',
                script_sha256=sha(__file__), config_sha256=sha(t / 'config.json'),
                checkpoint_sha256='a0b2da5c7497f7338db55d8f95d0795ec00aa92099a20aaf06179dff7097a750',
                seed=seed, sources=sources, excluded_source='T75: avoid forge game-lane contention',
                selected_blocks=blocks, warmup_steps=warmup,
                timed_steps=timed, operator_profile_steps=profiled, batch=16384,
                hidden=512, king_buckets=8, material_buckets=8, precision='bf16',
                float32_matmul_precision='high', cpu_threads=2, cpus=[12, 14],
                memory_max_bytes=4 * 1024**3, swap_max=0, nice=19,
                no_checkpoint_or_export=True, automatic_training=False, automatic_games=False)
(p / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
status('sample-start', '16 fixed blocks; bounded view, original loader mechanism')

# Retain only this diagnostic's blocks; do not invoke the full-corpus constructor.
archive = object.__new__(Archive)
archive.maps = []
for source in sources:
    path, count = source['path'], source['records']
    archive.maps.append(RemoteArray(path, ARCHIVE_DTYPE) if path.endswith('.archive.json') else
                        np.memmap(path, dtype=ARCHIVE_DTYPE, mode='r', shape=(count,)))
archive.blocks, archive.heldout, archive.split = blocks, [], 'train'
sample_counts = [0] * len(sources)
started = time.perf_counter()
for source, start, end in blocks:
    rows = archive.maps[source][start:end]
    selected = np.flatnonzero((board_hash(rows, 20261006) & np.uint64(1023)) == 0).astype(np.uint32)
    archive.heldout.append(selected)
    sample_counts[source] += end - start - len(selected)
sample_seconds = time.perf_counter() - started
status('sample-done', dict(training_rows_by_source=sample_counts, seconds=sample_seconds))

torch.set_num_threads(2)
torch.set_num_interop_threads(1)
torch.set_float32_matmul_precision('high')
torch.manual_seed(seed)
checkpoint = torch.load(t / 'best.pt', map_location='cpu', weights_only=True)
assert checkpoint['epoch'] == 58
model = NGNN(512, 8, config['king_map']).cuda().train()
model.load_state_dict(checkpoint['model'])
del checkpoint
optimizer = torch.optim.AdamW(model.parameters(), lr=.0001, weight_decay=.01)
batches = iter(prefetched(archive.batches(16384, seed)))
stage_names = ['transfer', 'decode', 'forward_loss', 'backward', 'optimizer_clip']
events, cpu_times = [], []


def update(measure=False):
    cpu_start = time.perf_counter()
    raw_numpy = next(batches)
    loaded = time.perf_counter()
    raw = torch.from_numpy(np.ascontiguousarray(raw_numpy)).pin_memory()
    pinned = time.perf_counter()
    marks = [torch.cuda.Event(enable_timing=True) for _ in range(6)] if measure else None
    if measure:
        marks[0].record()
    raw = raw.to('cuda', non_blocking=True)
    if measure:
        marks[1].record()
    optimizer.zero_grad(set_to_none=True)
    with torch.autocast('cuda', dtype=torch.bfloat16):
        features, stm, score, result, bucket = decode_gpu(raw, 8, model.king_map, 8)
        if measure:
            marks[2].record()
        loss = blended_loss(model(features, stm, bucket), score, result, .75, 400.)
    if measure:
        marks[3].record()
    loss.backward()
    if measure:
        marks[4].record()
    optimizer.step()
    model.clip_quantized()
    if measure:
        marks[5].record()
        events.append(marks)
        cpu_times.append([loaded - cpu_start, pinned - loaded, time.perf_counter() - pinned])
    return loss


for _ in range(warmup):
    update()
torch.cuda.synchronize()
torch.cuda.reset_peak_memory_stats()
status('timing-start', dict(steps=timed))
started = time.perf_counter()
for _ in range(timed):
    last_loss = update(True)
torch.cuda.synchronize()
seconds = time.perf_counter() - started
stage_ms = {name: sum(row[i].elapsed_time(row[i + 1]) for row in events)
            for i, name in enumerate(stage_names)}
peak = torch.cuda.max_memory_allocated()
assert torch.isfinite(last_loss).item()

status('operator-profile-start', dict(steps=profiled))
with torch.profiler.profile(activities=[torch.profiler.ProfilerActivity.CPU,
                                       torch.profiler.ProfilerActivity.CUDA], record_shapes=True) as profile:
    for _ in range(profiled):
        update()
        profile.step()
    torch.cuda.synchronize()
averages = profile.key_averages(group_by_input_shape=True)
(p / 'operators.txt').write_text(averages.table(sort_by='self_cuda_time_total', row_limit=40))
operators = [dict(name=row.key, calls=row.count, input_shapes=row.input_shapes,
                  cpu_total_us=row.cpu_time_total, cpu_self_us=row.self_cpu_time_total,
                  device_total_us=row.device_time_total, device_self_us=row.self_device_time_total)
             for row in averages]
(p / 'operators.json').write_text(json.dumps(operators, indent=2) + '\n')
summary = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               steps=timed, rows=timed * 16384, seconds=seconds,
               positions_per_second=timed * 16384 / seconds,
               cuda_stage_ms=stage_ms,
               cpu_loader_wait_seconds=sum(row[0] for row in cpu_times),
               cpu_pin_seconds=sum(row[1] for row in cpu_times),
               cpu_enqueue_seconds=sum(row[2] for row in cpu_times),
               gpu_peak_allocated_bytes=peak, dense_features_bytes=2 * 16384 * 6144 * 4,
               sample_training_rows_by_source=sample_counts, sample_seconds=sample_seconds,
               last_loss=float(last_loss), torch=torch.__version__, gpu=torch.cuda.get_device_name(),
               limits='bounded16-block T80/T76/T74 sample; excludes forge T75 network latency; diagnostic overhead; CPU/GPU times overlap; no numerical or Elo verdict',
               automatic_training=False, automatic_games=False)
(p / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
status('profile-done', dict(positions_per_second=summary['positions_per_second'], cuda_stage_ms=stage_ms))

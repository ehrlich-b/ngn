"""Measure the new occupied-board union once; retain bottom-k samples."""
import datetime
import hashlib
import ctypes
import json
import math
import os
from pathlib import Path
import select
import time

import numpy as np
from trainer.archive import ARCHIVE_DTYPE, board_hash

s = Path('/work/lanes/ngn-data-step2-20261011')
K = 65536

# Conversion and the independent old-corpus stream may finish in either order.
libc = ctypes.CDLL(None, use_errno=True)
fd = libc.inotify_init1(os.O_CLOEXEC)
assert fd >= 0 and libc.inotify_add_watch(fd, os.fsencode(s), 0x8 | 0x80) >= 0
poll = select.poll()
poll.register(fd, select.POLLIN)
while not (s / 'stream.done').exists():
    assert poll.poll(12 * 3600 * 1000), 'old-corpus stream wait expired'
    os.read(fd, 65536)
os.close(fd)


def update(sample, hashes):
    if len(sample) == K:
        hashes = hashes[hashes < sample[-1]]
    if not len(hashes):
        return sample
    values = np.unique(np.concatenate((sample, hashes)))
    if len(values) > K:
        values = np.partition(values, K - 1)[:K]
        values.sort()
    return values


def estimate(sample):
    if len(sample) < K:
        return dict(distinct=len(sample), exact_below_k=True)
    n = (K - 1) * 2**64 / (int(sample[-1]) + 1)
    relative = 1.96 / math.sqrt(K - 2)
    return dict(distinct=n, approx_ci95=[n * (1 - relative), n * (1 + relative)],
                relative_standard_error=1 / math.sqrt(K - 2), exact_below_k=False)


union = np.empty(0, dtype=np.uint64)
results = {}
started = time.monotonic()
for name in ['t80', 't76', 't74full', 't75']:
    if name in ['t80', 't76']:
        receipt = json.loads((s / (name + '-stream.json')).read_text())
        sample = np.load(s / (name + '-bottom65536.npy'))
        union = update(union, sample)
        results[name] = dict(bytes=receipt['bytes'], records=receipt['bytes'] // 32,
                             **estimate(sample))
        continue
    elif name == 't74full' and (s / 'storage-resumed.json').exists():
        receipt = json.loads((s / 't74full-rig-copy.json').read_text())
        sample_pin = json.loads((s / 't74full-sample.json').read_text())
        assert all(receipt[key] == sample_pin[key] for key in ['bytes', 'sha256'])
        assert sample_pin['seed'] == 20261006 and sample_pin['k'] == K
        assert hashlib.sha256((s / 't74full-bottom65536.npy').read_bytes()).hexdigest() == sample_pin['sample_sha256']
        sample = np.load(s / 't74full-bottom65536.npy')
        union = update(union, sample)
        results[name] = dict(bytes=receipt['bytes'], records=receipt['bytes'] // 32,
                             **estimate(sample))
        token = (s / 'token').read_text().strip()
        descriptor = dict(bytes=receipt['bytes'], sha256=receipt['sha256'],
                          url='http://127.0.0.1:8769/' + token + '/t74full.bf')
        (s / 't74full.archive.json').write_text(json.dumps(descriptor, indent=2) + '\n')
        print(json.dumps(dict(source=name, storage='rig exact copy', **results[name])), flush=True)
        continue
    else:
        path = s / (name + '.bf')
        rows = np.memmap(path, dtype=ARCHIVE_DTYPE, mode='r')
        count, size = len(rows), path.stat().st_size
    sample = np.empty(0, dtype=np.uint64)
    for start in range(0, count, 262144):
        sample = update(sample, board_hash(rows[start:min(start + 262144, count)], 20261006))
    np.save(s / (name + '-bottom65536.npy'), sample)
    union = update(union, sample)
    results[name] = dict(bytes=size, records=count, **estimate(sample))
    print(json.dumps(dict(source=name, **results[name])), flush=True)
np.save(s / 'union-bottom65536.npy', union)
after = estimate(union)
ratio = after['distinct'] / 4625841241.681954
j = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
         method='bottom65536 occupied-board hashes; ngn-board-mix64-v1/seed20261006',
         sources=results, combined=after, previous_distinct=4625841241.681954,
         estimated_unique_ratio=ratio, records=sum(r['records'] for r in results.values()),
         seconds=time.monotonic() - started,
         limits='occupied boards, not games or symmetry classes; correlated approximate errors',
         training_eligible=1.9 <= ratio <= 2.3)
(s / 'coverage.json').write_text(json.dumps(j, indent=2) + '\n')
print(json.dumps(j), flush=True)
payload = dict(coverage=j, descriptors={name: json.loads((s / (name + '.archive.json')).read_text())
                                       for name in ['t74full', 't75']})
# Existing archive-server descriptor endpoint waits for this atomic publication.
(s / 't74.archive.json.tmp').write_text(json.dumps(payload, indent=2) + '\n')
(s / 't74.archive.json.tmp').replace(s / 't74.archive.json')

"""Read-only, fixed-prefix diagnostic on the existing closed gen0b shards."""

import hashlib
import json
from pathlib import Path
import time

import numpy as np

from trainer.ngnp import RECORD_DTYPE
from trainer.ngnp_split import ContentShards


root = Path('/home/ehrli/ngn-data/night-20261007/session7')
data = Path('/home/ehrli/ngn-data/gen0b/data/run-20261006T032358Z-ewE9tg')
paths = sorted(data.glob('*.ngnp'))
assert len(paths) == 14
cap, seed, modulus = 65536, 20261008, 1024
sizes = {str(p): p.stat().st_size for p in paths}
assert all(size % 32 == 0 and size // 32 >= cap for size in sizes.values())
before = {str(p): hashlib.sha256(p.open('rb').read(cap * 32)).hexdigest() for p in paths}
counts = {str(p): cap for p in paths}
started = time.monotonic()
train = ContentShards(list(counts), record_counts=counts, modulus=modulus, seed=seed,
                      require_kings=True)
holdout = train.holdout()


def keys(records):
    # This diagnostic compares exact occupied-board wire bytes. Verify unused
    # nibbles are zero, rather than treating arbitrary padding as board content.
    occupied = records['occupancy'].copy()
    count = np.zeros(len(records), dtype=np.uint8)
    for bit in range(64):
        count += ((occupied >> np.uint64(bit)) & np.uint64(1)).astype(np.uint8)
    nibble = np.arange(32)[None, :]
    codes = (records['pieces'][:, nibble[0] // 2] >> ((nibble % 2) * 4)) & 15
    assert np.all(codes[nibble >= count[:, None]] == 0)
    raw = records.view(np.uint8).reshape(-1, 32)
    return np.ascontiguousarray(raw[:, :24]).view('V24').reshape(-1)


legacy = []
for group in (paths[:7], paths[7:]):
    parts = []
    for path in group:
        records = np.memmap(path, mode='r', dtype=RECORD_DTYPE, shape=(cap,))
        eligible = ((records['flags'] & 15) == 0) & (records['ply'] >= 0)
        parts.append(keys(records[eligible]))
    legacy.append(np.unique(np.concatenate(parts)))
old_overlap = len(np.intersect1d(*legacy))
partitions = []
for reader in (train, holdout):
    parts = [keys(batch.reshape(-1).view(RECORD_DTYPE))
             for batch in reader.batches(16384, seed, shuffle=False)]
    assert sum(map(len, parts)) == reader.count
    partitions.append(np.unique(np.concatenate(parts)))
overlap = len(np.intersect1d(*partitions))
assert overlap == 0
assert len(np.union1d(*legacy)) == sum(map(len, partitions))
after = {str(p): hashlib.sha256(p.open('rb').read(cap * 32)).hexdigest() for p in paths}
assert before == after
assert sizes == {str(p): p.stat().st_size for p in paths}
receipt = dict(data=str(data), prefix_records_per_shard=cap, shards=14,
               scope='first fixed prefix of each closed gen0b shard; not a full-corpus audit',
               prefix_sha256=before, unchanged_prefixes_and_sizes=True,
               split=train.split_receipt,
               legacy_seven_shards_each_unique_boards=list(map(len, legacy)),
               legacy_cross_shard_shared_boards=old_overlap,
               content_partition_unique_boards=list(map(len, partitions)),
               content_partition_shared_boards=overlap,
               seconds=time.monotonic()-started, cpu=[13,15], nice=19)
output=root/'data-sample-receipt.json'
with output.open('x') as stream:
    stream.write(json.dumps(receipt,indent=2)+'\n')
print(json.dumps({k:v for k,v in receipt.items() if k not in ('prefix_sha256','split')}))

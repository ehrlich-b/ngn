# NGN K4 accepted-corpus finalizer

`cmd/ngnk4finalize` closes the gap between deterministic pre-label candidates
and exact accepted training counts. The sampler deliberately includes 20%
headroom because the frozen Stockfish teacher rejects some inputs. The
finalizer follows sampler shard order, validates the sampler input, completed
label output, independent packer receipt and packed BF hash, then takes an
exact accepted prefix.

Artifact names are fixed by sampler shard ID:

- `LABELS/SHARD.labels.jsonl`
- `PACKED/SHARD.bf`
- `PACKED/SHARD.pack.json`

The pilot command writes exactly 1,000,000 accepted train records and exactly
100,000 records for each fixed holdout:

```text
ngnk4finalize pilot -sampler SAMPLER/manifest.json -labels LABELS \
  -packed PACKED -output NEW_PILOT_DIR
```

The main command copies the hash-verified pilot train corpus first, then takes
exactly 19,000,000 accepted records from the disjoint main-expansion series.
Thus the pilot is an exact byte-for-byte prefix and subset of the 20,000,000
record main corpus:

```text
ngnk4finalize main -sampler SAMPLER/manifest.json \
  -pilot PILOT/manifest.json -labels LABELS -packed PACKED \
  -output NEW_MAIN_DIR
```

Both modes refuse an existing output directory. Their manifests bind ordered
sampler, label and pack artifacts, accepted identity streams, exact BF hashes,
the frozen SF18 executable and network provenance, and any parent pilot.

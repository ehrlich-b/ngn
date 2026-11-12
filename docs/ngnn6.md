# NGNN6 residual output format

NGNN6 uses magic `NGN6` and version 6. It retains the NGNN3 KB8
accumulator, king orientation, material bucket selection, and original linear
head. A shared 16-unit bottleneck adds a residual in centipawns. This is a
fixed layout, with eight king buckets and eight output rows; it is not a graph
format. Only H32 (small fixtures) and H768 are reserved accumulator widths.
All other widths, dimensions, flags and scales are rejected. No NGNN4/5 data
or external weights are used.

`ReadNGNN6(io.Reader)` returns a separate immutable layout. The scalar
`evaluateBucket` method is an internal format reference. `ReadNGNN1` and UCI
`EvalFile` accept explicit NGNN6 selection and report backend `ngnn6`. Both
compact and wide workers execute the residual before the final clamp. The
embedded default remains NGNN3. Fixture success
makes no speed or Elo claim. Changing default play requires a completed
confirmation on the compute host for that separate decision.

## Features and arithmetic

For each perspective, flip Black's squares by XOR 56. If that perspective's
king is on files e–h after the flip, mirror all files by XOR 7. The 64-byte
map indexes the transformed king square and selects one of eight blocks.
The feature index within a block is `relativeColor*384 + (pieceType-1)*64
+ transformedSquare`, with own color first and pawn through king numbered
1–6. Each block has 768 rows. Accumulate the int16 input bias plus active
int16 rows into int32, as in NGNN3; at most one piece occupies each of 64
squares. Accumulators are stored White then Black. Concatenate side-to-move
then opponent to form a vector of length 2H and clip each entry to [0,255],
giving `x`. A malformed position without a king uses square zero, as NGNN3
does. The map can be any 64 values in [0,7], without a symmetry constraint.

Count all pieces including kings and select
`b = min(7, max(0, trunc((pieceCount-2)*8/32)))`.
All divisions below are signed integer divisions **toward zero**, performed
in int64. All products, sums, conversions to cp and the final addition use
int64. The original branch is exactly:

```
S = sum(i=0..2H-1, x[i]*x[i]*L[b,i])
linear_cp = trunc((trunc(S/255) + LB[b])*400 / (255*64))
```

In particular, retain the inner truncation and add the linear bias after it.
The residual uses unsquared clipped accumulator values, shared across all
output buckets:

```
T[j] = HB[j] + sum(i=0..2H-1, x[i]*A[j,i])       # j=0..15
z[j] = clip(trunc(T[j]/64), 0, 255)
P = RB[b] + sum(j=0..15, z[j]*R[b,j])
residual_cp = trunc(P*400 / (255*64))
score = clip(linear_cp + residual_cp, -24999, 24999)
```

The input division 64 converts the bottleneck weights' fixed scale into
activation units. Residual activations have the same cap 255 as the input;
the residual output uses fixed weight scale 64 and cp scale 400. These
constants are explicit header values, not tunable fields. Never merge the
two branches' truncations or clamp linear cp before adding residual cp.
Setting **both** `R` and `RB` to zero yields exactly the NGNN3 function,
regardless of the accepted bottleneck weights and biases. A nonzero residual
output bias alone is not a zero projection.

## Binary layout

All numeric fields are little-endian. Weights are signed two's-complement
int16. Head biases are signed int32; the inherited accumulator bias stays
int16 to preserve the NGNN3 accumulator. No padding or optional sections
exist. Matrices are row-major.

| Header byte offset | Type | Required value |
| --- | --- | --- |
| 0 | 4 bytes | ASCII `NGN6` |
| 4 | uint32 | version 6 |
| 8 | uint32 | H: 32 or 768 |
| 12 | uint32 | king buckets: 8 |
| 16 | uint32 | output rows: 8 |
| 20 | uint32 | QA: 255 |
| 24 | uint32 | linear QB: 64 |
| 28 | uint32 | linear cp scale: 400 |
| 32 | uint8[64] | king bucket map, each <8 |
| 96 | uint32 | bottleneck width: 16 |
| 100 | uint32 | bottleneck input divisor: 64 |
| 104 | uint32 | bottleneck activation cap: 255 |
| 108 | uint32 | residual output divisor: 64 |
| 112 | uint32 | residual cp scale: 400 |
| 116 | uint32 | reserved flags: 0 |

After the 120-byte header:

| Order | Array | Elements | Bytes |
| --- | --- | --- | --- |
| 1 | input weights W[8*768,H] | 6144H int16 | 12288H |
| 2 | accumulator bias B[H] | H int16 | 2H |
| 3 | linear weights L[8,2H] | 16H int16 | 32H |
| 4 | linear biases LB[8] | 8 int32 | 32 |
| 5 | bottleneck weights A[16,2H] | 32H int16 | 64H |
| 6 | bottleneck biases HB[16] | 16 int32 | 64 |
| 7 | residual projection R[8,16] | 128 int16 | 256 |
| 8 | residual biases RB[8] | 8 int32 | 32 |

The payload length is `2*(6144H+H+16H+32H+128) + 4*(8+16+8)`
= **12386H+384** bytes. Follow it with one uint32 IEEE CRC-32 (Go
`crc32.ChecksumIEEE`, Python `zlib.crc32`) over **every preceding byte**:
magic, complete header including map/flags, and payload. The CRC itself is
excluded. File length is exactly **12386H+508** bytes: 396,860 for H32 and
9,512,956 for H768. EOF is required immediately after the CRC. Truncation,
trailing data, read errors and checksum mismatches return no partial layout.

## Bounds and resource budget

The parser validates the complete fixed header before any payload allocation.
It reads at most the declared file length plus one trailing-byte probe. Raw
file storage is at most 9,512,956 bytes. Decoded numeric storage is exactly
the payload size (9,512,832 bytes at H768); raw and decoded arrays coexist
during parsing, for 19,025,788 bytes plus small object/slice overhead. The raw
buffer is not retained. Added residual numeric storage over NGNN3 is
`64H+352` bytes: 2,400 at H32 and 49,504 at H768.

The inherited accumulator is bounded by `[-65*32768,65*32767]` =
[-2,129,920, 2,129,855], which fits int32. At H768 the original linear
sum has absolute value at most `1536*255^2*32768` = 3,272,815,411,200.
After dividing by 255 and adding any int32 bias, the absolute bound is
14,982,053,888; multiplying by 400 is at most 5,992,821,555,200, well
inside int64. This permits the full int16 linear weights and int32 biases.

Version 6 additionally **reserves signed int32 affine ranges** for both
residual stages, while performing their accumulation in int64. For every
row with bias `q` and weights `w`, compute
`lo=q+255*sum(min(w,0))`, `hi=q+255*sum(max(w,0))`.
Reject the file unless `lo >= -2147483648` and `hi <= 2147483647`.
Apply this independently to all 16 A rows and all eight R rows. This proves
every possible clipped input, not merely observed chess positions. It can
conservatively reject correlated inputs; that is part of the format contract.
The unvalidated bottleneck interval calculations themselves are bounded by
`2^31+1536*255*32768` = 14,982,053,888, so the proof cannot overflow int64.
Endpoint biases -2^31 and 2^31-1 are valid with zero row weights.

Consequently `T` and `P` fit int32, clipped z fits [0,255], and
`abs(P*400) <= 858,993,459,200`. Residual cp has absolute bound
52,634,403; linear cp at H768 has absolute bound 367,207,203. Their sum
fits int64 (and int32) before the final clamp. H32 is strictly smaller.

At H768 the extra arithmetic is **1536*16 = 24,576 input MACs plus
16 selected-output MACs per evaluation**, with 16 bottleneck biases,
divisions/clips and one selected output bias/conversion. Only one output
row is evaluated. This is an arithmetic budget, not a measured speed
prediction; accumulator maintenance and the original linear branch remain.

## Reproduction

`experiments/wave2-1104/residual-format/generate.py --check` regenerates
the synthetic H32 bytes and independent Python scalar oracle in memory and
checks the committed files. No training data or downloaded network is used.
The Go fragment covers both turns, all eight output rows, independent
accumulators/scores, exact signed truncation, zero-projection identity,
clamp ordering, boundary arithmetic and malformed controls:

```
GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run '^TestWave2ResidualFormat' -timeout 15s
```

The executable `experiments/wave2-1104/residual-format/run.py` runs that
fragment at nice level 10 with two CPU threads and scrubbed credential
variables. `result.txt` is the captured fixture-run result, not a benchmark.

Production inference is checked by the separate int64 oracle and special-move
walk in `engine/ngnn6_eval_test.go`. The executable
`experiments/wave2-1104/residual-infer/run.py fragment` runs only
`^TestWave2ResidualEval` with a 15-second timeout and two CPU threads. The
committed H32 fixture has an independent Python constructor (`generate.py
--check`); `fixture.json` includes signed arithmetic and walk inputs. The
zero-residual check also wraps the actual embedded H768 linear weights and
compares both storage paths to the NGNN3 reference. No speed or strength claim
follows from these tests.

## Byte-exact warm start and independent oracle

`python3 -m trainer.residual_export SOURCE OUTPUT --receipt RECEIPT` lifts
an owned NGNN3 H32/H768 KB8/B8 file into NGNN6. It validates the original
length, header and CRC, copies its complete numeric payload and king map
without requantizing, and initializes the shared bottleneck using PCG64 seed
20261104. Both residual projection arrays are zero, preserving the original
integer function. `trainer.residual_refeval` independently reads NGNN3/6 and
evaluates the specified arithmetic in NumPy int64; it also supports packed
NGNP1 records and explicit nonzero residual arrays.

`experiments/wave2-1104/residual-convert/` contains reproducible H32 zero and
nonzero models, 32 packed records, cross-language parity cases, captured test
output, and the accepted H768 conversion/search receipt. The executable
`run.py h768` writes the large converted model only to external scratch and
requires exact equality of four depth-four searches. A later nonzero trained
model must pass the same independent oracle; zero-lift identity alone makes
no loss or strength claim and does not authorize changing the embedded net.

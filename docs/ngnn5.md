# NGNN5 sparse threat inputs

NGNN5 is opt-in through EvalFile. The embedded network and default play remain
NGNN3. It retains NGNN3's KB8 piece-square inputs, hidden layer, quantization,
material output buckets, and exact integer output arithmetic.

Each perspective adds directed occupied-square attacks by its own pieces,
including defenses. A tuple is `(attacker type, attacker square, target type
and relative color, target square)`. Attackers have six types; targets have
twelve (friendly pawn through king, then enemy pawn through king). Pins do not
remove attacks. Sliders stop at the first occupied square. Pawns attack forward
diagonally, never by pushes. En passant rights alone create no feature.

Apply the existing perspective rank flip and king-context horizontal mirror
to both squares. Within each king bucket, enumerate attacker types pawn through
king, source A1 through H8, target types as above, then target A1 through H8.
Remove geometrically unreachable pairs, pawn sources/targets on ranks 1 or 8,
and king-to-king pairs (impossible on a legal board). Remaining tuples receive
consecutive row indices. Bucket `k` adds `k * F` to that index, where F is the
enumerated count (42,564). Piece-square row indices remain unchanged in a separate block.
Threat rows contain signed int8 weights and add to the same int32 hidden
accumulators as the signed int16 piece-square rows. Missing rows are all zero.
Even 64 occupied squares with maximal attacks fit safely in int32.

All integers below are little endian. The 112-byte header is:

| Offset | Value |
| --- | --- |
| 0 | ASCII `NGN5` |
| 4 | uint32 version 5 |
| 8 | uint32 hidden width H, 16..2048, divisible by 16 |
| 12 | uint32 king buckets KB, 1..8 (KB8 for production) |
| 16 | uint32 output buckets NB, 1/2/4/8 |
| 20,24,28 | uint32 QA=255, QB=64, scale=400 |
| 32 | 64 uint8 king-bucket map entries, each less than KB |
| 96 | uint32 F, exact enumerated threat count per king bucket |
| 100 | uint32 number of stored nonzero threat rows R |
| 104,108 | uint32 sparse encoding 1, reserved 0 |

Payload: `KB*768*H` int16 piece-square weights; R records each containing a
uint32 threat row index and H int8 weights; H int16 hidden biases;
`NB*2*H` int16 output weights; NB int32 output biases; then uint32 IEEE CRC32
over every preceding byte. Records must have strictly increasing indices below
`KB*F` and contain a nonzero weight. There is no padding or trailing data.
Dimensions, constants, map, indices, length and checksum are validated before
publication. Sparse encoding keeps zero-threat conversions close to NGNN3 size.

The converter preserves all NGNN3 values and stores no threat records, giving
exact scores and deterministic search parity without training or default adoption.


Each real move snapshots the small pre-move board and prepares NGNN3 piece
rows in the old context. After make, changed mailbox squares identify attackers
that moved or whose empty-board attack reach intersects a changed square.
Only their old/new occupied attack edges are compared. Changed piece identities
also replace edges when occupancy stays set, as on captures and promotions.
This includes distant slider discoveries, both castling pieces and the en
passant victim square. A king-context crossing refreshes only its perspective.
The other perspective receives exact deltas. Each ply owns its accumulator;
unmake restores the parent frame. Null plies alias the unchanged parent, and
real children write separate frames. NGNN5 uses int32 accumulators throughout.

Validation uses nonzero signed int8 threat rows and extreme int16 piece rows,
20,000 deterministic random positions, guaranteed special-move seeds, full
refreshes and an independent mailbox ray oracle after every make/unmake. The
walk asserts NGNN5 format identity and Threads=1. The previous NGNN3 walk is
retained. `testdata/ngnn5/positions.json` fixes the 20-score and 12-search panels.

```sh
go run ./cmd/ngnn5-convert -in engine/nets/ngn-h512-81215229.nnue -out converted.nnue
go test -p 2 -short ./engine -run '^TestNGNN5' -count=1 -v
NGN_NGNN5_PARITY=1 go test -p 2 -short ./engine -run '^TestNGNN5SearchParity$' -count=1 -v
```

The optional parity check runs 36 depth-10 pairs: fixed, cold iterative and
warm iterative on each of twelve positions. Nodes, score, best move and PV must
match exactly. Setting `NGN_NGNN5_REPORT_DIR` writes score, search timing and
active-feature JSON to the chosen output directory. The feature report samples
both perspectives at every exact search node-counter increment in fixed depth-4
searches, including parent counts after undo; weights need not be nonzero for
an input to count. Shared-host search timings are diagnostic measurements;
`scripts/speedcheck.py --duration 10 --repeats 4` remains a separate quiet-host
measurement. No strength claim, training, or embedded network change is implied.

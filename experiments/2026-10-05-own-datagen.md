# NGN-owned datagen — 2026-10-05

Implementation: `own/datagen`, based on independent HCE commit
`74d63ee`. New generator/reader/driver code is written from first principles;
positions, search labels, and outcomes come only from NGN. No historical
`output/` data, snapshots, opponent labels, networks, or third-party training
tools are inputs. This experiment measures throughput, not playing strength.

## Format

NGNP1 is headerless, little-endian, exactly 32 bytes per record. Offsets:

| Bytes | Value |
|---|---|
| 0–7 | Occupancy; a1 is bit 0 |
| 8–23 | Piece nibbles in ascending occupied-square order, low nibble first; white P/N/B/R/Q/K = 0–5, black = 6–11; unused nibbles zero |
| 24–25 | Signed int16 White-POV search score in cp |
| 26 | White result: loss 0, draw 1, win 2 |
| 27 | STM: white 0, black 1 |
| 28–29 | uint16 game ply before the move |
| 30 | Flags: check/capture/promotion/clamped-or-mate at bits 0/1/2/3 |
| 31 | Reserved zero |

The fixed format omits castling, en passant, halfmove clocks and repetition
history. Decode-to-FEN preserves board/stm (first two FEN fields); remaining
fields are neutral. Sidecars identify format, committed source and executable
SHA-256, complete settings and seed, counts, game plies/results, filtering,
terminations, search work, timestamps, and final data SHA-256. Names are
exclusively reserved; an initial sidecar is explicitly incomplete.

## Decisions

- One receiver, HCE explicitly selected, one search worker and GOMAXPROCS=1.
  `NewGame` clears hash/history before each opening attempt. Hash and history
  then persist within the game, including the 1,000-node opening verification.
- `-games 0` runs until SIGINT/SIGTERM. Positive `-games` counts completed
  accepted games. Default openings choose 8 or 9 independent random legal plies;
  `-random-plies -1` selects this behavior. Verification rejects scores strictly
  outside ±400 cp, terminal openings, or searches without a completed iteration.
- Every move search uses the existing exact NGN node cap; labels and moves come
  from its last complete iteration. Proven mates can terminate early. No
  completed iteration means drop the whole game, counted as unscored.
- Store quiet, non-check, non-promotion, non-mate positions before moves only.
  Convert STM search scores to White POV before filtering/adjudication. All
  retained positions receive the final game result together. Filter counters
  overlap and include unfinished/dropped games; position counts include only
  committed games.
- Six consecutive scores ≥2,000 or ≤−2,000 cp adjudicate wins/losses. The entire
  12-ply ±8 cp draw streak must start at or after ply 80. Apply the searched move
  before score adjudication; checkmate/stalemate, rule draws and maxply take
  priority. Game lengths include random opening plies. Insufficient material
  includes promoted bishops when all surviving bishops share a square color.
- Signals drop the current game at the next bounded-search boundary, sync and
  close the shard, then finish the checksum sidecar. A failed game append rolls
  back only its unwritten/incomplete tail. SIGKILL/power-loss recovery is outside
  this orderly-close guarantee; incomplete sidecars fail validation.
- `datastat` validates every record and, by default, closed sidecar counts and
  checksums. It checks ≤32 pieces, ≤16/color, pawn and promotion budgets, exactly
  one king/color, no pawns on end ranks, check-flag consistency and that the side
  not to move is not checked. These are necessary legality tests, not a proof of
  historical reachability. Histograms use 100-cp and 20-ply bins.
- Duplicate identity is board/stm only; labels and unstored move rights are
  excluded. A bounded SHA-256-derived 64-bit hash sample retains ≤100,000 distinct
  keys and their occurrence counts. Below the bound it counts all positions;
  above it, it progressively halves the deterministic sample. Hash collisions
  and sampling error are not measured. Driver reruns use fresh run directories.

## WSL commands and validation

All builds, tests, engine execution and statistics run only on WSL, using
`/usr/local/go/bin/go` 1.25.5 and `GOAMD64=v3`. The source worktree is
`/home/ehrli/wt/datagen`; the primary checkout is never checked out or edited.
Sync uses commits and `origin/own/datagen`, not copied source files.

```bash
git -C /home/ehrli/repos/ngn fetch origin
git -C /home/ehrli/repos/ngn worktree add --detach /home/ehrli/wt/datagen origin/own/datagen
# For an existing clean worktree: git -C /home/ehrli/wt/datagen checkout --detach origin/own/datagen
cd /home/ehrli/wt/datagen
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3
go test ./internal/ngnp ./internal/datagen ./cmd/datagen ./cmd/datastat -count=1
go test -short ./... -count=1
go test -short -race ./internal/ngnp ./internal/datagen ./cmd/datagen ./cmd/datastat -count=1
mkdir -p build
go build -trimpath -ldflags "-X main.engineCommit=$(git rev-parse HEAD)" -o build/datagen ./cmd/datagen
go build -trimpath -o build/datastat ./cmd/datastat
scripts/datagen_run.sh -out /home/ehrli/ngn-data/pilot -workers 14 -seed 2026100500 -nodes 5000
# After 600 wall seconds, SIGTERM only this driver; it forwards to its children.
build/datastat /home/ehrli/ngn-data/pilot/RUN_DIRECTORY
```

Validated source: `645c45fb971e9e465b1a0496a98229193c624979`. All three
specified test commands passed on WSL with Go 1.25.5, `GOAMD64=v3`, and
`GOMAXPROCS=2` for the test runner. The narrow run includes the non-short
end-to-end test (two repeated two-game runs, identical bytes for identical
seed/settings). The complete short suite passed, including `engine` (8.009s).
The touched packages passed the short race check. Both CLI builds succeeded
from a clean committed worktree.

Additional executable checks passed: two driver invocations with identical
seeds produced four closed shards in two distinct reserved directories; all
records and checksums validated. SIGINT after 0.5s exited successfully with a
closed `interrupted` sidecar and a validated checksum. Logs and receipts are
under `/home/ehrli/ngn-data/datagen-validation-bZa501/`.

Datagen binary SHA-256:
`5964e4e3b39b5e39c5c14551165d749b3f154d58e28e05904fb4fc1b83d3d72c`.
Datastat binary SHA-256:
`eed3110d43c7e75dabdf1e8f53d1402a5e9b7212d9a983d1b93dbee31b4c11e0`.

## Pilot

The pilot uses the command above, fourteen distinct seeds starting at
2026100500, default 8/9 opening plies, 1,000 verification nodes, 5,000 nodes per
move, maxply 400, and 16 MB hash per worker. The harness uses a native
600-second subprocess timeout, sends SIGTERM to this driver alone, and waits
for its children to close. A 120 MiB inherited per-file limit bounds each
shard; no unrelated process is signaled. Started at 2026-10-05 22:40:36 UTC;
SIGTERM was sent at 600.001089 seconds, and the driver/children exited with
status zero at 600.064607 seconds. The throughput denominator includes orderly
shutdown. All fourteen worker PIDs were verified gone afterward.

Data remains at
`/home/ehrli/ngn-data/pilot/run-20261005T224036Z-woQn6X/`.
The validation directory contains `pilot-receipt.json`, `pilot-driver.log`, and
`pilot-stat.json` (full score, piece-count, and ply histograms). The latter was
produced with:

```bash
build/datastat -wall-seconds 600.064607061 /home/ehrli/ngn-data/pilot/run-20261005T224036Z-woQn6X
```

| Measurement | Result |
|---|---:|
| Validated shards / sidecars | 14 / 14 |
| Completed positions / games | 1,328,267 / 13,636 |
| Positions/hour / games/hour | 7,968,743.94 / 81,807.19 |
| Mean total game length | 145.426 plies (72.713 moves) |
| Data bytes / bytes per position | 42,504,544 / 32 |
| Total pilot bytes including logs/sidecars | 42,539,094 |
| White game wins / draws / losses | 5,041 (36.97%) / 3,465 (25.41%) / 5,130 (37.62%) |
| White record wins / draws / losses | 460,805 (34.69%) / 392,693 (29.56%) / 474,769 (35.74%) |
| Duplicate board/stm estimate | 2.5484%; 82,993 sampled records, 80,878 unique, 1/16 sample |
| Mean score / recorded ply | −6.638 cp / 87.274 |
| Opening attempts / rejections | 14,755 / 1,105 |
| Searches / total search nodes | 1,883,020 / 9,249,129,102 |

No invalid records, checksum failures, unscored searches, node-cap/thread
contract violations, or retained noisy records were observed. Exactly fourteen
unfinished games were dropped on shutdown (one per worker). 30,076 searches
(1.60%) ended below the node cap; the engine permits early search termination.
Worker position totals ranged from 92,195 to 96,940.

Game endings: score win/loss 8,290; checkmate 1,881; repetition 1,402;
insufficient material 745; fifty-move 656; score draw 621; stalemate 23;
maxply 18. Thus 0.132% of games needed the maxply fallback. Filtering counters
(overlapping and including dropped games): captures 329,973; in-check 195,089;
mate scores 43,242; promotions 10,346. Full histograms are in the statistics
receipt; retained piece counts span 3–32 and recorded plies span the opening
through the 380–399 bin. Training quality/Elo is not established by this run.

The pilot occupied about 41 MiB, with no large generation run started.
C: had about 14 GiB available afterward. The measured binaries/data identify
source `645c45fb971e9e465b1a0496a98229193c624979`; the subsequent experiment-note
commit changes documentation only and does not alter that provenance.

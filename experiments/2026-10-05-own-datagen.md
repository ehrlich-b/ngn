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

Validation and pilot results will be added after the committed implementation
is synced to the dedicated WSL worktree.

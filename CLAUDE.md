# CLAUDE.md

Development guidelines for Claude Code working on this repository.

## Commands

```bash
make build          # Build the engine
make test           # Run tests with coverage
make test-short     # Run tests without coverage (faster)
make eval           # 2-minute comprehensive evaluation (use sparingly)
make format         # Format Go code
make lint           # Run linter
make dev            # Full cycle: format, test, build
```

## ELO Assessment

Default: Bayesian early-stop with 2500ms/move on macOS E-cores (`taskpolicy -b`).
At 200ms we only reached depth 4 in middlegame; at 2500ms we reach depth 6-7,
which is meaningful for tactical accuracy. Mac stays cool on E-cores.

```bash
# Standard cadence (~15-30 min, cool, depth 6-7 reached):
./build/elo-assess -elo 1500 -movetime 2500 -min 6 -games 14 -ci 300

# Quick smoke test (~3-5 min, depth 4 only):
./build/elo-assess -elo 1320 -movetime 200 -min 6 -games 14 -ci 300

# Legacy presets:
./build/elo-assess -level 1   # 8 games @ 250ms
./build/elo-assess -level 5   # 20 games @ 400ms
./build/elo-assess -level 10  # 40 games @ 600ms
```

## Development Rules

### Testing Cadence
- **Default to 14-game Bayesian tests** at 2500ms on E-cores. Takes 15-30 min. Detects ±80 ELO.
- **Don't chase small effects** (< 50 ELO). 14-game CI is ~300-400 ELO wide — such deltas are pure noise. Resolving them needs 150+ games of direct A/B + SPRT.
- **Iteration loop:** surgical fix → `go build` → 14-game test → commit if visibly better or within noise; revert if visibly worse.
- **Commit checkpoints** after every confirmed win. Regressions bisect via `git stash` on one file at a time (search.go vs eval.go).
- **Never run on P-cores** (default `-lowpower=true` routes to E-cores). Don't override.
- **Avoid long fixed-game runs** unless explicitly asked. Prefer Bayesian early-stop.

### Always Update TODO.md
After ANY task, update TODO.md to reflect current status.

### Fix Bugs Before Features
Fix search/eval correctness issues before adding new features. See DIAGNOSIS.md for the current bug list.

### Verify Claims
Don't trust ELO/NPS claims in documentation without verification.

## Current Status

- **NPS:** ~350K (verified)
- **ELO:** ~1400-1500 (65% vs Stockfish 1320)
- **Recent Fix:** Threefold repetition detection - games now complete

## Project Structure

```
engine/
  search.go     - Search algorithm (alpha-beta, pruning, extensions)
  eval.go       - Position evaluation (material, PST, pawn structure)
  position.go   - Game state management
  bitboard.go   - Bitboard representation
  uci.go        - UCI protocol
  time.go       - Time management
cmd/
  elo-assess/   - ELO assessment tool
  gametest/     - Engine vs engine testing
```

## What's Working

- Search: All major techniques implemented
- Move generation: Perft validated
- UCI: Full protocol support
- Opening book: Embedded + Polyglot
- Tablebases: Syzygy support

## What's Working

- Repetition detection: Fixed - games complete properly
- All search techniques verified working

## Evaluation (Complete)

All classical evaluation features implemented:
- Material + piece-square tables
- Pawn structure (doubled, isolated, passed, chains)
- Bishop pair, mobility, king safety
- Knight outposts, piece coordination
- Rook on open file, tapered eval (MG/EG)

## GM Without NNUE

Classical engines CAN reach 2500+ ELO. Examples:
- Crafty (~2600)
- Fruit (~2700)
- Pre-NNUE Stockfish (~3000)

Focus on getting classical eval right before considering NNUE.

## Reanchor Command

When user says "reanchor", re-read README.md, TODO.md, and CLAUDE.md.

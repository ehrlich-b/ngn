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

```bash
./build/elo-assess -level 1   # Quick: 1 min, 4 games
./build/elo-assess -level 5   # Standard: 5 min, 10 games
./build/elo-assess -level 10  # Thorough: 10 min, 20 games
```

## Development Rules

### Always Update TODO.md
After ANY task, update TODO.md to reflect current status.

### Test After Every Change
Run `make test` after each modification.

### Fix Bugs Before Features
Current critical bug (repetition detection) blocks all meaningful progress.

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

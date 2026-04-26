# ngn - Chess Engine

A chess engine written in Go, targeting Grandmaster level (2500+ ELO) with classical evaluation (no NNUE).

## Current Status

**ELO:** ~1600-1650 (93% vs Stockfish 1320, 68% vs Stockfish 1500 at 200ms)

## Verified Performance

| Metric | Value |
|--------|-------|
| NPS | ~350K |
| Depth in 5s | 10 |

## Features

### Search (Working)
- Iterative deepening with aspiration windows
- Alpha-beta with transposition table
- Null move, futility, LMR, LMP pruning
- SEE for move ordering and pruning
- History, killer, counter move heuristics
- Check, singular, recapture, passed pawn extensions
- Probcut, IID

### Evaluation (Partial)
- Material + piece-square tables
- Pawn structure (doubled, isolated, passed, chains)
- Opening book (embedded + Polyglot)
- Syzygy tablebase support

### Evaluation (Complete)
All classical evaluation features implemented:
- Material, piece-square tables, pawn structure
- Bishop pair, mobility, king safety, knight outposts
- Rook on open file, tapered eval (MG/EG interpolation)

## Quick Start

```bash
make build    # Build the engine
make test     # Run tests
```

## ELO Assessment

Default cadence is `make smoke`: tactical suite + 5-game asymmetric match,
records to `output/smoke_history.jsonl`, prints a rolling ELO estimate that
tightens with every run.

```bash
make smoke              # ~5 min, battery-friendly (E-cores)
make smoke-tactical     # ~5 sec, tactical only
./build/smoke -rolling  # Print rolling estimate without running
```

Full Bayesian test (only when on AC power):

```bash
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

## Development

See `TODO.md` for roadmap and `CLAUDE.md` for development guidelines.

## Can Classical Engines Reach GM Level?

**Yes.** Crafty (~2600), Fruit (~2700), and pre-NNUE Stockfish (~3000) all used classical evaluation. NNUE adds ~300-400 ELO but is not required for GM strength.

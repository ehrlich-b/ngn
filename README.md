# ngn - Chess Engine

A chess engine written in Go, targeting Grandmaster level (2500+ ELO) with classical evaluation (no NNUE).

## Current Status

**ELO:** ~2380 CCRL Blitz — measured vs published-CCRL Blunder anchors (`make gauntlet`); Blunder 7.4.0 (CCRL 2532) is the working anchor and is ~+150 ahead. The Stockfish `UCI_Elo` slider reads ~+400 inflated, so disregard old slider-based "1600"/"2500" figures.

## Verified Performance

NPS is phase- and core-dependent — it is **not** a scalar; always state the position and core type (the engine runs on E-cores, `taskpolicy -b`):

| Metric | Value |
|--------|-------|
| NPS (E-core, middlegame) | ~307K |
| NPS (E-core, endgame) | ~540K–860K |
| NPS (P-core) | ~1M |

vs Blunder 7.4.0 (same box, both Go, both E-cores): ~2x slower **per node** everywhere, plus a **2.5–5x larger search tree in endgames** (a pruning-efficiency gap, not eval). Canonical breakdown: `docs/10-performance.md`; reproduce with `scripts/h2h.sh`.

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

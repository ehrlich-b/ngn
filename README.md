# ngn - Chess Engine

A chess engine written in Go, targeting Grandmaster level (2500+ ELO) with classical evaluation (no NNUE).

## Current Status

**ELO:** ~2540 CCRL Blitz (soft) — measured vs published-CCRL Blunder anchors (`make gauntlet`): Blunder 7.2.0→2549 and 7.4.0→2538 agree, so NGN is ~even with Blunder 7.4.0 (CCRL 2532), at/near the GM line. Soft pending a 100/anchor rerun (the prior samples were forfeit-capped). The Stockfish `UCI_Elo` slider reads ~+400 inflated, so disregard old slider-based "1600"/"2500" figures.

## Verified Performance

NPS is phase- and core-dependent — it is **not** a scalar; always state the position and core type (the engine runs on E-cores, `taskpolicy -b`):

| Metric | Value |
|--------|-------|
| NPS (E-core, middlegame) | ~307K |
| NPS (E-core, endgame) | ~540K–860K |
| NPS (P-core) | ~1M |

vs Blunder 7.4.0 (same box, both Go, both E-cores): **~1.5x slower per node** (engineering, recoverable — Blunder is also Go); the search tree is **~1x in middlegames and ~0.85x in quiet endgames** (NGN often *smaller*). The old "2.5–5x larger endgame tree / pruning collapse" claim was REFUTED (2026-06-06): thermal-NPS noise plus one mislabeled position (h2h's "quiet-end" is tactically won). Reproduce with `scripts/h2h.sh` — but disregard its "quiet-end" row.

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

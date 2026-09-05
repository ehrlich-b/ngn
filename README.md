# ngn - Chess Engine

A chess engine written in Go, targeting at least 2800 with classical evaluation (no NNUE).

## Current Status

**Last measured strength: 2726, reported 95% CI [2668, 2784]**, from a 200-game,
five-anchor real-clock gauntlet on September 4, 2026, using the repaired release.
This is a local estimate calibrated against historical CCRL anchors, not an
official rating. It clears the original 2600 gate; 2800 remains unconfirmed.
See the [run record and limitations](experiments/2026-09-04-strength-check.md)
and [current work state](TODO.md).

The internal-castling correction is deployed after a 400-game correctness gate
(+2.6 [−20,+25], zero flags/errors; no strength-gain claim). The
[six-angle review](experiments/2026-09-04-classical-review-prompt.md) and
[follow-up evidence](experiments/2026-09-04-review-second-pass.md) identify the
next qsearch/TT repair and the remaining training-data limitations.


## Historical Performance Measurements

The figures below describe earlier Mac measurements. Current repair timings
are recorded in the [September validation](experiments/2026-09-04-fast-correctness.md).

NPS is phase- and core-dependent — it is **not** a scalar; always state the position and core type (the engine runs on E-cores, `taskpolicy -b`):

| Metric | Value |
|--------|-------|
| NPS (E-core, middlegame) | ~307K |
| NPS (E-core, endgame) | ~540K–860K |
| NPS (P-core) | ~1M |

vs Blunder 7.4.0 (same box, both Go, both E-cores): **~1.5x slower per node** (engineering, recoverable — Blunder is also Go); the search tree is **~1x in middlegames and ~0.85x in quiet endgames** (NGN often *smaller*). The old "2.5–5x larger endgame tree / pruning collapse" claim was REFUTED (2026-06-06): thermal-NPS noise plus one mislabeled position (h2h's "quiet-end" is tactically won). Reproduce with `scripts/h2h.sh` — but disregard its "quiet-end" row.

## Features

### Search
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

### Current handcrafted evaluation
Current handcrafted terms:
- Material, piece-square tables, pawn structure
- Mobility, king safety, knight outposts
- Rook on open file, tapered eval (MG/EG interpolation)

## Quick Start

```bash
make build    # Build the engine
make test     # Run tests
```

## Strength Assessment

Use paired games at real tournament clocks for changes to search, evaluation,
or time management. A move-generation test, node count, NPS measurement, or
short smoke match does not establish Elo. Same-engine preflights check the
match setup before comparing different binaries.

Absolute strength is measured against the pinned five-opponent ladder. See the
[120+1 gauntlet manifest](experiments/2026-09-04-strength-check.md) for the exact
command, opponent labels, checksums, exclusions, and confidence-interval rule.
The older asymmetric Stockfish-slider and rolling smoke estimates are legacy
instruments and should not be used to claim a rating or accept an engine change.

For a quick tactical check:

```bash
make smoke-tactical
```

## Development

See `TODO.md` for roadmap and `CLAUDE.md` for development guidelines.

## Can Classical Engines Reach GM Level?

**Yes.** CounterGo and Chess-3 provide concrete classical Go implementations to
study. See the [pinned source comparison](experiments/2026-09-04-classical-review-findings.md).
Their results do not predict an automatic gain for NGN; this campaign targets
2800 without NNUE.

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

**Default loop is `make smoke`** (~3-5 min on E-cores, battery-friendly). It
runs a 30-position tactical suite plus a 5-game asymmetric match (NGN 1s vs
SF1500 200ms) and appends results to a rolling-ELO history at
`output/smoke_history.jsonl`. The cumulative Wilson CI tightens with every run.

Older runs decay by commit distance from HEAD: full weight at distance 0, then
−10%/commit through commit 5 (weight 0.5), then −5%/commit through commit 15
(weight 0). After 15 commits without a smoke, that data point falls off
entirely and confidence opens up again. Run smoke regularly to keep effective
sample size high.

The rolling-ELO at this config (~1430 baseline) is a **regression detector,
not the engine's absolute strength**. The 1s/200ms ratio is much tighter than
the 2.5s/200ms baseline used for "true" assessment; NGN looks weaker here on
purpose so we get faster feedback. What you watch for is the rolling estimate
not falling much below the running baseline.

```bash
make smoke              # Tactical + 5 games + rolling estimate (~5 min)
make smoke-tactical     # Tactical only (~5 sec) — for in-flight changes
./build/smoke -rolling  # Just print the rolling estimate, no run
```

Only fall back to the full `elo-assess` Bayesian test when on AC power and you
need a tight one-shot CI for publication or before a major decision:

```bash
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

Legacy presets exist (`-level 1/5/10`) but are not the default cadence.

## Development Rules

### Testing Cadence
- **Default to `make smoke`** after every change. ~5 min, runs in-place on
  E-cores, records a rolling-ELO data point.
- **Smoke fails** on tactical solve rate < 40% or 0/5 game outcome (catches
  catastrophic regressions). Mid outcomes are NEUTRAL and you continue.
- **Rolling estimate is the source of truth for current ELO**, not any single
  smoke. Once you have 6-10 runs at the default config the cumulative CI is
  tighter than a single 14-game Bayesian.
- **Don't chase small effects** (< 50 ELO). Even with the rolling estimate,
  resolving 30-50 ELO deltas needs many runs. Big wins should appear within a
  few smoke runs; if it doesn't show up across 4-5 smokes, it's noise.
- **Commit checkpoints** after every smoke that doesn't fail.
- **Never run on P-cores** (default `-lowpower=true` routes to E-cores).

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

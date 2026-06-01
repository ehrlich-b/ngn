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

Two distinct jobs, two cadences:

- **A — relative regression, per change (fast):** `make smoke` /
  `make smoke-tactical` + self-play `sprt`. Tells you if a change broke or helped;
  NOT an absolute rating.
- **B — absolute CCRL calibration, occasional (slow):** `make gauntlet` plays NGN
  against published-CCRL-rated Blunder anchors and pools a real rating. This is the
  **single source of truth for NGN's absolute strength.** Stockfish's UCI_Elo
  slider is NOT — we measured it ~+400 inflated vs CCRL (NGN scored 70% vs "SF2400"
  but ~7% vs Blunder 7.4.0, CCRL 2532).

**Default dev loop is `make smoke`** (~3-5 min on E-cores, battery-friendly). It
runs a 30-position tactical suite plus a 5-game asymmetric match (NGN 1s vs
SF2400 200ms) and appends results to a rolling-ELO history at
`output/smoke_history.jsonl`. The cumulative Wilson CI tightens with every run.

Older runs decay by commit distance from HEAD: full weight at distance 0, then
−10%/commit through commit 5 (weight 0.5), then −5%/commit through commit 15
(weight 0). After 15 commits without a smoke, that data point falls off
entirely and confidence opens up again. Run smoke regularly to keep effective
sample size high.

The rolling-ELO at this config is a **regression detector, not the engine's
absolute strength**. The opponent was raised SF1500 → SF2400 on 2026-05-29
because the engine had saturated the weaker targets: NGN scored 100% vs SF1500
and 13-1 (93%) vs SF2000, so neither could discriminate. At SF2400 NGN sits at
~71% (9W-2D-3L over the calibration match) — a clear favorite, but SF now wins
real games and draws appear, so drops are visible. The estimated 50% crossover
at this 1s/200ms time config is ~SF2560; raise the target again if NGN climbs
back toward saturation. What you watch for is the rolling estimate not falling
much below the running baseline. For NGN's ABSOLUTE rating use `make gauntlet`
(Cadence B) — this SF rolling number is relative only.

```bash
make smoke              # Tactical + 5 games + rolling estimate (~5 min)
make smoke-tactical     # Tactical only (~5 sec) — for in-flight changes
./build/smoke -rolling  # Just print the rolling estimate, no run
```

**Cadence B — absolute CCRL calibration (the real number):**

```bash
make build-anchors                               # one-time: clone+build the Blunder ladder
make gauntlet ARGS="-games 24 -movetime 1000"    # ~real CCRL number (~3-4h, E-cores)
make gauntlet ARGS="-games 60 -movetime 2500"    # tighter, closer to CCRL Blitz (overnight)
./build/gauntlet -rolling                        # commit-decayed pooled trend, no run
./build/gauntlet -only v7.4.0 -ngn opponents/blunder_740   # calibration sanity (~50%/~2532)
```

The gauntlet number is a movetime proxy for CCRL Blitz (2'+1"): it under-times the
anchors, biasing NGN ~+30-80 optimistic. A true CCRL-conditions run needs NGN to
play a real clock — blocked by the tournament time-management bug (top TODO);
finishing that work lets you drop the bias caveat.

`elo-assess` is now ad-hoc Stockfish sparring (relative to the SF slider, NOT
absolute — use the gauntlet for absolute):

```bash
./build/elo-assess -elo 2600 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

Legacy presets exist (`-level 1/5/10`) but are not the default cadence.

## Development Rules

### Testing Cadence
- **Default to `make smoke`** after every change. ~5 min, runs in-place on
  E-cores, records a rolling-ELO data point.
- **Smoke fails** on tactical solve rate < 40% — the deterministic
  catastrophic-regression gate. The game phase is now a *measurement*, not a
  pass/fail: vs SF2400 a 0/5 shutout is within variance (it prints a SHUTOUT
  note but does not fail), so watch the rolling estimate for real drift.
- **The smoke rolling estimate is a RELATIVE regression tripwire** (vs the SF
  slider), NOT NGN's absolute ELO — `make gauntlet` (Cadence B) owns the absolute
  CCRL number. Once you have 6-10 smokes at the default config its cumulative CI is
  tighter than a single 14-game Bayesian, which is what makes it a good tripwire.
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

- **ELO:** ~2380 CCRL Blitz (`make gauntlet` vs Blunder anchors; the SF `UCI_Elo` slider reads ~+400 inflated — disregard old slider numbers). Blunder 7.4.0 (CCRL 2532) is the working anchor, ~+150 ahead.
- **NPS:** phase/core-dependent, NOT a scalar — ~307K E-core middlegame, ~540K-860K E-core endgame, ~1M P-core.
- **Gap vs Blunder (both Go, E-cores):** ~2x slower per node everywhere (engineering tax) AND a **2.5-5x larger search tree in endgames** (a pruning-efficiency collapse — the bigger, more surprising lever; middlegame tree size is ~1x). Node counts are deterministic/exact; nps is thermally noisy on the M4 (~2x). Canonical breakdown: `docs/10-performance.md`; reproduce: `scripts/h2h.sh`.

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

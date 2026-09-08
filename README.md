# NGN

NGN is a UCI chess engine written in Go.

> **Project goal:** build the strongest Go-based chess engine in the world.
>
> NGN is not there yet. The current integrated candidate is provisionally around
> 3100--3150 CCRL-style Elo, while Rodent V 1.1 is listed at 3522 on CCRL 40/4.
> The gap is still several hundred Elo, and only independent rating-list testing
> can settle the comparison.

## Features

### Search

- Iterative deepening negamax with principal-variation search and aspiration windows
- Transposition table with mate-score normalization and synchronized shared access
- Quiescence search with check-aware terminal handling
- Null-move, reverse-futility, futility, late-move, history, SEE and ProbCut pruning
- Late-move reductions, internal iterative deepening and singular extensions
- Check and passed-pawn extensions with a bounded extension budget
- Main, follow-up, capture and correction histories; killer and counter-move ordering
- Static exchange evaluation and staged move picking
- Repetition, fifty-move, mate-distance and terminal-position handling
- Clock, increment, move-time, node-limit and stop support

The search has accumulated years of small, measured changes, but it still has
known technical debt. The design notes track the exact invariants and open joints
instead of treating the usual engine techniques as automatically correct in this
implementation.

### Real multicore search

- Lazy SMP with `Threads` configurable from 1 to 64
- Persistent worker pool with worker-private positions, histories and evaluator state
- Shared transposition table with explicit search-session ownership and joined shutdown
- Unified cancellation, time control and deterministic primary-worker result publication
- HCE, NGN-v1 and Counter 5.5 evaluators supported across workers

On the WSL reference machine, eight HCE workers produced 6.41x--6.69x the
aggregate nodes per second of one worker in the accepted fixed-time matrix. In a
separate paired strength test, HCE8 beat HCE1 63-3-36. These results demonstrate
useful multicore search; they do not imply linear Elo scaling. Pure fixed-node
searches deliberately use one worker so their node budgets remain deterministic.

### Pluggable evaluation

NGN can select an evaluator at process startup or through UCI while idle:

- **`hce`** -- the built-in tapered handcrafted evaluator. It covers material,
  piece-square tables, pawn structure, mobility, king safety, outposts, rook-file
  activity, threats and endgame draw scaling. It needs no external model.
- **`ngn-v1`** -- NGN's compact Chess768-to-128 network format with squared
  clipped-ReLU output, checked serialization, incremental accumulators, an exact
  portable integer path, plus a bounded-output AVX2 kernel in amd64-v3 builds.
- **`counter-5.5`** -- a compatibility backend for the proven Counter 5.5 network
  format, with worker-private incremental state, search-score adaptation and
  portable feature updates accelerated with AVX2 in amd64-v3 builds.

Model identity participates in worker, history and transposition-table lifecycle,
so switching models cannot silently reuse state produced by a different evaluator.

The repository also contains independently checked compatibility packages for
Stockfish 18 SMALL and BIG network files. SMALL has scalar and incremental
reference evaluation; BIG currently has strict model loading. These packages are
validation and integration foundations. Arbitrary Stockfish `.nnue` files are
not yet selectable as general NGN search backends.

### UCI and engine integration

- Standard UCI command loop for chess GUIs and tournament runners
- Configurable hash, thread count, move overhead and evaluator
- Embedded opening book with an `OwnBook` switch
- Legal UCI move parsing and full principal-variation validation
- Linux and optimized amd64-v3 Windows builds
- Strict startup failure when a requested neural model is missing or incompatible

`SyzygyPath` is exposed for forward compatibility, but the current tablebase
decoder is deliberately disabled and should not be treated as working Syzygy
support. External Polyglot books are also not advertised as supported.

### Testing, tuning and measurement

- Perft, make/unmake, hashing, legality, terminal-state and regression suites
- Race-tested evaluator, worker-pool, transposition-table and UCI lifecycle tests
- Full-refresh versus incremental NNUE parity tests across moves and undos
- Reference-oracle checks for Counter and Stockfish network arithmetic
- Tactical, fixed-node, NPS and profiling tools
- Paired match runners with reversed colors, fixed openings and independent PGN audits
- SPRT, SPSA, Texel, ACPL and multi-opponent gauntlet tooling

## Current strength

The best current number is a **provisional 3130 CCRL-style Elo** for the
integrated Counter-evaluator/Lazy-SMP candidate. A reasonable working range is
about **3000--3220**. This is a local calibration, not a CCRL result, and engine
ratings do not transfer exactly across hardware, time controls, books or thread
counts.

The estimate is supported by several distinct measurements:

| Configuration and opponent | Result | What it says |
|---|---:|---|
| Classical HCE on native Windows against the fixed historical-anchor ladder | 2884, reported 95% interval 2834--2934 | Latest fully audited local calibration of the classical line |
| Counter evaluator, 8 workers vs HCE, 8 workers | 66 wins, 27 draws, 5 losses | Combined NNUE plus SMP candidate is decisively stronger than the HCE configuration |
| Integrated candidate, 1 worker vs Counter 5.5 | 1 win, 25 draws, 74 losses; -323 relative Elo | About 3010 if mapped directly onto Counter's published 3333 anchor |
| Integrated candidate, 8 workers vs Counter 5.5 | 1 win, 49 draws, 50 losses; -186 relative Elo | About 3147 before correcting for the rating-list/thread mismatch |

The completed Counter cross-checks disagree materially by width, which is normal
for this kind of local calibration and is why the project should not claim a
precise official rating. A separate Rodent V 1.1 comparison is still running and
is excluded from the table until terminal. The working estimate will be replaced
by the completed external matrix and eventually an independent rating-list result.

The integrated source is now on `main`; deployment of the packaged candidate is
still gated on the remaining external comparison matrix and release verification.

## Where NGN is going

The target is the top of the Go-engine field, not merely 2800 or 3000. Rodent V,
CounterGo, Chess-3 and Zahak show that Go is not the limiting factor. NGN still
needs substantial work in several areas:

1. Finish external calibration and ship the reproducible NNUE/SMP release.
2. Replace dependence on a compatibility network with a first-rate NGN-owned
   data, training, export and validation pipeline.
3. Complete practical Stockfish-network evaluation where licensing and exact
   architecture compatibility allow it.
4. Improve Lazy SMP playing efficiency at 2, 4 and 8 threads, then attack
   contention, duplication and transposition-table policy with profiles.
5. Close the remaining search-correctness and search-efficiency gaps against
   mature engines, one measured change at a time.
6. Optimize Go hot paths, data layout, allocations and architecture-specific
   kernels without weakening the portable reference implementation.
7. Add production tablebases and the remaining tournament-engine features, then
   submit a tagged release for independent testing.

This project deliberately studies successful engines rather than assuming every
missing feature needs a novel solution. Transplants still require parity tests,
correct lifecycle integration and real games: several standard techniques have
lost strength when copied into NGN without the surrounding assumptions that made
them work elsewhere.

## Build and run

NGN requires Go 1.21.2 or newer.

```bash
git clone https://github.com/ehrlich-b/ngn.git
cd ngn
make build-release
./build/ngn
```

The default executable starts with the built-in handcrafted evaluator. It can be
used directly by any UCI-compatible GUI, or driven from a terminal:

```text
uci
isready
position startpos
go movetime 1000
quit
```

Build both the local Linux binary and an amd64-v3 Windows binary with:

```bash
make build
```

To start with a compatible neural model:

```bash
./build/ngn -eval-backend counter-5.5 -eval-file /path/to/network.nn
./build/ngn -eval-backend ngn-v1 -eval-file /path/to/model.ngn
```

Neural model files are not embedded in the repository. NGN validates the file
before starting and refuses a backend/file mismatch.

## UCI options

| Option | Default | Description |
|---|---:|---|
| `Hash` | `128` | Transposition-table size in MiB |
| `Threads` | `1` | Lazy-SMP worker count, 1--64 |
| `Move Overhead` | `100` | Safety margin in milliseconds |
| `OwnBook` | `true` | Enable the embedded opening book |
| `EvalBackend` | startup selection | `hce`, `ngn-v1` or `counter-5.5` |
| `EvalFile` | empty | Compatible model file for a neural backend |
| `SyzygyPath` | empty | Reserved; probing is not production-ready |

Search parameters are also exposed as UCI spin options for controlled tuning.

## Development

Useful checks and tools include:

```bash
make test-short       # Fast Go test suite
make test             # Short suite with race detector and coverage
make perft            # Move-generation validation
make smoke-tactical   # Fast tactical smoke test
make benchmark        # Deterministic fixed-position benchmark
make gauntlet         # Historical-anchor strength calibration
```

Changes that can affect play are judged with paired games at real clocks. NPS,
node count, tactical suites, training loss and a favorable partial score can
diagnose a change, but none alone establishes Elo. The experiment records retain
the exact source, binaries, models, openings, commands, hashes and rejected runs.

Start with the [search design documentation](docs/README.md), the
[NNUE and multicore roadmap](experiments/2026-09-05-next-stage-roadmap.md), and
the [release-confirmation protocol](experiments/2026-09-06-release-confirmation-plan.md).
Historical measurements and rejected ideas live under [`experiments/`](experiments/).

## Prior art

NGN learns heavily from mature chess engines and their published techniques,
especially [CounterGo](https://github.com/ChizhovVadim/CounterGo),
[Stockfish](https://github.com/official-stockfish/Stockfish),
[Rodent V](https://github.com/nescitus/Rodent-V), and
[Chess-3](https://github.com/paulsonkoly/chess-3). Compatibility does not mean
that NGN embeds another engine behind its UCI process: the search, state ownership,
protocol handling and evaluator adapters run inside the Go engine.

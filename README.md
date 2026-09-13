# NGN

NGN is a UCI chess engine written in Go.

> **Project goal:** build the strongest Go-based chess engine in the world.
>
> NGN is not there yet. Completed local comparisons place the current Counter
> candidate roughly around 3000--3100 at one thread when oriented against the
> pinned published ratings. This is not an official rating. At eight threads,
> NGN remains about 186 Elo behind Counter8 in the completed matched test.

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
  feature updates and exact-order output evaluation accelerated with AVX2 in
  amd64-v3 builds, retaining portable fallbacks.

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

The current Counter-evaluator candidate is **roughly 3000--3100 at one thread**
when mapped onto the pinned published opponent ratings. This is an orientation,
not a confidence interval or an official rating. The former 3130 headline mixed
one-thread and eight-thread comparisons and has been retired.

Each external row below is a completed 100-game test over 50 reversed-color
opening pairs at 30+0.3. Intervals resample complete pairs and express local
match uncertainty only; they do not cover transfer between rating lists, hardware,
time controls, opening books, tablebases or opponent personalities.

| Configuration and opponent | NGN wins / draws / losses | Relative Elo, paired 95% interval |
|---|---:|---:|
| Counter evaluator, 1 worker vs Counter 5.5, 1 worker | 1 / 25 / 74 | -323 [-402, -263] |
| Counter evaluator, 1 worker vs Rodent V1.1 Anand/testers | 1 / 15 / 84 | -413 [-512, -338] |
| Counter evaluator, 8 workers vs Counter 5.5, 8 workers | 1 / 49 / 50 | -186 [-230, -143] |

At one thread, Counter's published 3333 label maps to about 3010, while Rodent's
pinned 3522 label maps to about 3109. Counter's 3359 label in the pinned Blitz
snapshot shifts the first estimate to 3036. This sensitivity and the unverified
match between Rodent's testers/Anand artifact and its rated entry prevent a
precise combined estimate. The anchor-specific intervals span roughly 2930--3200;
that envelope is not a combined 95% interval.

At eight threads, **25.5% against Counter8** is the useful baseline. There is no
matched eight-thread absolute anchor in this evidence. Applying Counter's 3333
label would produce 3147, but that assumes a rating transfer we have not verified.
The difference between the one-thread and eight-thread relative scores is also
not NGN's isolated SMP gain: both engines changed width.

The historical classical result remains **2884 [2834,2934]**, from the separately
audited 400-game native-Windows calibration of `53e4d1b`. It is not pooled with
these WSL games. The accepted Counter8/HCE8 result (66 wins, 27 draws, 5 losses)
establishes an internal improvement, not an absolute rating increment.

The [September 12 rating reassessment](experiments/2026-09-12-rating-reassessment.md)
records artifact identities, paired calculations, limitations and the decision
to spend the next games on engine improvements. The failed Rodent V1.2 partial
cell contributes nothing to these estimates.

The subsequent [exact-order output kernel](experiments/2026-09-13-counter-output-kernel.md)
reduced one-thread cold fixed-search time by 27.85% across the declared fixture
aggregate, with identical search trajectories. That speed result does not
establish a new Elo rating or eight-thread gain.

The integrated source is now on `main`; deployment of the packaged candidate is
still gated on the remaining external comparison matrix and release verification.

## Where NGN is going

The target is the top of the Go-engine field, not merely 2800 or 3000. Rodent V,
CounterGo, Chess-3 and Zahak show that Go is not the limiting factor. NGN still
needs substantial work in several areas:

1. Prioritize measured engine speed and high-impact gaps against direct
   competitors, testing changes against the accepted Counter baseline.
2. Use existing networks. NGN-owned NNUE training, training-data generation and
   training-pipeline development are deferred.
3. Evaluate practical support for stronger existing networks, including
   Stockfish formats, where architecture compatibility and integration cost
   justify it.
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

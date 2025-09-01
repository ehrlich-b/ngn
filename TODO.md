# NGN Chess Engine - Performance Critical TODO

## 🚨 CRITICAL PERFORMANCE CRISIS - WE ARE BELOW 100K NPS RED LINE!
**Current Status: 78,176 NPS (FAILING!)**
**Target: >100,000 NPS immediate, >250,000 NPS working target**

## IMMEDIATE HANDOFF INSTRUCTIONS FOR NEXT DEVELOPER

### Work In Progress (Partially Complete)
We were fixing the major performance bottleneck: `getSetBits()` function allocating slices on every call.
- **DONE**: Fixed `evaluatePieceProtection()`, `evaluateOutpostSquares()`, `countPassedPawns()`, pawn chain/shield/storm evaluations
- **TODO**: Still need to fix remaining `getSetBits()` calls in:
  - `evaluatePieceBatteries()` (lines 1410-1412)
  - `evaluateMultiPieceAttacks()` (lines 1475-1478)  
  - `evaluateBishopPair()` (line 1715)
  - Any remaining calls - grep for `getSetBits(`

### Next Critical Performance Fixes (In Order)

1. **Complete getSetBits() Removal** (~20-30K NPS gain expected)
   - Replace ALL remaining `getSetBits()` with bitboard iteration
   - Pattern: `for bb := bitboard; bb != 0; bb &= bb - 1 { square := bits.TrailingZeros64(bb) }`
   - This avoids slice allocations in hot paths

2. **Remove Map Allocations in Eval** (~10-20K NPS gain expected)
   - `evaluatePawnStorm()` uses `make(map[int][]int)` - replace with bitboard operations
   - `evaluatePieceBatteries()` uses maps for fileRooks/rankRooks - use bitboards instead
   - Any `make(map...)` in evaluation is a performance killer

3. **Simplify or Remove Complex Eval Terms** (~20-30K NPS gain expected)
   - Consider removing entire piece coordination evaluation (adds <5 Elo, costs tons of CPU)
   - Simplify king safety to basic pawn shield + open files
   - Remove or simplify outpost evaluation

4. **Profile Again and Fix Next Bottlenecks**
   - Run: `go test ./engine -bench BenchmarkSearchPerformance -benchmem -run ^$ -cpuprofile cpu.out`
   - Analyze: `go tool pprof -text cpu.out | head -20`
   - Look for functions taking >5% CPU time

## Performance Guardrails
- Red line: `TestPrecise5SecondPerformance` ≥ 100,000 NPS at all times.
- Working target: ≥ 250,000 NPS on dev; long-term ≥ 1,000,000 NPS.
- If NPS regresses >5% vs parent commit, revert and profile before merging.
- Hot-path rules: no `map` or growing `[]` in eval/movegen/inner search; prefer bitboards and stack data.
- Quiescence is captures-only by default; add non-capture checks only if net speedup is proven.
- Legality is checked at make/unmake in search; generators remain pseudo-legal.

Quick perf commands
- Precise 5s: `go test ./engine -run TestPrecise5SecondPerformance -v`
- Sanity perf: `go test ./engine -run TestSearchNodesPerSecond -v`
- Microbenches: `go test ./engine -bench . -benchmem -run ^$`
- Profile: `go test ./engine -bench BenchmarkSearchPerformance -benchmem -run ^$ -cpuprofile cpu.out -memprofile mem.out && go tool pprof -top cpu.out`

Perf acceptance budgets
- `BenchmarkEvaluation`: ≤ 3000 ns/op, ≤ 16 allocs/op (stretch: ≤ 1500 ns/op, ≤ 4 allocs/op)
- `BenchmarkMoveGeneration`: ≤ 400 ns/op, ≤ 2 allocs/op
- `BenchmarkLegalMoveGeneration`: ≤ 1500 ns/op, ≤ 4 allocs/op
- `BenchmarkSearchPerformance`: ns/op should trend down; not up

## Performance Testing Commands
```bash
# Quick performance check (5 seconds)
go test ./engine -run TestPrecise5SecondPerformance -v

# Profile CPU usage
go test ./engine -bench BenchmarkSearchPerformance -benchmem -run ^$ -cpuprofile cpu.out
go tool pprof -text cpu.out | head -20

# Check microbenchmarks
go test ./engine -bench . -benchmem -run ^$
```

## Feature Development (ONLY AFTER PERFORMANCE IS FIXED!)

### Critical Missing Features

#### 1. Opening Book Integration **+100-150 Elo**
- Download Polyglot book (e.g., ProDeo.bin)
- Implement Polyglot format reader
- Probe book in UCI position command
- Add book move randomization
- Fall back to search when out of book

#### 2. Endgame Tablebases **+50-100 Elo**
- Integrate Syzygy tablebase probing
- Download 3-4-5 piece tablebases
- Probe at root for exact scores
- Probe in search at low depths
- Handle WDL scores correctly

#### 3. Principal Variation Search **+50 Elo**
- Search first move with full window
- Search remaining moves with null window
- Re-search if null window fails high
- Track principal variation for better move ordering

#### 4. Aspiration Windows **+30 Elo**
- Use previous iteration score as starting point
- Start with ±50 centipawn window
- Widen window on fail high/low
- Fall back to full window if needed

## Development Principles

### Performance First
1. **NO ALLOCATIONS IN HOT PATHS** - No maps, no growing slices, no `getSetBits()`
2. **MEASURE EVERYTHING** - Profile before and after every change
3. **SIMPLE > COMPLEX** - Simple eval + fast search beats complex eval + slow search
4. **100K NPS RED LINE** - Nothing ships if we're below this

## Completed Work Summary

### Performance Fixes Applied Today
- Partially fixed `getSetBits()` allocation issue (improved from 57K to 78K NPS)
- Fixed evaluatePieceProtection, evaluateOutpostSquares, countPassedPawns
- Fixed pawn evaluation functions to use bitboard iteration
- Still need ~22K NPS to reach 100K red line

### Already Working Features
- [x] Transposition Table fully integrated 
- [x] Repetition detection fixed
- [x] Move ordering (MVV-LVA, killers, history)
- [x] Search enhancements (null move, LMR, futility pruning)
- [x] Basic evaluation (material, PST, pawns, king safety)
- [x] UCI protocol working with Arena Chess

## Phase 0: Development Tooling and Infrastructure (COMPLETED)

### 0.1 Build System and Tooling
- [x] Create Makefile with ergonomic commands
  - [x] `make build` - Build the engine binary
  - [x] `make test` - Run all tests with coverage
  - [x] `make lint` - Run linting and formatting checks
  - [x] `make format` - Format all Go code
  - [x] `make clean` - Clean build artifacts
  - [x] `make install-tools` - Install required development tools
- [x] Set up Go linting with golangci-lint
  - [x] Configure .golangci.yml with appropriate rules
  - [x] Enable important linters (govet, errcheck, gofmt, etc.)
  - [ ] Set up pre-commit hooks (optional but recommended)
- Perf gate: add `make perf` target that runs the quick perf tests/benches.

### 0.2 Testing Infrastructure
- [x] Create testing framework for chess engine
  - [x] Test utilities for board setup (FEN parsing/generation scaffolding)
  - [x] Position equality and comparison functions
  - [x] Move list validation helpers (scaffolding)
  - [x] Performance testing utilities (perft framework scaffolding)
- [x] Set up test organization
  - [x] Unit tests for each engine component
  - [x] Integration tests for move generation (scaffolding)
  - [x] Benchmark tests for performance tracking
  - [x] Test data files for known positions
- [x] Create continuous testing approach
  - [x] Table-driven tests for systematic coverage
  - [ ] Property-based testing for move generation
  - [ ] Regression test suite for bug prevention
- Perf gate: perf tests run in CI (non-blocking logs at minimum).

### 0.3 Documentation and Workflow
- [x] Update CLAUDE.md with Makefile commands
- [x] Create development workflow documentation
- [x] Set up code coverage reporting
- [ ] Create debugging and profiling helpers
- Perf note: include profiling cookbook (pprof commands above).

## Phase 1: Core Engine Functionality (Foundation)

### 1.1 Move Generation
- [x] Implement pseudo-legal move generation for all piece types
  - [x] Pawn moves (single, double, captures, en passant)
  - [x] Knight moves using precomputed attack tables
  - [x] Bishop/Rook sliding piece moves with proper bounds checking
  - [x] Queen moves (combination of bishop + rook)
  - [x] King moves (single square + castling)
- [x] Create move generation tests with known positions
- [x] Add perft (performance test) function for move generation validation
- [x] Implement legal move validation (filtering out moves that leave king in check)
- Perf gate: `BenchmarkMoveGeneration` and `BenchmarkLegalMoveGeneration` stay within budgets.

### 1.2 Basic Search Algorithm
- [x] Implement minimax search with alpha-beta pruning
- [x] Add iterative deepening framework
- [x] Implement quiescence search for tactical positions
- [x] Add basic move ordering (captures first, then quiet moves)
- [x] Integrate existing transposition table into search
- Perf gate: quiescence remains captures-only by default.

### 1.3 Basic Evaluation Function
- [x] Material counting with piece values
- [x] Piece-square tables for positional evaluation
- [x] Basic pawn structure evaluation (doubled, isolated, passed pawns)
- [x] King safety evaluation (simple version)
- [x] Mobility evaluation (number of legal moves)
- Perf gate: no `GenerateLegalMoves` inside eval; no maps/slices per node.

## Phase 2: Engine Interface and Testing (Playability)

### 2.1 UCI Protocol Implementation
- [x] Implement UCI (Universal Chess Interface) protocol
  - [x] `uci` command and engine identification
  - [x] `isready` and `readyok` commands
  - [x] `position` command (fen, startpos, moves)
  - [x] `go` command with time controls
  - [x] `stop` and `quit` commands
- [x] Add FEN (Forsyth-Edwards Notation) parsing and generation
- [x] Implement algebraic notation move parsing
- [x] **CRITICAL BUG FIX**: Fixed runtime panic in move generation (index out of bounds in KingAttacks array)
- [x] **ENGINE NOW STABLE**: Successfully completing Arena Chess tournament games
- Perf note: ensure UCI logging can be disabled; no perf impact during search.

### 2.2 Time Management
- [x] Basic time management algorithm
- [x] Support for different time controls (bullet, blitz, rapid, classical)
- [x] Move time allocation based on game phase
- [x] Emergency time handling
- Perf gate: `TimeManager.ShouldStopSearch` calls remain amortized/cheap (rate-limited).

### 2.3 Testing Infrastructure
- [x] Created comprehensive test harness suite for tournament position testing
- [x] Implemented raw UCI protocol logging for debugging
- [x] Integration with chess GUIs (Arena Chess confirmed working)
- [ ] Automated testing against known tactical puzzles
- [ ] Engine vs engine testing framework
- [ ] Benchmark positions for performance regression testing
- Perf note: keep perf baselines and expected ranges documented.

## Phase 2.5: Code Cleanup and Optimization (COMPLETED)
- [x] **DEBUG CLEANUP**: Removed 500K node limit (now 50M)
- [x] **DEBUG CLEANUP**: Fixed ultra-aggressive quiescence limit (2→8 levels)
- [x] **DEBUG CLEANUP**: Removed depth 20 hard limit and ABORT messages
- [x] **DEBUG CLEANUP**: Cleaned up excessive debug logging
- [x] **TIME MANAGEMENT**: Set 15-second maximum time limit for any move
- [x] **RESIGNATION CONTROL**: Added `--allow-resignation` CLI flag (defaults to off)
- [x] **CRITICAL BUG FIX**: Fixed runtime panic in hash.go:208 (index out of range with NoPiece)
- [x] **CRITICAL BUG FIX**: Fixed runtime panic in moveorder.go:87 (negative depth in IsKillerMove)
- [x] **CRITICAL BUG FIX**: Fixed infinite recursion in alphaBeta with negative depth underflow
- [x] **CRITICAL BUG FIX**: Fixed runtime panic in hash.go updateHash function (en passant array bounds)
- [x] **PERFORMANCE ANALYSIS**: Identified mobility evaluation as major bottleneck (see performance_brief.md)
- Perf note: maintain perf notes alongside fixes so future work doesn’t regress.

## Phase 3: Tactical Strength (Expert Level ~2000)

### 3.1 Enhanced Search
- [x] Null move pruning for position evaluation
- [x] Late move reductions (LMR) for search efficiency
- [x] Futility pruning in leaf nodes
- [x] Check extensions in search
- [x] Mate distance pruning
- Perf gate: changes must reduce nodes at equal depth without NPS regression.

### 3.2 Improved Move Ordering (COMPLETED)
- [x] History heuristic for quiet moves
- [x] Killer move heuristic
- [x] Counter-move heuristic
- [x] MVV-LVA (Most Valuable Victim - Least Valuable Attacker) for captures
- Perf gate: verify improved cutoffs; stable allocations.

### 3.3 Enhanced Evaluation
- [x] **CRITICAL PERFORMANCE FIX**: Removed mobility evaluation (achieved 2.4x speedup: 59K→143K nps)
- [x] More sophisticated pawn evaluation
  - [x] Pawn chains and support
  - [x] Weak squares around pawns
  - [x] Pawn storms
- [x] Improved king safety
  - [x] Enhanced pawn shield evaluation
  - [x] Attack patterns on king zone evaluation
- [x] Basic endgame knowledge
  - [x] Opposition in king and pawn endgames
  - [x] Basic mate patterns (KQ vs K, KR vs K)
  - [x] King activity evaluation in endgames
- Perf gate: eval must avoid per-node allocations; use bitboards not maps/slices.

## Phase 4: Positional Understanding (Master Level ~2200)

### ⚠️ IMPORTANT: DO NOT WORK ON THIS UNTIL EMERGENCY PHASE IS COMPLETE

### 4.1 Advanced Evaluation Features (LOW PRIORITY - Worth <50 Elo total)
- [x] **COMPLETED**: Piece coordination and harmony (Note: Adds ~5 Elo, overengineered)
  - [x] Mutual piece protection evaluation
  - [x] Outpost square recognition for knights and bishops
  - [x] Piece battery detection (rook behind queen, doubled rooks)
  - [x] Multi-piece attack coordination
  - [x] Centralization bonus system
- [x] Bishop pair bonus (Note: Worth ~10 Elo, could be a single constant)
- [ ] ~~Rook on open/semi-open files~~ (SKIP until Emergency Phase done)
- [ ] ~~Space evaluation (central control)~~ (SKIP - marginal impact)
- [ ] ~~Tempo and development bonuses~~ (SKIP - marginal impact)
- Perf note: any new eval term must keep `BenchmarkEvaluation` within budget.

### 4.2 Opening Preparation
- [ ] Opening book integration
  - [ ] Polyglot book format support
  - [ ] Book move selection with variety
  - [ ] Out-of-book detection
- [ ] Basic opening principles in evaluation
  - [ ] Development bonuses
  - [ ] Center control
  - [ ] King safety in opening
- Perf note: book lookup should be O(1) per probe; no impact inside search loop.

### 4.3 Endgame Improvements
- [ ] Endgame tablebase integration (3-4-5 piece Syzygy)
- [ ] Endgame-specific evaluation adjustments
- [ ] Recognition of drawn endgames
- [ ] Endgame piece values (activated kings, etc.)
- Perf note: tablebase usage behind low-latency probes and time guard.

## Phase 5: Grandmaster Level Refinement (~2400+)

### 5.1 Advanced Search Techniques
- [ ] Principal variation search (PVS)
- [ ] Aspiration windows
- [ ] Multi-cut pruning
- [ ] Razoring for positions with low evaluation
- [ ] Singular extensions for important moves
- Perf note: validate node reduction vs re-search overhead; keep NPS steady or improved.

### 5.2 Sophisticated Evaluation
- [ ] Passed pawn evaluation with race conditions
- [ ] Advanced king safety with attack weights
- [ ] Piece mobility with square control
- [ ] Weak square identification and exploitation
- [ ] Dynamic vs static evaluation balance
- Perf note: any term adding maps/slices is a non-starter; use bitboards.

### 5.3 Strategic Understanding
- [ ] Pawn structure patterns recognition
- [ ] Piece imbalances evaluation
- [ ] Long-term positional factors
- [ ] Compensation evaluation (material vs positional)
- Perf note: prefer precomputed masks/bitboards; zero allocations.

## Phase 6: Optimization and Polish

### 6.1 Performance Optimization
- [ ] Profile-guided optimization
- [ ] Memory layout optimization
- [ ] SIMD instructions for bitboard operations where beneficial
- [ ] Parallel search implementation (lazy SMP)
- Perf target: push toward ≥ 1M NPS without compromising Elo.

### 6.2 Engine Personality and Tuning
- [ ] Parameter tuning using genetic algorithms or other methods
- [ ] Style parameters (aggressive vs positional)
- [ ] Contempt factor to avoid draws
- [ ] Game phase recognition for evaluation weights
- Perf note: tuning runs should include perf gating to prevent regressions.

### 6.3 Validation and Benchmarking
- [ ] CCRL rating list submission
- [ ] Tournament testing
- [ ] Rating calculation vs known engines
- [ ] Bug fixing and edge case handling
- Perf note: keep a living perf baseline doc with hardware notes.

## Milestones and Targets (REVISED)

### Current Reality Check
- **Actual Current Strength**: ~1600 ELO (due to missing TT, repetition detection, book, tablebases)
- **After Emergency Phase**: ~2100 ELO (+500 from critical fixes)
- **Current Perf Snapshot**: ~60K NPS (needs to clear 100K red line; aim for 250K→1M).

### New Realistic Targets
- **Emergency Phase**: Fix critical bugs → 2100 ELO; hold ≥100K NPS
- **Phase 5 (Search)**: PVS, aspiration, extensions → 2300 ELO; improve nodes/depth without NPS loss
- **Phase 6 (Polish)**: Tuning, parallel search → 2400+ ELO; push toward ≥1M NPS
- **Evaluation improvements**: Maybe +50 ELO total (not priority) and must meet eval budget

## Development Principles (UPDATED BASED ON REALITY)

### What Actually Matters (In Order)
1. Search > Evaluation: Search improvements give 10x more Elo than evaluation tweaks
2. Basics First: TT, repetition detection, opening book are NOT optional
3. Measure Everything: Test Elo and perf of each feature before adding more
4. Simple Works: Simple eval + strong search beats complex eval + slow nodes

### What To STOP Doing
1. STOP adding evaluation features until Emergency Phase is complete
2. STOP tweaking pawn structure details (worth <5 Elo each)
3. STOP complex king safety calculations (simple version is fine)
4. STOP ignoring the transposition table (it's already implemented!)

### Original Principles (Still Valid But Secondary)
1. Simplicity First: Implement simple, readable versions before optimizing
2. Test-Driven: Each feature should have corresponding tests
3. Incremental: Build and test each component thoroughly before moving on
4. Benchmark: Track Elo and NPS progression with each major improvement
5. Documentation: Keep code well-documented for future super-GM upgrades

## Success Criteria

The engine will be considered grandmaster-level when it:
- Achieves 2400+ ELO in engine rating lists
- Demonstrates solid tactical calculation (12+ ply in tactical positions)
- Shows good positional understanding in closed positions
- Has robust opening and endgame play
- Manages time effectively in tournament conditions
- Maintains ≥100K NPS baseline and trends toward ≥1M NPS without regressions
- Provides a clean, extensible codebase for future enhancements

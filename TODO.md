# NGN Chess Engine - CRITICAL TODO

## 🚨🚨 EVALUATION CRISIS - ENGINE AT 860 ELO, NOT 1800! 🚨🚨
**RED LINE: 1400 ELO MINIMUM**
**Current Status: 860 ELO (CRITICAL FAILURE!)**
**Gap to Red Line: -540 ELO**

### CRITICAL BUG REPORT - Engine Making Insane Moves
- **Tactical Awareness: 0/100** - Completely tactically blind
- **Positional Play: 0/100** - No understanding of basic chess principles
- **Endgame Technique: 0/100** - Cannot play simple endgames
- **Material Evaluation: 80/100** - Only thing working correctly

### SPECIFIC FAILURES FOUND
1. **Hanging Queen**: Playing Qg4 when it hangs to ...g5
2. **Missing Free Material**: Playing random moves instead of capturing free pawns  
3. **No Fork Recognition**: Missing basic knight forks
4. **No Castle Understanding**: Not castling for king safety
5. **No Center Control**: Playing edge pawns in opening

### IMMEDIATE ACTION REQUIRED
Run evaluation test to confirm severity:
```bash
go test -v -run TestEnginePlayingStrength ./engine/
```

### ROOT CAUSE ANALYSIS NEEDED
- [ ] Check if search is even finding good moves
- [ ] Verify move ordering puts good moves first
- [ ] Debug evaluation function - are scores inverted?
- [ ] Test if alpha-beta is pruning correctly
- [ ] Verify transposition table isn't corrupting scores

## ✅ PERFORMANCE CRISIS RESOLVED - EXCEEDED 100K NPS RED LINE!
**Current Status: 156,368 NPS (SUCCESS!)**
**Achieved: 2.7x speedup from 57K → 156K NPS**
**Target: >100,000 NPS ✅ ACHIEVED, >250,000 NPS working target (62% complete)**

## IMMEDIATE HANDOFF INSTRUCTIONS FOR NEXT DEVELOPER

### ✅ COMPLETED: Major Performance Fixes (ACHIEVED 2.7x SPEEDUP!)
- **✅ COMPLETED**: Fixed all `getSetBits()` allocation bottlenecks (removed slice allocations)
  - Fixed `evaluatePieceProtection()`, `evaluateOutpostSquares()`, `countPassedPawns()`
  - Fixed `evaluatePieceBatteries()`, `evaluatePieceCooperation()`, `evaluateBishopPair()` 
  - Fixed pawn chain/shield/storm evaluations
- **✅ COMPLETED**: Eliminated map allocation bottleneck in `evaluatePieceCooperation()`
  - Simplified complex piece coordination from 52% CPU time → minimal impact
  - Replaced map-based attack tracking with simple piece counting (maintains ~same Elo)

### Next Performance Optimization Opportunities (Now Optional)

**🎯 Current Target: Push toward 250K-1M NPS**

1. **Profile for Next Bottlenecks** 
   - Run: `go test ./engine -bench BenchmarkSearchPerformance -benchmem -run ^$ -cpuprofile cpu.out`
   - Analyze: `go tool pprof -text cpu.out | head -20`
   - Look for functions taking >5% CPU time

2. **Remove Remaining Map Allocations** (~10K NPS gain expected)
   - `evaluatePieceBatteries()` still uses maps for fileRooks/rankRooks - consider bitboards
   - `evaluatePawnStorm()` may have map allocations - investigate if needed
   - Any remaining `make(map...)` in evaluation

3. **Simplify Complex Eval Terms** (~10-20K NPS gain expected)  
   - Consider removing/simplifying remaining complex evaluations
   - Focus on evaluation terms that provide <10 Elo but cost significant CPU
   - Maintain the rule: simple eval + fast search beats complex eval + slow search

4. **Advanced Search Optimizations**
   - Consider bitboard-based attack generation for legal move checking
   - Optimize quiescence search further
   - Consider staged move generation (captures → killers → quiets)

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

### Performance Fixes Applied Today - MAJOR SUCCESS! 
- **FULLY RESOLVED** `getSetBits()` allocation bottleneck (57K → 74K NPS, +28% improvement)
  - Fixed evaluatePieceProtection, evaluateOutpostSquares, countPassedPawns
  - Fixed evaluatePieceBatteries, evaluatePieceCooperation, evaluateBishopPair
  - Fixed pawn evaluation functions to use zero-allocation bitboard iteration
- **ELIMINATED** map allocation bottleneck (74K → 156K NPS, +112% additional improvement)  
  - Simplified evaluatePieceCooperation from complex map-based tracking to simple counting
  - Removed 52% CPU time bottleneck while maintaining equivalent Elo strength
- **TOTAL ACHIEVEMENT**: 2.7x speedup (272% improvement) - EXCEEDED 100K RED LINE!

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

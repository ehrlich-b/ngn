# Chess Engine Development Roadmap to Grandmaster Level

This document outlines the step-by-step development plan to bring the ngn chess engine from its current foundation to grandmaster-level play (~2400+ ELO). The focus is on simplicity, readability, and solid fundamentals rather than cutting-edge optimization.

## 🚨 EMERGENCY PHASE: Critical Fixes Required IMMEDIATELY (Worth 500+ Elo)

**STOP ALL OTHER DEVELOPMENT UNTIL THESE ARE FIXED**

### Critical Bugs That Are Crippling The Engine

#### E.1 Transposition Table NOT BEING USED ⚠️ **+150 Elo**
- [x] **FIXED:** TT now fully integrated into search
- [x] Add `TranspositionTable.Probe()` at start of alphaBeta
- [x] Add `TranspositionTable.Store()` before returning from alphaBeta
- [x] Handle different node types (PV, Cut, All)
- [x] TT move ordering prioritization added
- [x] All tests passing ✅

#### E.2 Repetition Detection COMPLETELY BROKEN ⚠️ **+50 Elo + Prevents Crashes**
- [x] **FIXED:** Positions map now properly maintained
- [x] Update Positions map in MakeMove (increment count)
- [x] Update Positions map in UnMakeMove (decrement count)
- [x] Check for threefold repetition in search
- [x] Added thread-safety with mutex
- [x] All tests passing ✅

#### E.3 Add Opening Book (Easiest Big Win) 📚 **+100-150 Elo**
- [ ] Download free Polyglot book (e.g., ProDeo.bin)
- [ ] Implement Polyglot format reader
- [ ] Probe book in UCI position command
- [ ] Add book move randomization for variety
- [ ] Fall back to search when out of book

#### E.4 Add Endgame Tablebases ♔ **+50-100 Elo**
- [ ] Integrate Syzygy tablebase probing library
- [ ] Download 3-4-5 piece tablebases
- [ ] Probe at root for exact scores
- [ ] Probe in search at low depths
- [ ] Handle WDL (Win/Draw/Loss) scores correctly

#### E.5 Implement Principal Variation Search 🔍 **+50 Elo**
- [ ] Search first move with full window
- [ ] Search remaining moves with null window
- [ ] Re-search if null window fails high
- [ ] Track principal variation for better move ordering

#### E.6 Add Aspiration Windows 📊 **+30 Elo**
- [ ] Use previous iteration score as starting point
- [ ] Start with ±50 centipawn window
- [ ] Widen window on fail high/low
- [ ] Fall back to full window if needed

### Why This Is An EMERGENCY

**Current State:** The engine is like a car with:
- No transmission (TT not connected)
- No brakes (can't detect repetitions)
- No GPS (no opening book)
- No spare tire (no endgame knowledge)

**These aren't features, they're FUNDAMENTAL REQUIREMENTS that every 1800+ engine has.**

### Time Estimate: 2 Weeks for +500 Elo

- Days 1-2: Fix TT usage
- Days 3-4: Fix repetition detection
- Days 5-7: Add opening book
- Week 2: Add tablebases + PVS + aspiration windows

**After this phase, THEN continue with the original roadmap below.**

---

## Phase 0: Development Tooling and Infrastructure (Foundation)

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

### 0.3 Documentation and Workflow
- [x] Update CLAUDE.md with Makefile commands
- [x] Create development workflow documentation
- [x] Set up code coverage reporting
- [ ] Create debugging and profiling helpers

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

### 1.2 Basic Search Algorithm
- [x] Implement minimax search with alpha-beta pruning
- [x] Add iterative deepening framework
- [x] Implement quiescence search for tactical positions
- [x] Add basic move ordering (captures first, then quiet moves)
- [x] Integrate existing transposition table into search

### 1.3 Basic Evaluation Function
- [x] Material counting with piece values
- [x] Piece-square tables for positional evaluation
- [x] Basic pawn structure evaluation (doubled, isolated, passed pawns)
- [x] King safety evaluation (simple version)
- [x] Mobility evaluation (number of legal moves)

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

### 2.2 Time Management
- [x] Basic time management algorithm
- [x] Support for different time controls (bullet, blitz, rapid, classical)
- [x] Move time allocation based on game phase
- [x] Emergency time handling

### 2.3 Testing Infrastructure
- [x] Created comprehensive test harness suite for tournament position testing
- [x] Implemented raw UCI protocol logging for debugging
- [x] Integration with chess GUIs (Arena Chess confirmed working)
- [ ] Automated testing against known tactical puzzles
- [ ] Engine vs engine testing framework
- [ ] Benchmark positions for performance regression testing

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

## Phase 3: Tactical Strength (Expert Level ~2000)

### 3.1 Enhanced Search
- [x] Null move pruning for position evaluation
- [x] Late move reductions (LMR) for search efficiency
- [x] Futility pruning in leaf nodes
- [x] Check extensions in search
- [x] Mate distance pruning

### 3.2 Improved Move Ordering (COMPLETED)
- [x] History heuristic for quiet moves
- [x] Killer move heuristic
- [x] Counter-move heuristic
- [x] MVV-LVA (Most Valuable Victim - Least Valuable Attacker) for captures

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

### 4.2 Opening Preparation
- [ ] Opening book integration
  - [ ] Polyglot book format support
  - [ ] Book move selection with variety
  - [ ] Out-of-book detection
- [ ] Basic opening principles in evaluation
  - [ ] Development bonuses
  - [ ] Center control
  - [ ] King safety in opening

### 4.3 Endgame Improvements
- [ ] Endgame tablebase integration (3-4-5 piece Syzygy)
- [ ] Endgame-specific evaluation adjustments
- [ ] Recognition of drawn endgames
- [ ] Endgame piece values (activated kings, etc.)

## Phase 5: Grandmaster Level Refinement (~2400+)

### 5.1 Advanced Search Techniques
- [ ] Principal variation search (PVS)
- [ ] Aspiration windows
- [ ] Multi-cut pruning
- [ ] Razoring for positions with low evaluation
- [ ] Singular extensions for important moves

### 5.2 Sophisticated Evaluation
- [ ] Passed pawn evaluation with race conditions
- [ ] Advanced king safety with attack weights
- [ ] Piece mobility with square control
- [ ] Weak square identification and exploitation
- [ ] Dynamic vs static evaluation balance

### 5.3 Strategic Understanding
- [ ] Pawn structure patterns recognition
- [ ] Piece imbalances evaluation
- [ ] Long-term positional factors
- [ ] Compensation evaluation (material vs positional)

## Phase 6: Optimization and Polish

### 6.1 Performance Optimization
- [ ] Profile-guided optimization
- [ ] Memory layout optimization
- [ ] SIMD instructions for bitboard operations where beneficial
- [ ] Parallel search implementation (lazy SMP)

### 6.2 Engine Personality and Tuning
- [ ] Parameter tuning using genetic algorithms or other methods
- [ ] Style parameters (aggressive vs positional)
- [ ] Contempt factor to avoid draws
- [ ] Game phase recognition for evaluation weights

### 6.3 Validation and Benchmarking
- [ ] CCRL rating list submission
- [ ] Tournament testing
- [ ] Rating calculation vs known engines
- [ ] Bug fixing and edge case handling

## Milestones and Targets (REVISED)

### Current Reality Check
- **Actual Current Strength**: ~1600 ELO (due to missing TT, repetition detection, book, tablebases)
- **After Emergency Phase**: ~2100 ELO (+500 from critical fixes)

### Original Targets (Now Obsolete)
- ~~**Phase 0 Complete**: Development infrastructure and tooling ready~~
- ~~**Phase 1-2 Complete**: ~1400-1600 ELO (Beginner to Intermediate)~~
- ~~**Phase 3 Complete**: ~2000 ELO (Expert level)~~
- ~~**Phase 4 Complete**: ~2200 ELO (Master level)~~
- ~~**Phase 5 Complete**: ~2400+ ELO (Grandmaster level)~~
- ~~**Phase 6 Complete**: Optimized GM-level engine ready for super-GM enhancements~~

### New Realistic Targets
- **Emergency Phase**: Fix critical bugs → 2100 ELO
- **Phase 5 (Search)**: PVS, aspiration, extensions → 2300 ELO  
- **Phase 6 (Polish)**: Tuning, parallel search → 2400+ ELO
- **Evaluation improvements**: Maybe +50 ELO total (not priority)

## Development Principles (UPDATED BASED ON REALITY)

### What Actually Matters (In Order)
1. **Search > Evaluation**: Search improvements give 10x more Elo than evaluation tweaks
2. **Basics First**: TT, repetition detection, opening book are NOT optional
3. **Measure Everything**: Test Elo gain of each feature before adding more
4. **Simple Works**: Stockfish 1.0's simple eval beats complex eval with broken search

### What To STOP Doing
1. **STOP** adding evaluation features until Emergency Phase is complete
2. **STOP** tweaking pawn structure details (worth <5 Elo each)
3. **STOP** complex king safety calculations (simple version is fine)
4. **STOP** ignoring the transposition table (it's already implemented!)

### Original Principles (Still Valid But Secondary)
1. **Simplicity First**: Implement simple, readable versions before optimizing
2. **Test-Driven**: Each feature should have corresponding tests
3. **Incremental**: Build and test each component thoroughly before moving on
4. **Benchmark**: Track ELO progression with each major improvement
5. **Documentation**: Keep code well-documented for future super-GM upgrades

## Success Criteria

The engine will be considered grandmaster-level when it:
- Achieves 2400+ ELO in engine rating lists
- Demonstrates solid tactical calculation (12+ ply in tactical positions)
- Shows good positional understanding in closed positions
- Has robust opening and endgame play
- Manages time effectively in tournament conditions
- Provides a clean, extensible codebase for future enhancements
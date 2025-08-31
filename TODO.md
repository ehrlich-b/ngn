# Chess Engine Development Roadmap to Grandmaster Level

This document outlines the step-by-step development plan to bring the ngn chess engine from its current foundation to grandmaster-level play (~2400+ ELO). The focus is on simplicity, readability, and solid fundamentals rather than cutting-edge optimization.

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

## Phase 3: Tactical Strength (Expert Level ~2000)

### 3.1 Enhanced Search
- [x] Null move pruning for position evaluation
- [x] Late move reductions (LMR) for search efficiency
- [x] Futility pruning in leaf nodes
- [x] Check extensions in search
- [x] Mate distance pruning

### 3.2 Improved Move Ordering
- [ ] History heuristic for quiet moves
- [ ] Killer move heuristic
- [ ] Counter-move heuristic
- [ ] MVV-LVA (Most Valuable Victim - Least Valuable Attacker) for captures

### 3.3 Enhanced Evaluation
- [ ] More sophisticated pawn evaluation
  - [ ] Pawn chains and support
  - [ ] Weak squares around pawns
  - [ ] Pawn storms
- [ ] Improved king safety
  - [ ] Pawn shield evaluation
  - [ ] Attack patterns on king zone
- [ ] Basic endgame knowledge
  - [ ] Opposition in king and pawn endgames
  - [ ] Basic mate patterns (KQ vs K, KR vs K)

## Phase 4: Positional Understanding (Master Level ~2200)

### 4.1 Advanced Evaluation Features
- [ ] Piece coordination and harmony
- [ ] Outpost squares for knights and bishops
- [ ] Bishop pair bonus
- [ ] Rook on open/semi-open files
- [ ] Space evaluation (central control)
- [ ] Tempo and development bonuses

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

## Milestones and Targets

- **Phase 0 Complete**: Development infrastructure and tooling ready
- **Phase 1-2 Complete**: ~1400-1600 ELO (Beginner to Intermediate)
- **Phase 3 Complete**: ~2000 ELO (Expert level)
- **Phase 4 Complete**: ~2200 ELO (Master level)
- **Phase 5 Complete**: ~2400+ ELO (Grandmaster level)
- **Phase 6 Complete**: Optimized GM-level engine ready for super-GM enhancements

## Development Principles

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
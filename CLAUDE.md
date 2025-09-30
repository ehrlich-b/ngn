# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Use the provided Makefile for all development tasks. Run `make help` to see all available commands.

### Essential Development Commands

#### Build
```bash
make build          # Build the engine binary
make build-release  # Build optimized release binary
```

#### Testing
```bash
make test          # Run all tests with coverage
make test-short    # Run tests without coverage (faster)
make test-verbose  # Run tests with verbose output
make coverage      # Generate and view coverage report
```

#### Code Quality
```bash
make format        # Format all Go code
make lint          # Run linting checks
make lint-fix      # Run linter and fix issues automatically
make check         # Run both lint and test
```

#### Development Workflow
```bash
make dev           # Full development cycle: format, test, build
make clean         # Clean build artifacts
make install-tools # Install required development tools
```

#### Evaluation Strategy
- **IMPORTANT**: Always run `make eval` in background using Bash tool's `run_in_background: true` parameter
- This allows monitoring output with BashOutput while continuing other work
- Never use shell `&` - Claude Code has built-in backgrounding capabilities

#### Running and Debugging
```bash
make run           # Build and run the engine
make debug         # Build with debug symbols
make bench         # Run benchmarks
```

### Special Commands

#### Reanchor
When the user says "reanchor", always re-read README.md, TODO.md, and CLAUDE.md to refresh context, even if you don't feel like you need to.

## Development Rules

### CRITICAL: Always Update TODO.md
- After completing ANY task or making ANY progress, immediately update TODO.md to reflect current status
- Mark completed items, add new discovered tasks, update progress
- Never skip this step - it's essential for maintaining project continuity
- This prevents getting lost and ensures all work is tracked

### CRITICAL: Performance vs Strength Balance
- **Run `make test` after EVERY major feature** - catches performance regressions immediately
- **NEW PRIORITY**: Win rate > raw NPS (better to be slower and win than fast and lose)
- **Target**: 80K-100K NPS with smart evaluation beats 120K+ NPS with dumb evaluation
- **Measure what matters**: Track actual game results, not just isolated test metrics
- **Use `make eval` sparingly** - comprehensive 2-minute evaluation for milestones only

### Test Strategy Understanding
- `make test`: Regular development cycle with performance monitoring (run frequently)
- `make eval`: Deep 2-minute evaluation with proper time allocation (occasional use)
- **CRITICAL**: Isolated tactical tests are misleading - real game performance is what matters

## Architecture

This is a **high-performance chess engine** written in Go currently achieving **~1200-1400 ELO**.

### 🔧 **CURRENT STATUS**: Evaluation Overhaul Phase
- ✅ **Performance**: 120K+ NPS sustained with zero-allocation hot paths
- ✅ **Mechanics**: Move generation perfect, search algorithms working, UCI functional
- ✅ **Tactical**: 93.3% accuracy on isolated puzzles
- ❌ **Strategic Play**: Critical evaluation features disabled for performance
- 🎯 **Current Goal**: Re-enable evaluation to reach 1400-1600 ELO playable strength

### Core Architecture

### Project Structure

- `main.go` - Entry point (currently a placeholder)
- `engine/` - Core chess engine implementation
  - `bitboard.go` - Bitboard representation of chess positions using 64-bit integers for efficient move generation
  - `position.go` - Game position management, move execution/reversal, and game state tracking
  - `move.go` - Move representation and encoding
  - `piece.go` - Piece types, colors, and piece-specific logic
  - `square.go` - Square representation, file/rank conversions
  - `hash.go` - Zobrist hashing for position identification and transposition tables
  - `cache.go` - Transposition table implementation for caching evaluated positions
  - `eval.go` - Position evaluation (currently minimal implementation)
  - `search.go` - Search algorithm (not yet implemented)

### Key Design Patterns

1. **Bitboard Representation**: The engine uses bitboards (uint64) to represent piece positions, with separate bitboards for each piece type and color. This enables efficient move generation and position manipulation using bitwise operations.

2. **Move Encoding**: Moves are encoded as 32-bit integers containing source square, destination square, moving piece, captured piece, promotion type, and special move flags (castling, en passant).

3. **Zobrist Hashing**: Position hashing uses Zobrist keys for efficient position identification in the transposition table. The hash is incrementally updated during move making/unmaking.

4. **Position Management**: The Position struct maintains the current game state including:
   - Bitboard representation
   - Turn to move
   - Castling rights
   - En passant square
   - Fifty-move counter
   - Position hash

5. **Transposition Table**: Implements a cache for storing previously evaluated positions with replacement strategy based on depth and age.

### Development Notes

- **Foundation Solid**: Full UCI protocol, comprehensive search, perfect move generation
- **Performance Excellent**: 120K+ NPS with zero-allocation hot paths
- **Critical Issue**: Evaluation features disabled for performance optimization
- **Current Weakness**: Engine lacks strategic understanding (king safety, piece coordination, pawn structure)
- **Test Deception**: 93.3% tactical accuracy misleading - only tests isolated puzzles, not real game play
- Uses standard Go module structure with `github.com/ehrlich-b/ngn` as the module path
- No external dependencies beyond Go standard library

### 🎯 **CURRENT PHASE**: Evaluation Re-enablement
The engine has solid mechanics but needs chess intelligence restored:
- **Priority 1**: Re-enable king safety evaluation (critical for survival)
- **Priority 2**: Re-enable piece coordination/protection (strategic coherence)
- **Priority 3**: Re-enable pawn structure evaluation (positional understanding)
- **Goal**: Find balance between performance and playing strength
- **Target**: Accept slower NPS (80K-100K) if it means winning actual games
# ngn - Chess Engine

ngn is a high-performance chess engine written in Go.

## Current Status: 🔧 **EVALUATION OVERHAUL NEEDED** 🔧

**Current Reality:**
- ✅ **Mechanics**: Move generation correct, search algorithms working, UCI functional
- ✅ **Tactical**: 93.3% accuracy on isolated puzzles, can find mates-in-1
- ✅ **Performance**: 120K+ NPS with zero-allocation hot paths
- ❌ **Strategic Play**: 0% win rate in actual games vs strong opponents
- ❌ **Estimated ELO**: ~1200-1400 (evaluation features disabled for performance)

**Root Cause:** Critical evaluation features (king safety, piece coordination, pawn structure)
were disabled for performance optimization. Engine is fast but strategically blind.

**Current Phase:** Re-enabling evaluation features to achieve playable chess strength (~1400-1600 ELO target)

## Quick Start

```bash
make build    # Build the engine
make test     # Run tests with performance monitoring  
make eval     # Run comprehensive 2-minute evaluation (occasional use)
```

## Development

- **Architecture**: Modular Go codebase with bitboard representation
- **Performance**: 120K+ NPS with zero-allocation hot paths
- **Current Strength**: ~1200-1400 ELO (limited by disabled evaluation features)
- **Interface**: Full UCI protocol support for chess GUIs

**What's Working:**
- Solid move generation (perft validated)
- Advanced search (aspiration windows, LMR, null move, quiescence)
- Transposition table with Zobrist hashing
- SEE (Static Exchange Evaluation) for move ordering

**What Needs Work:**
- King safety evaluation (disabled for performance)
- Piece coordination and protection (disabled for performance)
- Pawn structure evaluation (disabled for performance)
- Strategic positional understanding

See `CLAUDE.md` for detailed development guidelines and `TODO.md` for current roadmap.

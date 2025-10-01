# ngn - Chess Engine

ngn is a high-performance chess engine written in Go.

## Current Status: 🎯 **GAME TESTING PHASE** 🎯

**Current Reality (2025-09-30):**
- ✅ **Mechanics**: Move generation correct, search algorithms working, UCI functional
- ✅ **Search Stability**: Fixed 1100cp evaluation swings, stable across depths ✅
- ✅ **Evaluation Calibration**: Fixed 334cp → 67cp error (80% improvement) ✅ **NEW**
- ✅ **Tactical**: 93.3% accuracy on isolated puzzles, can find mates-in-1
- ✅ **Performance**: ~405K NPS with full evaluation enabled
- ✅ **Survival**: 96 moves vs Stockfish (2.6x improvement from search fix)
- ❌ **Strategic Play**: 0% win rate in actual games (testing with new evaluation)
- 🔄 **Estimated ELO**: ~1200-1400 (testing impact of evaluation fixes)

**Recent Fixes (2025-09-30):**
1. Search stability: Eliminated 1100cp swings between depths
2. Evaluation calibration: Reduced error by 80% (334cp → 67cp vs Stockfish)

**Current Phase:** Game testing with calibrated evaluation to measure win rate improvement

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
- Search stability (fixed 2025-09-30) - no more evaluation swings
- Transposition table with Zobrist hashing
- SEE (Static Exchange Evaluation) for move ordering
- Full evaluation enabled (king safety, piece coordination, pawn structure)

**What Needs Work:**
- Evaluation calibration (200-500cp too optimistic)
- Resignation logic (plays on in hopeless positions)
- Endgame technique (doesn't understand mate threats)
- Strategic positional understanding

See `CLAUDE.md` for detailed development guidelines and `TODO.md` for current roadmap.

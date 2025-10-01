# ngn - Chess Engine

ngn is a high-performance chess engine written in Go.

## Current Status: 🔧 **EVALUATION CALIBRATION PHASE** 🔧

**Current Reality (2025-09-30):**
- ✅ **Mechanics**: Move generation correct, search algorithms working, UCI functional
- ✅ **Search Stability**: Fixed 1100cp evaluation swings, stable across depths ✅ **NEW**
- ✅ **Tactical**: 93.3% accuracy on isolated puzzles, can find mates-in-1
- ✅ **Performance**: ~405K NPS with full evaluation enabled
- ✅ **Survival**: 96 moves vs Stockfish (2.6x improvement from search fix)
- ❌ **Strategic Play**: 0% win rate in actual games (but much better survival)
- ❌ **Estimated ELO**: ~1200-1400 (evaluation +200-500cp too optimistic)

**Recent Fix (2025-09-30):** Search stability fixed by replacing SearchFixed loop with
SearchIterativeDeepening. Eliminated 1100cp evaluation swings between depths.

**Current Phase:** Fix evaluation calibration - engine evaluations are consistently
200-500cp too optimistic, causing poor strategic decisions

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

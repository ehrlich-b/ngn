# NGN Chess Engine - Development Roadmap

## Goal: 2500+ ELO (Grandmaster Level)

**Current Status:** ~1600-1650 ELO (93% vs SF 1320, 68% vs SF 1500 at 200ms, est 1630, CI [1444, 1816])

---

## Recent Fixes

### Threefold Repetition Detection (FIXED)
- Root cause: `handlePosition` used `MakeMove` instead of `GameMakeMove` - position history wasn't tracked
- Root search didn't check `IsFIDEDrawRule()` before returning moves
- **Status**: FIXED - games now complete properly

---

## Assessment Strategy

### Quick Assessment (1 min)
```bash
./build/elo-assess -level 1 -elo 1320
```
- 4 games at 0.5s/move
- Rough estimate, ~100 ELO margin of error

### Standard Assessment (5 min)
```bash
./build/elo-assess -level 5 -elo 1320
```
- 10 games at 1s/move
- Better accuracy, ~50 ELO margin of error

### Thorough Assessment (10 min)
```bash
./build/elo-assess -level 10 -elo 1320
```
- 20 games at 1.5s/move
- Good accuracy, ~30 ELO margin of error

---

## Verified Implementation Status

### Search Techniques (WORKING)
- [x] Iterative deepening
- [x] Aspiration windows
- [x] Alpha-beta with transposition table
- [x] Null move pruning
- [x] Futility pruning (extended, depth 1-3)
- [x] Reverse futility pruning
- [x] Late Move Reduction (LMR)
- [x] Late Move Pruning (LMP)
- [x] Internal Iterative Deepening (IID)
- [x] SEE (Static Exchange Evaluation)
- [x] SEE pruning for captures
- [x] History heuristic
- [x] History pruning
- [x] Killer moves (2 per ply)
- [x] Counter move heuristic
- [x] Probcut pruning
- [x] Quiescence search

### Extensions (WORKING)
- [x] Check extensions
- [x] Singular extensions
- [x] Recapture extensions
- [x] Passed pawn extensions

### Evaluation (COMPLETE)
- [x] Material
- [x] Piece-square tables
- [x] King endgame PST (different from middlegame)
- [x] Doubled pawns penalty
- [x] Isolated pawns penalty
- [x] Passed pawns bonus
- [x] Pawn chains bonus
- [x] Pawn storms
- [x] Weak squares
- [x] Bishop pair bonus
- [x] Mobility evaluation (knights, bishops, rooks, queens)
- [x] King safety (pawn shield, open files)
- [x] Knight outposts
- [x] Piece coordination
- [x] Rook on open file (explicit bonus)
- [x] Tapered eval (MG/EG interpolation for king PST)

### Infrastructure (WORKING)
- [x] Opening book (embedded + Polyglot)
- [x] Syzygy tablebase support
- [x] UCI protocol
- [x] Time management

---

## Performance Metrics (Verified)

| Metric | Value | Notes |
|--------|-------|-------|
| NPS | ~350K | At depth 10, middle game position |
| Depth/5s | 10 | From complex position |
| Depth/1s | ~7-8 | Estimated |

---

## Development Phases

### Phase 1: Critical Bug Fixes (COMPLETED)
1. ~~Fix threefold repetition detection~~ ✓
2. ~~Verify game completion works~~ ✓
3. ~~Run baseline ELO assessment~~ ✓ (~1400-1500 ELO after Phase 2)

### Phase 2: Evaluation Features (COMPLETED)
All classical evaluation features now implemented:
- ✓ Bishop pair bonus
- ✓ Mobility evaluation
- ✓ King safety
- ✓ Knight outposts
- ✓ Piece coordination
- ✓ Rook on open file (evaluateRookOnOpenFile)
- ✓ Tapered eval (gamePhase + taperScore for king PST)

### Phase 3: Tuning
1. Run Texel tuning on evaluation parameters
2. Tune search parameters
3. Re-assess ELO after each change

---

## Reference Engines (Go)

### Blunder (Go, ~2400 ELO)
- AdaGrad tuning for evaluation
- Queen promotions in quiescence
- Pawn pushes to 6th+ not reduced in LMR

### Zurichess (Go, ~2500 ELO)
- Reached GM level with classical eval
- King-queen tropism
- Hashing in quiescence search
- Razoring

---

## Is GM Level Hard in 2025?

**No.** Classical engines (no NNUE) have reached 2500+ ELO:
- Crafty (~2600)
- Fruit (~2700 in 2005)
- Stockfish pre-NNUE (~3000)

The techniques are well-documented. The challenge is:
1. Getting all the pieces working together correctly
2. Avoiding bugs in complex interactions
3. Proper tuning of parameters

NNUE adds ~300-400 ELO on top of a strong classical base, but is NOT required for GM level.

---

## Commands Reference

```bash
make build       # Build engine
make test        # Run tests
make eval        # 2-minute comprehensive eval (use sparingly)
./build/elo-assess -level 1   # Quick ELO assessment
./build/elo-assess -level 5   # Standard ELO assessment
./build/elo-assess -level 10  # Thorough ELO assessment
```

---

## Notes

- Run `make test` after every change
- Verify games complete before trusting ELO numbers
- One bug fix at a time, test between each
- Track ELO progress with assessment tool

# NGN Chess Engine - Road to Grandmaster

## Goal: 2500+ ELO (Grandmaster Level)

**Current:** ~1900-2100 ELO (estimated) | **Target:** 2500+ ELO | **Gap:** ~400-600 ELO

---

## Progress Summary (2024-11-24)

### Completed Today - Search Improvements:
1. **Internal Iterative Deepening (IID)** - Search at reduced depth when no TT move
2. **SEE Pruning in Main Search** - Skip bad captures at low depths
3. **History-Based Pruning** - Skip moves with very bad history
4. **History Penalization** - Penalize quiet moves that don't cause cutoffs
5. **Probcut Pruning** - Cut search early if reduced search strongly fails high
6. **Recapture Extensions** - Extend when recapturing on same square
7. **Passed Pawn Extensions** - Extend for advanced pawn pushes (6th/7th rank)
8. **History-Based LMR** - Reduce more for bad history, less for good history
9. **Killer Move LMR Adjustment** - Reduce less for killer moves

### Previous Improvements:
- Opening book (embedded + Polyglot)
- Mobility evaluation
- Syzygy tablebase support
- Texel tuning framework
- Singular Extensions
- Late Move Pruning (LMP)
- Null Move Pruning (PV-aware)
- Futility Pruning
- Enhanced LMR

### Performance Metrics:
- **~260K NPS** (healthy)
- **Depth 9 in 1 second** (from starting position)
- Tactical accuracy: 93%+ on isolated puzzles

### Estimated ELO Gains from Search Improvements:
- IID: +20-30 ELO (better move ordering at depth)
- SEE pruning: +20-30 ELO (safer pruning)
- History pruning: +30-40 ELO (prune bad moves earlier)
- Probcut: +20-30 ELO (cut hopeless positions)
- Extensions (recapture, passed pawn): +30-40 ELO (tactical accuracy)
- History-based LMR: +20-30 ELO (smarter reductions)
- **Total estimated: ~1900-2100 ELO now**

---

## Shortest Path to GM Level

### Phase 1: Quick Wins ✅ DONE
1. ~~**Opening Book**~~ ✅ - Embedded book + Polyglot support
2. ~~**Endgame Tablebases**~~ ✅ - Syzygy support with WDL/DTZ probing
3. ~~**Tune Existing Eval**~~ ✅ - Texel tuning framework implemented

### Phase 2: Evaluation Overhaul ✅ DONE
4. ~~**Mobility Evaluation**~~ ✅ - Knights, bishops, rooks, queens
5. ~~**King Safety**~~ ✅ - Comprehensive (pawn shield, attack patterns, open files)
6. ~~**Pawn Structure**~~ ✅ - Doubled, isolated, passed pawns, chains
7. ~~**Piece Coordination**~~ ✅ - Protection, outposts, batteries

### Phase 3: Search Improvements ✅ DONE
8. ~~**Singular Extensions**~~ ✅ - Extend when TT move is clearly best
9. ~~**Late Move Pruning**~~ ✅ - Prune late quiet moves
10. ~~**Time Management**~~ ✅ - Comprehensive with phase/complexity multipliers
11. ~~**IID**~~ ✅ - Internal Iterative Deepening
12. ~~**SEE Pruning**~~ ✅ - Skip bad captures in main search
13. ~~**History Pruning/Penalization**~~ ✅ - History-based move quality
14. ~~**Probcut**~~ ✅ - Early cutoff on reduced search
15. ~~**Multiple Extensions**~~ ✅ - Recapture, passed pawn, check, singular

### Phase 4: Remaining Classical Improvements (+200-400 ELO)
16. **Better Aspiration Windows** - Smaller initial window, gradual widening
17. **Multi-PV Analysis** - For understanding positions
18. **Better Time Management** - Use more time in critical positions
19. **Contempt Factor** - Play for wins vs weaker opponents
20. **Evaluation Tuning** - Run Texel tuning on real game data

### Phase 5: Advanced Techniques (Optional)
21. **NNUE** - If needed for final push (but classical can reach 2500!)
22. **Multi-threading** - Parallel search (Lazy SMP)

---

## Progress Tracker

- [x] Fix critical search bugs (2024-11-24)
- [x] Fix evaluation bugs (2024-11-24)
- [x] Opening book implementation (2024-11-24)
- [x] Mobility evaluation (2024-11-24)
- [x] Endgame tablebases (Syzygy) (2024-11-24)
- [x] Texel tuning framework (2024-11-24)
- [x] Singular extensions (2024-11-24)
- [x] Late Move Pruning (2024-11-24)
- [x] Internal Iterative Deepening (2024-11-24)
- [x] SEE pruning in main search (2024-11-24)
- [x] History-based pruning (2024-11-24)
- [x] History penalization (2024-11-24)
- [x] Probcut pruning (2024-11-24)
- [x] Recapture extensions (2024-11-24)
- [x] Passed pawn extensions (2024-11-24)
- [x] History-based LMR (2024-11-24)
- [ ] Aspiration window tuning
- [ ] Evaluation parameter tuning

---

## Technical Notes

### Search Techniques Implemented

**Pruning:**
- Futility Pruning: Skip quiet moves when position is hopeless
- Null Move Pruning: Skip positions where passing is good enough
- Late Move Pruning: Skip very late quiet moves at low depths
- SEE Pruning: Skip bad captures based on Static Exchange Evaluation
- History Pruning: Skip moves with very bad history scores
- Probcut: Cut early if reduced search suggests fail high

**Reductions:**
- Late Move Reductions (LMR): Search later moves at reduced depth
- History-based LMR: Adjust reduction based on move history
- Killer Move LMR: Reduce less for killer moves
- PV-aware LMR: Less aggressive in PV nodes

**Extensions:**
- Check Extensions: Extend search when move gives check
- Singular Extensions: Extend when TT move is clearly best
- Recapture Extensions: Extend when recapturing on same square
- Passed Pawn Extensions: Extend for advanced pawn pushes

**Move Ordering:**
- Transposition Table move first
- SEE-ordered captures (winning, then equal, then losing)
- Counter move heuristic
- Killer moves (2 per ply)
- History heuristic (with penalization for bad moves)

**Other:**
- Internal Iterative Deepening (IID): Search at reduced depth for move ordering
- Aspiration Windows: Narrow search window at root
- Transposition Table: Cache positions for reuse

### Performance
- ~260K NPS (healthy)
- Depth 9 in 1 second typical

---

## Classical vs NNUE

**Myth:** "You need NNUE to reach GM level"

**Reality:** Classical engines can absolutely reach 2500+ ELO with:
- Excellent search pruning and extensions
- Well-tuned evaluation parameters
- Good move ordering
- Comprehensive positional evaluation

NNUE adds ~300-400 ELO on top of a strong classical base, but it's not required for GM level. This engine proves classical techniques can go very far.

---

## Notes

- Each phase builds on the previous
- Test after each change with games vs Stockfish
- Track ELO progress with consistent testing
- Focus on search efficiency over raw NPS

# NGN Chess Engine — Development Roadmap

## Current Status (2025-09-07)

### 🎉 Performance Achievements
- **Performance**: 424K-728K NPS (2x+ over 330K baseline, 6x over 120K minimum)
- **Strength**: 1628 ELO (exceeds 1400 minimum requirement)
- **Stability**: All major illegal move bugs fixed, zero allocations in hot paths

### ✅ Recent Completions
- [x] **SEE Implementation**: Complete Static Exchange Evaluation with capture ordering ✅
- [x] **Move Generation**: Fixed all sliding piece illegal move wrap bugs ✅
- [x] **Search Foundation**: Dynamic LMR, IID, null-move, quiescence pruning ✅
- [x] **Performance**: Achieved target 2x speed improvement ✅

## Immediate Priorities (Next 2-3 commits)

### 🧪 **PRIORITY 1: Testing Infrastructure**
*Should have done this before SEE - let's fix it now*

1. **Benchmark Harness** (1-2 days)
   - [ ] Create deterministic benchmark CLI tool with fixed FEN corpus
   - [ ] Report: NPS, nodes, depth, time for 100-500 mixed positions
   - [ ] Automated regression detection (>5% NPS drop = fail)
   - [ ] Export results to CSV/JSON for tracking

2. **Tactical Test Suite** (1 day) 
   - [ ] Add EPD-based tactical test framework
   - [ ] Include 50-100 tactical positions with expected moves
   - [ ] Automated pass/fail scoring (currently: 12/100 tactical strength)
   - [ ] Performance tracking: solve time vs. depth

3. **Validation Framework** (1 day)
   - [ ] Perft validation for move generation correctness
   - [ ] Position evaluation regression tests
   - [ ] Memory leak detection in long searches

### 🚀 **PRIORITY 2: Move Ordering Optimization** 
*Address tactical weaknesses revealed by comprehensive eval*

4. **Single-Pass Move Ordering** (2-3 days)
   - [ ] Replace multi-scan ordering with single-pass classifier
   - [ ] Better SEE integration: order captures by SEE value
   - [ ] Optimize quiet move selection using history/killers
   - Expected: 15-25% node reduction, better tactical play

### ⚡ **PRIORITY 3: Search Speed Improvements**

5. **isInCheck Optimization** (1-2 days)
   - [ ] Replace per-step loops with ray masks + bit scans  
   - [ ] Target: reduce 16-18% CPU hotspot to <10%
   - [ ] Keep pawn/knight/king checks as bitboard operations

6. **TT Atomic Overhead** (1 day)
   - [ ] Add single-thread non-atomic TT mode for benchmarks
   - [ ] Measure impact of atomic operations (currently 12-20% CPU)

## Performance Targets & KPIs

### Current Metrics (Baseline)
- **NPS**: 424K-728K (excellent)
- **Tactical**: 12/100 (needs improvement)
- **Material**: 100/100 (perfect) 
- **Positional**: 33/100 (needs work)
- **Endgame**: 66/100 (good)

### Target Improvements
- **NPS**: Maintain 400K+ (no regressions)
- **Tactical**: 40+ / 100 (3x improvement via move ordering)
- **Node Reduction**: 20-30% via better ordering
- **Search Depth**: +1 ply average at fixed time

## Future Features (After Testing Infrastructure)

### Phase A: Advanced Search (3-5 days)
- [ ] Aspiration windows improvements  
- [ ] Check extensions refinement
- [ ] Futility/razoring in main search
- [ ] PV-node vs non-PV optimizations

### Phase B: Evaluation Enhancements (2-4 days)
- [ ] Piece-square table tuning based on game phase
- [ ] Pawn structure improvements (chains, storms, weakness)
- [ ] King safety refinements 
- [ ] Piece coordination bonuses

### Phase C: Opening/Endgame (3-6 days)
- [ ] Opening book integration (stub exists)
- [ ] Basic endgame tablebase support
- [ ] Specialized endgame evaluation

## Development Guidelines

### Performance Standards
- **Never regress NPS >5%** without explicit approval
- **Run `make test` after every change** for regression detection
- **Use `make eval` sparingly** - expensive 2-minute comprehensive evaluation

### Code Quality
- Zero allocations in hot paths (search, movegen, eval)
- Crash handlers wrap all UCI and search entry points  
- Comprehensive testing for move generation correctness
- Clear TODO tracking and commit discipline

### Testing Workflow  
1. `make dev` - format, test, build cycle
2. `make test` - regular performance monitoring
3. `make eval` - milestone comprehensive evaluation
4. Performance regression = immediate revert and analysis

## Notes

**Why Testing First**: The recent eval showed tactical weaknesses (12/100) that proper testing infrastructure would have caught earlier. Move ordering improvements will likely fix this, but we need systematic testing to validate.

**Current Strengths**: Material evaluation (100/100) and performance (728K NPS) are excellent. Engine has solid foundation.

**Risk Areas**: Tactical play and move ordering. The engine finds good moves but not always the *best* moves in tactical positions.
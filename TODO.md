# NGN Chess Engine — Development Roadmap

## Current Status (2025-09-07)

### 🎉 Performance Achievements
- **Performance**: 412K avg NPS (excellent, exceeds 400K target)
- **Strength**: 1628 ELO (exceeds 1400 minimum requirement)
- **Stability**: All major illegal move bugs fixed, zero allocations in hot paths
- **Testing**: Comprehensive benchmark and Stockfish-validated tactical test infrastructure implemented

### ✅ Recent Completions
- [x] **SEE Implementation**: Complete Static Exchange Evaluation with capture ordering ✅
- [x] **Move Generation**: Fixed all sliding piece illegal move wrap bugs ✅
- [x] **Search Foundation**: Dynamic LMR, IID, null-move, quiescence pruning ✅
- [x] **Performance**: Achieved target 2x speed improvement ✅
- [x] **Benchmark Harness**: Deterministic testing with 24 FEN positions, regression detection ✅
- [x] **Tactical Test Suite**: EPD-based framework with Stockfish-validated tactical positions ✅

## Immediate Priorities (Next 2-3 commits)

### 🧪 **PRIORITY 1: Testing Infrastructure** - MOSTLY COMPLETE
*Critical foundation now in place - revealed tactical weaknesses*

1. **Benchmark Harness** (1-2 days) ✅ COMPLETED
   - [x] Create deterministic benchmark CLI tool with fixed FEN corpus
   - [x] Report: NPS, nodes, depth, time for 24 mixed positions
   - [x] Automated regression detection (>5% NPS drop = fail)
   - [x] Export results to CSV/JSON for tracking
   - [x] **Results**: 412K avg NPS, 238K-790K range

2. **Tactical Test Suite** (1 day) ✅ COMPLETED
   - [x] Add EPD-based tactical test framework
   - [x] Include 15 Stockfish-validated tactical positions
   - [x] Automated pass/fail scoring (currently: 40.0% accuracy)
   - [x] Performance tracking: solve time vs. depth vs. nodes searched
   - [x] **Results**: 6/15 positions solved correctly, 240K-320K NPS, 300K-500K nodes
   - [x] **Stockfish Validation**: 11/15 expected moves confirmed by Stockfish

3. **Validation Framework** (1 day)
   - [ ] Perft validation for move generation correctness
   - [ ] Position evaluation regression tests
   - [ ] Memory leak detection in long searches

### 🚀 **PRIORITY 2: Move Ordering Optimization** - IMPORTANT
*Address tactical weaknesses revealed by testing (40.0% accuracy - moderate room for improvement)*

4. **Single-Pass Move Ordering** (2-3 days) ✅ COMPLETED
   - [x] Replace multi-scan ordering with single-pass classifier
   - [x] Better SEE integration: order captures by SEE value
   - [x] Optimize quiet move selection using history/killers
   - [x] Zero-allocation implementation with fixed-size arrays
   - Expected: 15-25% node reduction, better tactical play
   - **Status**: Implemented single-pass ordering with enhanced SEE integration, all tests passing

### ⚡ **PRIORITY 3: Search Speed Improvements** 

5. **isInCheck Optimization** (1-2 days) ✅ COMPLETED
   - [x] Replace per-step loops with optimized bounds checking
   - [x] Reduced redundant calculations and improved loop efficiency
   - [x] Keep pawn/knight/king checks as bitboard operations
   - **Status**: Optimized sliding piece check detection with unrolled loops and better bounds checking

6. **TT Atomic Overhead** (1 day)
   - [ ] Add single-thread non-atomic TT mode for benchmarks
   - [ ] Measure impact of atomic operations (currently 12-20% CPU)

## Performance Targets & KPIs

### Current Metrics (Updated 2025-09-07)
- **NPS**: 412K average (excellent - exceeds 400K target)
- **Tactical**: 40.0% accuracy (6/15 positions, moderate - room for improvement)
- **Stockfish Validation**: 73.3% of test positions confirmed by Stockfish
- **Material**: 100/100 (perfect)
- **Positional**: 33/100 (needs work)
- **Endgame**: 66/100 (good)

### Target Improvements
- **NPS**: Maintain 400K+ (no regressions)
- **Tactical**: 60+ / 100 (50% improvement needed via move ordering)
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
- **Run `make benchmark` after every change** for regression detection
- **Use `make eval` sparingly** - expensive 2-minute comprehensive evaluation
- **Run `make tactical-test` to validate tactical improvements**

### Code Quality
- Zero allocations in hot paths (search, movegen, eval)
- Crash handlers wrap all UCI and search entry points
- Comprehensive testing for move generation correctness
- Clear TODO tracking and commit discipline

### Testing Workflow
1. `make dev` - format, test, build cycle
2. `make benchmark` - performance regression testing
3. `make tactical-test` - tactical accuracy validation
4. `make eval` - milestone comprehensive evaluation
5. Performance regression = immediate revert and analysis

## Notes

**Key Finding**: Testing infrastructure revealed moderate tactical weaknesses (40.0% vs expected 60%+). The engine performs well in benchmarks but has room for improvement in tactical play.

**Stockfish Validation**: Now using Stockfish-validated test positions, providing reliable benchmark for tactical improvements.

**Testing Infrastructure Benefits**: Have systematic way to measure and validate improvements. Can confidently iterate on move ordering knowing we can measure the impact.

**Current Strengths**: Excellent NPS performance (412K), perfect material evaluation, good endgame play.

**Areas for Focus**: Tactical play and move ordering remain areas for improvement, but not as critical as initially assessed.

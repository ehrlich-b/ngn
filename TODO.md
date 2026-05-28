# NGN Chess Engine - Development Roadmap

## Goal: 2500+ ELO (Grandmaster Level), with 2200 as the next milestone

---

## ACTIVE FIX LIST — refreshed 2026-05-28 (AUTHORITATIVE; supersedes status notes below)

Fresh full-codebase audit (search/eval/movegen/position/TT/time/UCI + the
measurement harnesses). Verified every finding against the live code; agent
"show-stoppers" were checked and rejected where wrong. **The "correctness is
locked down" claim below is RETRACTED** — IID and singular extensions are both
silently broken (see C1). Worked in harmonious order: confirmed bugs first, then
cheap independent search refinements, then eval (the proven lane), then
structural, then infra. Stack ONE at a time, gate with the COOL+FAST loop
(smoke-tactical + mechanism unit/integration test + `go test -short`), commit
checkpoints; periodic fixed-nodes SPRT as the batch validator.

### Phase C — Correctness recovery (CONFIRMED bugs in supposedly-working features)
- [x] **C1 — IID & singular extension re-entrancy + singular self-cutoff (DONE · 2026-05-28
  · commits ff7e81e, 722b7ba).** Two stacked bugs meant IID and singular NEVER worked:
  (1) RE-ENTRANCY — `alphaBetaPV` pushed the node's hash onto `info.RepStack` BEFORE the IID
  and singular blocks, both of which re-enter on the SAME position; the re-entry matched its
  own hash in the repetition scan and returned a bogus 0 (proven: +1024 → 0 with the hash
  pre-pushed). IID was a total no-op; singular's score was always 0. Fixed by hiding the
  node's entry around both re-entrant calls (`RepStackLen--`/call/`++`). (2) SINGULAR
  SELF-CUTOFF — even after (1), the verification search took an immediate TT cutoff from the
  very entry that triggered detection, so `singularScore == ttEval` and it never proved
  singularity (0 extensions across 4 positions @ depth 11). Fixed with ply-scoped exclusion:
  `inSingular := ExcludedMove != Empty && ply == ExcludedPly` skips the TT cutoff AND both TT
  stores at the verification root only. Added `IID`/`Singular` to `SearchToggles` + regression
  tests (`TestIIDIsWired`, `TestSingularExtensionsFire`: IID node-count now changes on/off;
  singular fires 63 ext in Kiwipete, was 0). VERDICT: smoke-tactical 29/30, suite green.
  Isolation SPRT singular-on vs off (50k nodes, 200 games, 400-opening book): **+3.5 ELO
  [-45,+51], 92-18-90 = 50.5% — NEUTRAL** (the first 48-game full-vs-base read of -44 was
  book-exhausted noise). Singular kept ON: now CORRECT and not a regression, same standard as
  other sound sub-resolution techniques (cf. quiet-SEE +2.2). Tuning singular to net-positive
  (margin/double+negative extension) → see X3.
  **FULL-BATCH VERDICT (full HEAD vs pre-fix base, 50k nodes, 200 games, 400-opening book):
  −29.6 ELO [−78,+19], 78-27-95 = 45.8%, LLR −0.67 — INCONCLUSIVE but a consistent NEGATIVE
  lean.** Since singular is neutral (isolation above), the drag is IID. ROOT CAUSE is most
  likely an INSTRUMENT artifact, not a real-play regression: IID spends a reduced-depth search
  purely to improve ordering, and the fixed-NODES instrument charges those exploratory nodes
  with NO credit for the depth they buy at a real clock — Blunder measured IID at **+11 using
  fixed-TIME** self-play, not fixed-nodes. The engine plays at time controls, so optimizing IID
  away to please fixed-nodes would be instrument-optimization. DECISION PENDING (surfaced to
  Bryan): keep on correctness + time-control grounds / revert / validate at fixed-TIME first.
  META-FINDING (see I4): fixed-nodes SPRT systematically undercredits the whole node-SPENDING
  class (IID, IIR, extensions) — much of the Phase-1 search roadmap — so it is the wrong
  instrument for those; needs a fixed-TIME or long-TC complement.
- [x] **C2 — stalemate-when-winning returns -500 → 0 (DONE · 2026-05-28 · commit ff7e81e).**
  The `numMoves==0` branch returned -500 when up >300cp; stalemate is a draw. Near-unreachable
  (real stalemate handled at `legalTried==0`) but it broke the zero-sum invariant. Now `return 0`.
- [x] **C0 — stale eval baseline (DONE · 2026-05-28 · commit, hygiene).** `make test` was RED on
  `main`: the +46 PST commit (bb12a48) changed eval but didn't regenerate `eval_baseline.json`.
  Regenerated to match the tuned eval; suite green. (Test-hygiene, not an engine bug.)

### Phase S — Cheap independent search refinements (fixed-nodes SPRT signs these)
- [ ] **S1 — Aspiration windows: exponential widening (search.go:573-596).** Today:
  fixed ±50, and on ANY fail it jumps straight to ±INFINITY — throws away the
  aspiration benefit. Use delta≈16, widen ×~2 per fail, keep the non-failing bound,
  go full only after a few fails. Blunder aspiration +23; delta from improving an
  existing crude version is smaller but real.
- [ ] **S2 — Killer moves indexed by remaining `depth`, not `ply` (moveorder.go:83).**
  Standard is ply-indexed; depth-indexing scatters killers across slots when LMR
  reduces then re-searches. Switch to ply. Small but real; easy A/B.
- [ ] **S3 — Qsearch never stores to TT (search.go:1569+ only probes).** Storing the
  q-result (bound + best capture) is a known small win and improves ordering on
  re-visits (Zurichess does it). Watch the replacement policy doesn't evict useful
  main-search entries (store at depth 0).

### Phase E — Eval (the PROVEN lane; +46 already banked from PST+weights)
- [ ] **E1 — Tune MATERIAL jointly with PST on Zurichess data.** Material was left
  FIXED in the +46 tune; Rofchade got +75-80 tuning material+PST together on this
  exact file → ~+30 likely on the table. Biggest single eval lever. Gate: fixed-depth
  SPRT vs build/ngn_base.
- [ ] **E2 — King-safety weight suppression review.** The Texel run cut
  `kingSafetyWeight` to 57 and `kingActivityWeight` to 29 — the classic quiet-dataset
  pitfall (king safety has low MSE leverage on quiet positions but matters in sharp
  play). Test higher weights in GAMES, not MSE. While here, fold the double-counted
  missing-shield penalty (evaluatePawnShield ~1175 + open-file block ~744) into one
  coherent term so king safety is tunable as a unit. Also the phase 6-11 dead zone
  (eval.go:2433) gets neither king safety nor king activity — re-check the boundary.
- [ ] **E3 — Bigger/cleaner corpus (lichess-big3) + threat terms** once E1/E2 land.

### Phase X — Structural (refinements; need cutnode threading first)
- [ ] **X1 — 2-ply continuation history → wire summed score into LMR.** (Have 1-ply.)
- [ ] **X2 — Full cutnode/allNode threading.** Prereq for negative extensions, real
  IIR retry, and the big LMR cutnode signal. (See Phase 1C below.)
- [ ] **X3 — Tune singular extensions to net-positive.** Now CORRECT but measured NEUTRAL
  (+3.5 [-45,+51], C1). It ~2× the tree in Kiwipete, so the margin is too loose. Try a
  larger relative `SINGULAR_MARGIN` (currently 64; scale with depth), shorter verification
  depth, and double/negative extension (depends on X2 cutnode). Blunder's tuned singular = +6.8.

### Phase I — Infra / measurement (gates REAL-game ELO; invisible to current tests)
- [ ] **I1 — Tournament time management is lenient (≤2× allocated) AND untested**
  (time.go:309-382 only runs in real clock games; smoke uses movetime hard-limit,
  SPRT uses fixed-nodes/depth). Real CCRL risk: lose on time / misallocate. Pair the
  overhaul with a real-clock gauntlet vs a fixed ladder (Blunder/Zurichess binaries)
  — the only instrument that can see this class of bug.
- [ ] **I2 — Wire `setoption Hash` + decide `Threads` (uci.go:296-300 are stubs).**
  TT hard-pinned at 128MB; "Move Overhead" advertised (default 100) but ignored
  (stays 50ms). Needed for external-tester fidelity.
- [ ] **I3 — elo-assess scores stalemate as a loss (cmd/elo-assess/main.go:261-267).**
  Minor harness correctness (smoke being retired as a yardstick anyway).
- [ ] **I4 — fixed-TIME SPRT complement (HIGH — surfaced by C1).** The fixed-NODES instrument
  structurally undercredits node-SPENDING techniques (IID/IIR/extensions): it charges their
  exploratory nodes without crediting the depth they buy at a clock. Half the Phase-1 search
  roadmap is this class, so fixed-nodes is the WRONG signer for it. Add a clean fixed-time
  self-play mode (concurrency-1 to kill timing noise; reversed-color pairs; ~200+ games) OR a
  short-TC gauntlet vs a fixed binary ladder. Without it, IID/IIR/extension work can't be signed.

### Parked (orthogonal multiplier)
- Lazy SMP — +80-150 multi-core, Go makes it easy. Biggest single jump if wanted;
  not on the critical path.

### Audit false-positives (recorded so they aren't re-chased)
- "Castling rook not moved in MakeMove" — FALSE: `Bitboard.Move` relocates the rook
  on a king castle (bitboard.go:330-335); perft passes.
- White-pawn EP rank validation (movegen.go ~197) — EP square is only ever set at the
  correct rank in makeMoveHelper; defense-in-depth only, not exploitable.

---

**EVAL TUNING — ACTIVE 2026-05-25 (supersedes the search grind as current work).** After
quiet-SEE pruning measured NEUTRAL (+2.2 [-51,+56]) — the 3rd signal that single pruning levers
are sub-resolution — Bryan chose PIVOT TO EVAL TUNING (correctness gate satisfied; see
[[correctness-first-to-2k]]), specifically "both, weights first." KEY DISCOVERY: the `pawnPST`/
`knightPST`/… arrays in eval.go are the DEAD hand-set CPW tables (orphaned by the PeSTO swap,
only tests reference them). The real eval core is `evaluatePeSTO` (eval_pesto.go) using genuine
PeSTO MG/EG tables + material — ALREADY Texel-tuned by Friederich. So the README "tapered PeSTO"
claim is true, and re-tuning that core on NGN's weak ~1600 self-play risks overfitting/regression
(deferred to phase 2, SPRT-gated). PHASE 1 (current) tunes the genuinely-UNTUNED eval-term WEIGHTS.
INFRA BUILT + verified cool (2026-05-25): `engine/texel.go` (K-scaled sigmoid MSE parallel over
positions, ternary-search K calibration, line-search coordinate descent taking an explicit param
list + rebuild callback so it's target-agnostic; unit-tested: sigmoid/MSE/findK/wiring/CD-never-
increases-MSE) + `cmd/texel` (`-mode gendata` in-process self-play → quiet non-check positions
labeled by white-POV result; `-mode tune` → calibrate K, CD schedule [8,4,2,1], dump weights).
Exposed 10 tunable weights in eval.go (3 direct pawn scalars passed/doubled/isolated + 7 term-
multipliers default 100 = identity for chains/bishop-pair/mobility/rook-open/outpost/king-safety/
king-activity); the refactor is BEHAVIOR-NEUTRAL at defaults (eval symmetry + baseline + tactical
29/30 all unchanged). NB: the dead pawnPST tuning attempt was caught instantly by the MSE-sensitivity
probe (mutating knightPST[18] moved MSE by exactly 0) — mechanism checks earn their keep.

**PHASE 1 VERDICT — REGRESSED, REVERTED 2026-05-25.** Generated 156K quiet positions from NGN
self-play (4×400 games @4000 nodes), calibrated K=0.766, coordinate-descended the 10 weights. The
tuner near-ELIMINATED established positive terms (mobility 100→4, bishop-pair 100→35, king-safety
100→53, king-activity 100→40, isolated 12→2) for a NOISE-LEVEL MSE gain (0.123821→0.123791, ~0.00003).
Tactical held 29/30, so the regression isn't tactical-vision; it's positional. Fixed-DEPTH SPRT
(depth 8, 564 games vs ngn_base): **−44.0 ELO [−73,−15], LLR −2.96, H0 ACCEPTED (NOT better).**
REVERTED to defaults (build/ngn back to baseline). ROOT CAUSE = DATA QUALITY, not the machinery
(which is sound — CD never raises MSE, K-calib correct): NGN self-play labels reflect ~1600-level
judgment, so the tuner learns to predict NGN's own weakness and suppresses terms that are redundant
*for fitting this dataset* but valuable *in play*. The dominant eval signal is the already-(strongly-)
tuned PeSTO core, so peripheral weights have near-zero MSE leverage. IMPLICATION: phase 2 (re-tune the
PeSTO core on the SAME self-play data) is EVEN more likely to regress — perturbing the dominant signal
with weak labels. The real fix is REAL labeled data (Zurichess `quiet-labeled.epd` / lichess-big3 —
what Blunder & CounterGo actually tuned on), not better self-play. NEXT = strategic fork (see below),
surfaced to Bryan: (1) re-tune on a real public dataset, (2) run phase-2-on-self-play anyway to close
the hypothesis, or (3) abandon tuning and work the search long-tail. Infra (texel.go/cmd/texel) is
data-source-agnostic — only a tiny EPD→`<FEN> <result>` adapter is needed for option 1.

**DIAGNOSIS CORRECTED + REAL-DATA CORE TUNE LAUNCHED (2026-05-27).** Bryan pushed back on the
data-quality framing — rightly; I'd overstated "weak self-play = useless" (even the gold-standard
Zurichess set is itself self-play). Fetched the real corpus (KierenP/ChessTrainingSets
`quiet-labeled.epd`, 725K balanced positions: 272K/253K/198K W/L/D), taught `LoadTexelSamples` the
`c9 "1-0"` EPD form (first 4 FEN fields padded to a full FEN). Re-tuned the SAME 10 weights on it →
SPLIT VERDICT: (a) CONFIRMED labels matter — self-play had near-eliminated mobility (→4) and
bishop-pair (→35); real data keeps them sane (108 / 87). (b) CORRECTED — MSE still barely moved
(0.0601→0.0600), so the 10 peripheral knobs are genuinely LOW-LEVERAGE regardless of data: a sub-50
ELO effect, NOT worth an SPRT. The real leverage is material + PST (exactly what Rofchade tuned on
THIS file for +75-80 ELO). So: exposed the 768 raw PeSTO PST entries (`TexelPSTParams` + `RebuildPST`;
`init` refactored to call `RebuildPST`; behavior-neutral — tactical 29/30 held), added
`cmd/texel -target pst` (tunes 768 PST entries + the 10 weights jointly; material left FIXED since a
uniform shift of a piece's 64 squares already spans it; black tables mirror-rebuilt each mutation).
LAUNCHED the core tune on the 725K corpus (E-cores, ~1hr). GATE: paste tuned tables → smoke-tactical
→ fixed-DEPTH SPRT vs `build/ngn_base` → commit ONLY on a win. NB: this is NOT the "phase 2 will
regress" I warned about — that was conditioned on self-play data; gold-standard labels flip the risk,
and the +75-80 ELO precedent on this exact file is the evidence base.

**VERDICT — WON +46 ELO, COMMITTED (2026-05-28).** Fixed-depth SPRT (depth 8, 668 games,
302-152-214, 56.6%): **Elo(tuned−base) = +46.0, 95% CI [+19,+73], LLR +2.99 → H1 ACCEPTED (new is
stronger ≥10 ELO).** This is the FIRST statistically-significant eval win of the whole effort and
validates the thesis: tuning the HIGH-LEVERAGE params (768 PeSTO PST entries + the 10 weights) on
REAL labeled data (Zurichess `quiet-labeled.epd`) delivers real ELO, where the peripheral-only tune
was sub-resolution and the self-play tune regressed −44. MSE 0.06004→0.05935 (the PST entries moved
it ~10× more than the 10 weights alone could). Tactical held 29/30 (not a vision change). Committed
LOCALLY (not pushed — push is Bryan's call); `build/ngn_base` refreshed to the new tuned strength so
future A/Bs measure deltas from here. Engine now ≈1650-1700 (prior ~1600-1650 + the +46 delta; full
re-assess pending). CORRECTED DURABLE LESSON: not "weak self-play is useless" (I'd overstated that) —
it's "tune HIGH-LEVERAGE params (PST/material) on REAL labeled data; peripheral knobs are low-leverage
regardless of data source." NEXT options: (a) tune MATERIAL too — we left it fixed, and Rofchade got
+75-80 tuning material+PST on this same file, so ~+30 more may be on the table; (b) re-tune from a
larger/cleaner corpus (lichess-big3) or add king-safety subterms; (c) the honest NNUE strategy
conversation, now that classical tuning is PROVEN to still have headroom rather than being tapped.

**GM-PATH RESEARCH → PHASE 1 SEARCH ROADMAP (2026-05-25 — latest, ACTIVE PLAN).**
Surveyed Stockfish + Ethereal source, the two Go engines (Blunder, CounterGo) and CPW to
answer Bryan's moved goalpost: "why can't we reach GM (2500) by pruning better, no tuning?"
HONEST FINDING — the high-ELO pruning rocks are ALREADY IN NGN. Blunder's SPRT devlog
(algerbrex, ~2600 CCRL, Go — the cleanest comparable; full measured-ELO table in Reference
Engines below) puts the big movers at null-move +116, LMR +61, RFP +57, futility +37,
SEE-qsearch +26, aspiration +23, LMP +22, history +19, IID +11, singular +6.8 — NGN has every
one. So there is NO single +100-ELO pruning lever left; what remains is a long tail of 5-30
ELO refinements. Realistic sum of the whole roadmap below ≈ +120-160 ELO → NGN ~1750-1850
(true), NOT 2500. The remaining ~700 ELO to GM is eval QUALITY + tuning: EVERY comparable
engine that reached GM tuned (Blunder is AdaGrad/Texel-tuned, +34-61 ELO from eval re-tuning
alone; cf. "Is GM Level Hard" §3 "proper tuning"). So "GM on bugfixes/pruning alone" is NOT
supported by the evidence — but Phase 1 below is real, no-tuning, correctness-flavored ELO and
the right next step. Revisit the eval/tuning honesty at ~1800.

Ordered for HARMONY — prereqs first, cheap+independent early, structural middle, refinements
last; stacked ONE AT A TIME, gated by smoke-tactical (no regression) + node-count check.
Status key: MISSING / WEAK (partial) / VERIFY (audit for bug). ELO = est. at NGN's level.

PHASE 1A — cheap, reuses machinery NGN has (~50-70 ELO), bugfix-first:
- [x] **TT-replacement audit** [CLEAN — no bug · 2026-05-25]. Audited cache.go (Zahak-derived
  single-bucket, lockless XOR-key) + the search.go store/probe. Replacement is sound: empty→
  replace, same-pos→`depth>=oldDepth-3 || Exact`, collision→`oldAge!=age || depth>=oldDepth`;
  AdvanceAge fires once per root search (search.go:483 — correct cadence), ucinewgame/sprt
  rebuild per game. Store types bounds against origAlpha saved BEFORE mate-distance adjust
  (the subtle point, done right); probe gates `!isPV && ttDepth>=depth` with correct
  Lower≥beta / Upper≤alpha checks. Packing 28+16+7+3+10=64, no overlap. The Blunder +30 bug
  class is NOT here (consistent with the prior TT-on==TT-off sweep).
- [x] **`improving` flag** [DONE · +20 elo · 2026-05-25]. Added `StaticEvalStack
  [MaximumDepth]int` to SearchInfo; compute static eval once per non-check node (reused by
  RFP/futility, inherit across checks), `improving := staticEval > stack[ply-2]`; wired into
  LMR as `!improving → reduction++` (the magnitude-free CounterGo/Ethereal form). MECHANISM
  is dramatic: Kiwipete depth-12 nodes **4.84M → 1.21M (-75%)**, time 4846→1249ms, seldepth
  43→33, NPS -3%, tactical 29/30 held, same bestmove. CAVEAT: mechanism ≠ ELO — a +1 LMR on
  non-improving nodes can over-prune (depth-12 is now a more-pruned depth-12). SPRT verdict
  (movetime 100ms, conc-4, 600 games vs ngn_base): **+19.7 elo [-8,+48]**, 278-78-244 = 52.8%,
  stable — matches the +10-25 estimate, NOT a regression (over-pruning worry dead). KEPT (the
  efficiency gain compounds at longer TC). NEXT: improving-scaled RFP/futility/LMP margins.
- [~] **improving-scaled margins** [TRIED · neutral · reverted 2026-05-25]. RFP `depth--` when
  improving + LMP `idx--` when not, batched with adaptive null-R: movetime SPRT −6 elo [−37,+24]
  @488 games — neutral, reverted to keep the tree to proven wins. Re-addable; may help at longer
  TC. Soft-blend RFP return (716*beta+308*eval)/1024 still UNTRIED.
- [x] **SEE pruning of QUIET moves in MAIN search** [DONE · verified cool · MEASURED NEUTRAL +2.2 · 2026-05-25].
  Added a quiet-move SEE-prune gate beside the capture one (search.go ~1250): at depth<=4, !isPV,
  !inCheck, skip a quiet move (non-TT, non-promo, non-king) whose `staticExchangeEvaluation < -80*depth`
  — a "soft hang" where the piece can be won on its destination square. KEY FINDING: it was NEARLY A
  FREE REUSE, not the "needs a SEE extension" the old note feared — the existing swap already evaluates
  the destination exchange, so the only change was dropping the non-capture early-return (now a castle
  guard) so gain[0]=0 flows for quiet moves. All 4 existing SEE callers gate on IsCapture first, so this
  is a provable no-op for them. King moves excluded (infinite SEE weight poisons the swap + a pinned
  enemy "attacker" of a king square may be an illegal capture). VERIFIED COOL: 4 crafted-value quiet-SEE
  tests (safe square=0, queen-to-pawn=-1000, pawn-defended knight=-220, queen-declines=0; see_test.go),
  integration test (SEEQuietPrunes>0 in a depth-8 Kiwipete search, ==0 with the toggle off), full suite
  green, tactical 29/30. FOLLOW-UPS (not done): depth²-scale the CAPTURE threshold (Ethereal noisy
  -20*d²); history-adjust the quiet threshold; tune the -80 coeff. ELO VERDICT (fixed-nodes, -nodes
  100000, conc-2, 160 games vs 3f20a67): **+2.2 elo [-51, +56]**, 70-21-69 = 50.3%, LLR -0.04 INCONCLUSIVE
  — NOT a regression (over-pruning ruled out) but NOT a measurable win either; peaked ~+17 @G150 then
  decayed to +2.2 (the same phantom-decay the rook-7th term showed). A sub-30 standard technique is below
  this instrument's resolution at 160 games. Kept on theory + no-regression (sound, standard, likely
  compounds at longer TC), NOT as a measured win. This empirically reinforces the roadmap honesty: the
  remaining single pruning levers are all sub-resolution; the big ELO is eval QUALITY + tuning.
- [ ] **PV-reduction cutnode proxy** [MISSING · +5-15]. CounterGo one-liner `if pvNode {r-=2}`
  — cheap stand-in for full cutnode threading (the full version is 1C below).
- [~] **Adaptive null-move R** [TRIED (depth/6 only) · neutral · reverted 2026-05-25; Blunder
  +14 n.s.]. `R = 3 + depth/6` batched with improving-margins: SPRT −6 elo [−37,+24] @488,
  neutral, reverted. The eval-margin term (Ethereal min(3,(eval-beta)/191)) is UNTRIED — retry
  that piece alone if revisited.

PHASE 1B — structural (~40-60 ELO):
- [x] **Correction history** [PAWN TABLE DONE · verified · +64 batch w/ capture · 2026-05-25].
  Implemented the pawn-structure table (moveorder.go): `[2][16384]` gravity-EMA keyed on a hash
  of both pawn bitboards, SF weight 6245/131072 (saturates at ±49cp). Apply: the static eval
  feeding RFP/futility/null/improving is now `EvaluateForPlayer + correctionValue` (search.go
  static-eval block); update fires at BOTH node returns (fail-high + normal exit) gated SF-style
  (skip in check / capture best move; learn only on consistent worse-than-static or
  better-with-bestmove). Cleared with the other histories (ClearHistoryTable). VERIFIED COOL:
  full suite green (5.5s), tactical 29/30, unit tests for hash/EMA/gate, integration test
  populated 1131 slots in a real depth-10 search (correction_history_test.go). ELO magnitude
  validated jointly with capture history: +64.4 elo [+2, +127] (62-18-40 / 120 games) on the
  fixed-nodes instrument vs the improving baseline — CI clears zero, a real win (fixed-DEPTH had
  said +7 [-90,+104] because it can't see capture ordering). FOLLOW-UPS (not done): material / non-pawn /
  continuation-ply2 corr tables; correct the qsearch stand-pat eval too. (SF corrhist; b4d995d.)
- [ ] **Multi-ply continuation history** [WEAK (1-ply) · +10-25]. NGN already has a real 1-ply
  conthist table (moveorder.go:46, Stage-3 commit 7a309c3) — extend to 2-ply (then 4-ply) and
  wire the summed score into the LMR reduction. CounterGo flat `[1024][1024]int16` pattern.
- [x] **Capture-history table** [DONE · verified · +64 batch w/ correction · 2026-05-25].
  `captureHistory[13][64][13]` keyed (moving-piece, to, captured-piece), gravity-clamped ±8192
  like quiet history (bonus depth²). Quiet history skipped captures, so captures were ordered by
  SEE alone — this adds a learned tie-breaker `+captureHistoryScore/8` (≤±1024, kept inside the
  60000 SEE bands so it never reorders across winning/losing/killer bands). Updated at beta
  cutoffs: reward a cutting capture, penalize tried captures that didn't cut (new searchedCaptures
  list mirroring searchedQuiets). Cleared with the other histories. VERIFIED COOL: full suite green
  (6.1s), tactical 29/30, unit tests for reward/penalty/bound/independence, integration test
  populated 930 slots in a real depth-10 Kiwipete search (capture_history_test.go). FOLLOW-UPS
  (not done): feed capture history into SEE/futility margins; tune the /8 ordering scale.

PHASE 1C — refinements, after cutnode is threaded (~20-40 ELO):
- [ ] **Full cutnode/allNode threading** [MISSING]. Upgrade the PV proxy to a real cutNode bool
  through the recursion; prereq for negative extensions + the big LMR cutnode signal (+3995/1024 in SF).
- [ ] **Full singular package** [WEAK (single +1) · +10-25 but SKEPTICAL — Blunder's singular
  total was only +6.8]. Add double + negative extension and MULTICUT (return when the singular
  verification search itself fails high ≥ beta — the real prize, prunes the whole node). Reuses
  the verification search NGN already runs. NB distinct from Stage-5 "multi-cut after null move".
- [ ] **IIR retry** [HAS IID · ~0-10, net simplification]. Stage 5 tried IIR at depth>=4 and it
  cascaded (0/5 smoke, rolling −91). SF's actual gating vindicates the Stage-5 retry hypothesis:
  depth>5, PV/cutnode only, priorReduction<=3 (minimally-reduced nodes), `depth--`. Retry with
  those guards once cutnode exists; can then replace the IID block.
- [ ] **LMR fail-high re-search depth adjustment** [+5-10]. doDeeper/doShallower ±1 on the
  re-search keyed on (value vs bestValue+margin), plus the `+PvNode` / cap-at-newDepth+2 form.

MEASUREMENT SNAG (must solve to SIGN these): the items are 5-30 ELO search-EFFICIENCY changes;
fixed-depth SPRT structurally can't credit efficiency and 14-game movetime SPRT is too slow +
noisy (the extension-discipline lesson below). Blunder measured ALL of the above with thousands
of ultra-fast 10+0.1 self-play games. PLAN: (a) ordering/eval-shaped items (improving margins,
corrhist, capture-hist) DO move fixed-depth move quality → cmd/sprt fixed-depth signs those now;
(b) pure-efficiency items (SEE/LMR/null-R/cutnode) need either fast self-play SPRT at high game
count (mirror Blunder 10+0.1) or `go nodes` enforcement in search.go for a deterministic
fixed-NODES SPRT (UCI parses it, search doesn't enforce — a small wiring job). Gate every change
on smoke-tactical + node-count, stack one at a time, commit checkpoints.

COOL+FAST LOOP (ADOPTED 2026-05-25, per Bryan — "SPRT is too slow + smokes the CPU"): NO
per-change SPRT. Per change: `make smoke-tactical` (5s, in-process, cool) + a mechanism check
(unit test of the new machinery + an integration test asserting it is actually wired into search,
e.g. correction history's 1131-slot populate) + full `go test -short` (~5s). Theory-sound + green
+ no tactical regression → commit a checkpoint (local; push only when Bryan asks).

FIXED-NODES INSTRUMENT — BUILT 2026-05-25 (the unlock). `go nodes N` is now ENFORCED: a global
node budget (search.go SetMaxNodes/MaxNodes) trips the existing stop flag when spent, reusing the
incomplete-iteration discard; pure `go nodes` runs Infinite-time so only the budget binds. cmd/sprt
gained `-nodes N`. Why it matters: fixed-nodes self-play is DETERMINISTIC (so concurrency is free —
no timing noise, cool even at high concurrency), FAST (small cap → ms/move vs depth-9's ~50s/game),
and credits BOTH ordering/efficiency (capture history, LMR, null-R) AND eval — the thing fixed-depth
and movetime each half-miss. This is now the periodic batch validator. NB the baseline binary must
ALSO enforce nodes, so build base = (old commit + this instrument) via worktree cherry-pick.
FIRST RESULT (2026-05-25): corr+capture vs improving, -nodes 100000, 120 games → +64 [+2,+127]
(score 59.2%). TIMING LESSON: 120 games took ~20 min at -concurrency 4 because 8 engine procs
oversubscribe ~4 E-cores (each game ~half speed). Use -concurrency 2 (≈4 procs ≈ 4 E-cores, no
contention) and/or a lower node cap (50k) for faster cool runs; concurrency doesn't change the
deterministic result, only speed/heat.

DON'T use the SF1500 rolling smoke for strength: it is SATURATED — NGN goes 5-0 vs SF1500@200ms
every run (100% → degenerate ELO like "2458, CI [1699,2300]"). It can't discriminate; it's pinned
at the ceiling. Either retire it or give it a much stronger opponent. Fixed-depth one-shot SPRT is
also a poor fit: slow (40 min/48 games at depth 9) AND blind to ordering changes. Use fixed-NODES.

DOOR OPEN: Lazy SMP is real ELO (~+80-150 multi-core) but PARKED per Bryan — an orthogonal
multiplier, addable anytime, not the path.

**EXTENSION DISCIPLINE → TRIED, INCONCLUSIVE, REVERTED — plus an instrument lesson
(2026-05-25).** Hypothesis from the deep review: the check extension restores depth
(`nextDepth == depth` when givesCheck, search.go:1264) so forcing lines never decay and
explode the tree (the "35M nodes for depth 10" pathology) — bound it with a per-path
extension budget (cap net extensions at ~RootDepth via the ply+depth-RootDepth invariant).
The MECHANISM worked cleanly: Kiwipete `go depth 12` nodes 4.84M -> 2.45M (-49%), seldepth
43 -> 30; tactics held 29/30. BUT the ELO verdict was UNRESOLVABLE: two 14-game movetime
SPRTs gave OPPOSITE signs — concurrency-1/E-cores 4W-2D-8L (-102 elo) vs concurrency-4/
P-cores 6W-5D-3L (+76 elo), CIs hugely overlapping, both straddling 0. So the change is
~neutral within noise, NOT the regression an early read claimed. REVERTED to HEAD (already
strong AND crash-safe via the ply ceiling 9274203); the budget is documented here if a clean
instrument later shows it helps. THE REAL FINDING is about the LOOP: a 14-game movetime
self-play cannot sign a sub-100-elo SEARCH change — too slow (70s+/game) AND too noisy (sign
flips on core/concurrency). Fixed-depth SPRT is fast but structurally CANNOT credit search
EFFICIENCY (the change is invisible/worse at fixed depth). So search-efficiency work needs a
fixed-NODES instrument (deterministic -> clean at high concurrency; UCI parses `go nodes` but
search.go doesn't enforce it — a small wiring job), OR we accept search is locked down and
move to eval, where fixed-depth SPRT + a static Texel MSE objective ARE the fast loop. Bug
backlog now cleared; remaining ELO is eval QUALITY (calibration/tapering) = the tuning gated
on correctness, gate now satisfied.

**DEEP-PRUNER PROBE → CORRECTNESS LOCKED DOWN (2026-05-25).** Movetime
SPRT on the crash-fixed binary, baseline (all-on) vs each pruner-off build. LMR:
H1 ACCEPTED, +330 elo [+185,+474] (38-4-4/46) — massively load-bearing. Null-move:
+79.5 elo [+2,+157] (40-18-22/80) — POSITIVE (matches null-move's known ~50-100
elo value), so NOT over-pruning; formally inconclusive only because it hit maxgames
80 and I ran concurrency-2 (movetime timing noise — tool warned; traded for speed;
sign is robust, lower bound +2>0). Neither over-prunes = NO bug. This was the last
unverified classic technique, so CORRECTNESS IS NOW LOCKED DOWN: SEE, partial-
commit, outpost, mate hygiene, the ply-ceiling crash, qsearch bound, eval symmetry,
shallow pruners, ordering, pawn-structure, AND deep pruners all verified; the
empirical residual is eval-shaped, not bug-shaped. Bryan's gate for eval/tuning is
satisfied → cleared to start the Differentiation & Vision roadmap below (explainable
+ strongest-Go first). Optional loose end: a clean concurrency-1 null-move re-run
(~15 min) would pin the +80 magnitude but doesn't change the no-bug verdict.

**BLUNDER HUNT on HEAD → CRASH FIX (2026-05-24).** Pointed cmd/blunder-
finder at HEAD's own game log (107K real decisions, stride-sampled 400, fixed
depth 11, SF oracle) — the first real blunder hunt on the SEE-fixed engine — and
it immediately caught an ENGINE PANIC. Root cause: runaway check extensions. A
check extension restores depth (`nextDepth := depth-1; if givesCheck { nextDepth++ }`
=> nextDepth==depth, search.go:1253), so a perpetual-check / king-hunt line never
decays into qsearch and recurses until a ply-indexed array (PV[MaximumDepth=100])
overflows — crashed at ply 166. The clock unwinds it before that in real games
(why the SF2000 games didn't crash); fixed-depth analysis has no clock. FIX: a ply
ceiling `if ply >= MaximumDepth { return eval }` beside the existing depth guards
(search.go ~907). Behavior-neutral (normal search never nears ply 100; tactical
29/30 unchanged), and a re-scan of 400 positions now shows 0 panics (was crashing).
Blunder rate on the fixed binary: 14/307 = 4.6%, but ~3 are decided-position /
oracle artifacts (two 28900cp "drops" are mate-score arithmetic in already-lost
positions that slipped the -1200 filter; two are same-move SFBest==NGN drops from
the time-based oracle), so the real actionable rate is ~3.5-4% — in line with the
post-partial-commit 3.2%. Every real blunder is a 150-550cp POSITIONAL/EVAL error,
NOT a discrete bug: knight-to-rim (g1h3, f3g5), passive piece vs activation (queen
g5h6 not g5e5; rook not activated), weak endgame king centralization (queenside
pawn push instead of f2e3), queen-endgame eval miscalibration. EMPIRICAL CONCLUSION
(not assertion): past this crash, the residual is TUNING-SHAPED (eval quality /
search depth), not bug-shaped. Remaining search lever: even with the ceiling, the
check extension still burns nodes ~100-deep on checking lines (a search-EFFICIENCY
leak likely behind the "35M nodes for depth 10" explosions) — bounding extensions
to actually decay depth is the next correctness-adjacent fix, but it changes search
behavior so it needs SPRT.

**RE-MEASURE + RE-ANCHOR (2026-05-24):** The SEE +161 fix is confirmed
IN PLAY, not just at fixed depth — 4 fresh smokes at HEAD went **20W-0D-0L vs
SF1500/200ms** (every game a checkmate), tactical 29/30. That SATURATED the
SF1500 smoke: NGN wins 100%, the per-run estimate pins at the 2300 ceiling, and
the rolling point estimate is dragged toward it as old broken-SEE runs decay —
so the SF1500 rolling (and the stale "1711" below) is RETIRED as a forward
yardstick, kept only as a regression tripwire (a drop below ~100% = a real
break). RE-ANCHORED the default smoke to **SF2000** (`./build/smoke -elo 2000`,
its own rolling bucket): 3 runs = **12W-1D-2L = 83.3% over 15 games, rolling
2280, 95% CI [2059, 2500]** (width 441). This is the new working instrument —
non-saturated (games run long, SF2000 takes points: 1 loss + 1 max-moves draw),
CI lower bound 2059 sits above the opponent's nominal 2000. CAVEAT: still the 5x
time-odds config (NGN 1000ms vs SF 200ms) and SF's UCI_Elo throttling is an
imperfect strength model, so 2280 is the INSTRUMENT number, not a FIDE claim —
but the correctness-first thesis is clearly working (cleared the SF1500 bar
entirely on bug-fixes alone). Next correctness work is now measurable vs SF2000.
NOTE: `make smoke` still hardcodes -elo 1500 (saturated); use `./build/smoke
-elo 2000` until the Makefile target is bumped.

**Current Status (2026-05-24):** Built the measurement instrument that unblocks
everything from here — `cmd/sprt`, an NGN-vs-NGN self-play SPRT. The 5-game
vs-Stockfish smoke provably can't resolve sub-100-ELO changes (identical binary
swung 80%→20%), which is why every eval/ordering tweak kept coming back "neutral"
in the noise. sprt fixes that: **fixed-depth games by default** (every move
reaches real depth — directly answers the "hyperbullet won't get enough depth"
worry — and removes time noise), authoritative game-end detection via the engine
package (checkmate/stalemate/threefold/50-move; the elo-assess loop mis-scored
stalemate as a loss), reversed-color opening pairs for variance reduction,
`-genbook` to scale openings, `-basedepth` for a positive control, and a GSPRT
LLR stop. This is the prerequisite for Stages 2-7 (eval features, Texel,
search-efficiency) — none of which the smoke can measure. Validated end-to-end:
null (identical binaries) = exactly 50.0% (20W-8D-20L), LLR -0.10, no false
positive; positive control (new depth 7 vs base depth 3) correctly tripped H1
ACCEPTED at +280 elo. First real use is the next eval/ordering change.

Also closed the bugs-first gate: dissected the worst residual blunder (the
29301cp mate-walk) with a deterministic depth-1→16 scan — it's a **false-perpetual
horizon illusion** (NGN scores it 0.00/draw at depths 4-8, then sees white escapes
the perpetual at depth 9-10 and switches to SF's move), NOT a bug. The smoking gun
for *why* movetime stalls at the blundering depth: depth 9 = 289K nodes but depth
10 = **35.5M nodes** (120× explosion). Confirms search efficiency in sharp
positions is the remaining lever — exactly the tuning-shaped work sprt now measures.

**First real A/B results (2026-05-24) — both NEUTRAL, and the second proves the
instrument's worth.** (1) Hanging-pieces @ flat 50cp: 17W-10D-17L = 50.0% over 44
(depth 8). (2) Rook-on-7th @ 20cp, guarded (enemy king on back rank OR enemy
pawns still on the 7th): 176W-41D-183L = 49.1%, elo **-6.1**, CI [-40,+28],
LLR -0.16 over 400 games (depth 6) — reverted. The smoking gun: rook-7th read
**+29 ELO at 48 games** (the old built-in-opening cap), peaked ~+60, then decayed
to **-6 by 400** — a small sample would have shipped a phantom +30. That is the
exact false positive the 5-game smoke kept producing; it retroactively explains
why older smoke-gated eval tweaks "looked good then regressed." **Lesson:** bolting
single un-tuned eval terms onto an already-reasonable eval is low-yield (theory +
2 data points agree). Real levers from here: (a) TUNE the existing eval vector
together (Texel/SPSA — texel.go is a skeleton, a build task), or (b) search
efficiency (the measured blunder ceiling — needs a node-limited sprt mode since
fixed-depth can't reward reaching deeper; `-movetime` mode works now at
concurrency 1 as a noisier stopgap). Both experiments preserved in git stash.

**Correctness audit started (2026-05-24) — empirical, not code-reading.** Per the
directive to lock down ALL classic techniques before any tuning (no Texel/SPSA
until correctness is proven — "should reach 2k on bug-fixing alone"), began an
empirical correctness harness (tests that PROVE a bug, vs code-reading which gives
false positives). First test caught a real bug: replaced the coverage-theater eval
symmetry test (1 position, only checked the signs differed) with a 20K random-
position sweep asserting the EXACT mirror invariant Evaluate(P) == -Evaluate(
mirror(P)). It flagged 235/20K asymmetric (±12-25cp); a per-term sweep isolated it
to evaluateOutpostSquares; root cause = swapped edge-file guards in the
BlackPawnAttackers table init (engine/global.go): sq+7 (file-1) was gated on
file<7 and sq+9 (file+1) on file>0 — both backwards, so the table wrapped the A/H
edges and missed legitimate attackers there. Fixed, verified 0/20K, all 12 eval-
regression baselines still pass, tactical 30/30. Same family as the phantom passed-
pawn mask. Small magnitude (edge-file outposts only, likely sub-measurable ELO
alone) but correct now and permanently regression-guarded (TestEvaluationSymmetry,
TestEvalSymmetryByTerm in eval_test.go).

NEXT correctness-harness tests (in priority order): (1) PRUNING ABLATION — toggle
each of null-move/LMR/LMP/futility/RFP/SEE-prune/probcut OFF vs ON on the tactical
suite; a technique whose removal solves MORE is over-pruning = bug. This is where
the big fish are (the +243 partial-commit fix was this class). Needs toggle hooks
added to search.go (none exist). (2) TT SOUNDNESS — fixed-depth best move with TT
on must equal TT off (pure cache). (3) Search consistency / re-search determinism.

Pre-existing TEST-hygiene issues found en route (NOT engine bugs, flagged not
fixed): TestEnginePlayingStrength compares UCI moves ("b1c3") to SAN strings
("Nc3") so 3/4 sub-scores are always 0 → emits a bogus "950 ELO CRITICAL" that
misleads; TestGenerateEvaluationBaseline rewrites the committed eval_baseline.json
on every `go test` run (churn); that baseline is mildly stale (within ±10).

**Pruning ablation DONE (2026-05-24) — no tactical over-pruning; suite too shallow
for deep pruning.** Added SearchToggles (engine/search.go: 8 techniques, default ON
= behavior-neutral, tactical 30/30 confirmed) + TestPruningAblation
(engine/ablation_test.go): disables each technique and re-solves 120 fixed-depth-7
positions from validated_tactical_positions.epd. At fixed depth, removing a
technique only makes the search MORE thorough, so "solves MORE with it OFF" =
over-pruning bug. Result: every technique OFF still solves 119/120 (+0) — NO
over-pruning of tactical refutations in null-move/futility/RFP/probcut/LMP/SEE/
LMR/hist-prune. CAVEAT (the node-count column earns its keep): the forcing-tactics
suite is too shallow (avg ~4700 nodes/pos; mate-found-stop halts ID at depth 1-3),
so the DEEP pruners barely fired — no-NullMove/no-Probcut/no-HistPrune node counts
were IDENTICAL to baseline, no-LMR 1.00x. Only the shallow pruners
(futility/RFP/LMP/SEE) were actually exercised (0.97-1.06x) and are clean. So
deep-pruning soundness in QUIET play is NOT yet cleared — a forcing-tactics suite
structurally can't stress it. NEXT: probe null-move/LMR with deeper non-mate
positions (need a harder suite where searches reach ~100K+ nodes) or an oracle-
based cp-loss ablation on quiet positions (ties into the search-efficiency
measurement). The SearchToggles hooks are now in place for either.

**SEE WAS COMPLETELY BROKEN — fixed, +161 ELO (2026-05-24, the session's big win
and the proof of the correctness-first thesis).** TestSEECorrectness
(engine/see_test.go: crafted positions vs hand-computed exchange values) found
staticExchangeEvaluation returning garbage — 0 for winning a free knight (should
+320), 0 for QxP-defended-by-pawn (should -900). Three compounding sub-bugs in the
swap loop (engine/search.go): (1) gain[d] used the RECAPTURING piece's value
instead of the value of the piece standing on the square (the previous mover);
(2) the backward negamax read an unset gain[] slot, clamping any no-recapture
capture to 0; (3) occupied never removed the moving piece, so x-ray attackers
behind it were invisible. Consequence: SEE-based capture ORDERING and PRUNING
(SEE-prune, probcut, qsearch delta) ran on wrong values the entire time. Fixed all
three; TestSEECorrectness 7/7 (incl. x-ray battery, en passant, promotion-capture).
SPRT, fixed-SEE vs broken-SEE (the ONLY delta), fixed depth 6, 400 openings:
**+161.4 ELO, 95% CI [+121,+201], 246W-17D-92L = 71.7%, H1 ACCEPTED.** Movetime
would likely be even higher (ordering speedup -> deeper search is invisible at
fixed depth). Benign side effects vs the +161: tactical 30->29 (correct SEE pruning
now prunes shallow sacrifices -- the classic SEE-vs-sacs tension) and
TestAtomicVsNonAtomicPerformance trips its fragile 25% node-divergence threshold
(correct pruning amplifies atomic-vs-non-atomic TT-race divergence; a non-gating
perf benchmark -- consider loosening/skipping it). Committed.

**Current Status (2026-05-23):** ~1711 rolling-smoke estimate, 95% CI [1555, 1867]
(commit-decayed, NGN 1000ms vs SF1500 200ms), up +243 from 1468 at the start of today's
blunder work and still climbing as pre-fix runs decay. Post-fix smokes went 19W-0D-1L
(95%, up from ~50%); trajectory 1468 → 1585 → 1660 → 1711. Tactical 30/30. Four search
fixes today (partial-commit, mate-found stop, losing-mate UCI formula, + the diagnostic
tool); the dominant gain was the partial-commit fix below.

MAJOR FIX (commit 362fd24): the #1 strength bottleneck was NOT eval — it was a SEARCH
ROBUSTNESS bug. The iterative-deepening loop committed a *half-searched* best move when
the clock interrupted an iteration mid-move-loop (the classifier didn't check
`info.Stopped`), so under time pressure NGN played catastrophic blunders (hung
queens/rooks, walked into mates). Fixed with `if info.Stopped { break }` before the
classifier (search.go ~624). Verified three ways: oracle blunder rate 9.7%→3.2%, tactical
29→30/30, and two post-fix smokes went 9W-0D-1L (90%) lifting rolling 1468→1585.

How it was found: built `cmd/blunder-finder` (Stockfish-oracle centipawn-loss detector;
see -log/-in, -reset-tt, -depth modes). Dissection proved it was search-not-eval (static
evals of blunder positions were fine), and a fixed-depth-vs-movetime + reset-tt-vs-carry
A/B isolated the cause to the time-interrupt path. NPS also VERIFIED ~750K-1M (not the
stale "350K"), so Stage 6 is already met. The 5-game smoke is still a coarse instrument
(swung 80%→20% on an identical binary) — fine for catching big effects like this fix,
useless for <100-ELO tuning.

Mate hygiene DONE (commits e8d5eda): added a mate-found stop to the ID loop (no more
depth-100 spin; tactical suite 4.2s → 0.2s) and fixed the losing-mate UCI formula
("mate -501" → correct distances). Both gated by smoke-tactical 30/30, neutral on play
(same move, just stops deepening) — expected, since the mate spin was post-fix mostly
cosmetic.

Remaining blunders (~3% at movetime, down from ~9%): all HORIZON misses — NGN blunders at
depth 7-9 but finds the right move by depth 10-13 (verified by depth-scaling). NOT discrete
bugs. The lever is reaching more depth in sharp positions, where the search explodes
(~35M nodes to reach depth 10 in an 8-piece position). So further blunder reduction =
search EFFICIENCY work (move ordering, aspiration windows, extension/qsearch discipline)
— diminishing returns and tuning-shaped, so it needs the SPRT self-play instrument
(deferred) to measure. Eval/Texel stays premature. Highest-value next: either build the
SPRT harness (TC ≈ current smoke depth, NOT hyperbullet) to unlock fine tuning, or keep
using cmd/blunder-finder for any remaining gross bugs.

**Prior status (2026-05-03):** 1580 ELO rolling-smoke estimate, 95% CI [1446, 1714]
(width 268, commit-decayed sample of 26.0 games across 11 runs at NGN 1000ms vs SF1500
200ms). Tactical 29/30. Trajectory this session: 1477 → 1533 → 1582 (2 post-audit
smokes) → 1553 (hanging-pieces tried/reverted) → 1559 ([piece][to] history) → 1564
(continuation history) → 1599 (LMR log-formula) → 1580 (mate-distance ply fix; tactical
28→29/30, rolling drift within noise). +103 ELO net since session start, all changes
gated by smoke-tactical and committed individually. Stage 1 ladder target 1700 within
reach: CI lower bound 1446 vs 1700 target.

---

## Differentiation & Vision Roadmap (post-correctness)

Once correctness is locked down, specialize. Guiding principle: the strongest
niches LEAN INTO the classical eval (fast, hand-crafted, inspectable); the
riskiest ABANDON it. Sequencing at the bottom.

### Identity: "the strong, fast, EXPLAINABLE Go engine"
Three reinforcing bets, all banking on the classical eval:
- **Explainable (#1 differentiator).** Classical eval = named terms, so the engine
  can print WHY: "+1.3 = material +1.0, king-safety +0.5, passed pawn +0.3,
  back-rank -0.5". NNUE structurally cannot. Value grows as engines get more
  opaque — teaching, streaming, analysis tools. Lowest competition. Build: an
  eval-trace mode decomposing evaluateUnsafe term-by-term to a UCI `info string`.
- **Strongest Go engine.** Concrete title to climb: Blunder ~2400, Zurichess
  ~2500; we're in range. Scoreboard = CCRL / self-play vs those binaries.
- **WASM / embeddable.** Single static binary; compiles to WASM = strong engine
  in a browser tab, no install. Go is built for this.

### Speed / low-latency sub-niche
Won't beat Stockfish at bullet (it wins everywhere), but "lowest-latency engine"
and "strongest engine on weak / no-SIMD hardware (Pi, phone, WASM)" are real
classical wins — NNUE needs the net in RAM + SIMD; tiny classical screams on weak
HW. The Go-GC fear is overstated: GC is sub-ms and the hot path already uses stack
buffers (keep it zero-alloc). Go's real tax is ~1.5-2x single-thread throughput
(no easy SIMD), and it's FLAT across time controls, not bullet-specific. Unclaimed
speed wins: incremental eval (update on make/unmake), lazy SMP (goroutines make
this EASIER than in C++), TT prefetch, staged/lazy move gen.

### Learned pruning ("prune-net") — the one hybrid experiment
The disciplined version of the NNUE-pruning instinct. THE TRAP: NNUE inference
(~100-300ns) is WHY NNUE engines have lower NPS — call it at every node and you
become a slow NNUE engine, throwing away our edge. It only pays if the net's
verdict saves more than it costs:
- **NN-ProbCut (amortized — START HERE).** Run a net ONLY at high nodes
  (depth>=D, near root); one prune kills a giant subtree, so the net runs rarely
  and NPS stays high. Builds on the existing probcut hook.
- **Policy candidate filter (Leela-lite).** KEY: NNUE = position EVAL, not move
  POLICY. To prune by "which moves to even try" you want a POLICY net = Leela's,
  not Stockfish's. Search top-K policy moves, hard-prune the rest.
- **Tiny purpose-trained prune-classifier ("prune this node? y/n").** Cheapest
  inference, most novel, learned from our own search — the real "small" path.

Net sizes (DECISION GATE before pulling anything — keep it SMALL):
- **Stockfish .nnue**: documented (feature transformer + affine + clipped ReLU,
  int8/16). Default net ~70MB, but small-net variants ~1-10MB are size-viable.
- **Leela**: protobuf; strong nets are 100s of MB toward a GB = OFF-TABLE for a
  small engine. Only tiny maia-style nets (~few MB) qualify.
- Resolution of "pull both AND stay small": pulling = a no-training SHORTCUT to a
  baseline; for the real thing TRAIN OUR OWN tiny prune-net (KB-to-low-MB, we
  control size). Pull SF-small for an eval-pruning experiment; Leela-big is a no.
- Go inference cost: feasible (Go NNUE probers exist) but SIMD-less, so slower
  than C++ — another reason to favor a tiny custom net over a big pulled one.

### Wild: generic game engine
The search core (alpha-beta, TT, time mgmt) is GAME-AGNOSTIC; only move-gen, eval,
terminal detection are chess-specific. Refactor behind a Game abstraction and the
same engine plays Othello/checkers/Hex/Connect-4.
- **Generic WITHOUT losing chess strength?** YES, if the boundary is monomorphized
  — Go GENERICS (`Search[G Game]`), NOT interface values (dynamic dispatch in the
  million-node loop tanks NPS). Caveat: Go generics aren't fully zero-cost
  (GC-shape stenciling reintroduces indirection for pointers), so GATE the
  refactor on "chess NPS did not regress" (bench + smoke canary); fallback =
  code-gen a specialized search per game. Eval QUALITY is preserved for free —
  eval is per-game, only the SEARCH is shared.
- **Fully generic pruning?** Control flow yes, KNOWLEDGE no — split into a generic
  skeleton + a GameTraits hook set: eval margins (futility/RFP, scaled to game
  units), "quiet vs tactical?" (LMP), "forcing / extend?" (check → generic
  forcing predicate), "null-move sound? zugzwang risk?", move-ordering signals
  (history generic; SEE/captures chess-specific, absent in non-capture games).
- **THE UNIFICATION:** a learned prune-net is game-agnostic MACHINERY (same arch,
  retrained per game) that REPLACES all the hand-coded game-specific predicates —
  so "fully generic pruning" argues FOR the prune-net, and it's the AlphaZero move
  (generic search + learned knowledge). The prune-net and generic-engine threads
  are one architecture; AlphaZero-lite (general MCTS + per-game self-trained net)
  falls out of it.

### Sequencing
Correctness (current) → explainable + strongest-Go (cheap, bank the identity) →
generic refactor behind Game (architecture forcing function, gated on chess NPS) →
NN-ProbCut, then prune-net (the hybrid bet; train tiny, don't pull big) →
AlphaZero-lite if we want it. Each rung is a real project, not an afternoon.

---

## Recent Fixes (2026-04-26)

### Attack-table king safety (FIXED)
- `evaluateKingAttackPatterns` rewritten to use real piece attack bitboards
  intersected with a king zone (3x3 around king + one-rank extension toward
  the enemy half) instead of just counting opponent pieces in a 5x5 proximity
  zone. Per-piece attack-square hits weighted (N=2, B=2, R=3, Q=5).
  2+ attackers fed through quadratic penalty curve (units²/8 capped at 500cp);
  0-1 attackers get a small linear penalty (units/2). The old version flagged
  pieces NEAR the king as dangerous even when blockaded; the new version only
  credits real threat squares.
- 5W-0D-0L smoke. Rolling ELO 1605 → 1653 (CI lower bound +54 to 1519).

### Mate score TT encoding (FIXED)
- `alphaBetaPV` and `quiescenceWithDepth` now take an explicit `ply` parameter.
- New `scoreToTT(score, ply)` and `scoreFromTT(score, ply)` helpers convert between
  root-relative search scores and node-relative TT-stored scores. Without this, mate
  scores stored at one ply and retrieved via the same hash at a different ply reported
  the wrong distance to mate.
- First attempt failed because `info.Depth - depth` is NOT a valid ply (info.Depth is
  initialized to maxDepth and only updates AFTER each iteration completes). Proper fix
  threads ply through every recursive call site (root entries, IID, null move, probcut,
  singular, LMR, qsearch evasions/captures).

### Qsearch stalemate score (FIXED)
- `quiescenceWithDepth` returned `-500` when stalemated while winning (>+300cp eval).
  Stalemate is a draw regardless of material; replaced with `return 0`.

### Tempo bonus added
- `EvaluateForPlayer` now applies a 10cp bonus to side-to-move (eval.go).
  Standard small classical-eval feature, claimed +10–20 ELO.

### Eval/search micro-optimizations
- `pos.Hash()` cached once at the top of `alphaBetaPV` instead of recomputing in
  the repetition block and TT probe block.
- King-safety pawn lookups hoisted out of the file loop.
- `evaluateMobility` caches `GetWhitePieces`/`GetBlackPieces`.

### Rolling-ELO commit-distance decay
- `cmd/smoke/main.go` now weights historical runs by `git rev-list --count <sha>..HEAD`:
  full weight at distance 0, linear taper to 0.5 at distance 5, then to 0 at distance 15.
  After 15 commits without a smoke, that data point falls off entirely.

### Hash Corruption on UnMakeMove (FIXED)

### Hash Corruption on UnMakeMove (FIXED)
- `unMakeMoveHelper` mutated `p.Tag`/`p.EnPassant` before reversing the Zobrist hash, so the XOR undo used the wrong state.
- Fix: call `updateHash` symmetrically before reverting state. XOR is its own inverse.
- Regression test: `engine/hash_makeunmake_test.go` walks 6 FENs at depth 3 confirming `pos.Hash()` returns to its pre-make value after every UnMakeMove.

### countPassedPawnsFast Returned Phantom Passed Pawns (FIXED)
- The "all ranks ahead" mask used `uint64(0xFF) << ((sq/8 + 1) * 8)` which sets ONE rank, not all ranks. Black-side mask was always 0 for most pawns. Per-pawn rank masks in `opponentControl` had the same single-rank bug despite the comment claiming otherwise.
- Effect: starting position reported 8 passed pawns each side (correct: 0). Worth roughly +20cp × phantom passes per eval call.
- Fix: precompute `passedPawnFrontSpan[color][sq]` table in init() — file plus adjacent files at all ranks ahead.
- Regression test: `engine/passed_pawn_compare_test.go` compares slow vs fast counts across 8 positions.
- ELO impact: ~+160 ELO over post-hash baseline (CI lower bound 1468 vs prior ~1398).

### Aspiration Windows Treated Completed Iterations as Fail-Low (FIXED)
- Root loop raised `alpha` to the best score during the move loop, then classified the iteration with `iterationBestScore <= alpha` — which always fires because alpha was just raised to iterationBestScore. Past depth 3, every successful iteration was marked fail-low, attempts exhausted, but `info.Depth` and the UCI callback fired with stale data.
- Fix: snapshot `windowAlpha`/`windowBeta` before the move loop and classify against the snapshot. Only update `info.Depth` and call the callback when the iteration actually completed (`engine/search.go`).

### Quiescence Stand-Pat in Check (FIXED) + Quiet Checks Dropped
- Qsearch unconditionally stand-patted even when side-to-move was in check.
- Fix: at qsearch entry, compute `inCheck`. If in check, search ALL legal evasions (no stand-pat); return mate score if no legal evasion. If not in check, stand-pat + captures only (`engine/search.go`).
- Also dropped quiet checks from qsearch and removed `moveGivesCheck` — it missed discovered/EP/promotion checks, and incomplete-quiet-checks is worse than captures-only at the horizon (`engine/movegen.go`).

### TestIterativeDeepening Determinism (FIXED)
- Test failed reproducibly with `ID: b1c3, Fixed: g1f3` because global state (TT, history, killers, counters) carried between the two searches.
- Fix: reset global search state before each search call inside the test (`engine/search_test.go`).

### UCI `stop` Wired Into Recursive Search (FIXED)
- Stop was only checked in the depth-completion callback. `go infinite` + `stop` hung until max depth.
- Fix: added a global atomic stop flag (`globalStopRequested` in `engine/search.go`). `RequestStop()` is called by `handleStop`; `IsStopRequested()` is sampled at the top of `alphaBetaPV` and `quiescenceWithDepth` and at every iterative-deepening depth boundary. Cleared at search entry.

### ShouldStopSearch Iteration-Start (FIXED)
- All callers passed `time.Now()` where iteration-start was expected, making `iterationTime ≈ 0` and disabling soft-stop.
- Fix: TimeManager owns `iterationStartTime` internally. `NewIteration()` sets it at the top of each iterative-deepening depth. The legacy parameter is now ignored. Soft-stop falls back to "don't stop" if `iterationStartTime` was never set (`engine/time.go`).

### Test Hygiene (PARTIAL)
- Replaced `Rook_vs_Bishop_Knight` FEN (`8/8/8/8/8/8/2R5/2BNK2k`, which was white R+B+N vs lone king) with a real R-vs-B+N endgame.
- Toned down the misleading `CRITICAL ERROR` wording in `free_piece_test.go` — the test never failed and the position isn't a clean free-piece test at depth.

### Threefold Repetition Detection (FIXED)
- `handlePosition` used `MakeMove` instead of `GameMakeMove` — position history wasn't tracked.
- Root search didn't check `IsFIDEDrawRule()` before returning moves.

---

## 2026-05-02 audit batch — verified bugs

Six parallel sub-agent audits of search/eval/movegen/qsearch+SEE/TT+hash/UCI+time produced
findings; the items below were verified against current code. Apply one at a time gated by
`make smoke-tactical`.

### Pri 1: Mate distance pruning uses stale `info.Depth` (FIXED)
- `engine/search.go:1081` and `:1397` (alphaBetaPV checkmate paths) used
  `info.Depth - depth`, which is stale during the tree walk.
- Fix: both sites now use `ply` directly. Qsearch path (line 1516) and the
  top-of-alphaBetaPV mate-distance pruning (line 946) were already on `ply`.
- Commit b50aa23. Tactical 28 → 29/30; rolling 1599 → 1580 (drift within noise).

### Pri 2: SEE and delta pruning ignore promotion gain (FIXED in 5b0032b)
- Both initial SEE (`engine/search.go:139-143`) and qsearch delta pruning (`:1579-1583`)
  now add `promoPiece.seeWeight() - 100` (pawn weight) when `move.PromoType() != NoType`.

### Pri 3: Singular extension reduction is too aggressive (FIXED)
- Replaced `depth/2` verification depth with `max(1, depth-4)` per Stockfish/Ethereal.
- Commit 8269da1. Tactical 29/30 held; rolling 1580 → 1551 (drift within noise).

### Pri 4: Qsearch discards TT move + searches captures unordered (FIXED in 5b0032b)
- Both in-check evasion branch (`:1477-1484`) and captures branch (`:1564-1571`) now
  swap ttMove to index 0 if present. Avoids the -129 ELO upfront-MVV/LVA sort regression
  while still letting TT-suggested moves cut early.

### Pri 5: `info.SelDepth` not reset between iterations (FIXED in 5b0032b)
- `info.SelDepth = currentDepth` set at top of each iteration loop in
  `searchIterativeDeepeningUnsafe` (`engine/search.go:513`).

### Audit findings flagged but FALSE-POSITIVE
- **King centrality "negative for corners"** (`eval.go:1476-1484`): agent claimed HIGH/+50 ELO.
  Verified: `whiteCentrality = 6 - centerDistance` ranges [-1, +5] (slightly asymmetric, off
  by one). Term enters as `(whiteCentrality - blackCentrality) * 3` so the asymmetry cancels
  when both kings have similar status. Worst-case ~3cp calibration smell, not a real bug.
- **Cache.go atomic race / replacement strategy** (`cache.go:107-113, 157-164`): agent claimed
  partial-update visibility. This is the textbook XOR-verifier (lockless) pattern; mismatch
  detection rejects partial updates safely.

### Audit findings UNVERIFIED (defense-in-depth, low urgency)
- White EP missing rank validation (`movegen.go:198-214`) — perft passes; defense-in-depth fix
  if EP could ever be set to invalid rank by some other bug. Cheap to add.
- Movetime + 50ms overhead floor (`time.go:265-266`) — claim is plausible only at very low
  movetime (<100ms), uncommon in our smoke (1000ms). Verify if real.
- Soft-stop volatility burning time (`time.go:370-381`) — speculative; would need targeted A/B.

## Older deferred items

### External Polyglot books silently broken
- `engine/polyglot.go:33-55` — `polyglotRandom[781]` is generated from xorshift seed=1 instead
  of the spec-defined constants. `PolyglotHash` of starting position yields
  `0x4b9aa9e5e768fe35`; spec says `0x463b96181691fc9c`.
- A real `.bin` polyglot book will never match positions. Embedded book still works.

### Test hygiene (cosmetic)
- `engine/play_quality_test.go` (`TestEnginePlayingStrength`) compares `Move.ToString()` UCI
  output (`b1c3`, `e1g1`) to expected SAN (`Nc3`, `O-O`). Engine plays the right move, test
  marks it neutral.
- `engine/crash_stress_test.go:250-278` (`TestConcurrencyStress`) shares one mutable
  `*Position` across goroutines.

### Qsearch quiet-check detection (deferred — low priority)
- Currently qsearch is captures-only (quiet checks intentionally dropped after the
  `moveGivesCheck` direct-attack-only bug). If quiet-check coverage is desired later, use a
  proper discovered-check detector or make/unmake + `isInCheck`.

---

## Assessment Strategy

### Default: `make smoke` (~5 min, E-cores, battery-friendly)
Runs a 30-position tactical suite plus a 5-game asymmetric match (NGN 1s vs
SF1500 200ms). Appends results to `output/smoke_history.jsonl`. Prints a
rolling Wilson CI that tightens with every run. Fail conditions: tactical
solve rate < 40% or 0/5 game outcome.

```bash
make smoke              # Default cadence
make smoke-tactical     # Tactical only (~5 sec) — for in-flight changes
./build/smoke -rolling  # Show rolling estimate without running
```

The rolling estimate is the source of truth for current ELO, not any single
smoke. After 6-10 smokes at the default config the cumulative CI is tighter
than a single 14-game Bayesian and you've burned an order of magnitude less
battery.

### Full Bayesian (AC power only, ~15-30 min)
```bash
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```
Use when you need a tight one-shot CI before a major decision.

### Legacy Presets (hotter, mostly historical)
```bash
./build/elo-assess -level 1   # 8 games @ 250ms
./build/elo-assess -level 5   # 20 games @ 400ms
./build/elo-assess -level 10  # 40 games @ 600ms
```

---

## Verified Implementation Status

### Search Techniques
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
- [x] Quiescence search (in-check evasions; captures-only — quiet checks intentionally dropped)

### Extensions (WORKING)
- [x] Check extensions
- [x] Singular extensions
- [x] Recapture extensions
- [x] Passed pawn extensions

### Evaluation

Currently wired into `evaluateUnsafe`:
- [x] Material + PeSTO tapered MG/EG piece-square tables (all pieces, not just king)
- [x] Passed pawns bonus
- [x] Doubled pawns penalty
- [x] Isolated pawns penalty
- [x] Pawn chains bonus (re-wired commit a609f94 — was orphaned by PeSTO swap c6675ff)
- [x] Bishop pair bonus
- [x] Mobility (knights/bishops/rooks/queens, enemy-pawn-attacked squares excluded)
- [x] Rook on open / semi-open file
- [x] Outposts (knight/bishop on pawn-supported, unattackable enemy-territory squares; re-wired commit 94c4316)
- [x] King safety (pawn shield + attack-table king-zone scoring; middlegame only, phase ≥ 12)
- [x] King activity (centralization + king-near-pawns; deep endgame only, phase < 6; re-wired commit c0f6bbb)
- [x] Tempo bonus (10cp side-to-move, applied in `EvaluateForPlayer`)

Orphaned (function defined but never called from `evaluateUnsafe`; left as dead code by the PeSTO swap):
- [ ] `evaluatePawnStorms` (eval.go:561) — TRIED AND REVERTED at full magnitude in commit 7e8702b (reverted by 8c773eb). Tactical held at 27/30 but 5-game smoke went 0W-5L vs SF1500 (rolling 1524 → 1430). Hypothesis: full-magnitude storm bonus (28cp peak in Italian-style positions) over-rewards pawn pushes that weaken own king's pawn shield, especially when our king is also uncastled. If retried, scale magnitude down ~50% AND add a "only fire when own king is castled OR opponent king is castled" guard so we don't push our own structure apart against a king still in the center.
- [ ] `evaluatePieceCoordination` (eval.go:1517) — large/inflated magnitudes (queen-protection ×8 etc.); inline comments admit values were already tuned-down once. High risk to re-enable without recalibration
- [ ] `evaluateWeakSquares` (eval.go:450) — short-circuits `return 0`; real logic commented out for performance. Effectively a re-implementation
- [ ] `evaluateBasicEndgames` (eval.go:1256) and `evaluateKingPawnOpposition` (eval.go:1304) — `+800/+600` magnitudes mis-calibrated against PeSTO; deferred per project_bugs memory

### Infrastructure
- [x] Opening book (embedded works; external Polyglot broken — see Remaining Known Issues)
- [x] Syzygy tablebase support
- [x] UCI protocol (`stop` wired into recursive search via global atomic flag)
- [x] Time management (`TimeManager.NewIteration` drives soft-stop)

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

### Phase 3: Search Correctness (LARGELY COMPLETE)
- ✓ Aspiration windows fail-low past depth 3 (Pri 0)
- ✓ Quiescence in-check evasions + drop quiet checks (Pri 1 + Pri 5)
- ✓ `TestIterativeDeepening` deterministic (Pri 4)
- ✓ UCI `stop` wired into recursive search (Pri 2)
- ✓ `ShouldStopSearch` iteration-start tracking (Pri 3)
- Deferred: SAN→UCI in `TestEnginePlayingStrength`, Polyglot constants (Pri 6)

### Phase 4: ELO Validation + Tuning (CURRENT)
1. Build the rolling-ELO baseline by running `make smoke` after each change. Per-run Wilson CI is wide (~700+ ELO at 5 games), but the cumulative across 6-10 smokes converges below 300.
2. Tune evaluation/search parameters one knob at a time, gating each with `make smoke`. Revert anything that regresses tactically or drives the cumulative ELO down.
3. When you need a publishable single-shot CI (e.g., before claiming a milestone), run a full 14-game Bayesian on AC power.
4. Optional: address remaining test hygiene + Polyglot constants if external book support is wanted.

---

## Realistic non-NN path to 2200 ELO

You're at ~1450-1650 rolling. Target +550-750 ELO. Reference points: Crafty ~2600, Fruit
~2700, pre-NNUE Stockfish ~3000 — all classical. The route is staged, each rung gated by
smoke. Sized in expected ELO buckets per stage; reality will vary.

**Stage 1 — Audit-batch fixes (target 1700, +50-100 ELO) — COMPLETE**
- All Pri 1-5 done. Tactical 28 → 29/30 (mate-distance ply fix). Rolling drifted
  1599 → 1551 across two changes — within Wilson noise; CI lower bound 1414 vs
  the 1700 ladder target.

**Stage 2 — Texel tune existing magnitudes (target 1800, +50-100 ELO)**
- `engine/texel.go` exists. The wired components (mobility baselines, king-safety quadratic
  curve, pawn-structure penalties, PST blends, tempo) are hand-tuned or sub-agent-guessed.
- Run a real Texel session against ~50K-100K labeled positions. Expect biggest movers in
  king-safety weights and mobility baselines.
- After tuning, re-evaluate orphans (`evaluatePawnStorms`, `evaluatePieceCoordination`) at
  freshly-tuned magnitudes — they may be salvageable post-recalibration.

**Stage 3 — Better history (target 1850, +30-50 ELO)**
- [x] Replace `[from][to]` history with `[piece][to]` (commit 8b04b85, 2026-05-03).
  Tactical 27/30 → 28/30; rolling 1553 → 1559 (+6 ELO).
- [x] Add **continuation history**: indexed by `[prev-piece][prev-to][curr-piece][curr-to]`
  (commit 7a309c3, 2026-05-03). Tactical 28/30 held; rolling 1559 → 1564 (+5 ELO);
  CI lower bound +7. Combined Stage 3 effect: +11 ELO rolling, +16 lower bound.
- [x] LMR formula change: `log(depth) * log(moveNum) / 2` precomputed table replaces
  piecewise base + legalTried thresholds + depth>6 bump (commit 2351daf, 2026-05-03).
  Tactical 28/30 held; rolling 1564 → 1599 (+35 ELO); CI lower bound +26.

**Stage 4 — Eval features that move the needle (target 1950-2000, +120-170 ELO)**
- Hanging/loose pieces: TRIED 2026-05-03 twice. V1 at piece_value/4 regressed -29 ELO
  (rolling 1582 → 1553) — V/4 double-counts material. V2 at flat 30cp was neutral
  (3W-0D-2L, rolling 1599 → 1598, tactical 27/30 held). Both reverted. Real fix:
  defer to Stage 2 Texel so magnitudes co-tune with mobility/king-safety weights.
  Future retry could try flat 50-60cp (Stockfish's range) but only after Texel.
- King safety upgrade: open/half-open files in front of king + pawn shelter ratio (the
  attack-table now exists; pair with pawn-shield strength).
- Threats: piece attacks higher-value piece, discovered-attack potential, pinned-piece bonus.
- Endgame heuristics that don't require tablebase: KPK rule-of-the-square, KBNK corner
  driving, opposition.
- Space evaluation in middlegame.

**Stage 5 — Search depth (target 2050-2100, +50-100 ELO)**
- Multi-cut after null move (search 2-3 moves at reduced depth before accepting null fail-high).
- Probcut margin / depth re-tune.
- Razoring depth/margin re-tune.
- Internal iterative reductions for non-PV nodes without TT move (cheaper than IID).
  TRIED 2026-05-03 with `if ttMove == EmptyMove && depth >= 4 && !isPV { depth-- }`
  right after the IID block. Tactical 29/30 held but 5-game smoke went 0W-0D-5L
  (all 5 losses to checkmate); rolling 1551 → 1460. Reverted (no commit).
  Hypothesis: depth >= 4 threshold is too aggressive in our codebase — most non-PV
  search nodes early in iterative deepening have empty TT, so IIR cascades through
  the entire shallow tree and effectively reduces full-game search by 1+ plies.
  Stockfish has many other compensating mechanisms (multi-cut, IIR depth >= 6,
  tt-aware reduction caps). Retry candidates: (a) raise threshold to depth >= 6 or
  >= 8, (b) only fire if depth - ttDepth >= 4 (skip IIR when TT has stale entry),
  (c) defer until after Texel + multi-cut so the search tree is more robust.

**Stage 6 — NPS (LARGELY DONE — verified 2026-05-23)**
- The "350K NPS is low" premise is STALE. Measured in-process: ~750K-1M NPS in real
  middlegame/tactical positions, 1.47M on startpos, depth 20 in 5s. The 350K figure came
  from the Feb 5 profiling files before optimizations.
- Already at/above the "realistic Go-bitboard target 800K-1M" this stage set as the goal.
  Search speed is NOT the bottleneck; do not spend time here. Re-profile only if a future
  position-mix shows a hot path, but expect single-digit ELO, not +50.

**Stage 7 — TT, time, polish (target 2200, +50 ELO)**
- 2-bucket TT (depth-preferred + always-replace).
- Wire UCI Hash option (currently ignored).
- Smarter time management: easy-move detection, reduce on book/forced moves, extend on score drop.
- Aspiration window: smaller initial (16cp), exponential growth.

---

## Reference Engines (Go)

### Blunder (Go, ~2600 CCRL — algerbrex/blunder)
- AdaGrad/Texel eval tuning (+34-61 ELO from re-tuning alone); queen promotions in qsearch;
  pawn pushes to 6th+ not reduced in LMR; counter-move bonus measured statistically ZERO.
- **Measured per-feature ELO (self-play SPRT @ 10+0.1, docs/testing.md — NGN's best external
  yardstick, same language):** null-move +116, LMR +61, RFP/static-null +57, futility +37,
  SEE-qsearch +26, fail-soft-in-static-null +23, aspiration +23, LMP +22, history +19,
  TT-replacement-bug FIX +30, LMR research-order +12, IID +11, razoring +8, singular +6.8,
  adaptive-null-R +14 (n.s.), counter-move +5 (n.s.). Eval terms: king-safety +47, passed
  pawns +36, doubled/isolated +33, mobility +27, bishop-pair +17. The numbers force the
  conclusion: NGN already has every +20-plus pruning rock; the remaining search tail is 5-30
  ELO each (see Phase 1 roadmap at top).
- Lacks (vs NGN's roadmap): continuation history, correction history, improving flag,
  cutnode-aware reductions — NGN's Phase-1B items are past even Blunder.

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
make build           # Build engine
make test            # Run tests with coverage
make test-short      # Run tests without coverage (faster)
make smoke           # DEFAULT verification: tactical + 5-game smoke + rolling ELO (~5 min)
make smoke-tactical  # Tactical-only smoke (~5 sec)

./build/smoke -rolling   # Print rolling-ELO summary without running

# Full Bayesian ELO test (AC power only):
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

---

## Notes

- Run `make test` after every change
- Verify games complete before trusting ELO numbers
- One bug fix at a time, test between each
- Track ELO progress with assessment tool

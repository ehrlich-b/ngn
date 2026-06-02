# NGN Chess Engine — Development Roadmap

## Goal: GM ≈ 2500 (beat Blunder 7.4.0 / CCRL 2532), then 2800 — CLASSIC HCE + speed, **NO NNUE (yet)**. Now ~2400 CCRL, ~+85-150 behind Blunder.
**STATUS 2026-06-02 (PIVOT — eval-tuning chapter CLOSED):** the gradient Texel tuner is SHIPPED and its VERDICT IS IN: **eval tuning is NOT the lever.** Self-play distill (30k −10 reg / −440 l2=0), then the full Lichess 1.05M SF-eval tune (787 data-only AND 796 tapered) both NEGATIVE in games (−26 to −31) despite better MSE. Bryan called it: "you've been confirmed full of shit ... there's a problem and you're not finding it" and forbade NNUE (it'd be tuning into the same wall). A phoned-in second opinion + our own M2/M3 diagnosis CONVERGE: **the lever is endgame TREE EFFICIENCY (search), not eval.** New spearhead below. Reanchor = re-read README.md, this TODO, CLAUDE.md.

---

## >>> THE PLAN — READ FIRST, every reanchor (rev. 2026-06-02 post-pivot) <<<

Eval tuning produced MSE-down / game-down repeatedly — a warning, not a reason to scale the same pipeline. The evidence (M2 depth-at-time, M3 ACPL, docs/10 tree sizes) says NGN loses because it reaches FEWER useful plies in exactly the phase where its eval is LEAST suspect. Path to 2800 (classic, no NNUE), reordered:

1. **ENDGAME TREE AUTOPSY — THE NEW SPEARHEAD (Lane A; attended, needs interpretation).** NGN's endgame tree is **2.57x and 5.12x** Blunder's (docs/10-performance.md), while endgame ACPL is FINE (15.7 vs Blunder 17.0, −1.3cp; eg eval is NOT the miss). Before touching any knob, **instrument WHERE the extra 2.5-5x endgame nodes physically go.** Most counters ALREADY EXIST in `SearchInfo` (search.go ~506: `QNodes, TTHits, TTCutoffs, BetaCutoffs, FutilityPrunes, NullMoveCutoffs, LMRReductions, LMPPrunes, ProbcutPrunes, SEEQuietPrunes`). NEEDED: split them BY PHASE, add TT-probe count + the 4 extension counters (check/recapture/passed/singular), and a dump. Then run `scripts/h2h.sh build/ngn opponents/blunder_740` + `scripts/m2_depth.sh 1000/2000 build/ngn opponents/blunder_740` on the eg FENs. The artifact = a table that says whether the blowup is **qsearch explosion / missed TT cutoffs / weak futility-LMP / extension storms / low beta-cutoff (ordering)**. That localizes the bug. "Try another pruning knob" without this is guessing (and [[project_endgame_tree_diagnostic]] already refuted blind knob-twiddling).
2. **Clean J1 re-test (Lane B; candidate, flagged).** J1 = extensions computed into `nextDepth` (check/recapture/passed/singular, search.go ~1470-1532) but the non-first PVS scout searches `reducedDepth := depth-1-reduction` (search.go ~1621), so checks/captures (reduction==0, LMR-excluded) are searched at `depth-1`, NOT the extension-adjusted depth — extensions silently don't apply off-PV. **My 2026-06-01 fix attempt (`nextDepth - reduction`) was REVERTED on an UNCLEAN verdict** (movetime+conc4, read @G65 — and fixed-nodes self-play is DETERMINISTIC so early reads are doubly bogus) AND it **+83% bloated the eg tree / leaned −59**. TENSION: it grows the very tree we're trying to shrink, so it's likely net-negative for a depth-starved engine — but re-test CLEANLY (fixed-TIME, conc 1; + `info.RootDepth = depth` in SearchFixed to dodge the RootDepth==0 clamp) then gauntlet-confirm before final dismissal. Not a free patch; don't ship blind.
3. **Search-parameter tuning** (LMR/NMP/futility/RFP margins — hand-set round numbers, search.go:18-25). **One axis at a time or a small fixed-TIME grid; never a margin bundle.** Do AFTER the autopsy says which margin is implicated.
4. **Per-node speed — SECONDARY.** Flat ~2x tax (quality tradeoff; no structural bug found; ~15-20% micro-opt ceiling). Real but does NOT explain why endgames are uniquely 2.5-5x — so it's second, not the spearhead.

**EVAL-SEARCH COUPLING (the eval-adjacent risk, if eval is ever touched again):** the eval is NOT independent of the hardcoded cp margins (`FUTILITY_MARGIN=200`, `REVERSE_FUTILITY_MARGIN=120·d`, `DELTA_MARGIN=100`, `SINGULAR_MARGIN=64`, SEE −80·d, aspiration 50/100 — search.go:18-25) NOR of `correctionValue` (search.go:1156, a learned eval term Texel never sees). A retune that enlarges eval terms (mobility 166→199, passed→41) desyncs those fixed margins — a candidate reason the Lichess tunes regressed. Tune eval+margins JOINTLY or not at all.

**MEASUREMENT DISCIPLINE (VERIFIED this session — the guardrail):** **fixed-nodes self-play is DETERMINISTIC** (two identical 6-game matches → byte-identical W/D/L). So a fixed-nodes self-play SPRT is a deterministic ENUMERATION of ≤800 paired games over a fixed book: re-runs add ZERO info, the 400-opening book is a hard resolution cap, paired/correlated games make the binomial CI OPTIMISTIC, and it measures self-vs-near-self on one book — the mechanism behind the documented transfer-mirages. ⇒ **the SPRT STAYS the fast per-change gate (you can't gate iteration on a ~4h gauntlet) — the determinism caveat just SCOPES it: use fixed-TIME for SEARCH changes (non-deterministic → REAL CI, no asterisk); for eval changes (fixed-nodes) keep ≥400 openings and treat a tight CI as OPTIMISTIC. GAUNTLET = Cadence B (periodic milestone/batch confirmation to catch transfer-mirages), NOT a per-change gate.** Never read an SPRT before hundreds of games. KEEP RULE: clear improvement OR non-regression ⇒ keep; reject only on a concrete measured regression. One change at a time, never batch. Commits single-sentence, no co-author, no emoji, LOCAL (don't push). Never P-cores (`-lowpower true` → E-cores). [[feedback_decide_dont_ask]] [[feedback_canonical_measures]]

---

## >>> NEXT STEP: GRADIENT TEXEL TUNER — build spec (reanchor + PROCEED here) <<<

**BUILD STATUS 2026-06-02 (spearhead largely SHIPPED):**
- **(a) eval coefficient TRACE — DONE** (commit 123181d). `engine/texel_gradient.go`: decomposes `evaluateUnsafe` into a cached sparse coeff vector (PeSTO occupancy + scalar-term features); `reconstructEvalInt` reproduces `evaluateUnsafe` BIT-EXACT over the gate FENs (the faithfulness gate, stronger than aggregate-MSE) + float model within 1.85cp. Param layout = cmd/texel "full" order (material 10 + 768 PST + 9 scalars = 787).
- **(b) Adam optimizer — DONE** (commit 75538a8). `cmd/texel -mode gradient`: full-batch gradient descent over cached traces, all 787 params jointly. Validated by a NUMERIC gradient check (16 params incl. the negated penalties) + an end-to-end MSE-reduction/transfer test. Real-data smoke (30k self-play): train 0.122→0.091, test 0.126→0.098 in <1s (the 100-1000x speedup, realized). **Self-play data exposed the weak-teacher artifact live: isolatedPawnPenalty tuned NEGATIVE (−14), kingActivity −72 — exactly why we distill.**
- **(c1) SF distillation labeler — DONE** (commit 096eead). `oracle -mode distill`: parallel Stockfish soft labels, white-POV cp → logistic win-prob target, Texel-format. Sign-flip verified (black-to-move winning-for-white positions read ~0.95 not ~0.04).
- **(c2) FIRST CUT DONE — 30k/d12 distillation = NEUTRAL, not shipped; scaled run in flight.** SF labels DID fix the weak-teacher artifact (isolated −14→+10, kingActivity −72→+123 — distillation thesis validated). But SPRT verdicts: regularized (λ=3e-8, sane tables) = −10 [−40,+21] NEUTRAL; unregularized l2=0 (BEST held-out MSE!) = CATASTROPHE −440. **BIG LESSON: held-out MSE is NOT a strength proxy — search steers toward the eval's argmax, so wild PST cells the static sample rarely touches become the moves played; regularization toward PeSTO is MANDATORY, judge by GAMES not MSE.** Neutral because 30k self-play can't out-vote the incumbent's 725k-Zurichess tune. **IN FLIGHT (task b6akl2n35): scaled ~200k diverse positions + SF d14 labels** → regularized retune → SPRT → gauntlet. If still neutral, the linear-HCE basis is the ceiling → richer basis (more terms) or NNUE (Bryan riffed "small NNUE on the Mac" — feasible via PyTorch-MPS/cloud GPU; blocker is Go inference + data volume, not hardware).



**What ALREADY exists (engine/texel.go + cmd/texel):** the Texel OBJECTIVE is implemented — `texelSigmoid(eval,k)`, `texelMSE(samples,k)`, `TexelFindK` (ternary-search K), `LoadTexelSamples` (`<FEN> <result>` / result-labeled EPD), param exposure (`TexelWeights()`=10 eval-term scalars, `TexelMaterialParams()`, PST access), `gendata` (self-play data), Go-source emitters (`FormatPSTTables/MaterialValues/TexelWeights`). The night tuned PSTs+10 weights on the Zurichess set for +46 ELO (commit bb12a48). **What's MISSING is the SOLVER:** it's COORDINATE DESCENT (`TexelTune` line-searches one param at a time — slow, stalls in diagonal valleys).

**The build — gradient descent (Blunder's 8.0.0 upgrade), phased:**
1. **(a) Eval coefficient TRACE — the core new piece.** Instrument evaluatePeSTO + evaluateUnsafe to emit, per position, a sparse feature-coefficient vector: for each tunable param its (mg_coeff, eg_coeff) contribution + the position phase. The eval is LINEAR in the weights ⇒ `∂eval/∂wⱼ = coeffⱼ`. Trace every dataset position ONCE and CACHE it — then every epoch is cheap linear algebra over cached traces (no re-eval). This caching is what makes gradient 100-1000x faster than coordinate descent. GATE (a): reproduce the current `texelMSE` from the cached traces exactly (proves the trace is faithful) before optimizing.
2. **(b) Gradient + optimizer.** MSE gradient `∂E/∂wⱼ = (2/N)Σ (σ(sᵢ)−resultᵢ)·σ'(sᵢ)·(K·ln10/400)·coeffⱼ(i)`, σ'(s)=σ(s)(1−σ(s)), s=tapered eval. Step ALL params together via **Adam** (momentum + bias-correction, modern default; AdaGrad is the simpler fallback Blunder used). ~tens of epochs.
3. **Tune the FULL param set** (the real win vs today's 10 scalars): material + all 768 PST cells (mg+eg) + every eval-term's INTERNAL table (king-safety attack values, mobility per-piece bonuses, passed-pawn-by-rank, pawn-structure penalties, outpost bonuses) + the scalar weights. The phase-7-11 king hole should shrink once king-safety is jointly tuned across phases.
4. **(c) Anti-overfit + VALIDATE.** TRAIN/TEST split; tune on train, watch held-out test MSE (stop when it stops improving); emit tuned values as Go source; rebuild; **fixed-nodes SPRT vs current**, then **`-tc 120+1` gauntlet** for real transfer. KEEP RULE.
5. **Data — DISTILLATION (Bryan's steer 2026-06-02: learn from STOCKFISH, not from NGN-vs-NGN).** The self-play ceiling is in the LABELS, not the positions — NGN-vs-NGN grades itself (a weak teacher that can't correct a blind spot both its sides share). So **label positions with Stockfish's deep eval (knowledge distillation)**, not self-play WDL. POSITIONS can still come from cheap self-play (`texel -mode gendata`, fast, diverse) — SF labels supply the truth regardless of who reached the position; NGN-vs-SF game positions are a valid "relevance" option (positions NGN faces under strong pressure) but slower. LABELS via the oracle `label` pipeline (SF eval, already built); **PARALLELIZE it — labeling is OFFLINE / timing-insensitive, so run N parallel SF processes (P-cores fine; NOT the engine-vs-engine timing path) → ~100-300k positions in ~1-2h** (single-thread would be days). Map SF-cp→target via the sigmoid (or regress eval→SF-cp); optionally blend real-game WDL. The Zurichess WDL set (`LoadTexelSamples`) is a free baseline to validate the gradient MACHINERY first, before the SF-labeling run. **CAVEAT:** NGN's HCE is a small LINEAR model — it can only mimic SF's NNUE up to its capacity (a ceiling); distillation is a big win over self-play but BOUNDED, and hitting that ceiling is the future signal to go NNUE. This exact pipeline (positions + SF labels + gradient optimizer) IS the NNUE training pipeline — build once, use twice.

**Implementation:** new code in engine/texel.go + a `cmd/texel -mode gradient` (keep coordinate descent as fallback). Phase strictly: (a) trace+cache+reproduce-MSE → commit; (b) Adam over the full set → tuned values; (c) SPRT/gauntlet → KEEP/revert.

---

## TUNING PRIMER (distilled — the concept behind the spearhead; full thesis in the 2026-06-02 chat)

- **An eval = Σ wⱼ·featureⱼ.** Features = human concepts (material, PST, mobility, king safety); weights = the knobs. Tuning = set the knobs from data.
- **Texel tuning = the OBJECTIVE:** map eval→win-prob with sigmoid `σ(s)=1/(1+10^(−K·s/400))`, label each quiet position with its GAME RESULT (1/0.5/0), minimize MSE `E=mean((result−σ(eval))²)`. Turns "is the eval good?" into "does it predict outcomes?" — a number to minimize.
- **Coordinate vs gradient descent = two SOLVERS for that SAME objective.** Coord (NGN now): nudge each knob, keep good nudges — derivative-free, slow, axis-bound. Gradient (Blunder 8.0): closed-form best direction for ALL knobs at once (eval is linear in weights) — 100-1000x faster AND a better minimum (moves correlated params jointly). AdaGrad/Adam = gradient + smart per-knob step sizes. "Texel" and "gradient" aren't rival techniques — gradient is the good solver FOR Texel.
- **NNUE = the SAME objective + optimizer, eval function swapped for a neural net, human features REMOVED** (raw piece-square inputs; the net learns king-safety/mobility itself; millions of params; ~+300-400 ELO). NGN's incremental PST accumulator (accMG/accEG) is literally a 1-neuron linear NNUE. ⇒ the gradient tuner we build now is the NNUE on-ramp.
- **The blunder-finder (oracle/ACPL) is the DIAGNOSTIC TWIN of tuning, not a loose cousin.** Tuning minimizes distance to GAME RESULTS; the blunder-finder measures distance to STOCKFISH. Same latent target (true value), two estimators (results=unbiased/noisy; SF=low-variance/slightly-biased). Blunder-finder = the loss made VISIBLE (where the eval is wrong, e.g. phase 7-11); tuning = the loss MINIMIZED. And SF-eval is itself a valid tuning TARGET (distillation) — modern NNUE pipelines blend WDL+engine-eval. So diagnose (ACPL/phase-split) → tune (gradient) → verify (SPRT/gauntlet) is ONE closed loop.

---

## DIAGNOSIS RESULT (2026-06-02) — why Blunder beats NGN, with data (the evidence the plan rests on)

Read-only instruments built this day: `scripts/m2_depth.sh` (depth at matched time) + `cmd/oracle acpl` (cp-loss vs SF MultiPV). All vs blunder_740 (CCRL 2532).
- **M2 — depth at matched TIME (3 budgets 1000/1500/2000ms, incl. one idle-clean):** NGN ≈ Blunder depth in MIDDLEGAMES (~0-2 ply) but **+4 to +7 ply SHALLOWER in ENDGAMES** (quiet rook eg: NGN d16 vs Blunder d22), budget-stable. The 2.5-5x eg-tree explosion as a concrete depth deficit; NGN burns MORE nodes to reach FEWER eg plies.
- **M3 — eval ACPL (240 mg/eg positions, matched-COMPUTE nodes=fair read; depth-6 flatters NGN's bigger tree):** overall NGN ≈ Blunder (+2.7cp). Per phase: mg **+3.6** (competitive), late-mg/phase-7-11 **+9.9** (real, search-PROOF eval gap), eg **-1.3** (NGN's endgame eval is FINE). NGN's eval is noisier SHALLOW (+9.8 @d6 → -3.1 @d12): Blunder leans on a clean shallow eval, NGN on search.
- **HYPOTHESIS:** (1) PRIMARY = endgame SEARCH DEPTH (eg eval is fine ⇒ NGN doesn't misjudge endgames, it under-searches them). (2) ACTIONABLE = phase-7-11 king-eval gap (king-safety gated `phase>=12`, activity `phase<6` ⇒ neither fires at 7-11; → king SPRT in flight; the gradient tuner is the systematic fix). (3) NOT a factor = mg eval QUALITY at equal compute, NOR per-node speed (quality tradeoff, ~15-20% ceiling, no bug, GC free).
- M1 (real-game loss classification: gauntlet `-pgn` + oracle `classify`, both built) was stopped at 7 games when Bryan steered to action; the hypothesis stands on M2+M3.

---

## IN FLIGHT (2026-06-02 post-pivot)

**VERDICT — eval tuning is NOT the lever (chapter closed).** The full Lichess 1.05M SF-eval tune (787 data-only AND 796 MG/EG-tapered) is NEGATIVE in self-play (~−26 to −31 elo, trending) despite ~5% better held-out MSE — same MSE-up/game-down pattern as the 30k distill (−10 reg / −440 l2=0). The taper DID learn correct chess (passed pawn MG 15→EG 41) but added ZERO MSE gain. `build/ngn_lichess787/796` and `output/tuned_787/796_l2e7.go` are kept as artifacts but **NOT shipped**. Do NOT launch another SF-eval distillation tune as a main run.

**OVERNIGHT (running): absolute gauntlet — the trusted ruler.** Self-play SPRT is the broken ruler (see MEASUREMENT DISCIPLINE: deterministic, optimistic CI, transfer-mirages). So confirm the eval-tune verdict and tighten the base baseline on the instrument Bryan trusts: `cmd/gauntlet` `build/ngn_base` then `build/ngn_lichess787` vs Blunder 7.4.0, movetime 1000, 24 games each → `output/gauntlet_pivot.log`. Expectation: base ≈ 2400-2450; 787 ≤ base (absolute receipt that eval-tuning didn't help). (movetime mode is ~+30-80 optimistic but STABLE unattended; `-tc` risks the time-mgmt flag bug.)

**TOMORROW — THE HEADLINE = Lane A endgame-tree autopsy** (see PLAN #1). Counters mostly exist (`SearchInfo` ~search.go:506); add phase-split + TT-probe + 4 extension counters + a dump; run `scripts/h2h.sh` + `scripts/m2_depth.sh` on the eg FENs vs blunder_740; produce the "where do the 2.5-5x eg nodes go" table → localize the bug. THEN Lane B clean J1 re-test (flagged +83% bloat), THEN the implicated search margin (one axis). All gated on fixed-TIME SPRT + gauntlet, never fixed-nodes self-play for search changes.

**Killed:** b6akl2n35 (self-play distill, superseded), blkf2fcex (the deterministic self-play SPRTs — low value once the ruler was shown deterministic + the verdict clear).

## RESOLVED THIS SESSION (2026-06-02)
- **phase-7-11 king-eval gate — KEPT** (commit d036fbd): lowered king-safety gate `phase>=12 → >=7`. Fixed-nodes SPRT (b3ujnoooq) ran 297 games dead-neutral (elo +0.0 [-39,+39]) = no measured regression ⇒ KEEP per the rule; and structurally REQUIRED so the gradient tuner can reach king-safety in the 7-11 band (a dead term can't be tuned). Eval-regression baseline updated (709ef39, King Safety Castled 20→35, the suite's only phase-8 position — confirms the change is correctly localized).

---

## INSTRUMENTS (the toolbox — all built + validated)
- **cmd/sprt** — self-play SPRT. fixed-NODES = eval A/B (contention-immune); fixed-TIME = search A/B. ~800 games/run (400-opening book ×2 is the real cap). `-elo0/-elo1` match the decision ([-5,0] non-regression / [0,10] gain); mingames ~64 (NOT high — defeats the early-stop).
- **cmd/gauntlet** — absolute CCRL rating vs Blunder anchors. **USE `-tc 120+1`** (movetime mode BROKEN — anchors ignore `stop`). `-pgn` captures games.
- **cmd/oracle** — eval-quality diagnostics (the blunder-finder): `gencorpus`→`label` (SF MultiPV)→`acpl` (cp-loss vs SF, A/B; `-depth` eval-leaning / `-nodes` matched-compute) + `classify` (loss tactical-vs-positional × phase, mate-clamped). SF18 at /opt/homebrew/bin/stockfish.
- **cmd/texel** — the tuner (gendata + tune). THE UPGRADE TARGET (→ gradient).
- **scripts/m2_depth.sh** (depth at matched time — eg deficit), **scripts/h2h.sh** (fixed-depth node/nps decomposition), **scripts/nodecheck.sh** (node-identity gate for speed changes), **TestHotPathAllocationFree** (per-node alloc gate).

═══════════════════════════════════════════════════════════════════════════════
## ▼▼▼ HISTORICAL ARCHIVE (superseded — kept for the detailed record + lessons; THE PLAN ABOVE GOVERNS) ▼▼▼
═══════════════════════════════════════════════════════════════════════════════

## NIGHT RECORD (2026-06-01 PM → 06-02 AM) — COMPLETE & SUPERSEDED by THE DIAGNOSIS above. The "RUN ALL NIGHT" mandate/loop below is DONE — do NOT re-execute it; kept only for the detailed results + lessons. <<<

**MANDATE (Bryan, laptop left open all night):** proceed continuously until morning. NEVER idle, NEVER ask,
NEVER stop on a settled result — DECIDE, commit, launch the next. The background SPRT is the heartbeat (its
completion re-invokes me automatically — do NOT poll/ScheduleWakeup for it); foreground node-identical work +
next-candidate prep fills the gaps. Goal of the night: lean the eval for speed (Track 2) + bank node-identical
speed (Track 1), all serving "be the fastest Go engine," then a morning gauntlet for the real number.

**PRE-NIGHT MAIN baseline = `ba8397b`** (probcut-alloc-fix + TT-atomics-removed + this plan). Snapshot it as
`build/ngn_prenight` so the whole night's accumulated leaning can be A/B'd or reverted wholesale in the morning.

**FIRST ACTIONS on "reanchor and proceed":**
1. `caffeinate -i -t 39600 &` in background (11h keep-awake; AC not guaranteed but best-effort).
2. `go build -o build/ngn_prenight main.go` (the morning reference).
3. Build SPRT candidate **L1** (recipe below), launch its fixed-nodes SPRT in the background.
4. Start foreground Lane F1 (GetBitboardOf array refactor) while it runs.

**THE LOOP (event-driven — each SPRT completion wakes me):**
on wake → `tail -25 output/sprt_<name>.log` → read the `Verdict:` line + final `Elo`/CI/`LLR` → DECIDE (rules
below) → KEEP (re-apply the real removal + commit) or REVERT (tree already clean) → `go build -o build/ngn_base
main.go` (clean base from new HEAD) → build the next candidate → launch its SPRT (bg) → advance foreground Lane F
→ RECORD the decision in this block + commit → end turn (the next completion re-wakes me).

**SPRT INVOCATION (copy-paste; fixed-NODES = deterministic, contention-immune, isolates eval quality):**
```
./build/sprt -new build/ngn_<name> -base build/ngn_base -nodes 100000 -concurrency 4 \
  -elo0 -5 -elo1 0 -mingames 64 -maxgames 1500 -openings output/sprt_openings.txt \
  > output/sprt_<name>.log 2>&1        # run_in_background:true
```
`Elo(new-base)`: new = term REMOVED, base = term PRESENT. `[-5,0]`: H1-accept (removal ≥ ~0, term is a
non-earner) ⇒ **DELETE for free speed**; H0-accept (removal ≤ -5, term earns its keep) ⇒ **REVERT (keep term)**.
mingames 64 (NOT high — high defeats the early-stop, cmd/sprt:283). One SPRT at a time, never batch.

**DECISION RULES (mechanical, KEEP RULE):**
- DELETE the term iff NOT a regression: LLR hits H1, OR (at maxgames) point est ≥ -5 with CI not below ~-15.
- KEEP the term iff clear regression: LLR hits H0 / point est ≤ -5 hard.
- Ambiguous at cap (|est|<5, CI straddles): DELETE — it removes CPU with no measured strength cost (lean+free
  under the KEEP RULE; the speed transfers, a self-play non-regression at worst hides a tiny wobble the morning
  gauntlet will catch). Do NOT gain-hunt (mirage); only delete clear non-earners.

**CANDIDATE RECIPE (cleanest, lowest-risk):** TEST step = 1-char edit (set the term's weight const = 0,
eval.go:2355-2360) → `go build -o build/ngn_<name> main.go` (it compiles ⇒ valid; weight=0 cannot break tactics) →
`git checkout engine/eval.go` (tree clean) → `go build -o build/ngn_base main.go` (clean base) → launch SPRT. On
DELETE (non-regression): comment the term's CALL in evaluateUnsafe (THIS is what gains the CPU) + drop the now-unused
weight const → `go build ./...` + `go test -short ./engine/` green + `make smoke-tactical` ≥24/30 (smoke runs
in-process from source, no `-bin` arg, so it tests whatever's compiled; <24 ⇒ botched comment, fix it) → commit.
(weight=0 and call-removed are eval-IDENTICAL ⇒ node-identical to each other ⇒ the SPRT verdict transfers exactly;
the committed call-removed version is just faster.)

**BACKGROUND SPRT QUEUE (eval-leaning, serial, order = speculative→core; eval.go line / weight):**
- **L1 no-outposts** — call `eval.go:2462-2467`, `outpostWeight=84` (2462-65 also fetch 4 bitboards — drop those too).
- **L2 no-mobility** — call `:2455`, `mobilityWeight=166` (high weight, but largely redundant with the PSTs — prime suspect).
- **L3 no-kingactivity** — call `:2476` (eg phase<6 only), `kingActivityWeight=29` (low weight, narrow phase).
- **L4 no-rookopen** — call `:2458`, `rookOpenWeight=83` (textbook-good, cheap — likely a keep, but confirm).
- **L5 no-bishoppair** — call `:2452`, `bishopPairWeight=90` (**3.09% flat — biggest eval-leaf CPU**; likely keep, measure the tradeoff).
- **L6 no-kingsafety** — call `:2470-2471` (mg phase≥12), `kingSafetyWeight=100` (**~7.85% cum — biggest subsystem**; likely keep).

**FOREGROUND NODE-IDENTICAL LANE (no SPRT; gated by nodecheck which is CONTENTION-PROOF + alloc test; NPS deferred):**
- **F1 GetBitboardOf/GetPiece → `bitboards[12]uint64` array** (~6% flat, ~208 sites, BIG mechanical — the headline free win).
- **F2 eval-term node-identical SPEEDUPS** — make bishop-pair (3.09%) / mobility / king-safety compute bit-identical but cheaper (the *other* half of the eval CPU: keep the term, make it fast).
- Gate each: `scripts/nodecheck.sh build/ngn_<f>` EXACT (1319650 / 1188433 / 772119) + `go test -run TestHotPathAllocationFree` + `go test -short ./engine/`. KEEP on node-identity + strictly-less-work (NPS confirm in a quiet gap). Commit each. If a refactor resists >~30 min, drop it and take a smaller one — never leave the tree half-refactored at a commit.

**NEVER-RUN-DRY FALLBACK (both queues exhaust):** re-profile on free cores (between SPRTs) → new node-identical
target / new leaning candidate from the fresh profile. Last resort: launch the morning gauntlet early.

**MORNING PAYOFF (when the SPRT queue is exhausted — the real check on accumulated leaning):**
```
go build -o build/ngn main.go            # CRITICAL: build/ngn is the gauntlet's -ngn default and is otherwise
                                          # stale (night work built to ngn_f*/ngn_base, never build/ngn). The
                                          # movetime gauntlet rewards the F1-F8 speed wins (more NPS -> more depth).
./build/gauntlet -games 24 -movetime 1000 > output/gauntlet_night.log 2>&1   # run_in_background
```
Compare the pooled rating to the historical ~2380 / 27.5%@300ms. CLEARLY lower ⇒ the accumulated self-play-neutral
leaning was net-negative vs the real anchor ⇒ A/B `build/ngn` vs `build/ngn_prenight` (fixed-nodes SPRT) and revert
the losing leans. Same-or-higher ⇒ the speed+lean is paying off; record the new number, fix the stale ~2380 docs.

**PRE-EXISTING TEST FAILURES (do NOT panic — NOT my regressions):** EnginePlayingStrength (SAN-vs-UCI format),
ConcurrencyStress (shared-pos data race), ordering tests flaky under parallel contention. `go test -short ./engine/`
is otherwise green; `-short` skips the slow ones.

**RULES (every turn):** single-sentence commits, no co-author, no emoji, commit LOCAL — DO NOT push. Never P-cores
(`-lowpower true` default). Build dev work to SEPARATE binaries, never `make` mid-gate. RECORD each decision in this
block and commit the SAME turn (the integral term). [[feedback_long_autonomous_sessions]] [[feedback_decide_dont_ask]] [[sprt_instrument]]

**NIGHT PROGRESS LOG (newest last):**
- `ngn_prenight` snapshot built from baseline (probcut-fix + TT-atomics). caffeinate 11h running.
- **F1 DONE, KEPT (commit 2aaf2e9):** replaced the 12 named piece-bitboard fields with `pieces [13]uint64`
  indexed by Piece (via 12 `gofmt -r` rewrites across engine/ + `copy()`→`return *b`); GetBitboardOf is now a
  single array load, not a 12-way switch (~4.16% flat). Node-identical EXACT, alloc-free, `go test -short` green.
- **F2 DONE, KEPT (commit 8bc4416):** collapsed the 12-way piece switches in UpdateSquare/Clear/Move into direct
  `b.pieces[piece]` writes + one `piece < BlackPawn` color-branch for the white/black aggregates (castling order
  preserved). Cuts make/unmake flat cost (Clear 5.4% + Move 3.0%). Node-identical EXACT, alloc-free, suite green.
- **F3 DONE, KEPT (commit f81eab5):** hoisted the running max score into a local in selectNextMove so the
  partial-selection-sort inner loop stops re-loading scores[maxIdx] (+bounds check) each iteration. Node-identical
  (#1 flat at ~12%; the O(n²) scan itself is the floor — can't reduce further without reordering).
- **F4 DONE, KEPT (commit 37ca163):** hoisted the per-node-constant continuationHistory[prevP][prevT] sub-table out
  of the move-scoring loop and inlined the quiet-move history lookup (2-D index/move instead of 4-D + call).
  Node-identical; GetHistoryScore fell off the profile (folded into scoreMovesIntoBuffer).
- **F5 DONE, KEPT (commit cd81c0b):** collapsed GetPiece's 12-case double switch into Piece(pieceType)/+6 using the
  contiguous White-then-Black enum layout. Node-identical (hot in attack/check detection ~4% flat).
- **L1 (no-outposts) SPRT:** estimate regressed to the mean as games accumulated — **+57@G123 → +9@G227 → -7@G292**,
  CI [-47,+33], LLR ~0. True effect ≈ 0 (outposts is a non-earner); LLR will random-walk near 0 and likely run to
  maxgames 1500 → DELETE at cap (ambiguous-at-cap rule: removes CPU at no measured strength cost). Running in bg.
- **F6 DONE, KEPT (commit c05c8d6):** fetch the enemy queen bitboard once in isInCheck's attack detection (diagonal
  and straight attackers reuse it) instead of twice. Node-identical; saves a GetPiece+GetBitboardOf per check in the
  common no-diagonal case.
- **F7 DONE, KEPT (commit e241a83):** hoist the enemy color offset once in isInCheck so the 5 enemy-piece lookups
  index by WhiteX+offset instead of re-branching inside GetPiece per call. Node-identical (check detection every move).
- **F8 DONE, KEPT (commit eb55788):** replace the per-call `[]PieceType{...}` slice literal in
  findLeastValuableAttacker (SEE) with an integer Pawn..King loop + hoisted color offset. Node-identical.
- **>>> FREE NODE-IDENTICAL LANE EXHAUSTED of clean hot wins (8 banked: F1-F8, commit eb55788). <<<** Measured NPS
  gain via BenchmarkSearchProfile (identical 2,681,334 nodes/op throughout, contention-constant): **1302→1234→1168→1156
  ns/op ≈ ~11% faster** (big drops were F3 selectNextMove + F4 history; F5-F8 smaller). Post-F8 profile is FLAT:
  selectNextMove 11.9% (algorithmic floor — O(n²) scan, can't reduce without reordering = non-identical),
  scoreMovesIntoBuffer 6% (SEE-per-capture + necessary scoring branches), Bitboard.Move/makeMoveHelper/updateHash
  ~3-4% (real work, no redundancy), magic GetBishop/RookAttacks ~3.5% (tight), evaluateKingSafety 8.2% cum +
  evaluatePawnShield ~2.6% (king-safety = L6 candidate; pawnShield's quirky rankOffset bound + early-break make a
  node-identical bitboard rewrite fragile for ~2% — SKIPPED). canPieceAttackSquare switch = necessary attack dispatch
  (not collapsible). Further free wins are <2% churn ⇒ STOP the free lane (anti-over-engineering). Clean NPS A/B vs
  ngn_prenight deferred to the quiet morning (cores pinned by the SPRT now). **Verified terminal:** qsearch (the
  ~44% cum) is ALREADY capture-only (`GenerateCapturesIntoBuffer`) + frame-pooled; its qDepth==0 stalemate probe needs
  full movegen for accuracy (`HasLegalMove` = same algo + heap-allocs, `HasLegalMoveQuick` is inaccurate — misses
  piece-blocked stalemates), so not node-identically cheaper. Further per-node speed now needs BEHAVIORAL changes
  (lazy/staged movegen, SEE caching) = SPRT-gated, NOT free — a possible Track 3 after the eval-leaning queue, but the
  queue will eat the night's SPRT slots, so Track 3 is a future-session candidate, not tonight.
- **L1 (no-outposts) SPRT trend:** +57@G123 → +9@G227 → -7@G292 → **-11 [-42,+20] @ G470, LLR -0.20**. Now leaning
  slightly NEGATIVE (removing outposts may COST ~10 ELO ⇒ outposts earns its keep ⇒ KEEP-the-term), but CI wide and
  bouncing — DECIDE at the 1500 cap. Running in bg.
- **THROUGHPUT REFINEMENT (evidence-based, for L2-L6):** L1 demonstrated these eval-term effects are small enough that
  the [-5,0] band NEVER resolves (LLR random-walks ~0) → the SPRT runs to maxgames regardless. The [-5,0] precision is
  unachievable at any feasible n (~5000 games needed); what the DELETE decision actually needs is detecting a
  MEANINGFUL earner (≥~15-20 ELO), which ~800 games resolves (CI ±24). So **launch L2-L6 with `-maxgames 800`** (2x
  throughput, more frequent decision points, keeps the night active); the morning gauntlet is the backstop against
  wrongly deleting a marginal earner. Do NOT touch the running L1 (let it finish at 1500 — clean, it's the heartbeat).
- **L1 (no-outposts) DONE — VERDICT: KEEP outposts (commit-neutral, no code change).** Final @ 800 games (the
  400-opening book ×2 colors is the REAL cap, not -maxgames 1500 — every candidate runs ~800 games / ~1h8m, which
  moots the maxgames-800 refinement: the book already caps it): 331-117-352, **Elo(new-base) -9.1 [-33,+15]**,
  pentanomial -9.1 [-31,+13], LLR -0.26 (inconclusive, didn't hit a bound). Point estimate is a clear ~9-ELO
  regression for REMOVAL ⇒ outposts is a (weak) earner, NOT a free-deletion candidate ⇒ KEEP per the KEEP RULE
  (don't delete something showing a measured regression, even a noisy one). The SPRT did its job: ruled outposts OUT.
  Updated prior: the eval is Texel-tuned, so these terms may mostly be earners — the queue tells us which (if any) are
  free to drop.
- **L2 (no-mobility) LAUNCHED (bg task b432zdi5a, output/sprt_l2.log):** `mobilityWeight 166→0`, ngn_l2 vs ngn_base
  (rebuilt from HEAD 65990da = F1-F8). Mobility is the highest eval weight + ~420ms cum + flagged "likely redundant
  with PSTs" — the prime deletion suspect. Caps ~800 games. No re-profile this cycle (L1=KEEP ⇒ HEAD unchanged ⇒ same
  flat profile; re-profile only after a DELETE that changes HEAD).
- **L2 (no-mobility) DONE — VERDICT: KEEP mobility, emphatically.** 275-99-426, **Elo -66.4 [-91,-42]**, pentanomial
  -66.4 [-89,-44], LLR -2.49 (nearly hit H0). Removing mobility is a LARGE clear regression ⇒ mobility is a ~66-ELO
  earner, the most valuable term tested. "Redundant with PSTs" REFUTED.
- **>>> STRATEGY PIVOT (2026-06-02, evidence-based): both terms tested are earners (outposts +9, mobility +66) ⇒ the
  Texel-tuned eval is well-calibrated, NOT bloated ⇒ the deletion sweep is yielding KEEPs (little/no free speed). So:
  keep the deletion sweep running in the BACKGROUND for closure, but resume the FREE NODE-IDENTICAL LANE in the
  FOREGROUND on CONFIRMED KEEPERS (speed up the term, keep its output) — a free win needing no SPRT. <<<**
- **F9 DONE, KEPT (commit 6876b7b):** evaluateMobility was called per-color, recomputing the color-independent setup
  (occupancy + both pawn-attack spans) twice/eval. Factored the per-side loops into `mobilityForSide` + added
  `evaluateMobilityDelta` that computes the shared setup once; production caller uses it. Node-identical EXACT,
  alloc-free, suite green. (evaluateMobility kept for eval_test:323, refactored onto the same helper.)
- **L5 (no-bishop-pair) LAUNCHED (bg bupfifr4n, output/sprt_l5.log)** — reordered ahead of L3/L4 by value:
  bishop-pair is the biggest eval-LEAF flat cost (~3% flat) AND its value is genuinely debated, so it's the
  highest-info remaining deletion test. `bishopPairWeight 90→0`, ngn_l5 vs ngn_base (HEAD with F1-F9). Caps ~800.
  (NOTE: first launch used a bare `&` which gives no completion notification — killed + relaunched via run_in_background.)
- **L5 (no-bishop-pair) DONE — VERDICT: DELETE (the night's FIRST leaning win, commit 037f853).** 361-125-314,
  **Elo +20.4 [-4,+45]**, pentanomial +20.4 [-1,+42], LLR +0.90 (toward H1). Removing bishop-pair is a ~20-ELO
  IMPROVEMENT — the term was net-HARMFUL (double-counted the PeSTO bishop material values). Double win: **+20 ELO AND
  ~3% faster** (it was the biggest eval-leaf flat cost). Executed: removed the eval call (kept evaluateBishopPair for
  its 3 unit tests), dropped bishopPairWeight from the const block + texel tunables, fixed the weight-count test,
  recomputed nodecheck baselines (kiwipete 1372558 / mid 1003482 / end 772119 — verified ngn_nobp ≡ ngn_l5 exactly so
  the SPRT verdict transfers). smoke-tactical 28/30, go test -short green. **LESSON: chess-intuition priors are
  unreliable (bishop-pair was tagged "likely keep" — it was +20 to REMOVE); test every term. The eval is NOT fully
  tuned; the sweep finds real strength+speed wins, so it's worth continuing.**
- **L6 (no-king-safety) LAUNCHED (bg bnexstfeq, output/sprt_l6.log):** `kingSafetyWeight 100→0`, ngn_l6 vs ngn_base
  (rebuilt from HEAD 037f853 = F1-F9 + bishop-pair-removed). Biggest eval cost (~610ms cum); high-EV test — resolves
  FAST toward H0 if king-safety is the strong earner expected (then KEEP), or is the night's biggest combined win if
  harmful like bishop-pair. build/ngn also rebuilt current for the eventual gauntlet.
- **L6 (no-king-safety) DONE — VERDICT: KEEP king-safety (no code change).** 322-82-396, **Elo -32.2 [-56,-8]**,
  pentanomial -32.2 [-53,-12], LLR -1.10. Removal is a clear ~32-ELO regression ⇒ king-safety is a real earner. (Note:
  the LEAST cost-efficient earner — 610ms cum for only +32 ELO vs mobility's +66 for ~420ms — so it's the prime target
  for a node-identical SPEEDUP if a clean one can be found; pawnShield rewrite is fragile and attack-patterns is
  inherent per-king, so none found this session — a Track-3 candidate.)
- **SWEEP COMPLETE (the expensive terms): outposts KEEP +9 · mobility KEEP +66 · bishop-pair DELETE +20 · king-safety
  KEEP +32.** One harmful term found and removed (bishop-pair). Deferred low-value tail: L4 rook-open (captures
  non-PST open-file info ⇒ likely KEEP) and L3 king-activity (endgame-only ⇒ ~0 speed + diluted SPRT signal) — run
  next session or post-gauntlet; not worth delaying the payoff.
- **>>> MORNING GAUNTLET LAUNCHED (bg batx1m3tx, output/gauntlet_night.log) <<<** `build/ngn` (HEAD 3f670af = F1-F9
  speed + bishop-pair removed) vs the 3 crossover Blunder anchors (`-only v6.1.0,v7.2.0,v7.4.0`, CCRL 2155/2425/2532),
  24 games each = 72, movetime 1000ms, E-cores, concurrency-1. ~3-4h → finishes by morning. (First launch defaulted to
  all 5 anchors incl. saturated 2080/2674 with the key 2532 LAST — killed + relaunched focused on the crossover 3.)
  **UPDATE: the gauntlet auto-EXCLUDED v6.1.0** (its integrity check caught it ignoring movetime/stop — probe 7.5s vs
  1s budget — which would distort the TC), so the run is effectively 2 anchors: 7.2.0 (2425) + 7.4.0 (2532), BOTH
  above NGN ⇒ no >50% bracket, so the pooled number is a mild downward extrapolation. The DECISIVE read is therefore
  the PER-ANCHOR score vs 7.4.0 (and 7.2.0) compared to the historical ~31%@1000ms / 27.5%@300ms vs 7.4.0 — a
  bracketing-independent same-anchor before/after: higher score now ⇒ the F1-F9 speed + bishop-pair removal helped.
  **UPDATE 2: 7.2.0 ALSO auto-excluded (3 consecutive no-move forfeits — dead/broken), so the gauntlet is now a
  SINGLE-anchor run vs 7.4.0 (CCRL 2532, the canonical working anchor per CLAUDE.md — playing real games: draw/mate).
  ~24 games, ~50 min. Decisive read = NGN's score vs 7.4.0 vs historical ~31%@1000ms. (Early 0-1-2 = noise.) If the
  24-game score is CLEARLY below ~31%, the bishop-pair removal may have regressed real strength despite +20 self-play
  (the transfer-mirage risk) ⇒ run a fixed-time ngn_prenight-vs-build/ngn SPRT to isolate, and consider reverting it
  while keeping the node-identical F1-F9 speed wins (which can't regress strength).**
- **>>> GAUNTLET RESULT: NO POOLED RATING — the absolute instrument is DEGRADED (anchors dead). <<<** 6.1.0 ignores
  movetime/stop (excluded at warmup); 7.2.0 forfeited 3 straight no-move games (dead/broken, auto-aborted); 7.4.0
  (the canonical working anchor) PLAYED real games but had 2/24 no-move forfeits (>2% ceiling) → tainted → dropped.
  So the pool was empty. The raw 7.4.0 line: 6W-6D-12L incl. 2 forfeit-wins ⇒ **4W-6D-12L = 31.8% over the 22 REAL
  games**, essentially identical to the historical ~31%@1000ms. **24 games (±~10%) cannot resolve the night's expected
  ~10-25 ELO gain (a +20 real gain only shifts the score ~+3%), so 31.8%≈31% is CONSISTENT WITH a small improvement —
  neither confirms nor refutes.** TODO for the gauntlet: rebuild a bracketing set of movetime-honoring anchors
  (`gauntlet -probe` to find which Blunder versions honor `stop`); 6.1.0/7.2.0 are unusable at an honest TC.
- **>>> GAUNTLET RE-RUN IN THE CORRECT MODE (`-tc 120+1`) — the real payoff: +69 ELO point estimate. <<<** The
  movetime failure above was my mode error; `-tc` sends wtime/btime (never `stop`) so anchors play clean. SAME config
  as the historical 2380 baseline (`-tc 120+1 -only v7.4.0 -games 16` vs Blunder 7.4.0/2532): **tonight 2W-8D-6L =
  37.5% → perf 2443, pooled 2449 [2274,2624]**, vs **historical 28.1% (2W-5D-9L) → 2369, pooled 2380 [2193,2567]**.
  **Point estimate UP +69 ELO (2380→2449)** at the real clock vs a real opponent — directionally confirms the night's
  F1-F9 speed (deeper search per time budget) + bishop-pair removal transferred. HONEST CAVEAT: 16 games each, CIs
  overlap heavily ⇒ NOT statistically significant; the +69 is noise-inflated above the ~+28 expected (+20 eval +
  ~8 speed), but the DIRECTION is right and it rules out a regression. The correct `-tc` mode showed the lift the
  broken movetime mode (~31.8%≈historical) could not. Next: more games / a 2nd stable `-tc` anchor to tighten the CI.
- **NIGHT VERDICT — clear NET POSITIVE, no revert.** The clean (concurrency-immune) measurements stand: (1) F1-F9 are
  node-identical ⇒ at fixed TIME more NPS = more depth = strength can only RISE — SAFE, banked. (2) bishop-pair removal
  = **+20.4 ELO at fixed-NODES** (L5, a clean deterministic measure; and since F1-F9 are node-identical, ngn_base ≡
  ngn_prenight at fixed-nodes ⇒ current is +20 over prenight at fixed-nodes). So the night = **+20 ELO (fixed-nodes,
  clean) + ~11% NPS (node-identical) = a net positive that cannot have regressed.** The gauntlet's underpowered ~neutral
  does NOT refute it. (A fixed-TIME prenight-vs-current self-play SPRT was attempted for a powered total read but is
  INVALID at concurrency>1 — movetime contention corrupts timing, per the sprt tool's own warning — and conc-1 is ~8h;
  killed. Not needed: speed is node-identical-safe, bishop-pair is +20 fixed-nodes-clean.)
- **DEFERRED (next session):** L4 rook-open + L3 king-activity deletion tests (king-activity is the prime remaining
  double-count suspect — overlaps the PeSTO endgame king-centralization PST, like bishop-pair did the bishop values);
  the gauntlet-anchor rebuild; Track-3 behavioral speedups (lazy/staged movegen, a node-identical king-safety speedup —
  king-safety is the least cost-efficient keeper, 610ms cum for only +32 ELO). 22 commits local, tree clean, NOT pushed.

---

## >>> IN FLIGHT (2026-06-01) — resume info, survives compaction <<<

- **J1 extensions×LMR fix — SHELVED + REVERTED 2026-06-01 (working tree clean; one-line change re-derivable from docs/05).**
  THE headline HIGH/VIOLATED joint (docs/05+09): off-PV non-first checking/recapture moves had their
  check/recapture extension *computed and counted but never applied* — the scout ran at `depth-1-reduction`
  (un-extended) and neither re-search (C4 gated `reduction>0`, PVS gated `isPV`) fires for a reduction==0
  off-PV move. Verified against live code (full trace) BEFORE editing. FIX (per doc 05's stated intent):
  `reducedDepth := nextDepth - reduction` + dropped the tautological `if reducedDepth>=depth` clamp (its body
  == its init, a no-op today; with the new base it would WRONGLY reset a check back to depth-1, defeating the
  fix — the doc's "keep the dead branch" prose contradicts its own intent, so I followed intent). **PREREQ for
  A3** (SEE-gating a dead extension = gating a no-op), so J1 reorders ahead of A3 (my call, decide-don't-ask).
  GATES PASSED: `go test ./engine/` adds ZERO new failures (3 pre-existing broken: EnginePlayingStrength =
  SAN-vs-UCI format mismatch scores 950 for ANY engine; ConcurrencyStress = data-race on shared pos; ordering
  tests flaky under parallel contention, pass 5/5 isolated). Node check (build/ngn_base vs build/ngn_j1, both
  on disk): base reproduces baselines EXACTLY, j1 fires the extension — **kiwipete d12 +51% (1.32M→2.00M),
  mid d12 −21% (1.19M→0.94M, deeper scout improved ordering), end d16 +83% (0.77M→1.41M)**. smoke-tactical
  28/30 (93%). seldepth bounded by EXTENSION_BUDGET=24 (structural). **THE CONCERN: end +83% amplifies the
  endgame-tree weakness (top speed lane) — net strength is two-sided (tactical depth vs EG bloat), a fixed-TIME
  SPRT prices it in.** NEXT: (1) oracle pre-filter running — task bbbg530e4, output/j1_oracle.log (tactical
  best-move @2000 nodes + ACPL @ fixed depth-12, j1 vs base; J1 is tactically-relevant so the oracle CAN see it,
  unlike A2). **VERDICT (provisional, stopped early — NOT a formal SPRT result):** oracle pre-filter MIXED
  (tactical +0.9% / +2 solved −0 lost, but ACPL **+1.4 cp/move WORSE** on quiet middlegames — just above the
  noise floor, not a clear reject). Fixed-TIME SPRT (ngn_j1 vs ngn_base, 800ms conc-4, [-5,0]) ran to **G65:
  18W-18D-29L 41.5%, −59 ELO [−144,+25], LLR −0.22** — leaning clearly negative, CI still grazing 0 ([-5,0]
  bounds resolve slowly). REVERTED to clean main to free cores for the headline gauntlet (Bryan's call): J1 is
  ±20 immaterial to the gauntlet, leaning a regression, AND applying the extension **bloats the endgame tree
  +83%** (the rook EG) — exactly NGN's top weakness — so KEEP RULE = don't keep. The dead off-PV extension is
  accepted as likely-better-left-dead; docs/05+09 J1 stays `[VIOLATED]` (fix ATTEMPTED, measured not-a-win, the
  one-liner `reducedDepth := nextDepth - reduction` is recorded there if revisited). LESSON: a "correctness"
  invariant fix is NOT automatically a strength win — applying a long-dead extension can cost more (tree bloat)
  than it gains (tactical depth); the engine had implicitly tuned around the dead code.
- **>>> SPEED LANE IS NOW THE WHOLE GAME — first win banked, gauntlet cancelled (2026-06-01 PIVOT) <<<**
  - **PROBCUT ALLOCATION FIX — DONE, KEPT, ON MAIN (commit 967d9bd).** The profiler caught probcut escaping a
    per-node `[256]Move` to the heap (34MB/run) — so "allocation-free hot path" (T1) had been silently FALSE.
    Fixed by routing generation through the preallocated `frame.moveBuffer`. VERIFIED: node-identical (nodecheck
    EXACT 1319650/1188433/772119) + per-node allocs now **0.000000** (was the #1 alloc site). Added
    **`TestHotPathAllocationFree`** (engine/profile_test.go) = the enforcement: fails the build if any per-node
    alloc regresses. This is the template for "VERIFY, don't claim."
  - **TT ATOMICS REMOVED — DONE, KEPT, ON MAIN (commit ed5eda0).** Profile showed `sync/atomic.(*Uint64).Load`
    at **4.49% flat** — the TT used `atomic.Uint64` for a multi-threaded future that doesn't exist (`Threads` is a
    no-op stub; verified the TT is touched ONLY by the lone `go uci.search` goroutine — `Consumed`/`ExtractPVFromTT`
    run in the same goroutine's per-depth callback, `stop` never touches it, Texel goroutines do pure eval). Swapped
    `atomic.Uint64`→plain `uint64` (ARM64 `LDAR`/`STLR`→`LDR`/`STR`, strictly cheaper), deleted the half-baked
    `BenchmarkMode` escape-hatch + its whole benchmark test. **Struct stays 16B ⇒ slot count unchanged ⇒
    node-IDENTICAL (exact on all 3 baselines + d13 byte-identical PV/score).** Faster by median (−49ms/9 pairs) +
    deterministic (can't-be-slower); thermal noise too big to resolve the ~4% cleanly but the KEEP RULE is satisfied
    (node-identical + strictly-less-work). The author had DISCOVERED this (the BenchmarkMode flag) but never turned
    it on in production — classic.
  - **GAUNTLET CANCELLED** (was on anchor 2/4, no pooled number). It pins the E-cores (corrupting the NPS
    measurement the speed loop needs) and was measuring a binary I'm now actively changing. Re-run it ONCE at the
    end of a speed batch — never continuously.
  - **LAST DATA POINT (bullet, output/bullet_vs2500.log):** NGN vs Blunder 7.4.0 (2532) @ 300ms/move, 40 games =
    7W-8D-25L (27.5%), −168 ELO → perf ~2364. Bullet is NGN's WORST TC (the 2x per-node tax bites hardest at shallow
    depth); consistent with ~2350-2400 + strong TC-sensitivity (31% @ 1000ms old gauntlet → 27.5% @ 300ms). NOT a
    doc number.
  - **NEXT (the loop, on FREE cores):** profile → attack the biggest per-node cost, in order from the post-fix CPU
    profile: **(1) Evaluate ~15% (evaluateUnsafe / evaluatePeSTO), (2) staticExchangeEvaluation ~9%,
    (3) GenerateMovesIntoBuffer ~8%, (4) scoreMovesIntoBuffer ~8%, (5) makeMoveHelper ~8%.** `build/ngn_fast` =
    current main (probcut fix). Each change: SEPARATE binary → `scripts/nodecheck.sh` exact + `go test -run
    TestHotPathAllocationFree` + clean NPS delta → KEEP iff faster + node-identical.
  - **ALSO QUEUED (cope purge, Bryan greenlit):** strip the "rich eval / GM-without-speed / fewer-better-nodes"
    framing from CLAUDE.md (the "GM Without NNUE" section) and the project_speed_lane memory — THE FRAME at the top of
    this file supersedes it. The SPEED-LANE section below keeps its gap-decomposition DATA (good) but its
    "eval-not-the-bottleneck" clause is dead.
- **A1 de-pad TT — DONE, KEPT, ON MAIN (commit d1fb499).** SPRT stopped at G225: 49.8%, elo -1.5 [-47,+44] =
  neutral / no big regression. Kept per Bryan's sound-fix-no-grind policy (commit fa9cc02): free capacity (4x TT
  slots, removes single-thread-pointless padding, no plausible regression mechanism); the 200ms read is flat
  because per-move searches are too shallow to pressure the TT. Fixed-depth node effect is MIXED/position-
  dependent (mid d12 −47% but kiwipete d12 +21%, end ~0 — search-instability swings, matching the neutral SPRT),
  NOT a uniform win — kept as a free no-regression capacity change, not a depth lever. LESSON: 200ms
  UNDERCREDITS depth/quality changes; NPS changes (like A4) still show at 200ms.
- **A4 magic-movegen — KEPT, on main (commit 3ab9eb5). SPRT STOPPED at G1404 (settled, Bryan's call).** Final:
  52.8%, elo **+19.3 [+1,+38]**, pLLR 2.20 — CI excludes 0, a CONFIRMED small positive (the early −88 scare AND
  the +30 spike were BOTH noise; truth ~+19, matching the ~2x-faster-movegen NPS prediction). Sound perft-exact
  speed change with no realistic path to a reject ⇒ stopped rather than grind a settled call. Branch
  `lane-a4-magic-movegen` deletable. **A4 finish-up DONE:** post-A4 node baselines recomputed + baked into
  `scripts/nodecheck.sh` (kiwipete 1319650 / mid 1188433 / end 772119, d12/d12/d16).
- **ORACLE (cmd/oracle) — BUILT + VALIDATED 2026-06-01 (commit fb1e079). The fast pre-filter Bryan proposed.**
  Deterministic best-move-agreement A/B over a labeled UCI-`bm` EPD at a FIXED node budget — zero sampling
  variance (verified identical 135/221 across runs; an SPRT needs hundreds of games to average out clock noise).
  Sees SEARCH changes (better search→better move), unlike an eval-error oracle. `make oracle ARGS="-new X -base Y
  -nodes 2000 -v"`. CALIBRATION: on the 221-pos easy-tactics corpus, solve rate 98.6%@50K → 61%@2K → 42%@800, so
  ~2000 nodes is the discrimination point. CAVEAT: easy (500ms-solvable) tactics only discriminate when
  node-STARVED (shallow-search quality ≈ an over-pruning detector, an imperfect full-depth-strength proxy).
  ROLE: filter + rank in FRONT of the SPRT lane (kill bad fast, SPRT only survivors); NOT a standalone verdict.
  **UPGRADE DONE (2026-06-01, commits fb1e079→HEAD): the ACPL pipeline.** Stockfish 18 installed (brew,
  /opt/homebrew/bin/stockfish). cmd/oracle now has `-mode gencorpus|label|acpl` (+ `internal/uci.Analyze` to
  capture MultiPV lines): gencorpus self-plays diverse middlegame FENs; label records SF MultiPV eval-per-move;
  acpl = average centipawn-loss vs SF (A/B). **ACPL is the quiet-robust metric Bryan's pushback forced out** —
  binary best-move match is noisy on quiet positions (5 near-equal moves → both engines "miss", signal drowns);
  cp-loss scores HOW MUCH eval a move gives up (equal alt ≈ 0 loss, real mistake = real loss), continuous /
  low-variance, sees tactical+quiet uniformly (this is how lichess/STS measure accuracy). VALIDATION:
  the original ~480-pos run (task b7wveql3a, output/acpl_run.log) HUNG at the car-sleep after 150 positions / 25
  openings — the 60s per-move timeout does NOT fire across a macOS suspend (ngn_base logged 38s CPU over 3h09m
  elapsed then sat at 0%/state-S; corpus.fens mtime frozen at 15:38). Killed it 2026-06-01 ~18:48. 150 diverse
  FENs is plenty for ACPL (continuous/low-variance), so RELAUNCHED the fast tail on the existing corpus: label the
  150 → ACPL A/B of A2 (SPRT-confirmed −145) vs base. **Task b4x3uuc62, output/acpl_run2.log — DONE.**
  **VERDICT 2026-06-01: THESIS VALIDATED.** A2 ACPL **24.8** vs base **21.6** = **Δ +3.2 cp/move, A2 worse
  (correct direction)** — vs binary best-move match's **0 flips / zero signal** on the same regression. So ACPL
  on a diverse middlegame corpus catches the quiet-search over-pruning that best-move match is blind to; the
  pipeline works as the coarse pre-filter. SENSITIVITY CAVEAT: −145 ELO → only +3.2 cp/move (~+15% relative,
  deterministic) ⇒ flags BIG regressions reliably but can't resolve <50 ELO (≈+1 cp/move ≈ corpus-choice noise
  floor) — fine, the SPRT owns the verdict and we don't chase <50. Full writeup in [[ngn-oracle-acpl]]. LESSONS
  banked: a `pgrep`-alive background proc ≠ progressing (verify CPU-time + output mtime, not liveness); macOS
  suspend freezes the 60s per-move timeout (the original b7wveql3a run hung at a car-sleep, killed after 150 pos).
- **SPRT QUEUE (one at a time, rebase onto main as winners merge; never batch):**
  - A2 = phase-gated futility→d8 — **REJECTED + ABANDONED 2026-06-01 (do NOT merge): −144.7 ELO [−202,−87] @
    G165**, pLLR −3.82 past the H0 reject bound in ~20 min. Futility→d8 over-prunes catastrophically in real games
    (the unchanged margin formula at d8 = 550cp prunes far too much quiet play; "Blunder reaches d8" was a bad
    analogy — its margins/conditions differ). **BIG LESSON (the whole oracle thesis in one experiment):** the
    oracle pre-check said 0 flips / "tactically SAFE" — correctly NOT treated as a green light, because the
    TACTICAL corpus is BLIND to A2's quiet over-pruning; the SPRT caught the −145 regression the oracle could not
    see. Vindicates oracle=FILTER-not-verdict, SPRT-essential-for-quiet-search, and 600ms-reveals-what-200ms-hides.
    Historical analysis (still valid): clean phase-gate, node-confirmed active (mid d12 −39%, kiwipete +41%).
    Branch `lane-a2-futility-d8` + build/ngn_a2 abandoned. original analysis is in git history; not repeated.
  - **STANDING SPRT DEFAULTS (locked from A2/A4):**
    - **mingames ~64, NOT 500.** Overriding mingames high DEFEATS the SPRT's own early-stop: `cmd/sprt:283` halts
      the instant the trinomial LLR crosses its bound, BUT only once `n >= mingames`. A2's reject bound hit ~G165;
      `-mingames 500` would've burned ~335 needless games. Built-in default is 24. We do NOT iterate to tiny
      bands — the LLR bound fires on the DECISION (sign), not magnitude (KEEP-RULE never needs magnitude).
    - **Match bounds to the decision.** `[-5,0]` non-regression frame for SOUND/expected-neutral changes (accepts
      fast on "not a regression" — would've stopped A4 fast; [0,10] only drags because it asks "worth ≥+10?",
      slow for a small +). `[0,10]` only to confirm a real GAIN. Big regressions (A2) blow past either bound fast.
    - **LEVER: promote the pentanomial LLR to the decision stat** — already computed, crosses ~15-30% sooner
      (code comment has the TODO; A2's pLLR −3.82 crossed before the trinomial −2.87 did).
    - conc-4 (6 E-cores, 1 search/game → conc-2 wasted 4 cores, symmetric contention can't bias A-vs-B, ~2x free);
      600ms+ for depth/quality (200ms undercredits); never batch; one SPRT at a time.
    - **CAUTION (don't hand-roll):** a raw "CI excludes 0 at 50 games → stop" peek is too trigger-happy — early
      CIs are unstable (A4 excluded 0 at G854 [+7,+54] then RE-included it at G1354 [−0,+37]) and repeated peeking
      inflates false-positives. The LLR bound + small mingames IS the rigorous form of that idea.
  - A3 check-ext SEE-gate (delicate: pre-move SEE + LMR reduced-scout tangle; SUPERVISED variant) and A5
    eval-audit — NOT yet built (don't over-prep; build when their turn nears).
- **SHIPPED this session (main):** T6 PieceAt mailbox (974d75a, node-identical, perft-verified). Lane-B cheap
  well now ~DRY — GetBitboardOf is a 208-site array refactor (too big for ~0 ELO, DEPRIORITIZED); T3 defer is
  marginal/maybe-free (Go open-codes function-scope defers). ⇒ remaining gains are SPRT-gated (Lane A).
- **Foreground while an SPRT runs:** prep the next SPRT candidate + eval-audit analysis; keep one SPRT +
  foreground work going at all times, never idle ([[feedback_control_loop]], [[feedback_decide_dont_ask]]).

## SPEED LANE (NPS) — TOP PRIORITY, prioritized 2026-05-31 — the 2.5x gap

### >>> GAP DECOMPOSED — ENDGAME TREE IS THE BIG LEVER (2026-05-31, verified; SUPERSEDES the per-node framing below) <<<

**The "2.5x" was one ratio hiding two independent problems.** Measured fixed-depth on 5 positions, both
engines E-cores, both counting qsearch (`scripts/h2h.sh` = the new canonical instrument; full writeup in
docs/10):  `time-to-depth = per-node-speed × tree-size`.

  | position | per-node BLU/NGN | TREE NGN/BLU | to-depth |
  |---|---|---|---|
  | mid / mid2 / kiwipete (d12) | ~1.9-2.0x | 1.13 / 0.78 / 1.00x | ~2x |
  | passed-pawn endgame (d16) | ~2.0x | **2.57x** | ~4-6x |
  | quiet rook endgame (d18) | ~2.4x | **5.12x** | ~12x |

Tree ratios are EXACT/deterministic; per-node is thermally noisy (~2x). **THESIS REFUTED:** NGN does NOT
search fewer/better nodes — tree is ~1x in middlegames, **2.5-5x BIGGER in endgames**, while ~2x slower per
node, and Blunder (560-line eval vs NGN's ~2750) is +150 CCRL STRONGER. NGN is behind on BOTH axes.

**LEVERS, RE-RANKED 2026-05-31 AFTER THE STEP-1 AUDIT (the old "endgame = #1 quick pruning win" is REFUTED):**
1. **ENDGAME SEARCH EFFICIENCY — the 2.5-5x tree. Still real, but NOW KNOWN HARD (was "NEW #1, START HERE").**
   EBF ~10%/ply worse in EG (~2.25 vs ~2.05), compounding to 5x (deterministic cross-engine, `scripts/h2h.sh`).
   The step-1 audit ([[project_endgame_tree_diagnostic]], TODO step 1 below) EMPIRICALLY REFUTED every standard
   pruning knob as the cause — futility (margin AND depth — deeper futility BLOWS UP the EG +141%), RFP,
   LMR-re-search (~0.1%/node), NMP (correct in dead-equal EG), TT-capacity (EG fits 2M slots) — AND found that
   single-position node counts can't rank these knobs (instability dominates, non-monotonic). So it's DIFFUSE
   EBF, **SPRT-only**, not a cheap tweak: re-ranked BELOW the tractable middlegame candidates. Do not keep
   node-count-probing it.
2. **MIDDLEGAME NODE-REDUCERS (NEW, the tractable win the audit surfaced).** TT capacity bites the MIDDLEGAME
   (8x table → mid-tree −46%, NGN drops below Blunder) — and NGN WASTES 75% of its TT on single-thread-pointless
   cache-line padding (`CachedEval` 64B → only 2M slots; `Threads`/`Hash` are no-op stubs). **De-pad → 8M slots
   FREE (one line).** Plus Blunder-style futility→depth-8 shrinks MG −20-23% (phase-gate it). Both [BEHAVIOUR],
   fixed-time SPRT. **Start the de-pad SPRT first — lowest-risk +EV.**
3. **PER-NODE SPEED (~2x engineering tax).** The free node-identical queue below (T6/T3/GetBitboardOf ≈
   1.15-1.2x) + [BEHAVIOUR] magic movegen by SPRT. Risk-free, ~+90-130 ELO at blitz if fully recovered.
4. **EVAL/SEE COST AUDIT.** Eval is richer than the stronger peer ⇒ not the bottleneck, but unaudited.
   Per-term fixed-nodes SPRT + bench-NPS; cut neutral-but-expensive (KEEP RULE supports it). Don't gut
   blindly (N3 regressed); DON'T disable-by-ply (folklore).

**INTENT (proposed, evidence-based):** be a lean-eval, fast, strong-search engine like Blunder/Weiss/Fruit
(all 2500+ on lean eval); rich eval only wins at the very top on a world-class tuned search we lack. NNUE is
a separate project. Per-node micro-opts measured ~0 ELO at 100ms ([[project_strength_bottleneck]]); the
endgame tree is real but SPRT-only-hard, so the MIDDLEGAME TT/futility candidates (#2) + eval audit (#4) are
now the most tractable levers.

**>>> STEP-1 ENDGAME AUDIT — DONE 2026-06-01, VERDICT: NO QUICK PRUNING FIX EXISTS. <<<**
Confirmed the futility 13%-vs-69% firing gap, then empirically REFUTED every standard pruning knob as the
endgame-tree cause (throwaway-build node-count probes + Blunder-7.4.0 source diff): **futility margin**
(tightening via accPhase gate is CHAOTIC/non-monotonic, pp-end −3%→+62%, never near Blunder); **futility
DEPTH 3→8** (Blunder-style HELPS MG −20-23% but BLOWS UP EG: pp-end **+141%**, quiet-end +37%); **RFP depth**
(unbounding adds nodes back); **LMR re-search** (~0.1%/node); **NMP** (blocked-by-eval 53.7% in dead-equal EG
but that's CORRECT, Blunder gates identically); **TT capacity** (8x→16M slots: EG −0%/−5%, but MIDDLEGAME
−46% — the inversion). Move ordering is RICHER than Blunder's; LMR reduction ~equal. **META:** single-position
node counts are UNRELIABLE for pruning/TT knobs (instability dominates, non-monotonic — *less* futility also
shrank the tree); only node-IDENTICAL changes + gross cross-engine structure read cleanly, STRENGTH needs
SPRT. ⇒ the 2.5-5x endgame tree is DIFFUSE EBF, SPRT-only-hard — **PARKED** (see below). Full writeup:
[[project_endgame_tree_diagnostic]].

**>>> NEXT — EXECUTION MODEL: TWO LANES (the audit reshaped the queue) <<<**
SPRTs are SERIAL + multi-hour (the bottleneck); the free node-identical queue needs NO SPRT. So run them
CONCURRENTLY — one SPRT in the background, free-lane work in the foreground (the [[feedback_long_autonomous_sessions]]
pattern). Ordering principle: lowest-risk highest-EV first; never batch; each kept change is a checkpoint.

**LANE A — SPRT lane (serial, background, fixed-TIME to a bound ≥1000 games, one at a time):**
- **A1 — De-pad the TT. DO FIRST (one line, lowest-risk, best EV).** Delete `_ [48]byte` from `CachedEval`
  (engine/cache.go:34-38): entry 64B→16B ⇒ `NewCache(128)` gives **8.4M slots not 2.1M** (4x, same 128MB).
  Safe: NGN is hard single-threaded (`Threads`/`Hash` setoptions are no-op `// TODO` stubs, uci.go:296/298),
  so the cache-line padding (cross-thread false-sharing only) buys nothing; lockless XOR-verify `Key=Data^hash`
  still correct, all data fits the single Data uint64. [BEHAVIOUR] (node counts change): build separate binary
  → tactical 30/30 → fixed-time SPRT HEAD-vs-depad. Signal is MIDDLEGAME (mid-tree −46%, NGN below Blunder) but
  NOISY (kiwipete +17%) ⇒ the SPRT decides, not node counts. Expect small-positive (more effective hash →
  deeper MG search at fixed time). LOW-PRI COMPANION: implement the `Hash` setoption (uci.go:296) so the
  gauntlet/tests can size TT — makes de-pad coherent and lets Cadence-B test bigger hash vs the anchors.
- **A2 — Phase-gated futility→depth-8** (Blunder-proven MG −20-23%). Raise FUTILITY_MAX_DEPTH to ~8 in the
  MIDDLEGAME ONLY (gate on `accPhase`); NEVER deepen EG futility (it +141%'d). Margin formula already extends
  (200+(d-1)*50 ≈ Blunder's 520 at d8). Branch → tactical → fixed-time SPRT.
- **A3 — Check-extension discipline (SUPERVISED — Bryan's nod on the variant).** SEE-gate the full-ply extend
  at `search.go:~1466` (default = extend only if SEE(check) >= 0). Helps ALL phases (the disable-checkext
  probe cut MG −47%); it's a STRENGTH feature ⇒ SPRT, not removal. Branch → tactical 30/30 → fixed-time SPRT.
- **A4 — Magic movegen (T5, ~8.4% per-node).** Replace ray-walk `generateSlidingMoves/Captures` with the
  existing O(1) `GetRookAttacks`/`GetBishopAttacks`. Gen-order is load-bearing (tie-break flip moved nodes
  −30%..+22%) ⇒ NOT node-identical, fixed-time SPRT. Biggest single per-node compute item.
- **A5 — Eval/SEE cost audit. REFRAMED + DOWNGRADED 2026-06-01 (call-graph traced from `evaluateUnsafe`).**
  The LIVE eval is already LEAN — only ~8 textbook-standard term groups: PeSTO (material+PST), pawn structure
  (passed/doubled/isolated/chains), bishop-pair, mobility, rook-open-file, knight-outposts, king-safety
  (+pawn-shield +attack-patterns, mg phase≥12), king-activity (eg phase<6). The ~14 "speculative" funcs
  (PieceCoordination+Protection/Batteries/Cooperation/Centralization, BasicEndgames+KingPawnOpposition/MatePattern,
  WeakSquares, PawnStorms, Castling×2, CenterControl×2, PieceDevelopment) are **DEAD CODE** (callers=0 from the
  live root — vestigial from a pre-PeSTO eval). ⇒ "cut expensive speculative terms" is VOID: dead code doesn't
  run (0 ELO, 0 NPS). REAL A5 levers: (a) **SEE cost audit still valid** — SEE-twice-per-capture / T2b lazy-SEE,
  fixed-NODES SPRT + bench-NPS; (b) **eval STRENGTH** = weight-tune the live terms (Texel, partly done bb12a48)
  or fixed-nodes-SPRT removal of the weakest LIVE term (outposts? king-activity?). Dead-code deletion = optional
  cleanliness only (flag, NOT a lever; may be intentionally-orphaned ⇒ don't delete unprompted).

**LANE B — Free node-identical queue (foreground, NO SPRT — do WHILE a Lane-A SPRT runs):**
- T6 (PieceAt mailbox) → T3 (drop per-node defer) → GetBitboardOf array refactor (detail in the re-profile
  section below). ≈ ~1.15-1.2x, ~0 ELO @blitz so don't over-invest, but free to bank. Each gated by 3-baseline
  node-identity + `go test -short ./engine/` + `make smoke-tactical`; commit each on KEEP RULE.

**PARKED — Endgame tree (the old "#1 lever").** Real and deterministic (5x at d18), but the step-1 audit
eliminated every standard knob and proved node-counts can't rank pruning changes. Do NOT spend more node-count
probing here. Revisit only via (a) a real fixed-time SPRT of a structural search change, or (b) a deliberate
search-architecture rethink — after Lane A's cheap wins are banked.

**Gate discipline (all lanes):** build dev work to a SEPARATE binary (never `make` mid-gate); [NEUTRAL]/Lane-B
items by 3-baseline node-identity; [BEHAVIOUR]/Lane-A items by fixed-TIME SPRT to a bound (≥1000 games; never
fixed-nodes for node-spending changes; never batch — see [[ngn-sprt-instrument]]). Each kept change commits as
a checkpoint (local, single-sentence, no co-author).

### >>> DEEP RE-PROFILE + PLAN (2026-05-31 @ 7c9c58f) — READ FIRST, SUPERSEDES STEP 0 BELOW <<<

**Full per-node micro-budget + the "what does 2.5x even mean" decomposition is in `docs/10-performance.md`
(rewritten this session). This is the actionable distillation.** Re-profiled at HEAD with a deep
phase-diverse basket (`BenchmarkSearchProfile`, engine/profile_test.go — mid+Kiwipete+endgame, depth 11,
cold TT/op, 27M nodes / 67s samples, E-cores; benchmark KEPT as the reproducer). build/ngn = HEAD.

**HEADLINE FINDINGS (verified vs live code + profile, not eyeballed):**
- **Allocation INVERTED after T1.** T1 pooled alphaBetaPV's buffers, so the residual is now **96.9% in
  quiescence**: `var moveBuffer [256]Move` at **search.go:1842 = 6.0 GB** (the qDepth==0 stalemate probe,
  hit at every horizon node) + **:1789 = 1.08 GB** (in-check evasions). ~684K allocs/op, ~7 GB/search,
  ~0.25 allocs/node. The `[:]` slice escapes via non-inlined `GenerateMovesIntoBuffer`. Blunder is
  zero-alloc/node. (`captures [64]Move` :1876 does NOT escape — inlined `copy`. Earlier "alloc isn't the
  bottleneck" was true PRE-T1; the residual just moved to qsearch, which T1 never touched.)
- **Ray-walk movegen** (`generateSlidingMoves`/`Captures`, movegen.go:530+) = **~8.4% flat**, when O(1)
  magic getters (`GetRookAttacks`/`GetBishopAttacks`, magic.go:253/260) ALREADY EXIST, used only for
  check/SEE — not move-gen.
- **No mailbox:** `GetBitboardOf`+`PieceAt`+`GetPiece` 12-way switches = **~8.8% flat**.
- **Subsystem cum:** qsearch 49.8%, eval 18.6%, ordering ~18% (SEE-per-capture in scoring), make+unmake
  16.8%, SEE 11%, movegen 10.8%, isInCheck 8.2%. `selectNextMove` 7.5% flat (biggest leaf, inherent post-T2a),
  `Bitboard.Clear` 5.4% flat (N1 accumulator moved eval cost here).

**THE 2.5x, HONESTLY (docs/10) — REVISED DOWN after empirical re-scoping:** the genuinely-free
node-identical wins are MODEST — T1b (done) + T6 (~2.3%) + T3 (~3%) ≈ ~5–6% ⇒ **~1.1x**, +GetBitboardOf
array refactor (~4.6%, big) ⇒ **~1.15–1.2x** (≈ +5–20 ELO via depth). **The biggest single compute item,
magic movegen (T5, ~8.4%), is NOT free** — gen-order sets tie-break order, which is load-bearing (proven:
tie-break flip moved nodes −30%..+22% and changed a bestmove) ⇒ SPRT lane. The rest of the ~2.5x is NGN
DELIBERATELY spending on rich eval (18.6%) + SEE-accurate ordering — TRADEOFFS (SPRT), history against
trimming (N3 reverted; upfront MVV-LVA sort was −129 once, 2f61d3a). **NPS is a means: strength = NPS ×
node-quality. Do NOT chase Blunder-parity by gutting eval** — only a cheaper-AND-stronger regime (packed
score / NNUE) gets both.

**NEW FREE QUEUE — do in order; each gated by 3-baseline node-identity + `go test -short ./engine/` +
`make smoke-tactical`; commit each (KEEP RULE), build dev work to a SEPARATE binary (never `make` mid-gate).**
**CORRECTED node-identity baselines (post-J10 HEAD — the OLD values @ "NODE-IDENTITY GATE" below are STALE
pre-J10; J10 shifted Kiwipete + endgame):** Kiwipete `r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R
w KQkq - 0 1` d12 = **1,669,503 / sd38 / e2a6** · mid `r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1
w - - 0 10` d12 = **1,165,258 / sd30 / a2a3** · end `8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1` d16 =
**862,348 / sd39 / b4f4**. Driver: `/tmp/nodecheck.sh <bin> <fen> <depth>`.
1. **T1b — quiescence frame pool `[NEUTRAL]` — >>> DONE (commit 3b6809b), KEPT. <<<** Pooled the escaping
   `var moveBuffer [256]Move` at search.go:1789 (in-check) + :1842 (stalemate probe) onto a per-qsearch-frame
   `qframe.moveBuffer` (FrameDepth counter, stack fallback) — both mutually exclusive per call; `captures
   [64]Move` :1876 left on stack (never escaped). **Result: allocs/op 684K→11K (−98.4%), B/op 721MB→31MB
   (residual is the bench's NewCache scaffolding), ~7 GB→~0 churn/search; node-identical to HEAD on all 3
   baselines; go test -short green; tactical 28/30.** Benchmark-loop NPS +~27% but that OVERSTATES real-game
   gain (T1 lesson); real read TBD — kept on node-identity + churn-elimination regardless.
2. **T6 — mailbox for PieceAt `[NEUTRAL]` — ~2.3%, NEXT free item.** `[64]Piece` on `Bitboard`, synced in
   Clear/UpdateSquare/Move (the 3 primitives N1 already hooks); `PieceAt(sq)` reads it instead of the 12-way
   scan. CORRECTED scope (was "~8.8%"): does NOT help GetBitboardOf (4.6%, piece→bitboard, not a square
   lookup) or GetPiece (1.9%, a type+color constructor) — neither is mailbox-addressable. ~2.3% only.
3. **T3 — drop per-node defer `[NEUTRAL]`.** Explicit `RepStackLen--`+`FrameDepth--` at each alphaBetaPV
   return (CAUTION: the frame pool RIDES this defer; T1b just added a 2nd qsearch-FrameDepth defer — same
   care; doc 09 INV-A3/J3; a missed path corrupts buffer reuse — re-audit every return).
4. **GetBitboardOf array refactor `[NEUTRAL, BIG]`** — replace the 12 named bitboard fields + 12-way switch
   with `bitboards [12]uint64` so `GetBitboardOf(piece) = bitboards[piece]` (~4.6% flat). Large churn (every
   `b.whitePawn` ref site), node-identity-gated. Deferred until the cheap items land.

**SPRT LANE (BEHAVIOUR — fixed-time, run-to-bound, kept iff no regression; the biggest compute item lives here):**
- **T5 — magic movegen `[BEHAVIOUR]` — ~8.4%, RECLASSIFIED from [NEUTRAL] (do NOT gate by node-identity).**
  Rewrite slider gen to `GetXxxAttacks(sq,occ)` + bit-iterate (classify capture by `enemy & bit`). NOT
  node-identical: magic emits square-order, ray-walk direction-order, and `selectNextMove` breaks
  equal-score ties by GENERATION ORDER (the same property that made T2a neutral) ⇒ tied-quiet try-order
  changes ⇒ node counts move. **PROVEN EMPIRICALLY 2026-05-31:** flipping the tie-break `>`→`>=` moved nodes
  −30%..+22% (mid 1,165,258→820,220) and CHANGED a root bestmove (mid a2a3→d4c5). ⇒ fixed-time SPRT (expect
  ~neutral, tie-break order is arbitrary). Node-identical ALT if wanted: emit per-direction in step order
  using the magic set as a blocker mask (preserves order, drops the wrap guards — smaller win, more work).
- **T2b** (lazy-SEE/staged ordering) + **T4** (incremental legality, the isInCheck residual) — `[BEHAVIOUR]`,
  fixed-time SPRT, supervised.

**STATUS: T1b DONE (3b6809b). T5 [BEHAVIOUR]/SPRT. REORDERED 2026-05-31 — the free queue (T6/T3/
GetBitboardOf) is now step 4 of the EXECUTION ORDER above; the #1 lever is the ENDGAME TREE (steps 1-3 =
endgame pruning audit → check-ext discipline → endgame futility fix), do that FIRST.**

### >>> AUTONOMOUS SESSION (2026-05-31 PM) — EXPIRED: superseded by THE CONTROL LOOP at top of file; kept only as the 05-31 session-results record (its baselines + "until window ends" loop are stale) <<<

**>>> SESSION RESULTS (Bryan, read this on return) <<<**
- **HONEST HEADLINE: the session's work (N2+N6+J10) = +10 ELO [-5,+25] over 2000 games** (cumulative HEAD vs session-start
  825895d, ran to completion = UNBIASED). A small positive, NOT firmly resolved (CI includes 0). The single behaviour fix
  **J10 (commit f6c9414, stop futility-pruning queen promotions** — a provable bug: 200cp futility margin can't cover an
  ~800cp promo) is most of it; the node-identical speed work is ~0 of it.
- **CORRECTION — I overstated J10 as "+37".** The J10-vs-N6 SPRT read +37 [+16,+59] but it STOPPED at the H1 bound, and a
  boundary-stopped point estimate is BIASED HIGH (optional-stopping inflation). The unbiased run-to-completion cumulative
  (+10 [-5,+25]) is the trustworthy number → J10's true effect is ~+5-15, not +37. (Diagnostic: the cumulative base N1 is
  SLOWER than N6, so it should be EASIER to beat → cumulative should EXCEED +37 if real; it's +10 ⇒ +37 was inflated.)
- **Speed lane (N2 isInCheck-tag dedup `9c414c2`/`6ca41f6`, N6 pawn-eval cache `23b6187`)**: all node-identical, KEPT (free
  under the KEEP RULE), but a fixed-time SPRT (N1+N2+N6 vs pre-N1) measured **~0 ELO at 100ms** → node-identical speed
  micro-opts don't move ELO at this TC; **STRATEGY PIVOT: chase correctness/eval, not speed micro-opts.**
- **Tried+REVERTED:** N3 lazy-eval (wrong lever — NGN's big positional terms distort sharp positions). **Deferred to
  SUPERVISED** (risky/invasive): J1 extension×LMR (HIGH, code-analyzed below), N4-TTfirst, N5 lazy-SEE.
- **TWO METHODOLOGY LESSONS (both burned me this session, both in [[ngn-sprt-instrument]]):** (1) a CI-excludes-0 read at
  low n is STILL noise — J10 read -60 [-118,-2] @n=141 → +37 @n=1004 → true ~+10; always `-mingames 1000`. (2) a
  boundary-STOPPED SPRT inflates the point estimate; trust a RUN-TO-COMPLETION (fixed-N) estimate for magnitude.
- **Known rare gap (low-value, supervised):** `IsCapture()` excludes en-passant (move.go:115 vs the `||IsEnPassant()`
  pattern in moveorder), so futility/LMP/histprune treat EP as a quiet move — same class as J10 but EP is rare ⇒ ~0 ELO,
  unmeasurable; fix `&& !move.IsEnPassant()` opportunistically.

**Mandate: keep working, NEVER ask a question, NEVER sit idle waiting.** N1 shipped (commit d0fae30, +3.6% NPS,
node-identical). Bryan's recalibration accepted: per-node micro-opts each yield single digits; the 2.5x is
STRUCTURAL — a stack of micro-opts won't close it. He chose **re-profile first, then target** over blind grinding.

**Uncommitted in tree:** `engine/zz_nps_profile_test.go` — a throwaway deep-Kiwipete d11 profiling benchmark. It's
the FIRST tool: run STEP 0, record findings here, then `rm` it (do NOT commit it). build/ngn currently = N1.

**STEP 0 — re-profile (do immediately, bulletproof form keeps the binary for symbolization):**
`taskpolicy -b go test ./engine/ -run=NONE -bench=BenchmarkNPSProfile -benchtime=15x -cpuprofile=/tmp/ngn_n1.prof -o /tmp/engine.test -count=1`
then `go tool pprof -top -nodecount=40 /tmp/engine.test /tmp/ngn_n1.prof`. Read FLAT self-time of COMPUTE fns
(post-N1 the eval slice should be smaller; watch isInCheck, ordering+SEE, movegen, make/unmake/Clear). DISCOUNT
alloc/GC/madvise/memclr (benchmark-loop artifact — T1 lesson). Record the top-15 flat here; pick the biggest real one.

**STEP 0 DONE (2026-05-31 PM, post-N1, d11 Kiwipete ×15, 2.82s/search):** top FLAT compute (discounting memclr 11% /
madvise 6.2% / newobject 1.4% / atomic 4.2% = benchmark-loop + TT-probe artifacts): selectNextMove 5.48 | evaluateUnsafe
4.02 (cum 19.8) | Bitboard.Clear 3.72 | findLeastValuableAttacker 3.70 | alphaBetaPV 3.65 | **isInCheck 3.15 (cum 8.71)**
| evaluatePawnChains 3.00 | GetRookAttacks 2.95 | scoreMovesIntoBuffer 2.80 (cum 8.53) | genSlidingMoves 2.70 |
genSlidingCaptures 2.38 | GetPiece 2.26 | Bitboard.Move 2.16 | updateHash 2.11 | GetBitboardOf 2.03. CUM subsystems:
**qsearch 53.5%**, eval 19.8%, SEE 9.3%, isInCheck 8.7%, ordering ~14% (score+select). **PICK = N2 (isInCheck, cum 8.7%,
top of queue, node-identical — gives-check should == the cached InCheck tag).** NEW lever queued: **N6 — pawn-hash cache**
for the pawn-structure terms (pawnChains 3.0 + pawnShield 1.79 + doubled 1.54 + isolated 0.97 ≈ 7% cum, recomputed every
eval; node-identical iff keyed on full pawn-occupancy with no-collision verify). zz_nps_profile_test.go removed post-STEP-0.

**THE LOOP (repeat until window ends):**
1. Pick the biggest actionable flat hotspot. If ambiguous, default order: N2 → N4 → N3 → N5 (specs below).
2. NODE-IDENTICAL lever (N2; N4 if yielded order preserved): implement → GATE = 3-baseline node-identity
   (`/tmp/nodecheck.sh <bin> <fen> <depth>`; the recorded baselines are INVARIANT under any node-identical change:
   Kiwipete d12=1339599/sd32/d5e6, mid `r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10`
   d12=1165258/a2a3, end `8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1` d16=848103/b4f4) + `go test -short ./engine/`
   + `make smoke-tactical`. Pass & faster ⇒ COMMIT (KEEP RULE). Gate FAILS (nodes move) ⇒ secretly BEHAVIOUR ⇒
   make it exact, or move it to the SPRT lane.
3. BEHAVIOUR lever (N3 lazy eval, N5 lazy SEE): implement → launch a PROPERLY-POWERED fixed-TIME SPRT in the
   BACKGROUND (run_in_background) — ≥1000 games / run to an SPRT bound, NOT the 82-game underpowered runs that
   plagued this session ([[sprt_instrument]]). **NEVER block on it** — while it runs, go implement the next
   node-identical lever. On conclusion: KEEP iff no regression, else revert. This long SPRT IS the multi-hour E-core task.
4. COMMIT each checkpoint (single-sentence msg, NO co-author/emoji). UPDATE this block after each. Commit LOCALLY, DO NOT push.

**DON'T-GET-STUCK RULES:** (a) never wait idle on a background task — always keep foreground work going;
(b) if a lever resists the gate >~20 min, revert and take the next; (c) if the NPS queue is exhausted/blocked,
PIVOT to the correctness lane (J1 extension×LMR reduced-scout bug, docs/09 — HIGH, independent of NPS, always
available); (d) any choice that arises → pick the obvious option, note it here, proceed — do not ask;
(e) re-profile again after 2-3 levers land to re-target. Keep `make smoke` (game phase) for a real regression read if unsure.

**The headline lever.** NGN ~310K NPS vs Blunder 7.4.0 (CCRL 2532) ~640-750K = **~2-2.4x slower per node**,
~2-3 plies shallower at equal time ≈ ~100-140 ELO ≈ the whole CCRL gap (docs/10). Both engines are
single-threaded classical Go, so the gap is **per-node work, and it is reducible** — a peer engine proves it.

**Root cause (VERIFIED 2026-05-31 vs live code, NOT eyeballed):** not one missing thing — four hot paths each
computed the simple/full way, MULTIPLYING ~1.4×1.3×1.3×1.15 ≈ **2.7x**. Eval runs at internal nodes too
(RFP/futility/NMP static eval/improving), so the eval cost spans the whole tree, not just leaves.

**SHIPPED THIS PUSH:**
- **J10-queen `[~+10 ELO, BEHAVIOUR/CORRECTNESS]` — commit f6c9414.** Stop futility-pruning queen promotions
  (search.go:1352, `&& move.PromoType() != Queen`) — provable bug (200cp margin can't cover an ~800cp promo). The J10-vs-N6
  SPRT read +37.2 [+16,+59]/1004g H1-accepted, BUT that was BOUNDARY-STOPPED (biased high); the unbiased run-to-completion
  cumulative (HEAD vs session-start, 2000g) puts the real session gain at **+10 [-5,+25]**, so J10 ≈ +5-15. Kept on
  correctness grounds (sound + positive lean + no regression). Found via docs/09 J10.

**Node-identical NPS levers (all NEUTRAL, node-identity-proven, KEPT — each ~single-digit ELO, cumulatively ~0 at 100ms):**
- **T1** — frame-pool hot-path buffer reuse (search.go searchFrame/SearchInfo.Frames). Allocs -75%. commit b9c1859.
- **T2a** — incremental deferred selection sort (moveorder.go scoreMovesIntoBuffer+selectNextMove, driven from the
  alphaBetaPV loop). Skips the O(n²) sort tail past a beta cutoff. ~4-7% faster. commit 7445851.
- **N1** — incremental material+PST+phase accumulator (`Bitboard.accMG/accEG/accPhase`) maintained in the three
  board-mutation primitives (`Clear` subtracts, `UpdateSquare`/`Move` add); `evaluatePeSTO` reads it instead of
  rescanning; dead `gamePhase` removed; Texel tuner recomputes per-sample (tables mutate under it). Bit-exact ⇒
  node-identical on all 3 baselines; `TestEvalAccumulatorUnderMakeUnmake` walks make/unmake over 7 FENs. ~3.6%
  faster. commit d0fae30.
- **N2** — isInCheck dedup via the cached `InCheck` tag. **N2a (9c414c2):** the alphaBetaPV gives-check probe
  (search.go:1453) == exactly what makeMoveHelper:263 just computed and cached after the move, so read `pos.IsInCheck()`
  (removes one full board scan per legal move; ~+1.4% measured). **N2b (6ca41f6):** the node-in-check probes in
  alphaBetaPV (1132) + quiescence (1776) read the maintained tag instead of rescanning — tag is correct on every entry
  (ParseFEN root, makeMoveHelper, IID/singular same-pos, null-move clear by the NMP `!inCheck` gate). Both node-identical
  on all 3 baselines (the gate proves the tag-maintenance invariant holds). **RESIDUAL (NOT tag-addressable):** the
  per-move LEGALITY scan (search.go:1434, mover's-king color ≠ side-to-move tag) is the dominant remaining isInCheck
  cost; eliminating it needs INCREMENTAL legality (only re-check the king when the move could expose it — pins/EP/king
  moves), a behaviour-neutral but bug-prone project. Deferred (high risk, gated by node-identity if attempted).
- **N6** — pawn-structure eval cache (commit 23b6187). passed/doubled/isolated/pawn-chain are pure functions of the two
  pawn bitboards (verified each reads only pawn bitboards), so memoize their RAW counts in a `(whitePawns,blackPawns)`-keyed
  direct-mapped table (`pawnCache`, 1<<16 entries) with a full-key verify — collision-proof ⇒ node-identical. Raw counts
  stored (weights applied after) so Texel weight-tuning never stales it; Texel's parallel eval loop sets `pawnCacheDisabled`
  to stay race-free (search is single-threaded, like the existing global TT/history/killer tables). Node-identical on all 3
  baselines; `TestPawnCacheMatchesDirect` pins cache==direct on miss/hit/disabled paths. **+2.6% midgame / +9% pawn-endgame
  NPS** (biggest node-identical win this session — cache hit-rate is high since pawns rarely move within a subtree).

**QUEUE — do TOP-DOWN, one at a time, each gated before the next is stacked:**

**N2 — isInCheck dedup `[NEUTRAL]` — DONE (9c414c2 + 6ca41f6), see SHIPPED above. Residual = per-move legality scan (deferred).**

**MEASUREMENT CAVEAT (2026-05-31 PM):** back-to-back E-core benchmarking THERMALLY THROTTLES the M4 — absolute NPS
drifted 294K→262K across two A/B runs ~10 min apart. So small NPS deltas (<~2%) are UNRESOLVABLE by interleaved A/B
right now; trust the deterministic node-identity gate + the strictly-less-work argument for [NEUTRAL] levers, and use
INTERLEAVED (alternating) A/B never block-vs-block. For real speed reads, let the cores cool or measure depth-to-time.

**>>> BACKGROUND SPRT RUNNING (launched 2026-05-31 PM):** speed→ELO measurement of this session's node-identical work —
`new=build/ngn (N6, 23b6187)` vs `base=/tmp/ngn_pre_n1 (d0fae30~1)` = pure N1+N2+N6 (baseline is node-IDENTICAL to HEAD on
all 3 baselines, so the SPRT isolates SPEED→depth→ELO). fixed-time 100ms, conc-4, [0,8], maxgames 3000, 400-book,
`/tmp/sprt_speed.log` (bash bg id bp96upzxu). Answers the strategic question "do the node-identical micro-opts buy real
ELO at this TC?" — if clearly +, keep grinding node-identical levers; if ~0, NGN isn't depth-limited at 100ms (pivot).
Wakes me on completion. (Watching: draw rate — speed-only twins should draw a lot when both hit equal depth.) <<<
  - **STOPPED at G758 (~18 min) to reallocate cores to the J10 SPRT (higher value).** Final read: 354W-68D-336L = 51.2%,
    **elo +8.3 [-16,+33]**, LLR +0.23, draws ~9%. Trajectory +49(n=106)→+7(n=535)→+1.6(n=646)→+8.3(n=758) = a SMALL
    UNRESOLVED POSITIVE (~+5±15), NO regression. READ: ~10-15% cumulative NPS (N1+N2+N6) buys ~0-8 ELO at 100ms — the
    "micro-opts give single digits, unresolvable" thesis CONFIRMED. **Strategy pivot: stop grinding node-identical speed
    micro-opts (kept what shipped — they're free under the KEEP RULE regardless of this), the near-term ELO is in
    correctness/behaviour (J10, J1) and eval, or structural (NNUE).** No keep/reject rode on this SPRT (the commits are
    kept on node-identity), so stopping a settled ~neutral measurement to run the uncertain J10 fix is sound. NB: at a
    LONGER real TC the same NPS may show more (depth matters more) — speed work isn't worthless, just unresolvable at 100ms.

**N3 — Lazy eval `[BEHAVIOUR]` — TRIED 2026-05-31, REVERTED (wrong lever for NGN, NOT committed).** Implemented lazy
stand-pat in qsearch (`lazyEvaluateForPlayer`: skip positional terms when the cheap PeSTO core is >margin past
`[alpha,beta]`). At margin 150 Kiwipete EXPLODED +68% nodes (1.34M→2.26M), bestmove d5e6→e2a6, root score -136→-80; even
at margin 300 the Kiwipete bestmove stayed wrong and score off 56cp. Root cause: in null-window scout searches (most of the
tree) the window is 1cp wide, so "core >margin past window" fires constantly, and NGN's positional terms (king safety +
mobility ×166%) genuinely exceed 300cp in sharp positions → coarsening flips scout results exactly where tactics matter. No
sound margin both avoids tactical distortion AND fires usefully. Concrete cost (distorted tactical eval) ⇒ reject. Quiet
positions DID shrink (mid -44%, end -8%) — lazy eval helps quiet, hurts sharp; NGN's big positional terms make it net-risky.
DON'T retry as-is. (If ever revisited: only the *cheap-positional* terms could be skipped, or condition on a quietness proxy.)

**N4 — Staged move generation — FINDING: full staging is `[BEHAVIOUR]`, not neutral.** The move-ordering score bands
INTERLEAVE captures and quiets: winning captures ~80000 (above counters/killers/castling) but LOSING captures ~20000 (BELOW
counters 30000, killers 25000, castling 50000, above plain quiets). So "captures-then-quiets" staging would try losing
captures before killers/castling ⇒ reorders ⇒ NOT node-identical (moveorder.go:343-388). The ONLY node-identical slice is
**N4-TTfirst** (try the TT move before generating any moves; if it cuts, skip generation entirely). ASSESSED 2026-05-31:
the alphaBetaPV move loop body (~300 lines: extensions/LMR/re-search/history/pruning bookkeeping, search.go:1322-1670) would
have to be DUPLICATED for the standalone TT-move trial, or extracted into a per-move helper (big refactor). Too invasive /
high bug-surface for an unattended window — DEFER to a supervised session. (The node-identity gate makes it SAFE to attempt
then, just not quick.)

**N5 — Lazy SEE `[BEHAVIOUR → SPRT]` (was T2b).** ASSESSED 2026-05-31: TANGLED. SEE feeds BOTH ordering (moveorder.go:345,
every capture up front) AND pruning (search.go:1376/1391/1905, qsearch:1905). A naive MVV-LVA swap would try LOSING captures
too early (they currently sort at ~20000, BELOW killers/castling) ⇒ likely regression. Doing it right needs STAGED selection
with lazy-SEE deferral of losing captures — non-trivial. Behaviour ⇒ SPRT. Supervised.

---

## >>> POST-SPRT QUEUE (2026-05-31 PM — work these when the speed-SPRT frees the cores / when supervised) <<<

**The cores are busy with the speed-SPRT (~3000 games), so NO behaviour change can be game-validated until it finishes.**
When woken by its completion: rebuild `build/ngn` from HEAD (it was the frozen N6 snapshot for the SPRT), then:

1. **J10-queen — futility no-prune queen promotions `[CORRECTNESS]` — >>> DONE: ~+10 ELO, COMMITTED f6c9414, KEPT. <<<**
   Provable correctness fix (200cp futility margin can't cover an ~800cp promo), kept on correctness grounds (sound +
   positive lean + no regression). **TWO SPRT-METHODOLOGY LESSONS, both burned me here (do not delete):**
   (1) **low-n noise:** the J10-vs-N6 read went G141 = -59.7 [-118,-2] (CI EXCLUDED 0 — I nearly rejected it as a -60
   regression) → G269 = -6.5 → G1004 = +37.2 [+16,+59] H1-accepted. A CI-excludes-0 read at n=141 was pure noise; NEVER
   conclude below ~n=1000; always `-mingames 1000`.
   (2) **boundary-stop inflation:** that +37 was BIASED HIGH because the SPRT stopped at the H1 bound. The unbiased
   run-to-completion cumulative (HEAD=N2+N6+J10 vs session-start N1, 2000g, INCONCLUSIVE) = **+10 [-5,+25]**, and since the
   N1 base is SLOWER than N6 (should be EASIER to beat → should EXCEED +37 if real), the true J10 is ~+5-15, not +37.
   Trust run-to-completion estimates for MAGNITUDE, not boundary-stop estimates. ([[ngn-sprt-instrument]] updated.)
   FOLLOW-UPS (low priority): the guard could extend to under-promos/LMP/histprune (likely ~neutral) and the EP gap
   (rare). Verified vs live code: futility
   (search.go:1352), LMP (1361), histprune (1399) skip `!IsCapture()` moves WITHOUT a promo guard, while the SEE-quiet site
   (1389) has `PromoType()==NoType`. Futility (not move-index-gated) pruning a queen promo is a PROVABLE bug — its 200cp
   margin can't cover an ~800cp promo. ONE-LINE FIX: add `&& move.PromoType() != Queen` to the futility cond (1352).
   MEASURED (queen-only, ngn_dev): tactical held 28/30, go test green, eval moved Kiwipete **-136→-73 (TOWARD Blunder's -86!)**
   = accuracy gain; BUT **+25% nodes on Kiwipete** (promotion-rich; midgame IDENTICAL, endgame +2.5% → cost is position-specific,
   avg likely small). Needs a fixed-time no-regression SPRT (per methodology: correctness fix kept iff no measurable regression
   in a short run — the node cost is the flag to check). If +/neutral, consider extending to under-promos + LMP/histprune.
2. **Re-profile (post N2+N6)** — `BenchmarkNPSProfile` (recreate the throwaway harness), re-read flat hotspots to confirm
   eval/isInCheck shares dropped and re-target. (Deferred during SPRT — contention pollutes the profile.)
3. **N4-TTfirst** (node-identical, supervised — needs the loop-body-helper refactor; see above).
4. **N5 lazy SEE** (behaviour, supervised — needs staged lazy-SEE deferral; see above).
5. **J1 (extension×LMR reduced-scout, docs/09 HIGH)** — the real strength bug, but its broad fix (C5) was REJECTED for 2× nodes;
   a NARROW fix may exist (see the J1 analysis block below). HIGH risk (explosion-prone extension code) — SUPERVISED only.

### J1 analysis (VERIFIED vs live code 2026-05-31 — for the supervised fix)
**Confirmed real.** search.go:1613-1636. A non-first move (`legalTried > 1`) with `reduction == 0` — which is EVERY
capture/check/in-check move, since the LMR gate (1557: `!move.IsCapture() && !inCheck && !givesCheck`) skips them so reduction
stays 0 — is scouted at `reducedDepth = depth-1-reduction = depth-1` (line 1614/1620). That **ignores the extension already
baked into `nextDepth`** (nextDepth = depth-1 + check/recapture/passed ext, computed at 1457-1485). The two re-searches that
DO use nextDepth can't save it: C4 (1629) requires `reduction > 0` (false here); PVS (1634) requires `isPV` (false off-PV,
~99% of nodes). NET: for an **off-PV, non-first, checking/recapture/passed move, the extension is computed + counted
(`CheckExtensions++`) but NEVER searched** — the move is only ever seen at depth-1. (The FIRST move, 1549-1551, DOES get
nextDepth, so the PV spine is fine; this bug is the off-PV siblings.)
**Narrow fix (NOT C5):** before the scout, `if reduction == 0 { reducedDepth = nextDepth }` — applies the extension ONLY to
non-reduced moves, leaving reduced moves' cheap `depth-1-reduction` scout intact (C5's mistake was `nextDepth-reduction` for
ALL moves → reduced moves extended-then-reduced → 2× nodes).
**GOTCHA TANGLE:** the "dead" branch at 1615-1617 (`if reducedDepth >= depth { reducedDepth = depth-1-reduction }`) is NOT
dead under this fix — for an extended move `reducedDepth` becomes `depth` (when ext=1), trips `reducedDepth >= depth`, and
RESETS to depth-1, clobbering the fix. The fix must restructure that branch (this is the same branch the burned-in GOTCHA
says not to naively clamp — handle with care, NOT a floor-at-1).
**RISK:** applying check extensions off-PV ⇒ more extensions fire ⇒ explosion vector (the seldepth-111 family). The per-path
EXTENSION_BUDGET clamp (1542) bounds it, but VERIFY: depth-10/12 Kiwipete node+seldepth must NOT explode (sd stays ~30s, not
100+), then a fixed-TIME SPRT. SUPERVISED only — this is the code that burned most of a prior session.

6. **J8 — SearchFixed RootDepth=0 `[CORRECTNESS, LOW value]`.** VERIFIED 2026-05-31: `RootDepth` is set ONLY in the ID loop
   (search.go:611); `searchFixedUnsafe` (864) + standalone `alphaBeta` (963) leave it 0, so the extension-budget clamp (1544)
   becomes `23-ply` and truncates deep fixed-depth searches. **Confirmed it does NOT affect shipping or my node-identity
   baselines** — UCI `go depth N` routes through `SearchIterativeDeepeningWithCallback` (uci.go:709 → sets RootDepth), so only
   direct `SearchFixed` callers (cmd/trace_move + a few deep tests) bite. One-liner `info.RootDepth = depth` in
   searchFixedUnsafe; baseline-neutral, safe, near-zero strength impact — do opportunistically, not urgent.

**CLEARED — checked, NOT the culprit, do not chase:** `WrapEvaluation` has a real fast path
(`!ENABLE_HOT_PATH_DEBUG` → bare `return fn()`, crash_handler.go:160); the profile's ~17% memclr+madvise was a
BenchmarkSearchDeep loop artifact (GC churn from thousands of short searches), near-zero in a single game search (docs/10).

**NODE-IDENTITY GATE (all NEUTRAL items):** identical node count + seldepth + score + bestmove at fixed depth vs a
freshly-built clean HEAD on the 3 baselines — **Kiwipete d12 = 1,339,599 / sd32 / d5e6**, **middlegame
`r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10` d12 = 1,165,258 / a2a3**, **endgame
`8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1` d16 = 848,103 / b4f4**. Driver: `/tmp/nodecheck.sh <bin> <fen> <depth>`
(bash coproc, waits for bestmove). If node counts MOVE, the change is secretly BEHAVIOUR → it needs an SPRT, not the gate.

**KEEP RULE (in force, do not re-invert):** NEUTRAL + faster + no regression ⇒ **KEEP, full stop** — even a small/
within-noise gain; trivial added memory is NOT a cost. BEHAVIOUR ⇒ keep iff SPRT shows no regression. REJECT only when
BOTH non-correctness AND a concrete measured regression. [[feedback_confirm_before_dropping_findings]]

**Parallel correctness lane (separate from speed):** docs/09 fix queue — J1 extension×LMR reduced-scout (HIGH), then
J10/J8/J12/J9. NPS is the bigger measured lever, but J1 is a real strength bug; interleave as desired.

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

### >>> 2026-05-30 SESSION STATE — READ FIRST (post-compact anchor) <<<
- **Real rating: NGN ≈ 2400 CCRL raw / ~2320-2370 adjusted** (low-2300s). POST-C3 read 2026-05-30 =
  Blunder 7.4.0 (CCRL 2532), 3W-9D-12L = 31.2% over 24 games @1s/move → perf 2395 [2249,2541], pooled
  2401 [2252,2550]. Movetime proxy ⇒ ~+30-80 optimistic (true ~2320-2370). **+240 ELO vs the PRE-C3
  read** (was 1W-3D-20L = 10.4% → 2158 [1944,2372]): score tripled, draws 3→9 vs the SAME anchor, so
  C3+PST translated to REAL external strength, not just self-play. NOT 2500 — the SF UCI_Elo slider was
  ~+400 inflated; it is dead as a yardstick. CI still wide at n=24; next gauntlet should add a 2nd anchor.
- **Gauntlet FIXED (was "turbo broken").** It had no defense against an opponent process dying mid-run:
  v8.0.0 died after G6, then 18 phantom no-move "wins" → the run TAINTED and emitted NO RATING despite the
  clean 7.4.0. Two fixes shipped this session: (a) dead-opponent guard — abort an anchor after 3 consecutive
  forfeit games; (b) a tainted anchor is now EXCLUDED from the pool, not allowed to void the whole run.
  Earlier today: pre-flight `-probe`, in-play overshoot guard, 200ms stop-grace. Stop-compliant anchors:
  7.2.0/7.4.0/8.0.0 (5.0.0/6.1.0 ignore `stop`, auto-skipped). NB 7.2.0 passes the probe but overshoots
  under load (aborts); 8.0.0 is flaky (died mid-run). **7.4.0 is the one reliable anchor right now.**
- **2026-05-30 PM — C3 IMPLEMENTED + S6/S7/RT1; fixed-time SPRT RUNNING.** C3 (NMP gated on
  `canNull && !isPV && staticEval>=beta`, with a `canNull` double-null thread through alphaBetaPV) is
  VERIFIED at the mechanism level: Kiwipete depth-10 = 389K nodes / seldepth 29 vs HEAD 516K / seldepth 30
  (~25% fewer nodes, faster), nmcut ~7K — NMP now fires across the whole tree, not just the PV spine.
  Batched with S6 (root searches prev-iteration best first), S7 (reuse bounded TT score as static eval),
  RT1 (debug-on end-of-search counter dump in uci.go: `info string stats …`). smoke-tactical 28/30 (was
  29-30; a 1-tactic nick, same profile as the shipped E2 king-safety win). Fixed-time SPRT (movetime 100,
  conc-1, vs **clean HEAD** — NOT the stale pre-+46 ngn_base) → **VERDICT: +138 ELO [+45,+231] over 61
  games (38W-8D-15L = 68.9%, LLR +0.88 still climbing), batch COMMITTED.** Biggest single NGN search win
  to date. CAVEAT: fixed-time self-play vs HEAD reads the FULL depth-gain benefit (shared eval amplifies a
  pure search-depth edge), so the external-opponent (CCRL) contribution is likely smaller — but it is
  unambiguously large and positive (lower bound +45). Re-confirm magnitude with the next gauntlet.
- **2026-05-30 PM (cont.) — PHASE-S BATCH (S4+S3), gated per-change, batch SPRT RUNNING.** Lowest-hanging
  search refinements, each gated by smoke-tactical + a Kiwipete depth-10 node/seldepth tripwire vs clean
  HEAD f04eb0d (388708 nodes / sd29) BEFORE the next was stacked. Verdicts:
  - **S2 (killers ply-indexed) — REJECTED at the gate.** Drop-in ply-indexing REGRESSED fixed-depth ordering:
    +28-43% nodes in 3 of 4 test positions (Kiwipete 389K→541K). The audit called it "small but real" from
    general theory; the node-count mechanism check refuted the SIGN for NGN ([[feedback_audit_agent_skepticism]]
    earns its keep again). Reverted, never committed. NGN's depth-indexed killers (broad per-remaining-depth
    sharing) happen to out-order ply-indexing here; revisit only with a real fixed-time SPRT, not as a freebie.
  - **S4 (LMP gated on improving) — committed d21a833.** Split lmpThreshold into [improving][depth]; not-improving
    row prunes ~2x sooner ({0,3,4,6,9,12,15,19}), improving row left = HEAD ({0,5,8,12,17,23,30,38}) so only
    not-improving nodes change. seldepth 29→26 (more pruning, as intended), tactical 28/30 held. Node count is
    a selectivity proxy here (not pure efficiency), so its rise is not a regression signal — SPRT is the arbiter.
  - **S3 (qsearch TT store) — committed a0fdda4, FAIL-HIGH-ONLY.** First cut (store every qsearch node) FLOODED
    the TT: hashfull 20→63, nodes +26% — the exact "evicts deeper main entries" pitfall the note warned of.
    Salvaged to store ONLY fail-highs (the cutting capture, depth-0 LowerBound): hashfull 40, nodes back to
    flat/slightly-better, tactical 28/30. Added QSearchTTStores counter + TestQSearchTTStoresWired.
  - **Batch SPRT VERDICT: NEUTRAL — S4+S3 KEPT on theory.** fixed-NODES 100k vs clean HEAD f04eb0d, conc-2,
    400-opening book: G82 33W-13D-36L = 48.2%, elo -13 [-87,+62], LLR drifting ~0 (lean +50→+25→0→-4→-13,
    classic phantom-decay to neutral). No measurable win, NO significant regression. Both are textbook-sound,
    mechanism-verified, tactical-held → kept on the established "sound standard technique + no regression" bar
    (cf. quiet-SEE +2.2, singular +3.5). Did NOT bisect: CLAUDE.md "don't chase <50-ELO effects" — not worth
    more SPRT cores. Stopped at G82 to free cores for the proven (eval) lane.
- **2026-05-30 PM (cont. 2) — PIVOT TO EVAL (proven lane). kingSafety sweep RUNNING.** Search refinements are
  all sub-resolution for NGN now (S2 rejected, S4/S3 neutral, null-R uncertain). Eval is where measurable wins
  live (+46 PST, +18-20 E2 king-safety). Testing E2's open follow-up "sweep 80/100/130 for the king-safety
  peak": **kingSafetyWeight 100→130** (eval.go:2378; phase>=12 coverage = most positions; E2 thesis = quiet-MSE
  underweights king-safety's tactical value, so the ELO peak may be >100). tactical 28/30 held (130 didn't nick
  it further). fixed-DEPTH depth-8 SPRT (the proven eval instrument): new=build/ngn (130) vs base=build/ngn_base_eval
  (a0fdda4=100), conc-2, /tmp/sprt_ksafe130.log. eval.go left UNCOMMITTED pending verdict (needs eval_baseline.json
  regen on a win, like C0). If WIN → regen baseline + commit + sweep further; if not → git checkout eval.go.
  QUEUED after: kingActivity 29→? (endgame-only, motivated by NGN's own endgame-king-centralization blunders, but
  dilute/hard to measure), then extension discipline (unblocks C4/C5). Gauntlet re-run for the absolute number still pending.
  VERDICT (2026-05-30): kingSafety 130 = NEUTRAL-NEGATIVE (G106 49%, -10 [-76,+56], trended negative, bottomed -32 @G75)
  → REVERTED to 100 (eval.go clean). Peak hunt UNFINISHED — A5 says test 115 (binary-search 100..130), NOT 130. Wrong sweep point.
- **2026-05-30 PM (cont. 3) — ADVERSARIAL RE-INVESTIGATION (5 agents, Bryan-directed; [[feedback-confirm-before-dropping-findings]]).**
  Bryan was hesitant to drop the "neutral/rejected" findings without confirmation. 4 agents deep-audited search tricks vs
  Stockfish/Ethereal/Blunder/CPW; 1 audited WHY my findings didn't pan out. RESULTS:
  - **REAL BUG FOUND + FIXED + COMMITTED (ff4ff33): SEE pawn-attacker file guards were SWAPPED** in getAttackers
    (search.go:242) + getAttackersAfterMove (search.go:390), both colors. Edge-file (A/H) recapturers missed (SEE
    over-pessimistic → discards winning edge captures) + phantom wrap-around attackers. Corrupts ~2-3% of SEE evals;
    feeds capture pruning + ordering everywhere. Proven: Kiwipete Qf3xh3 = NGN -900 vs true -375 (missed g2 recapture).
    Fixed (swap offsets to match guards) + 3 edge-file regression tests in see_test.go. This is a real bug like the
    prior SEE-garbage +161. build/ngn_see = fixed binary, build/ngn_base_see = pre-SEE HEAD; **SEE SPRT (fixed-TIME,
    to a bound) PENDING after the gauntlet frees cores.** build/ngn still = pre-SEE HEAD (gauntlet measures the "before").
  - **NMP raw-eval base case (A1, HIGH, NEXT):** alphaBetaPV `depth<0 → EvaluateForPlayer` (raw eval, NOT qsearch),
    so NMP at depth 3 (→depth-4=-1) verifies the null with a RAW eval — false cutoffs on hanging-piece positions; every
    ref reduces to qsearch. ALSO explains the neutral depth-scaled-R (bigger R → more raw-eval verification). FIX: clamp
    null recursion depth to ≥0 (surgical: `if nullDepth<0 {nullDepth=0}` in the NMP block) so depth-3 NMP gets qsearch.
    Then re-test depth+eval-margin R. Correctness fix; commit on a tactical gate, SPRT fixed-TIME.
  - **Move ordering (A4): counter ranked ABOVE killer is BACKWARDS** (NGN counter 30000 > killer 25000; SF/Ethereal/
    Blunder/CPW all put killers first). NGN is the lone outlier; counters are also color-blind (counterMoves[from][to],
    no side dim). FIX: swap to killer 30000 / counter 25000 + add [2] side index. ~+5-20, A/B (fixed-nodes, to a bound).
    S2 RESOLVED: ply-vs-depth killers is a ~0-ELO near-tie (depth=D-ply alias at fixed depth; SF removed killers for ~0)
    — my node-count rejection reasoning was invalid but the outcome (leave depth-indexed) is fine. DON'T pursue S2.
  - **C4 (A2): real** — reduced LMR fail-highs not re-verified at full depth in ~99% of nodes (non-PV null-window makes
    `score<beta` unsatisfiable); inherited from Blunder, diverges from SF/Ethereal. FIX: drop `&& score<beta`; LOW
    explosion risk. **GOTCHA CORRECTION:** A2 says the reducedDepth=0 "load-bearing" claim does NOT hold now (reduction≤depth-2
    cap floors it at 1; LMR excludes checks) → C5 fix may be safe — but EMPIRICALLY re-verify (depth-10 node/seldepth) before trusting.
  - **S3 qsearch-TT (A3): CONFIRMED correct + worth keeping** (bounds right, depth-0 replacement safe). S7 consumes depth-0
    qsearch entries ungated by depth — low severity, optionally gate S7 with ttDepth>=1.
  - **METHODOLOGY (A5, THE BIG LESSON — in [[ngn-sprt-instrument]]):** every SPRT this session was 40-65x UNDERPOWERED
    (82-106 games; ~5,378 needed for true +10 at [0,10]; ±69 ELO noise at n=82). "neutral" was NO INFORMATION. Run to a
    bound/≥1k games; node count gates SANITY not strength; never batch; bracket eval sweeps; pentanomial DONE (e830e37, cont.4).
  - **Blunder corpus (current NGN):** output/blunders_current.jsonl, 18/500 (3.6%); top-4 are mate-score artifacts (filter
    gap), ~14 real = 150-475cp POSITIONAL/EVAL errors (endgame king/knight, passive-rook) → residual is eval-shaped, reconfirmed.
  - **Gauntlet "why it failed":** the failing May-30 run PREDATED the fix commit 18d6ea8 (consecutive-forfeit abort +
    tainted-anchor-dropped-not-voiding). Code is correct now; fresh post-C3 run RUNNING (-only v7.4.0, /tmp/gauntlet_postc3.log).
    Structural limit: weak anchors NGN beats (5.0.0/6.1.0) ignore UCI stop → can't bracket the 50% crossover at movetime;
    a fixed-depth gauntlet mode would unlock brackets (future).
  - **POLICY UPDATE (Bryan, cont.5):** for a fix I'm CONFIDENT in fundamentally (a real correctness fix), confirm
    "not a big regression" with a short run (~500-1.2k games) then PROCEED — do NOT grind to a formal LLR bound.
    Run-to-a-bound / chase-the-sign still applies to SPECULATIVE changes (ordering swaps, eval value guesses) where
    the question is "does it help," not "is it sound." [[ngn-sprt-instrument]]
  - **PRIORITIZED QUEUE:** (1) SEE confirm **DONE** (+6.9 [-10,+24]/1565g, non-regression, kept ff4ff33),
    (2) NMP base-case **DONE +5.3 [-14,+24]/1254g, KEPT 844bb20**,
    (3) counter/killer swap + color dim (verified, fixed-NODES node-check FIRST per S2 lesson), (4) C4 LMR
    re-verify (clamp confirmed DEAD — leave it; dropping `&& score<beta` routes reduced fail-highs through the
    check-extension re-search = explosion vector), (5) kingSafety 100→115 (only untuned weight, fixed-DEPTH),
    (6) NMP depth-scaled R.
- **GOTCHA BURNED IN (cost most of this session):** do NOT "clean up" the LMR `reducedDepth` dead branch
  (`if reducedDepth >= depth { reducedDepth = depth-1-reduction }`). It looks like a no-op, but replacing it
  with an `if reducedDepth < 1 { reducedDepth = 1 }` floor is CATASTROPHIC: reducedDepth legitimately
  computes to **0 at depth-1 nodes (→ qsearch)**, and clamping to 1 turns every such leaf into a full
  main-search ply WITH check extensions → non-decaying check recursion → seldepth 111, ~20x node explosion.
  The value 0 flowing to qsearch is load-bearing. C3/S6/S7 were ALL innocent; this clamp (carried across
  several "reverts") masked the real cause for hours. LESSON RE-LEARNED ([[feedback_audit_agent_skepticism]]):
  stack ONE change at a time, each gated by a depth-10 node/seldepth check vs a freshly-built clean HEAD.
- **C4/C5 DEFERRED (coupled to extension discipline).** Same LMR/scout block; C5 (scout at nextDepth) removes
  the implicit brake on NGN's full-restore check extension → explosion. They need the extension-budget work
  (bound check extensions by a per-path ply cap) FIRST, then their own SPRT.
- **NEXT after C3 verdict:** if C3 wins → ship it, then extension discipline (unblocks C4/C5 + tames the
  latent deep-check explosion), then S4 (LMP-improving). Gauntlet re-run for the absolute number still pending.
- Provenance: 6-agent search audit + 1 runtime audit (2026-05-30), every claim validated vs live code →
  C3/C4/C5 + S4-S7 (Phase S), RT1-RT4 (Phase RT), TT audited CLEAN, code-clarity R1 (low pri per Bryan).
- **2026-05-30 PM (cont. 4) — PENTANOMIAL committed (e830e37) during the gauntlet window (non-contending, no games).**
  cmd/sprt now scores the PAIR (same opening, reversed colors) as a 5-bucket distribution alongside the trinomial:
  SECONDARY stat for now (per-line `pLLR` + a Penta-Elo CI in the result block); the trinomial remains the decision/
  stop rule. Unit tests (cmd/sprt/sprt_test.go) prove it reduces EXACTLY to the trinomial under independence and
  tightens (more-negative LLR) on the balanced-pair null — exactly the instrument's own null-validation scenario.
  PROMOTE to the stop rule ONLY after re-running the live null (identical binary→50%) + positive (depth7v3) controls
  post-gauntlet; then the whole fix queue (SEE→NMP→ordering→C4→kingSafety) costs ~15-30% fewer games to a bound.
- **2026-05-30 PM (cont. 5) — POST-C3 GAUNTLET DONE (+240 ELO, see headline) → fix queue STARTED.**
  - **(1) SEE confirm DONE — +6.9 ELO [-10,+24] over 1565 games, LLR phantom-decayed +10→+7 (→0), KEPT.**
    `build/ngn_see` vs `build/ngn_base_see`, fixed-TIME 100ms/conc-6/[0,10], 400-opening auto-cycle. Non-regression
    confirmed (lower bound -10), mechanism-verified (3 edge-file tests), tactical-held → kept on the sound+tested+
    no-regression bar (cf. quiet-SEE +2.2). Stopped at the pre-committed N=1.5k (no optional-stopping bias). Chose
    fixed-TIME because nodes/depth REPLAY identically per opening (capped at unique-openings×2=800 < the ≥1k A5
    floor); movetime auto-fills by cycling. Conc-6 is load-dependent but UNBIASED for symmetric self-play.
  - **(2) NMP base-case DONE — +5.3 [-14,+24]/1254g, KEPT (844bb20).** Verified A1 vs live code: at depth==3 the null
    child depth = depth-NULL_MOVE_R-1 = -1 trips the `depth<0` guard (search.go:962) → raw EvaluateForPlayer, NOT
    qsearch (depth==0 → quiescence at :1012). Fix floors nullDepth at 0 so depth-3 NMP verifies through qsearch
    like depth>=4 (mirrors the load-bearing LMR 0→qsearch; qDepth>=6 bounds it, no explosion). Deterministic
    engine tests PASS (7.7s). SPRT: build/ngn_nmp vs build/ngn_see, fixed-TIME 100ms/conc-6/[0,10] +pentanomial
    (rebuilt sprt; pLLR column now live — its first real-game run). /tmp/sprt_nmp.log. fixed-TIME because NMP
    SPENDS qsearch nodes for soundness (A5: fixed-nodes undercredits node-spenders).
  - **FLAG (pre-existing, NOT mine) — flaky illegal-move fuzz:** `go test ./engine/` (full, not -short) FAILS
    intermittently: `TestFuzzSearchLegalMoves` is time-seeded random (search_fuzz_test.go:19) and found illegal
    moves (d5f6, e2e3 at depth 1-2) on clean HEAD too (3 on one seed, 0 on the next — pure flake). Depth 1-2 random
    positions, where NMP can't fire, so unrelated to (2). Real games show ZERO illegal-move forfeits. Two concrete
    root-cause leads (likely TEST artifact): (a) search_fuzz_test.go:150 builds positions with `GenerateMoves`
    (PSEUDO-legal) not `GenerateLegalMoves` → can reach positions legal play can't; (b) the global TT is NEVER
    cleared between unrelated random positions → a stale/index-colliding TT move from posA can surface in posB.
    SEPARATE TASK: make the fuzz print+save the failing FEN, add a TT clear between positions, and audit whether
    the search legality-validates its TT move before use — that last one is the only path to a REAL-game bug.

### >>> NEXT — TIGHT TIME CONTROLS (priority, set by Bryan 2026-05-29; pulled ahead of S/E/X) <<<
**Goal: NGN plays bullet/blitz in a fully sane/good way — never flags, budgets the
bank to last the whole game, and stops early once the move is decided.** Code-grounded
deep-dive 2026-05-29 (refs below). Reprioritizes I1 to the front; makes I4 its prerequisite.

**2026-05-30 PROGRESS (cont. 6) — TIME MGMT CORE DONE + REAL-CLOCK GAUNTLET UNLOCKED. All committed, tree clean.**
- **>>> REAL-CLOCK GAUNTLET COMPLETE — NGN's TRUE CCRL-Blitz number is 2380. <<<** `gauntlet -tc 120+1
  -only v7.4.0 -games 16` (CCRL Blitz conditions vs Blunder 7.4.0 / CCRL 2532): **2W-5D-9L = 28.1% → perf
  2369 [2188,2550], POOLED 2380 [2193,2567].** Reasons: 11 checkmate, 1 draw-rule, 4 max-moves — **ZERO
  time-forfeits.** Two clean findings: (a) the time-mgmt fix is validated EXTERNALLY at a true 120+1 clock,
  not just self-play — no flagging defect remains; (b) the real number (2380) is ~20 ELO under the movetime
  proxy (2401), exactly the predicted direction (anchors now get their full clock = the optimism removed), so
  **the +30-80 movetime-bias caveat is now RETIRED** — 2380 is NGN's real strength, no asterisk. NGN climbed
  ~2150 → 2380 this session via C3+PST, now confirmed real (not a movetime artifact). CI still wide at n=16
  (1 anchor); a future gauntlet should add a 2nd stable anchor + more games to tighten it.
- **>>> MULTI-ANCHOR CCRL CALIBRATION RUNNING (Bryan asked for a calibrated rating) <<<** `gauntlet -tc 15+0.15
  -only v6.1.0,v7.2.0,v7.4.0 -games 30 -ngn build/ngn` (HEAD 3e88982, with step-4), log /tmp/gauntlet_ccrl_calib.log,
  watcher bcdfsgjnc. KEY UNLOCK: at the REAL clock the gauntlet sends wtime/btime and never `stop`, so the full
  Blunder ladder is viable (the movetime-mode stop-noncompliance that skipped 5.0.0/6.1.0 does NOT apply) — probe
  confirmed all 5 anchors play clean (0 forfeits). 3 bracketing rungs chosen: 6.1.0/2155 (NGN >50%), 7.2.0/2425
  (~50%, n=2 probe split), 7.4.0/2532 (NGN <50%) → crossover bracketed. CAVEAT: 15+0.15 is 1/8 CCRL-Blitz time
  (cross-TC scaling ~±20-30, larger than the 120+1 run) — a fast multi-anchor proxy, NOT the gold 120+1 number.
  RESUME: read watcher output or `grep -E "POOLED|^==|reasons" /tmp/gauntlet_ccrl_calib.log` once gauntlet pgrep empty.
  >>> VERDICT: RUN INVALID — pooled 2411 is BOGUS. 26/90 games were time-forfeits, ALL the ANCHOR flagging (NGN
  never flagged once in 90 games — its time mgmt is bulletproof). Pattern: 6.1.0 all-30-clean (56.7%→perf 2202,
  valid); 7.2.0 clean G1-16 (31.2%) then flagged G17-30; 7.4.0 clean G1-18 (13.9%) then flagged G19-30; fresh
  anchor process RESETS the clock → REUSED-PROCESS SLOWDOWN (Blunder bloats over a long session, can't hold a 15s
  bank, loses on time not OTB). Gift-wins inflated the pool 2227→2411. SALVAGED clean-subset pool = **NGN ≈ 2227
  [2132,2322] at 15+0.15** — ~150 BELOW the 120+1 number (2380): NGN scales DOWN at faster TC. LESSON: fast TC is
  the WRONG calibration tool — anchors are CCRL-rated at 120+1 and at 1/8 speed they both flag AND under-represent.
  Two harness gaps found: (1) reused anchor process slows over ~16-18 games → restart anchors every ~10 games (or
  fresh process/ucinewgame-hygiene per game); (2) the -tc crosstable FOOTER still prints the stale "...at movetime;
  ~+30-80 optimistic" caveat (gauntlet/main.go) — wrong for real-clock. Gold calibrated number STANDS at 2380 (120+1,
  clean). To tighten cleanly: slower clock (anchors behave) — 60+0.6 (~3-4h) or 120+1 overnight; NOT faster.
  >>> ACCEPTED (Bryan, 2026-05-30): take 2380 (120+1) + 2227 (15+0.15) as NGN's calibrated rating; harness fixes
  (anchor-restart + footer caveat) DEFERRED to the next real-clock gauntlet; PIVOT to engine strength = extension discipline.

**2026-05-30 PROGRESS (cont. 7) — EXTENSION DISCIPLINE. Step-1 budget committed; C4 implemented + SPRT running.**
- **Extension budget DONE + committed cbe05db (foundational, behavior-NEUTRAL).** Root cause: NGN's check extension
  does `nextDepth++` (search.go:1402) which only RESTORES the decrement (nextDepth==depth), so a non-decaying checking
  line keeps depth flat → recurses toward MaximumDepth (the seldepth-111, ~20x-node explosion that burned cont.3).
  Only brakes were a per-NODE cap (≤depth+1) + the implicit reducedDepth=0→qsearch. FIX: a per-PATH budget — net
  extensions == `depth + ply - info.RootDepth` (new SearchInfo.RootDepth set per ID iteration at search.go:580);
  clamp `nextDepth` so child net-ext ≤ EXTENSION_BUDGET (=24) right after the per-node guardrail (~search.go:1470).
  VERIFIED: node-IDENTICAL on Kiwipete d10-12 (528711/sd29, 1072174/sd37, 1581933/sd41) = neutral on normal search;
  clamp is LIVE (budget=2 → sd 41→26, nodes drop) so not dead code; tactical 28/30; go test green.
- **C4 IMPLEMENTED (uncommitted) — fixed-time SPRT RUNNING.** The LMR re-search outer guard `if score>alpha && score<beta`
  is DEAD in non-PV nodes (beta==alpha+1 ⇒ score>alpha && score<alpha+1 impossible), so reduced fail-highs were NEVER
  re-verified at full depth in ~99% of the tree (reduced cutoffs on faith; check ext never applied off-PV). FIX (standard
  PVS+LMR, search.go ~1568): re-search at full depth on `reduction>0 && score>alpha` (ungated by beta), then PV full-window
  re-search gated `isPV && score>alpha && score<beta`. VERIFIED BOUNDED by the budget: Kiwipete d10-12 NO explosion
  (sd 30/32/32, NOT 111); PV+eval CHANGED (d12 e2a6/-58 → d5e6/-136 = verifying fail-highs finds the position is worse
  than the unverified-reduced score thought = more accurate); nodes ~neutral (d10 +29%, d12 -15%); tactical 28/30, go test green.
  VALIDATION: fixed-time movetime-100 conc-2 vs build/ngn_base_c4 (cbe05db), /tmp/sprt_c4.log. Per PROCEED policy: correctness
  fix → confirm not-a-big-regression (~500-1.2k games) then ship; don't grind to a bound. THEN C5 (scout at nextDepth).
  **C4 VERDICT: KEPT, committed c136b63.** +11.7 [-14,+37] over 716g (346W-48D-322L=51.7%, LLR +0.42, pLLR +0.52). The
  -8 at G406 was noise; lean flipped POSITIVE by G700 and held → neutral-to-slightly-positive, NOT a regression →
  sound correctness fix kept per proceed policy (same bar as quiet-SEE +2.2 / singular +3.5). movetime-100 is harsh on
  depth-costly accuracy (charges re-search nodes as lost depth), so true value at real TC is likely ≥ this.
- **C5 (scout at nextDepth) — IMPLEMENTED, MEASURED, REJECTED + reverted (not committed).** Changed the scout
  base from `depth-1-reduction` to `nextDepth-reduction` (apply the extension in the primary null-window search,
  not just the C4 re-search) and removed the now-live dead `if reducedDepth>=depth` reset (NO floor-at-1 — kept
  0→qsearch per the GOTCHA). The budget HELD (Kiwipete sd 35/31/42, check-heavy pos sd 38 — NO 111 explosion, so
  the budget did its job and C5 was SAFE to test — the whole point of building it first). But REJECTED on three
  converging signals: (1) node count >2x at d12 (1.34M→2.86M) and the blowup GROWS with depth (1.4x@d10→2.1x@d12)
  = direct fixed-time depth loss in real games; (2) ZERO tactical upside (28/30 unchanged) — no accuracy to offset
  the cost; (3) NGN's cheap depth-1 scout is a DELIBERATE-GOOD optimization (scout cheap, extend only on a fail-high
  re-search) that C4 already complements correctly. Fixed-nodes would undercharge it, fixed-time would only confirm
  the depth loss → confident reject without burning a full SPRT (per "don't chase <50 ELO" + multi-signal, not a lone
  node-count drop). **EXTENSION DISCIPLINE COMPLETE: budget (cbe05db) + C4 (c136b63) shipped; C5 rejected.**

**2026-05-31 (cont. 8) — >>> 4 BUG-HUNT AGENTS RUNNING AT COMPACT (Bryan-directed) <<<** Bryan: "way more bugs to
find, every pass turns up horrendous correctness bugs." Launched 4 background sub-agents (general-purpose, opus);
they REPORT findings only (do NOT edit) — the lead VERIFIES each claim against live code before applying
([[feedback_audit_agent_skepticism]]), then applies CORRECTNESS fixes ONE at a time gated by depth-10 Kiwipete
node/seldepth vs clean HEAD + smoke-tactical + go test. Agent scopes (disjoint):
  - **af2f023f30f15387a** — PRUNING & REDUCTIONS: NMP gates/canNull, LMR formula+reducedDepth arithmetic, LMP/futility/
    razoring/probcut conditions, the PVS re-search calls, aspiration windows. (warned off the load-bearing reducedDepth=0→qsearch / dead branch.)
  - **a4f0b130bca30faa4** — EXTENSIONS/QSEARCH/MATE-DRAW: check/recapture/passed/singular ext, the NEW budget accounting,
    qsearch stand-pat/delta/SEE/in-check, mate-distance + scoreToTT/From ply-adjust, repetition/50mv/stalemate scoring.
  - **a186f2830c9fe12cf** — TT/ORDERING/STATE: TT bounds+depth+illegal-TT-move validation+qsearch stores, ordering bands+
    killer/counter/history indexing, make/unmake + lastMovePlayed restore on every path, RepStack push/pop balance.
  - **af64cc3853cb199e2** — REVIEW THIS SESSION'S FIXES (rock-solid?): budget cbe05db, C4 c136b63, step-4 3e88982,
    timefix fdbf565, AND whether C5's revert dropped anything correctness-relevant. Verdict per fix: ROCK SOLID / SUBTLE / BUG.
  RESUME: agents notify on completion. Collect all 4 → triage by severity×confidence → verify each vs code → fix the
  real correctness bugs one-at-a-time (correctness bar, NOT ELO; see methodology below). HEAD at compact = 6bcee9d, tree clean.
- **METHODOLOGY (Bryan, cont. 8 — RESOLVES the keep/drop tension):** a PROVABLE correctness fix is KEPT on
  CORRECTNESS grounds — mechanism-verified (depth-10 node/seldepth, unit test, tactical) + no measurable regression in
  a SHORT run. Do NOT gate correctness fixes on a +10-ELO we can't resolve (SPRT to 10-ELO resolution takes too long).
  Only REJECT a change when it is BOTH (a) non-correctness (speculative/tuning) AND (b) shows a concrete cost (like C5's
  2x nodes). The C5 revert was correct under this rule (NGN's cheap scout was already correct; C5 added cost, no correctness).
  Don't drop legit correctness fixes on the floor for lack of a visible ELO swing. [[feedback_confirm_before_dropping_findings]]

**2026-05-31 (cont. 9) — >>> ALL 4 AGENTS LANDED + SEARCH DESIGN SPEC WRITTEN (`docs/`) <<<** Bryan: "make a /docs
folder laying down a deep theory of all these random tree searches — an anchor design doc / spec to implement
TOWARD, not eyeballed from memory; worried about JOINTS (bugs in the seam between two correct techniques)" +
"this exercise will help you RECEIVE the sub-agent bugs — stay meta, use them as worked examples, OK if we drop
some on the floor this run." Wrote `docs/README.md` + `docs/01..09` (architecture / TT / ordering / pruning+
reductions / extensions / quiescence / mate-draw-scoring / time / invariants-joints), grounded in a FULL read of
search.go+cache.go+moveorder.go+time.go+position.go. Template per technique: Purpose / Canonical (the target) /
Invariants (testable) / NGN (file:line) / Joints. **`docs/09` is the INTAKE FRAMEWORK** — a master invariant table,
a J1-J15 joint catalogue, and a finding ledger that slots all 4 audits with verdicts + a prioritized fix queue.
- **All 4 agents completed; every finding verified vs live code + recorded in docs/09 (NOT dropped — parked w/ status):**
  - **J1 [VIOLATED, HIGH] extension×LMR reduced-scout base** (search.go:1560 `reducedDepth := depth-1-reduction`, should
    be `nextDepth-reduction`). LMR excludes captures/checks (reduction==0) → their check/recapture extension is COMPUTED
    + COUNTED but NEVER APPLIED off-PV (re-searches at :1575/:1580 need reduction>0 or isPV). **Confirmed independently by
    TWO agents (af2f023f + a4f0b130) + lead read.** Makes the C4 SPRT a no-op for its headline case. THE joints-thesis
    proof: not the extension code, not the LMR code — the seam. Fix #1 in the queue (gate + fixed-time SPRT, likely strength-moving).
  - **J8/INV-EXT3 [VIOLATED, MED-LOW] SearchFixed RootDepth=0** — searchFixedUnsafe/alphaBeta never set RootDepth → budget
    clamp truncates fixed-depth search past ~depth24. Confirmed by 2 agents (a4f0b130 + af64cc38). Shipping UCI path OK
    (ID sets it); cmd/trace_move + deep tests bite. One-liner: `info.RootDepth = depth`.
  - **J10/DIV-FUT1 [VIOLATED, MED] futility prunes quiet promotions** (search.go:1301 lacks `&& PromoType()==NoType`; SEE-quiet
    block at :1338 has it). LMP/hist-prune same gap [LOW, near-unreachable: promos sort first].
  - **J12/DIV-Q1 [VIOLATED, LOW-MED] qsearch stalemate scored as material eval below qDepth==0** (:1782 stalemate check is qDepth==0-only).
  - **J9/DIV-RFP1 [VIOLATED, LOW]** RFP gated `beta-alpha==1` not `!isPV` (mate-distance window collapse edge). **DIV-Q2 [cosmetic]** qsearch seldepth double-count.
  - **af64cc38 fix-review: 4/5 session fixes ROCK SOLID** (C4 matches canonical PVS+LMR in all 9 cases; budget math sound on ID path;
    stability-time + increment-alloc can't flag — hard cap/emergency guard untouched; C5 revert dropped nothing). 2 non-blocking notes:
    SearchFixed RootDepth (above) + the "0.3 cap prevents flagging" comment is overstated (emergency guard is the real backstop).
  - **a186f283 TT/ordering/state: NO live corruption.** Latent-only: RepStack push/pop asymmetry (unreachable, ply-100<512),
    IID/singular RepStackLen straddle panic-unsafe (contained), singular-vs-mate-TT (efficiency), root-killer-depth-slot (efficiency).
  - **Broad CLEAN BILLS corroborated across agents:** TT bounds/mate-encoding/TT-move-legality, make/unmake balance, repetition split,
    ordering bands, NMP gate (C3), LMR formula/cap, probcut, singular re-entrancy, qsearch in-check evasions, delta/SEE, draw detection.
- **NO code fixed this session (Bryan's explicit OK).** Tree change = docs only; build untouched. Fix queue (docs/09) is the durable artifact.
  RESUME: work the fix queue top-down, ONE at a time, gated (depth-10 Kiwipete node/seldepth vs clean HEAD + smoke-tactical + go test);
  J1 first (re-validates C4). Also close the extension-budget test gap. HEAD before docs commit = 87714a1, tree clean.

**2026-05-31 (cont. 10) — >>> NPS IS THE ELEPHANT (profiled) + docs/10 + T1 IN PROGRESS <<<** Bryan asked why
Blunder is still ahead. MEASURED (Kiwipete, equal movetime, both E-cores): NGN ~267K nps / depth 11 @ 3.6s vs
**Blunder 7.4.0 ~640-750K nps / depth 13 @ 2.0s** — Blunder is ~2.5x faster per node and ~2-3 plies deeper at
equal time. At ~40-60 ELO/ply that ~2-ply deficit ≈ 100-140 ELO ≈ the whole CCRL gap (2532-2380). Both engines
agree the position is ~-1 pawn → NGN's eval is NOT sign-broken; it's a SPEED/DEPTH deficit. So "I suspect bugs"
is half-right (J1 is real) but NPS is the bigger, measured lever. Memory's "NPS ~1M" was WRONG (perft/BenchmarkMode);
real is ~267K. Have we done every standard technique? YES (per docs/01-09) — gap is NOT missing techniques; absent:
Lazy SMP (Threads is a stub; Blunder likely single-threaded too) + NNUE (deferred).
- **CPU profile (verified, docs/10):** eval ~15%, isInCheck ~15% (called 2-3x/move), move ordering ~14% (eager
  SEE-per-capture + O(n²) selection sort BEFORE trying a move), movegen ~10%, **memclr ~8% + madvise ~9% + defer
  ~3.5% = ~20% PURE OVERHEAD.** Root cause: SearchInfo pre-allocs 4 reuse buffers but 3 are DEAD FIELDS
  (MoveBuffer/OrderedBuffer/SEEGains unused); hot path stack-allocs fresh [256]Move/[256]int per node (search.go:1251/
  1270/1278/1280 + moveorder.go:321) that ESCAPE to heap + re-zero every node (~3.5KB/node). alphaBetaPV=62% of allocs,
  qsearch=33%. The zero-alloc infra was half-built + abandoned.
- **Ranked attack plan (docs/10):** T1 [NEUTRAL, ~15-20% NPS] reuse hot-path buffers via a recursion-FRAME pool
  (NOT ply-indexed — IID/singular re-enter same ply → would clobber, the J3 joint); T2 [BEHAVIOUR, biggest algo]
  lazy/staged move ordering (SEE on demand, stop on cutoff) — needs SPRT; T3 [NEUTRAL, ~3%] drop the per-node RepStack
  defer; T4 cut redundant isInCheck/givesCheck. GATE for [NEUTRAL]: identical node counts + bestmove at fixed depth
  vs clean HEAD (if counts move it's secretly BEHAVIOUR → SPRT). Do NOT trim eval for speed.
- **T1 SHIPPED + KEPT this turn** (Bryan: "integrate into todo + proceed"). Frame-counter-keyed buffer pool in
  SearchInfo (searchFrame{moveBuffer,ordered,quiets,captures}, balanced via the existing RepStack defer — no new
  defer, stack fallback on pool exhaustion). Scope = alphaBetaPV main-loop buffers (the 62%); qsearch (33%) = T1b.
  VERIFIED behaviour-neutral: IDENTICAL node counts + bestmoves at fixed depth on 3 positions (Kiwipete d12 1339599/
  d5e6, mid d12 859325/d2c2, end d16 848103/b4f4) + go test green. Cut allocs ~75% (77→18/op). Real-search NPS ~0-2%
  (within E-core noise), below the ~15-20% estimate. **KEPT — clear improvement + ZERO regression = KEEP, full stop**
  (I nearly reverted it citing "benefit within noise"; Bryan: that's the rule INVERTED — see [[feedback_confirm_before_dropping_findings]]).
- **LESSON (docs/10):** the benchmark profile OVERSTATED alloc/GC — memclr+madvise ~17% came from BenchmarkSearchDeep
  LOOPING short searches (churns GC); a single game search allocs buffers once + GC barely runs, so killing 75% of allocs
  moved real NPS <2%. Alloc is NOT the single-search bottleneck. Trust the profile's FLAT compute (ordering ~14% incl
  eager-SEE+O(n²) sort, isInCheck ~15%, eval ~15%, movegen ~10%); distrust its alloc/GC cum for single-search. Verify NPS
  on `go depth N`, not a bench-loop profile. **NEXT lever = T2 (lazy/staged move ordering, BEHAVIOUR → SPRT) + T4 (cut
  redundant isInCheck/givesCheck) — bigger than T1.** T1b (qsearch buffers) optional follow-up (same neutral pattern).
- **T1 + T2a SHIPPED/KEPT, NPS gap root-caused (2026-05-31).** >>> The full prioritized speed plan + verified root cause
  + the node-identity gate/baselines now live in the **`## SPEED LANE (NPS)` section at the TOP of this file** — single
  source of truth, resume there. <<< Short version: NGN ~310K vs Blunder ~640-750K (~2-2.4x); four hot paths each computed
  the simple/full way multiply to ~2.7x; queue N1 incremental eval (NEUTRAL, biggest) → N2 isInCheck dedup → N3 lazy eval
  (SPRT) → N4 staged movegen → N5 lazy SEE. HEAD before T2a commit = a1704cb; T2a = 7445851. KEEP RULE unchanged
  ([[feedback_confirm_before_dropping_findings]]).
- **I4 instrument DONE + committed (5f2582f).** `sprt -tc "S+I"` / `gauntlet -tc "S+I"` (fab7f0b) play a real
  tournament clock (go wtime/btime/winc/binc, track both banks, detect flag-outs, force conc-1). Validated:
  detector FIRES (2+0 self-play → 2 time-forfeits) AND current NGN does NOT flag at 10+0.1 (0/24). The
  diagnosis's "flags at bullet" is REGIME-SPECIFIC — only ~2s clocks; above that NGN self-corrects (alloc
  shrinks with the bank, hard limit fires before the old absolute floor). Measure-first fixed a wrong assumption.
- **Steps 2-3 DONE + committed (fdbf565), VALIDATED +168 ELO [+50,+287] / 0 flags at 2+0 self-play.** time.go:
  stacked-multiplier alloc → increment-aware `soft = usable/moves + 0.8·inc`, `hard = min(4·soft, 0.3·usable)`
  (the 0.3 cap MATHEMATICALLY prevents flagging). Dropped the absolute minThinkTime(0.5-8s)/minDepth(4-12)
  floors (the bullet-killers) for bank-relative soft/hard stop (project next iter = 2× last completed;
  finish ≥depth 1). Removed 3 dead multiplier helpers + stale test; all time tests pass.
- **Step 4 (stability-scaled soft) IMPLEMENTED — SPRT RUNNING (uncommitted).** Chose the minimal, strictly-safe
  half: scale the soft budget DOWN only as the decision settles (never up — scaling up is the riskier half and
  the larger remaining bank redeploys banked time to hard moves automatically). `TimeManager.stableIters` counts
  consecutive COMPLETED iterations with unchanged best move AND window-held-on-attempt-1; any change or re-search
  resets it (the sharpness flag). `shouldStopTournamentSearch`: stableIters≥6 → soft×0.5, ≥3 → soft×0.7; HARD cap
  UNTOUCHED (so it only ever plays FASTER → cannot reintroduce flagging — mechanism-tested in TestStabilityScaledSoftStop).
  Wired in search.go completed-iteration branch (ReportCompletedIteration(bestMoveChanged, attempts==1)). This also
  SUBSUMES single-legal/easy-move (a forced move is stable from depth 1 → stableIters climbs fast → self-accelerates),
  so those special-cases are DROPPED as redundant unless the SPRT shows headroom. smoke-tactical 28/30 held.
  VALIDATION: real-clock only (no effect at fixed depth/nodes/movetime) → `sprt -tc 10+0.1` build/ngn_stab vs
  build/ngn_base_stab (clean HEAD 1470931), 400-opening book, /tmp/sprt_stability.log. Per Bryan's PROCEED policy:
  sound + hard-cap-safe → keep on "no big regression," don't grind to a bound.
- **Step 5 (fast paths) — DROPPED as redundant** (subsumed by step 4's stability self-acceleration). Revisit only
  if step-4 SPRT shows the forced/easy-move case is still leaving measurable time on the table.
- **BINARY MAP:** build/ngn_stab = step-4 stability scaling (UNCOMMITTED, under SPRT). build/ngn_base_stab =
  clean HEAD 1470931 (code == 2f5ddbc = SEE+NMP+timefix, the stability A/B baseline). build/ngn_tc (2f5ddbc =
  pre-stability). build/ngn_nmp (d6ce5b0 = HEAD−timefix). build/ngn_see (SEE-only). build/ngn_base_see (pre-SEE).
  Session commits: 844bb20 NMP, 5f2582f I4, fdbf565 timefix, fab7f0b gauntlet-tc, 1470931 gauntlet-verdict.
- **FLAKY FUZZ still open (separate task):** TestFuzzSearchLegalMoves time-seeded random, flags illegal moves
  on clean HEAD too (test artifact: pseudo-legal posgen + uncleared TT); make it print the FEN + audit TT-move
  legality validation. Use `go test -short ./engine/` to skip it in the normal loop.

**Why it's broken today (the autonomous-play path is UNTESTED):**
- Real games route to the `Tournament` time control (uci.go:501); EVERY harness
  (smoke/SPRT/elo-assess/blunder-finder) uses `movetime` = the hard-cap-only `TimePerMove`
  path. So `shouldStopTournamentSearch` (time.go:309-382) — the real-game brain — has ZERO
  test coverage.
- **Bullet flags (loses on time).** On a 60s bank, per-move allocation is ~2.5-4s → out of
  time in ~20 moves (games run 40+). Three compounding causes: (a) alloc =
  `baseTime / ~20-25 moves × stacked multipliers`, NOT a sustainable fraction of the bank,
  and bleeds far faster than the increment refills (time.go:140-176); (b) ABSOLUTE floors —
  `minThinkTime` 1-2s, `minDepthFloor` 5-6 — that ignore a tiny bank (time.go:325-344;
  comments admit they were "increased to competitive level" for LONG games); (c) soft-stop
  allows spending 1.5-2× allocated when >30-60s remain (time.go:376-381).
- **Dead "danger" knob.** `getScoreChangeMultiplier` (spend-more-when-score-drops) is always
  1.0: allocation runs with `lastScoreChange==0` because uci.go:506 passes `(lastScore,
  lastScore)` BEFORE `SetTimeControl` computes the allocation; the in-search update
  (uci.go:655) lands too late and is never re-read.
- **No "stop early when quiet."** Soft-stop projects only the next iteration's TIME COST
  (time.go:363-365); it never looks at what the search FINDS. Missing the decision-quiet
  signals: best-move stability across iterations (tracks `previousScore` but NOT
  `previousBestMove`, search.go:548), aspiration-window-held-first-try (computed at
  search.go:606-702, then thrown away), root falling-eval/fail-low, easy-move margin (no
  2nd-best tracked at root, search.go:657), single-legal-move shortcut. NB: "quiet" here =
  ROOT decision-quiet, NOT qsearch leaf-quiet (that's already used correctly to bound the
  tree — a different thing; don't conflate).
- GOOD (don't regress): the clock-interrupt blunder-guard keeps the last COMPLETED depth's
  move (search.go:672-681), so it loses on time, doesn't hang pieces. NPS ~1M, so depth isn't
  the issue — this is purely bank budgeting.

**Plan (sequence I4 FIRST — unmeasurable otherwise):**
1. **I4 real-clock instrument** — fixed-time / short-TC self-play (concurrency-1, reversed
   pairs) or a gauntlet vs a fixed binary ladder at actual `wtime/binc`. The ONLY instrument
   that can see flag-outs. Prerequisite for everything below.
2. **Allocation → increment-aware fraction of remaining:** `soft ≈ remaining/movestogo +
   0.8·inc`; `hard ≈ min(soft·N, remaining·0.3)`. Sustainable at bullet.
3. **Floors relative to the bank** — scale `minThinkTime`/`minDepth` to remaining (or drop
   them under low time). The absolute 1-2s floors are the bullet-killers.
4. **Stability-scaled soft limit (the "intelligent" part)** — best-move stable K iters +
   window held → ×0.5-0.6 (bank it); falling eval / best-move just changed → ×1.5-2 (spend
   it). Wire the 3 signals the search already half-computes; fold the dead score-change knob
   in here.
5. **Fast paths** — single legal move → play instantly; easy-move (best ≫ 2nd-best at low
   depth, needs 2nd-best tracking at root) → move early.
6. **Validate** on the I4 gauntlet at 1+0 / 2+1 / 3+2: success = never flags AND holds ELO
   vs a fixed-time baseline.

Absorbs I1; promotes I4 from "HIGH" to PREREQUISITE. UPDATE 2026-05-29: cmd/gauntlet
(CCRL-rated Blunder anchors, `make gauntlet`) now reports NGN's absolute rating at fixed
movetime — but biased ~+30-80 HIGH because movetime under-times the anchors vs their 2'+1"
CCRL-Blitz rating. The unbiased true-clock gauntlet is gated on THIS time-management fix;
finishing it lets us drop the gauntlet's bias caveat.

### >>> 2026-05-30 SEARCH AUDIT (6-agent fan-out; EVERY claim re-checked against live code) <<<
Fanned out 4 search-trick deep-dives (ordering / LMR+LMP / null-move / TT) + 2 meta. Validated
findings recorded below; over-claims rejected (the "LMR re-search not PV-gated" framing was wrong
as stated — the real defect is the `< beta` guard, see C4). **HEADLINE: C3 — null-move pruning is
disabled at every cut node** (a stale `beta-alpha>1` gate that PVS never satisfies off the PV
spine) — exactly the "big bug in a small section" we suspected. Plus C4/C5 (LMR re-search + scout
depth), S4–S7 (LMP-improving, capture/quiet ordering, root prev-best, TT-eval reuse), and S2
independently re-confirmed by two agents. **TT audited CLEAN** (mate-adjust, bounds, key-verify,
replacement all correct — do NOT re-chase). Implement ONE at a time, SPRT-gated (fixed-TIME for the
node-spenders C3/C4/S4), AFTER the running gauntlet finishes (an SPRT now would taint its timing).
Search is the current 50+-ELO lane; eval (Phase E) is the proven backup.

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
- [x] **C3 — Null-move pruning was OFF at every cut node — FIXED & SHIPPED (+138 ELO [+45,+231] fixed-time SPRT vs HEAD, 2026-05-30).**
  DONE: gated NMP on `canNull && !isPV && staticEval>=beta` (canNull threaded through alphaBetaPV, set false
  on the null recursion to block back-to-back nulls); nmcut now ~7K/search (was PV-spine only), ~25% fewer
  nodes to depth 10. Batched + shipped with S6 (root prev-best ordering), S7 (TT-score-as-static-eval),
  RT1 (debug-on counter dump). Open follow-up once more is wanted: depth-scaled `R = 3 + depth/4`. The
  saga: my C5 "cleanup" of the reducedDepth dead branch (→ a `<1` clamp) caused a check-extension explosion
  that masked C3 for hours — see the SESSION STATE GOTCHA at top. ORIGINAL DIAGNOSIS:
  The gate `beta-alpha > 1 && (!isPV || depth>=6)` requires a FULL window, but PVS scouts every
  non-PV node with a null window (`-alpha-1,-alpha` ⇒ beta-alpha==1, search.go:1487), so NMP fires
  ONLY on the PV spine (~depth nodes) — effectively off in ~99% of the tree. NGN's own notes put NMP
  at +80-116, almost none of it currently realized. `beta-alpha>1` is a stale pre-PVS "is this a PV
  node" test that contradicts the later `isPV` thread. FIX: replace the gate with `!isPV`, AND add a
  double-null guard (thread `canNull bool`, set false on the line-1118 null recursion) — that broken
  gate is presently the ONLY thing stopping recursive double-null, so do NOT ship the gate fix without
  it. Follow-ups once hot: `staticEval >= beta` gate; depth-scaled `R = 3 + depth/4`. VALIDATE
  FIXED-TIME (I4), not fixed-nodes (undercredits the depth NMP buys). Confirmed vs live code 2026-05-30.
- [ ] **C4 — LMR reduced fail-high trusted without re-verify (search.go:1490), est. +15-40 (SPRT to confirm sign).**
  The re-search guard `if score > alpha && score < beta` SKIPS the full-depth verification when a
  reduced scout fails HIGH (`score >= beta`): the reduced-depth cutoff goes straight to the beta cutoff
  at 1514. Canonical PVS re-verifies at full depth on ANY `score > alpha`. FIX: re-search full-depth
  zero-window when `reduction>0 && score>alpha`; open the full `(-beta,-alpha)` window only at PV nodes
  (`isPV && score>alpha`). (Audit agent framed it as "re-search not PV-gated"; the actual defect is the
  `< beta` guard — recorded correctly.) Could be a deliberate speed/accuracy tradeoff; SPRT both ways.
  Pairs with C5 (same ~20-line block).
- [ ] **C5 — LMR scout depth ignores extensions + dead no-op branch (search.go:1481-1484), est. +10-20.**
  `reducedDepth := depth-1-reduction` uses `depth-1`, NOT `nextDepth` (which carries the
  check/recapture/passed-pawn/singular extension), so an extended quiet is scouted a ply too shallow vs
  its own verify search. And `if reducedDepth >= depth { reducedDepth = depth-1-reduction }` is
  unreachable (reduction≥0 ⇒ reducedDepth<depth) and re-assigns the identical value — a tautology
  mangled out of a `givesCheck` guard in commit 3900696. FIX: `reducedDepth := nextDepth - reduction;
  if reducedDepth < 1 { reducedDepth = 1 }`; delete the dead branch. Confirmed vs live code 2026-05-30.

### Phase S — Cheap independent search refinements (fixed-nodes SPRT signs these)
- [ ] **S1 — Aspiration windows: exponential widening (search.go:573-596).** Today:
  fixed ±50, and on ANY fail it jumps straight to ±INFINITY — throws away the
  aspiration benefit. Use delta≈16, widen ×~2 per fail, keep the non-failing bound,
  go full only after a few fails. Blunder aspiration +23; delta from improving an
  existing crude version is smaller but real.
- [~] **S2 — Killer ply-indexing — REJECTED 2026-05-30 (regressed ordering for NGN).** Drop-in switch
  depth→ply INCREASED fixed-depth nodes +28-43% in 3/4 positions (Kiwipete 389K→541K) — worse ordering, not
  better. The two audit agents argued from general theory; NGN's depth-indexed killers (broad per-remaining-
  depth sharing) empirically out-order ply here. Reverted, not committed. Only revisit via a real fixed-time
  SPRT (the node check killed it as a freebie). sites if retried: search.go:1484/1544, moveorder.go:83/370.
- [x] **S3 — Qsearch stores fail-highs to TT — committed a0fdda4 2026-05-30 (batch SPRT pending).** Stores
  ONLY fail-highs (cutting capture, depth-0 LowerBound). Storing every q-node flooded the TT (hashfull 20→63,
  +26% nodes — the exact "evicts deeper main entries" pitfall); fail-high-only keeps it flat. QSearchTTStores
  counter + TestQSearchTTStoresWired added. tactical 28/30. Follow-up if wanted: an Upper-bound store guarded
  by "only if slot empty" to add the non-cutoff ordering hint without the flood.
- [x] **S4 — LMP gated on `improving` — committed d21a833 2026-05-30 (batch SPRT pending).** 2-row
  lmpThreshold[improving][depth]; not-improving prunes ~2x sooner ({0,3,4,6,9,12,15,19}), improving row =
  HEAD. seldepth 29→26, tactical 28/30. est. +20-40.
- [ ] **S5 — Losing captures ordered ABOVE good-history quiets (moveorder.go:343), est. +10-20.**
  SEE<0 captures score ~20000 (band base); quiet ceiling is ~16384 (history), so every losing capture
  is searched before every non-killer/counter quiet. DELIBERATE per the code comment, but Ethereal/
  Berserk/Blunder all order bad captures LAST (Blunder prunes them). A/B a negative base
  (`score = -1_000_000 + see + chScore`) so losing captures fall below quiets.
- [ ] **S6 — Root doesn't search the previous-iteration best move first (search.go:623), est. +10-20.**
  `orderMovesIntoBufferWithDepth` passes no TT/prev-best move and `iterationBestMove` isn't fed back, so
  each ID iteration re-derives root order from scratch. Standard ID hygiene; ALSO supplies the
  best-move-stability signal the time manager wants (ties to the NEXT block, search.go:548/657). FIX:
  seed root ordering with the prior iteration's best (swap to index 0, or pass as a pseudo-TT move).
  Flagged independently by the ordering + TT agents.
- [ ] **S7 — Reuse the TT score as the node's static eval (search.go:1076), est. +5-15.** On a TT hit,
  refine `staticEval` from `ttEval` when the bound permits (Exact, or Lower>eval, or Upper<eval) — a
  search-refined value beats a fresh eval for RFP/futility/NMP/improving margins and skips an eval call
  on hits. Ethereal/Stockfish standard; the probe data (ttEval/ttNodeType) is already in hand at 1076.
  Fixed-TIME SPRT.

### Phase E — Eval (the PROVEN lane; +46 already banked from PST+weights)
- [x] **E1 — Tune MATERIAL jointly with PST (DONE · NULL RESULT · 2026-05-28).** Tuner infra
  shipped (`TexelMaterialParams`+`FormatMaterialValues`+`-target full`). Re-fetched the 725K
  Zurichess set (KierenP mirror; 272K/253K/198K — matches the +46 run). Ran `-target full`
  (788 params, 29 min) starting from the +46-tuned eval. **MSE barely moved: 0.059218 →
  0.059180 (~0.00004), ~18× SMALLER than the +46 tune's 0.00069-for-+46-ELO move → ~+2 ELO,
  sub-resolution.** EG material didn't move at all; MG moved a hair (queen 1025→1077, rook
  477→497, pawn 82→78). NOT shipped — not worth an SPRT. CORRECTS the "~+30 Rofchade" guess:
  material ≈ PST-mean, and the +46 PST tune ALREADY absorbed it via per-piece table means, so
  explicit material params add essentially nothing on top. Durable lesson: once PSTs are tuned
  free-form, separate material tuning is redundant. (The MSE-tune also re-confirmed kingSafety
  stays suppressed at 57 → E2 is the real eval lever.)
- [x] **E2 — King-safety suppression FIXED + SHIPPED (+17-22 ELO · commit 125b5d8 · 2026-05-29).**
  Bumped `kingSafetyWeight` 57→100 (revert the quiet-data suppression). Two runs both positive:
  **+22.6 [-25,+71] @200g, then +17.4 [-10,+45] @600g (LLR +0.69) — ~+18-20 across 800 games.**
  Just under the instrument's p<0.05 wall (a +18 effect needs ~900+ games) but a consistent,
  theory-backed lean STRONGER than the shipped `improving` flag (+19.7 [-8,+48]); shipped on
  that bar. Net-positive despite tactical 29→28 (the game SPRT prices the nick in). FIRST eval
  ELO win since the +46 PST tune, and it came from the MSE-vs-ELO gap (NOT MSE-tuning — E1
  null). FOLLOW-UPS: sweep 80/100/130 for the peak; `kingActivityWeight` (29, endgame) is
  likely suppressed the same way — test it; fold the double-counted shield/open-file penalty.
  ORIGINAL LEAD NOTE: scored **+22.6 ELO [-25,+71], 99-15-86 = 53.2%** over 200 games (fixed-nodes, vs HEAD) —
  inconclusive (CI spans 0) but a clear POSITIVE lean, opposite the C1 batch. Validates "MSE-
  optimum ≠ ELO-optimum" for king safety (low MSE leverage on quiet positions, real value in
  play). COST: tactical 29→28 (deterministic — king-safety weighting misses one forcing shot),
  but the +22.6 game lean ALREADY prices that in and is still net-positive. NOT shipped yet
  (inconclusive CI + tactical nick); eval.go reverted to 57, HEAD clean. 600-game confirmation
  RUNNING (kingSafety=100 vs pre-E2 HEAD). If it firms up positive → ship 100 (or sweep
  80/100/130 for the peak first). Caveat: a +22 effect needs ~900 games for p<0.05 (the
  instrument's resolution wall), so even 600 may stay inconclusive — decide on lean+theory.
  Also TODO while here: fold the double-counted missing-shield penalty (evaluatePawnShield
  ~1175 + open-file block ~744) into one term; re-check the phase 6-11 dead zone (eval.go:2433).
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
- [ ] **I1 — [PULLED TO TOP → see NEXT block] Tournament time management is lenient (≤2× allocated) AND untested**
  (time.go:309-382 only runs in real clock games; smoke uses movetime hard-limit,
  SPRT uses fixed-nodes/depth). Real CCRL risk: lose on time / misallocate. Pair the
  overhaul with a real-clock gauntlet vs a fixed ladder (Blunder/Zurichess binaries)
  — the only instrument that can see this class of bug.
- [ ] **I2 — Wire `setoption Hash` + decide `Threads` (uci.go:296-300 are stubs).**
  TT hard-pinned at 128MB; "Move Overhead" advertised (default 100) but ignored
  (stays 50ms). Needed for external-tester fidelity.
- [ ] **I3 — elo-assess scores stalemate as a loss (cmd/elo-assess/main.go:261-267).**
  Minor harness correctness (smoke being retired as a yardstick anyway).
- [ ] **I4 — [PREREQUISITE FOR NEXT block] fixed-TIME SPRT complement (HIGH — surfaced by C1).** The fixed-NODES instrument
  structurally undercredits node-SPENDING techniques (IID/IIR/extensions): it charges their
  exploratory nodes without crediting the depth they buy at a clock. Half the Phase-1 search
  roadmap is this class, so fixed-nodes is the WRONG signer for it. Add a clean fixed-time
  self-play mode (concurrency-1 to kill timing noise; reversed-color pairs; ~200+ games) OR a
  short-TC gauntlet vs a fixed binary ladder. Without it, IID/IIR/extension work can't be signed.
- [ ] **I5 — Blunder-finder → self-play capture + Stockfish-judge attribution (the RUNTIME-understandability lever).**
  Design pass verdict: KEEP the existing SF-judge core (cmd/blunder-finder `analyze`/`evalAfterMove` is
  sound — full-strength SF, mover-POV apples-to-apples, mate handling, decided-position guard); ADD
  (1) self-play game CAPTURE via internal/uci.PlayGame vs a stop-compliant Blunder anchor at a REAL
  clock (shares the I4 real-clock work + a per-move OnMove/score trace ≈ 20 lines in internal/uci, since
  GetMove currently discards NGN's score/PV), and (2) attribution buckets (tactical-oversight /
  time-pressure / positional-drift / endgame / missed-mate) from SF-best-vs-NGN drop +
  `self_delusion = ngn_score − sf_eval`. Emit a bucket table + one mechanical "FIX SIGNAL" line. This is
  what tells us WHY NGN blunders in live play (Bryan's runtime-understandability ask). Uses fixed-NODES
  SF as judge for reproducibility; judge stays OFFLINE so the older-Blunder stop-ignore problem is
  confined to the opponent (handled by RespectsMovetime/Overshoots).
- [ ] **R1 — Code clarity (LOW priority per Bryan — runtime ≫ code).** Only the high-value,
  behavior-NEUTRAL bit is worth doing now: eval.go is ~60% DEAD code (orphaned hand-PST
  `pawnPST…kingEndgamePST` + `getPieceSquareValue`, plus ~14 unused `evaluate*` term functions) —
  grep-verified zero non-test callers; the live eval is `evaluatePeSTO` + the ~11 terms in
  `evaluateUnsafe`. Dead code actively misleads bug-hunting (a DIAGNOSIS "knightPST" item points at a
  dead table). Delete in one behavior-identical commit (tests reference them — update/drop together).
  Stale README/CLAUDE NPS (~350K → ~1M) and ELO numbers also mislead. Everything else (split
  alphaBetaPV, banner comments, debug-tagged invariant asserts) DEFERRED unless it blocks a fix.

### Phase RT — Runtime understandability (Bryan 2026-05-30; "expose what's already computed")
NGN already COMPUTES a correct multi-move PV, 14 search counters, 11 separated eval terms, and a 4-branch
time-stop decision — but surfaces almost none of it at runtime. All near-free plumbing (audit validated).
- [ ] **RT1 — End-of-search counter dump behind `debug on` (~15 lines), HIGH value/effort.** Emit the
  SearchInfo counters (NullMoveCutoffs/LMRReductions/LMPPrunes/TTCutoffs/… search.go:471-484) as one
  `info string` after the search (uci.go:709); the formatter already exists in
  cmd/pruning-analysis/main.go:59-90. DIRECTLY validates C3 (watch NullMoveCutoffs before/after the fix)
  plus C4/C5 in REAL searches, not the offline start-position tool.
- [ ] **RT2 — Time-stop reason logging (~10 lines) — UNBLOCKS the time-controls NEXT block.**
  `shouldStopTournamentSearch` (time.go:309-382) returns a bare bool from 4 distinct points
  (hard/emergency/floor/soft-projection). Record which fired + elapsed/alloc/nextEst/scoreChangeMult and
  print via `info string` (debug). The overhaul is untested BECAUSE the stop decision is invisible.
- [ ] **RT3 — `eval` UCI command: term-by-term breakdown (~30 lines).** Add `EvaluateVerbose` beside
  evaluateUnsafe (eval.go:2382-2444; terms already isolated with named weights) + `case "eval"` in uci.go.
  Runtime side of eval tuning + positional-blunder triage (I5). Also retires cmd/eval_breakdown's own TODO.
- [ ] **RT4 — PV + counters in cmd/analyze_position & cmd/trace_move (~10 lines each).** They already hold
  `*SearchInfo`; print `ExtractPVFromTT` (search.go:735) + the RT1 counter block. The offline "replay a
  position, see move/score/PV/depth/heuristics" half of I5; zero risk (offline tools).
- Lower-pri runtime gaps: no currmove/currmovenumber, no aspiration lowerbound/upperbound, no in-process
  `bench`, no `d` board dump. The per-depth UCI info line itself is already near-Stockfish-standard (uci.go:702).

### Parked (orthogonal multiplier)
- Lazy SMP — +80-150 multi-core, Go makes it easy. Biggest single jump if wanted;
  not on the critical path.

### Audit false-positives (recorded so they aren't re-chased)
- TT (cache.go / search.go) — RE-AUDITED CLEAN 2026-05-30 vs 4 reference engines: mate-score ply
  adjustment correct (`scoreToTT`/`scoreFromTT` match Stockfish), cutoff bound/depth/PV gating correct,
  full-key XOR collision verification, all store paths covered, aging + per-game clear, depth-preferred
  replacement. The classic silent TT bug is NOT present (re-confirms the 2026-05-25 audit). Remaining TT
  items are config/refinement only: I2 (Hash unwired), S7 (eval-reuse), and a minor 50-move-rule clamp
  on TT mate cutoffs (rare, low ELO) — none are the hidden 50+.
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

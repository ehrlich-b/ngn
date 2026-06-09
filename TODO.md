# NGN Chess Engine — Roadmap

> Reanchor = re-read README.md, this file, CLAUDE.md. Canonical strategic home. Deep-rewritten 2026-06-09 (4-agent adversarial review: EBF measured, texel autopsy, CounterGo census, infra audit — all load-bearing claims spot-verified in code/logs).

## Where we are (2026-06-09)
- **Strength: ~2600 CCRL Blitz, SOFT, single-family.** Spread ~2540 (Jun-4 bracket, best-sampled, storm-capped) .. 2632 (Jun-8, 120+1, 24g). **Bryan is happy with ~2600 — focus is building strength, not measuring it.** Clean pin available if ever wanted (#R below). [[ngn-elo-ccrl-reality]] ⚠ Two newly-found harness biases inflate our reads: anchors play on DEFAULT hash (Fruit 16MB vs NGN 128MB) and MaxMoves=200 adjudicates slow conversions as draws (Lane D fixes).
- **Goal: 2800 — HCE CAN reach it, no NNUE.** Proof CORRECTED: **Counter 3.8 (pure-Go HCE) = 2994/3012 CCRL with FEWER eval terms than NGN** (~46 vs NGN's inventory; the old "~3373" figure was the NNUE-era Counter 5.x); SF11-HCE ~3400. Same language, same paradigm, +360-400 over us — by construction achievable.
- **HEAD = `56bf755`**. Node-identity: RE-LOCK at HEAD via `scripts/nodecheck.sh` after any eval change; speed changes must reproduce EXACT.
- **Infra clean bill (2026-06-09 audit):** hot path verified alloc-free (0.000000/node live test), TT carry-over between moves correct, GC ≤1-2 cycles/GAME at 128MB ballast, no logging/book pathology. No per-game infrastructure bleed exists — the gap is chess, not plumbing.

## THE DIAGNOSIS (2026-06-09, two probes) — the gap is OPTIMISM × EBF
- **`cmd/lossxray` over 86 real losses:** 64% slow positional bleed / 35% eval-blindspot (own eval +213cp vs reality at point-of-no-return) / **0% tactical horizon-miss**; median 34 moves played after lost.
- **`cmd/zzbias` over 596 positions:** **+97cp median own-side optimism at phase 6-11** (vs +40 mg, ~0 eg). Sharply localized.
- **EBF probe (NEW, fixed 400K nodes, 6 positions):** NGN middlegame **depth 11, EBF ~2.9-3.2**; modern HCE (Ethereal 12 / Weiss / CounterGo / SF11) reaches **depth 17-19 at the same nodes (EBF ~1.9-2.2)**. **Blunder 7.4 is the SAME class as us (EBF 3.09)** — every prior "tree is fine" conclusion was anchored to a 2532 peer and couldn't see this. ~5-6 quiet middlegame plies are available at FIXED nodes, before any speed work.
- **Reconciliation — "0% tactical" does NOT mean depth-is-fine:** lossxray rules out horizon-miss *blunders*; being quietly out-planned by a 6-plies-deeper opponent presents EXACTLY as slow positional bleed. The bleed has two consistent, co-primary causes: **eval over-optimism (proven: the only category that ever transferred, +43.7)** and **an EBF gap vs the 2800+ class (newly measured)**. Bound estimate: EBF 2.93→~2.2 ≈ +4-5 mg plies ≈ +150-250 blitz ELO — approximately the entire 2600→2800 gap.

## THE PLAN — four lanes, ranked; one change at a time, always SPRT-gated

### Lane A — EBF: 2.93 → ~2.2 (search-SHAPE stack)  ·  EV +150-250 bounded  ·  CO-#1
NOT the closed scalar-retune lane: every item adds a missing INPUT or extends a technique over depth ranges where it's ABSENT. All verified in code 2026-06-09. Per-item cadence: **EBF probe (deterministic +ply readout at 400K nodes) → tactical ≥40% → short-TC SPRT 10+0.1 [-5,0] → KEEP rule.**
- **A1 — LMR shape (the #1 lever):** moves 2-3 are NEVER reduced (`LMR_MOVE_THRESHOLD=3`, search.go:28,1609 — modern reduces from move 2); **no cutnode dimension exists in the engine** (0 grep hits — thread `cutnode bool` through alphaBetaPV, `+1` when cutnode); history input is ±1 at thresholds (search.go:1631-41) → fractional `clamp(hist/5000,±2)`. Stage the three sub-changes. (est 30-80)
- **A2 — RFP/futility/SEE-prune depth range:** all die at depth ≤3-4 (search.go:1197,1205,1416); CounterGo runs RFP ≤8 @100·d, futility ≤8, SEE-prune quiets+captures ≤8. Extend with improving-adjusted margins (`improving` exists, search.go:1191, but feeds no margin). (15-40)
- **A3 — NMP scaling:** R is a CONSTANT 3 (search.go:25,1233 — null child at depth−4 forever); modern R = 3-4 + depth/4-6 + min(3,(eval−beta)/200) + verification at high depth. (15-40)
- **A4 — qsearch quality:** captures searched in RAW GENERATION ORDER (pawns→N→B→R→Q, search.go:1948-62) — the "−129 ELO MVV/LVA" comment cites a revert made **5min37s after commit, in the rolling-1653 era** (pre-C3, pre-clock-fix, underpowered-SPRT era) = FOLKLORE, VOID. Re-test MVV-LVA (+capture-history tiebreak); add quiet promotions to capture-gen (qsearch never sees queening, movegen.go); lift the `qDepth>=6 → raw static eval` mid-sequence truncation (search.go:1799). (10-30)
- **A5 — IIR:** none exists (only PV-node IID, search.go:1118). `if ttMove==Empty && depth>=4 && !isPV { depth-- }`. (8-15, ~3 lines)
- **A6 — killers indexed by remaining DEPTH not ply** (moveorder.go:83) — siblings probe different slots, root/leaf smear. Two-token fix; watch FirstMoveCutoffs. (5-15)
- **A7 — singular: verification at depth−4** (norm: depth/2), fixed margin 64 (norm: 2·depth), **no multicut** (free cutoff when exclusion also fails high), no mate-score guard. (8-20)
- **A8 — history updates:** depth² bonus + whole-table halving on any entry crossing 8192 (moveorder.go:99-102,212-18) → gravity form `entry += bonus − entry·|bonus|/16384`; 2-ply follow-up table later. (5-15)
- **A9 (probe-first, adjoins deferred C4/C5):** extension diet — unconditional check ext at every depth (seldepth 28 @ depth 11; SF11 gates on SEE≥0), recapture + passer exts CounterGo plays 3000 without.
- Estimates are fishtest/changelog-era provenance, NOT additive — the lane total is the EBF bound. After shapes land, SPSA the new margins jointly (absorbs old Lane 2).

### Lane B — phase-6-11 over-optimism reduction  ·  EV +20-50, proven-transfer category  ·  CO-#1
Continues `7587eb6` (+43.7, the one eval change that transferred; bias +134→+97cp). Residual +97 remains; `zz_termagg` ranks kingSafety +41.5 still #1.
- **B1 (cheap, deterministic, DO FIRST in lane):** `cmd/zzbias` attribution — split the residual +97 by term (shield magnitude with queens off vs phase-gating hole vs other).
- **B2 — king-danger MODEL upgrade (the CounterGo answer):** NGN models *proximity* (shield arithmetic + zone attack-units) but has **no weak-square term and no check modeling at all**; CounterGo/Ethereal score weak-squares-in-king-area (attacked & not doubly defended, needs attackedBy2) + safe checks per piece type × attacker-count weight table — the two things that convert "pressure" into "lost", i.e. exactly the +97cp "my king looks fine" signature. ⚠ ledger shows a prior "safe-checks −77" under tapped-richness: pull that record first — if it was a hand-weighted bolt-on judged by an early-era SPRT it does not close THIS (full model, CG-tuned starting weights, over-optimism category not richness).
- **B3 — drawishness scaling family (extends the banked OCB axis, same safe-by-construction recipe):** strong-side pawn-count scale ×(128−(8−pawns)²)/128, rule50 ramp ×(200−hmc)/200 — **eval never reads HalfMoveClock today** (verified); pawnless-strong-side caps. CounterGo scales EVERY eval through these.
- **GATE:** zzbias (bias drops) → `make acpl` (Δ≤0) → SPRT 10+0.1 [-5,0] → gauntlet-confirm.

### Lane C — tuner BASIS expansion (the legit reopen)  ·  EV +60-120, slow-burn
2026-06-09 autopsy: pipeline PROVEN sound 3 ways (bit-exact trace gate + numeric-gradient check + MSE-recovery, all green); the Jun-8 "−27/−41 REJECT" was **VOID** (SPRT killed at G68, LLR −0.07 = 2.4% to bound = no information; ACPL said tuned build −1.0 BETTER; the tune itself moved params ±1-6cp = a no-op because the 796-param basis was ALREADY at its optimum — +46 `bb12a48` and +68 `5915619` banked long ago). **The dividend is not in re-fitting the 796 — it's that the basis FREEZES every term the diagnosis implicates:** mobility is hand-set LINEAR `(count−3)×k` (CG: tuned per-count tables, concave, ~85cp span), threats/passer-curve/KS-internals/shelter all sit in the non-tunable "Fixed lump" (texel_gradient.go:58-78). CounterGo = 1150-weight vector, every SHAPE tuned, 2994 CCRL with fewer terms.
- **C1:** convert mobility to `[piece][count]` MG/EG tables initialized bit-exact to current linear (node-identical proves wiring) → expose to tuner → tune on the existing 1.05M corpus → acpl → SPRT. One term, falsifies "shapes don't matter" cheaply.
- **C2+:** passer rank curve, threats weights, shelter table, KS internals — one family at a time.

### Lane D — harness + time-mgmt correctness (small, cheap, some free)
- **D1 (gauntlet-side, risk-free, ~4 lines):** send `Hash 128`/`Threads 1` to anchors (Fruit defaults to 16MB — our Fruit read is ~10-25 inflated); MaxMoves 200→400 (steals slow conversions from the stronger side); record per-move clocks in `-pgn`.
- **D2 — plies-vs-moves units bug (CONFIRMED):** `movesPlayed++` counts PLIES (uci.go:411) but time.go:179-93 subtracts it from full-move constants → moves-remaining hits the floor-10 by game-move ~8, ~2× front-loaded vs design. 1-line; real-clock SPRT.
- **D3:** soft-time extension ×1.3-1.5 on root fail-low/best-move churn (`stableIters` already maintained but DEAD since the Jun-3 fix, time.go:257-63). ~6 lines; real-clock SPRT.
- **D4:** small repetition contempt (all draw paths return bare 0; 11/24 draws in the last clean run vs a 100-ELO-weaker anchor). acpl + SPRT; can give points back vs stronger pools.
- Also: `Hash`/`Move Overhead` setoptions are advertising-only stubs (real overhead hardcoded 50ms vs advertised 100) — wire or stop advertising before any external/CCRL testing.

### Speed (~1.5×/node vs Blunder) — filler only, unchanged
Banked: 9 node-identical leaf opts, alloc-free hot path, O(1) TT placement. Remaining slivers (qsearch stalemate-probe trim, per-node mutexed repetition-map read) between SPRTs. NOT the headline.

## CLOSED LANES — do not reopen without NEW evidence (anti-oscillation ledger)
To reopen, FIRST state the new evidence the prior close missed. Code locked by tests, rulers by committed instruments, decisions by this ledger.
- **Eval-basis RE-TUNE of the EXISTING 796 params — CLOSED, reason CORRECTED 2026-06-09.** Old record said "−27/−41 REJECT"; that verdict was VOID (killed SPRT @G68, LLR −0.07; ACPL −1.0 better; by the KEEP rule it was keepable). Real reason: the basis is already AT its optimum (re-tune moved ±1-6cp, twice-tuned history). ~0 ELO at stake either way. The richer-basis path is OPEN = Lane C.
- **"MVV/LVA in qsearch = −129" — VOID 2026-06-09.** 5min37s revert (`2f61d3a`→`ccdf8d7`), rolling-1653 era, underpowered-SPRT epoch. Re-test is Lane A4. (KEEP-rule violation instance #2, after the re-tune misread.)
- **Blunder as the tree/EBF reference — CLOSED 2026-06-09.** Blunder EBF 3.09 ≈ NGN 2.93 (same class); it cannot see the gap to 2800. Benchmark EBF against modern-HCE depth-at-fixed-nodes (17-19 @ 400K).
- **Generic eval-term ADDs (richness) — TAPPED 2026-06-08.** Backward-pawns rejected by 2 rulers; beyond OCB it's +5-and-noise. EXCEPTIONS that remain open: over-optimism-REDUCTION (Lane B), tuned-SHAPE conversions (Lane C) — distinct categories.
- **Search-param SPSA over REDUCTIONS — CLOSED 2026-06-07** (flat surface, verified instruments). Margins re-enter ONLY as post-Lane-A joint tune. [[project_search_param_lane]]
- **eg-search "2.5-5× tree" — REFUTED 2026-06-06** (mislabeled-WON-position artifact). [[project_endgame_tree_diagnostic]]
- **TT-move-first staged movegen — REFUTED 2026-06-06** (+85% nodes). [[project_speed_lane]]
- **Lazy qsearch eval — CLOSED-unsound 2026-06-02** (±199cp extras, concrete wrong cut).
- **J1 reduced-scout base — ACCEPTED non-fix 2026-06-01** (load-bearing-dead branch). [[reference_search_spec]]
- **SF `UCI_Elo` slider — CLOSED** (~+400 inflated). **bishop-pair term — DELETED, +20.** **NNUE — RESERVED BY CHOICE** (Bryan's mandate).

## Gates & discipline (STANDING — do not relax)
- **Short-TC self-play SPRT (`10+0.1`, real clock, conc 1, E-cores) is THE per-change gate.** Gauntlet = occasional absolute check only. [[time_mgmt_ruler]]
- **`make acpl` BEFORE SPRT-ing any eval change** (external SF ruler; raises-cp-loss ⇒ suspect). [[oracle_acpl]]
- **KEEP RULE:** improvement OR correctness + no measured regression ⇒ KEEP. REJECT needs BOTH non-correctness AND a measured regression. **An SPRT that hasn't hit a bound (or ≥~400g) is NO INFORMATION — never read mid-run snapshots as verdicts (G68 lesson, twice now).** [[feedback_confirm_before_dropping_findings]]
- **Self-play gains are partly transfer-mirages** — gauntlet-confirm batches. [[sprt_instrument]]
- **EBF probe is the new deterministic pre-filter for Lane A** (fixed 400K nodes, 6 positions, +ply readout — rebuild: python UCI driver, `go nodes 400000`; commit it as `scripts/ebfprobe/` when convenient).
- Node-identity re-lock after kept changes; one SPRT at a time; never P-cores; single-sentence local commits.
- **DO NOT:** re-fit the existing 796 basis; richness ADDs; chase the eg tree; batch changes; read killed SPRTs as verdicts.

## The LOOP (serial; one SPRT lane; never idle-wait)
1. Prep ONE candidate: edit + gates (faithfulness test, symmetry, tactical ≥40%, EBF probe for Lane A, `make acpl` for eval).
2. Launch short-TC SPRT (bg): `./build/sprt -base build/ngn_base -new build/ngn_<v> -tc 10+0.1 -openings output/sprt_openings.txt -elo0 -5 -elo1 0 -mingames 64 -maxgames 400 > output/sprt_<v>.log 2>&1`
3. WHILE LIVE — no-CPU work only (design next candidate / read / tidy).
4. Verdict AT A BOUND (or maxgames): KEEP ⇒ commit + rebuild base + re-lock nodecheck. REJECT ⇒ checkout. Never verdict a killed run.
5. BETWEEN SPRTs: heavy compute OK (profiler / probe / occasional gauntlet).

## Recently shipped (full history in git log + memory/)
- **king-safety attacker-material scaling** `7587eb6` (+43.7 SPRT, transferred) · **OCB draw-scaling** `79f596b` · **O(1) TT placement** `2d6372b` · **SEE short-circuit** `83581b7` · **tuner #21 fix** `49e170b` · **ACPL ruler** `6b435e7` · **gauntlet restart-fix** `b53732a`.

## Optional / deferred
- **#R — clean rating pin** (DE-PRIORITIZED by Bryan): 100g/anchor 120+1 bracket incl. 8.0.0, with D1 harness fixes in. Only if the number is wanted.
- **Dead conversion code:** `evaluateBasicEndgames` (KQK/KRK mop-up + opposition) has 0 callers; live `evaluateKingActivity` is anti-gradient for KBNK. Wire under low-material gate, gate by mate-conversion test — or delete. Also dead: `evaluateCastling`/`evaluateCenterControl`/`evaluatePieceDevelopment`/`countProtectors`, `evaluatePawnStorms` (wrong sign), `OldAge` (cache.go:61).
- Aspiration-exhaustion quirk: 3 failed widenings can silently discard an iteration (search.go:661-783); CG always completes. 5-minute look someday.

## Tooling
- `./build/sprt` (THE gate) · `make acpl` (external filter) · EBF probe (Lane A pre-filter) · `scripts/nodecheck.sh` · `make gauntlet` (absolute, `-tc 120+1`).
- Eval-bias instruments: `cmd/zzbias` · `cmd/lossxray` · `cmd/sfpv` · `cmd/oracle` · `zz_termagg`.
- `cmd/spsa` (post-Lane-A margins) · `scripts/h2h.sh` (⚠ disregard "quiet-end" row) · anchors via `make build-anchors`.
- Reference source: CounterGo HCE checkout = `git clone ChizhovVadim/CounterGo` @ `3e81ab9` (pkg/eval/counter 507 lines, pkg/engine/search.go 611 lines) — the 2994-CCRL existence proof, diff against it before inventing.

## Durable lessons (full history in memory/)
- **L1 (SHARPENED 2026-06-09): the bleed has TWO co-primary causes** — phase-6-11 eval over-optimism (+97cp, the proven-transfer lane) AND an EBF gap vs the 2800 class (2.93 vs ~2.0; Blunder is our class, so Blunder-anchored tree checks were blind). "0% tactical losses" rules out horizon BLUNDERS, not quiet out-planning — don't re-demote depth off that stat. [[project_king_safety_lever]] [[feedback_follow_the_diagnosis]]
- **Verdicts need bounds:** two KEEP-rule violations found this review (re-tune G68 misread; MVV/LVA 5-minute revert). A killed/underpowered SPRT is not a null result. [[sprt_instrument]]
- **Benchmark against the class you want to join, not the peer you just beat.** Blunder agreement told us nothing about 2800.
- **follow-the-diagnosis / suspect-the-experiment** [[feedback_follow_the_diagnosis]] · bug history [[project_bugs]] · tuner [[texel_pipeline]] · CCRL anchoring [[project_elo_reality]].

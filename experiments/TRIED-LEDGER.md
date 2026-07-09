# NGN — Everything We've Tried (consolidated ledger)

Last updated 2026-07-08. This is the single inventory of every lever attempted, with verdicts,
so we stop re-grinding dead ends. Detailed records are in the per-experiment files and the memory
topic files; this is the index + verdict.

> **2026-07-01 CLOSURE-QUALITY AUDIT (`2026-07-01-method-revamp.md`) — corrections:**
> a closure below is only binding if it is POST-reset games evidence meeting the decision
> table. Re-graded: joint-SPSA = INVALID closure (iter-50 kill, param movement mathematically
> impossible, ~100 games); "LMR/NMP both directions regress" = PRE-RESET evidence (corrupted
> instrument), thresholds never game-tested post-reset; butterfly history = cold-proxy
> false-reject candidate (manifest admits under-credit); corrhist = closed against a
> MIS-PLUMBED base (correction never applied at qsearch stand-pat; learning target corrupted
> by S7 TT-refinement) — reopened as T4. Draw scaling below is STALE: it is LIVE in HEAD
> (eval.go:2684 drawFactor + 50-move damp, tests in eval_ocb_test.go).

## The pattern, in one line

**Every Elo gain in this engine's history has been a CORRECTNESS fix. Every speculative
eval / search-heuristic / tuning play has been null or negative.** That is the plateau.
**[FALSIFIED 2026-07-02/03 by T4 +9.1, T4np +6.4, T1b +13.1 (post-revamp keeps) — retain this
headline as a description of the PRE-revamp era ONLY.]**

Strength trajectory (CCRL, real-clock): 2380 → 2449 → 2632 → 2669 → **2666** [2627,2706] (multi-family, 2026-06-28)
→ **2689** [2649,2729] (2026-07-08 re-pin, +23 point-signal inside the ±40 delta floor).
The climbs were correctness fixes + the time-mgmt fix. Flat Jun-10→Jul-1 despite weeks of eval/search work;
climbing since the 2026-07-01 revamp (~+28-30 self-play kept).

---

## ✅ WHAT ACTUALLY MOVED THE NUMBER (kept)

| Change | Effect | Class |
|---|---|---|
| SEE evaluation was garbage → fixed | **+161 Elo** (ddcf82f) | correctness |
| NMP was OFF at every cut node (C3) → fixed | **+138 Elo** (f04eb0d, batched S6/S7/RT1) | correctness |
| Clock-interrupt committed half-searched moves → fixed | blunder 9.7%→3.2%, tactical 29→30/30 (362fd24) | correctness |
| IID & singular extensions silently broken (re-entrancy + self-cutoff) → fixed | restored both | correctness |
| Time management: only ~25% of soft clock used → soft-stop projection fix + easy-move-shrink removal | 25%→57% usage, +1 depth, **+~37 Elo** measured | correctness-adjacent |
| King safety E2 (enemy attacker material) | **+18–20 Elo** | eval-correctness |
| 2026-06-28 HARD RESET: P0 measurement bugs (stopped-search state pollution, qsearch-in-check, UCI stop race, Hash resize, SearchFixed authority, root-rep, EP-hash, promotion safety) | strength FLAT but instrument now trustworthy | correctness/measurement |
| Mobility per-count TABLES (C1) | kept (universal 10/10 in research) | eval shape |
| TT de-padded to 16B (was 64B, 48B wasted) | node-identical, 4× slot count | speed |
| TT make-move prefetch | **+0.8% NPS**, node-identical | speed |
| BCE-ordering pass on hot loop | **+1.0% NPS**, node-identical | speed |
| passersOf + pawn-shield micro-opts | ~+6.5% node-identical (early) | speed |

Takeaway: the only repeatable wins are (a) fixing things that were broken, and (b) tiny node-identical speedups in the single hottest loop.

---

## ❌ SEARCH heuristics / pruning / reductions — tried, did NOT move it

| Lever | Verdict |
|---|---|
| LMR continuous history scaling (replace crude ±1) | REJECT: /2048 flat (-8), /256 regressed (-45). `2026-06-28-lmr-conthist.md` |
| cutNode-based LMR reductions (extra reduction at cut nodes) | **REJECT -7.9** [-16,0] pLLR -2.87, fixed-nodes proxy 2984g (T3a value-1, `experiments/2026-07-02-t3a-lmr-cutnode.md`), post-reset **BINDING**. Blocked on move-ordering: NGN's FMC ~84% means MORE cutNode reduction over-reduces mis-ordered moves (silent fail-lows). REOPEN only after an ordering/FMC fix (T10); value 2/3 NOT retried (over-reduction isn't cured by more reduction). The [S] +6.2/+10 prior assumed elite ordering. |
| LMR / NMP single-axis retunes (both directions) | 3 SPRTs / 0 gains — both directions regress (well-tuned local optimum). **[pre-reset — NON-BINDING, reopened]** |
| Joint-SPSA over all 11 margin params (coordinated) | NO-OP: zero gradient at iter 50, params pinned. `2026-06-28-joint-spsa.md`. **[INVALID closure (iter-50 kill) — reopened as T8]** |
| Search-param SPSA (earlier) | CLOSED (flat). **[reopened / superseded by T8]** |
| J1 extension×LMR reduced-scout base fix | -59 Elo + endgame tree +83% → REVERTED (docs/09 J1 = accepted non-fix) |
| Dynamic NMP | REJECT. **(pre-reset evidence non-binding; adjacent to the queued T12 NMP polish)** |
| Multicut (return-beta) | REJECT (the A3 lesson). **[pre-reset REJECT; reopened as T9, prior +5.8 [S]]** |
| Deeper futility (margin+depth) | blew up endgame tree +141% |
| RFP / futility / LMP as the "endgame tree" fix | every standard pruning knob empirically refuted; eg-tree gap was a mislabeled-position artifact anyway |
| TT capacity as endgame fix | bites the MIDDLEGAME (-46%), not endgame |

Mechanism (2026-06-29 diagnostic): NGN's LMR re-search rate is ~1.1% (reductions ultra-conservative,
huge headroom) but every "reduce/prune more" regressed — because the re-search only catches moves that
BEAT alpha, so over-reduced good moves fail low SILENTLY. The gate is move-ordering precision (FMC ~84%,
not the ~90% of strong engines). See `2026-06-29` tree-shape measurement below.

---

## ❌ EVAL — tried, did NOT move it

| Lever | Verdict |
|---|---|
| Texel tuning, 10 untuned aux weights | NO-OP (±1-6cp, basis already at optimum) |
| Texel tuning, 787-param SF-d16 PST | NO-OP (proxy-artifact, never beat games) |
| Texel tuning, 1000-param | null |
| Linear texel tune, L2=0 | DEAD BY GAMES (= -440, commit e3e56f6) |
| Material weight tune (E1) | null |
| Eval optimism (reads +80 while SF says -320; ~160cp off SF) | STRUCTURAL/concentrated, outside the linear basis — texel CANNOT fix it |
| King-safety over-optimism fix (-38cp via zzbias) | gate-1 passed; SPRT/gauntlet never closed |
| Passed-pawn under-parameterization (flat-20, no rank knob) | identified as structural-add candidate; not confirmed by games |

Loss autopsy (30 losses vs Counter): 28 positional (slow-bleed + eval-blindspot), 1 tactical — looked
like "eval is the lever," but the 2026-06-28 search-bound proof reframed these as HORIZON errors (shallow
search trusting an optimistic static eval), i.e. downstream of search depth, not texel-fixable.

---

## CORRECTION HISTORY

| Table | Verdict |
|---|---|
| Pawn correction history | KEPT (in HEAD; the one corrhist that holds) |
| Pawn corrhist RE-PLUMB (T4: apply correction at qsearch stand-pat + learn from the pre-S7 CORRECTED static, gravity form) | **KEPT +9.1 [+0,+18] penta, H1 ACCEPTED** — real-clock 10+0.1 c8, pLLR +2.80, 0/0 flag-outs / 2325g (`2026-07-02-t4-corrhist-replumb.md`); FIRST outright SPRT accept of the campaign. The prior "corrhist closed" verdicts were all measured on the MIS-PLUMBED base (correction never applied at qsearch stand-pat; learn target corrupted by S7 TT-refinement). |
| Non-pawn corrhist RE-PLUMB (T4-nonpawn: the ORIGINAL @6245 form on the fixed T4 plumbing — same qsearch stand-pat application + pre-S7 corrected learn target) | **KEPT +6.4 [-1,+14] penta, H1 ACCEPTED** — real-clock 10+0.1 c8, pLLR crossed +2.94 at G3796 (peak +2.97, four lines ≥ bound, drain to +2.80), 0/0 flag-outs / 3804g (`2026-07-02-t4-nonpawn.md`). SAME @6245 form the two rows below rejected → the plumbing was the confound, not the corrector. |
| Aux corrhist (C-vs-B, 2026-06-23) | REJECT — and the run was INVALIDATED mid-flight by a ClearHistoryTable cross-game leak (Codex caught it), since fixed |
| Non-pawn corrhist @ weight 6245 (OLD, mis-plumbed base) | ~~REJECT -9.5 [-20,+1], 1569g~~ **SUPERSEDED** by the T4-nonpawn KEEP above: identical @6245 form on FIXED plumbing = **+6.4 H1**. `2026-06-28-nonpawn-corrhist.md` |
| Non-pawn corrhist @ weight 3000 (halved, mis-plumbed base) | ~~REJECT -3.8 [-11,+4], 3000g~~ **SUPERSEDED** — the halve was an over-correction remedy for the broken base; unnecessary on fixed plumbing (the @6245 form wins there) |
| Minor-piece corrhist | built + suite-green, never game-tested (parked when non-pawn shelved) — **T4c SPRT launched 2026-07-08, verdict pending** (corrhist family is 2-for-2 on fixed plumbing; same conventions + red→green test template) |

**Closure (2026-06-29): [SUPERSEDED — the 2026-07-02 T4 Update below overturns this.]** additive
eval-correction is DOWNSTREAM of the search fix. corrhist learns from `(search - static)`, so a
search-bound engine is a poor teacher → learns noise. Matches "conthist pays ≥3200".

**Update (2026-07-02, T4):** the 2026-06-29 closure was measured on a MIS-PLUMBED base (correction never
applied at qsearch stand-pat; learn target corrupted by S7 TT-refinement). Re-plumbed properly, PAWN
corrhist is **+9.1 H1** and NON-PAWN corrhist (identical @6245 form, STRICT H1-only gate) is **+6.4 H1 /
3804g** — the corrhist family is now **2-for-2 on fixed plumbing**. The "corrhist is negative / downstream
of search / poor teacher" belief was a PLUMBING CONFOUND, not a property of the corrector: the same @6245
non-pawn code that lost -9.5/-3.8 on the broken base WINS +6.4 on the fixed one. This **CLOSES the "non-pawn
corrhist is negative" lane** (it is positive) and de-risks the family — minor-piece corrhist (**T4c**) is
next, same conventions + red→green test template. The remaining caution: the family stacks on ONE eval, so
watch for diminishing returns / interaction when T4c lands (certify via games, not by assuming additivity).

---

## MOVE ORDERING

| Lever | Verdict |
|---|---|
| Full history machinery (main + continuation + followup + capture history, killers, counters) | present, mature |
| BCE-ordering pass | KEPT (+1.0% NPS) |
| qmvv ordering | +22.6 (early keep) |
| Staged / lazy quiet-move history scoring | REJECT (not node-identical) |
| MVV/LVA "−129" folklore | VOIDED |

**2026-06-29 measurement: FMC ~84% (mid 82%, eg 86%, tac 85%) — strong engines hit ~90%+.** This is the
identified next lever (the ordering precision gate, FMC as fast proxy). NOT yet attempted.

---

## PER-NODE SPEED (node-identical lane — "be the fastest Go engine")

Wins: BCE +1.0%, TT prefetch +0.8%, TT de-pad, passersOf/pawn-shield. **Lane is THOROUGHLY TAPPED.**

| Dud | Verdict |
|---|---|
| int16 history tables (4× smaller) | node-identical but NPS-FLAT — NGN is NOT memory-bound (96MB V-cache holds the 11MB tables) |
| eval-cache size sweep | bits=22 = -2.7% (locality), bits=18 = neutral → bits=20 already tuned. `speed-log.md` |
| int32 scores, qsearch-capScore-BCE, Move/Clear-masks, staged-scoring, pin-legality | all DUDS (added per-node work the OoO core already hides) |

Rule: a node-identical change pays ONLY in the single hottest tight loop with ~0 added instructions, or
when it hides real memory latency (prefetch). **Untapped:** LARGE PAGES (TLB, +5-10%, needs box admin / SeLockMemoryPrivilege).
"allocation-free hot path" was silently FALSE for ages (probcut) — fixed + locked by TestHotPathAllocationFree.

---

## TIME MANAGEMENT — KEPT (a real win)

Soft-stop projection fix + easy-move-shrink removal: clock usage 25%→57%, +1 depth, ~+37 Elo (transferred).
Counter still spends ~98% of clock vs NGN ~70-80% — a residual time-mgmt gap was identified but the 1-line
soft-stop projection refinement was STAGED, then CLOSED post-reset by T1a (below).

**2026-07-02 — the naive projection-drop 1-liner is CLOSED (post-reset, binding).** T1a fired the soft
stop at `elapsed >= soft` (spending the FULL soft budget vs ~2/3 under the projection): real-clock c8
SPRT, A/A-validated (penta -0.7 [-11,+9], 0/0 flag-outs) = **REJECT -7.8 [-15,-0], pLLR -3.22, 0/0
flag-outs / 2864g** (`experiments/2026-07-02-t1a-softstop.md`). The projection is LOAD-BEARING: spending
more time per move regressed strength with ZERO flag-outs — a uniform overspend starves the moves that
matter (the removed easy-move-shrink failure mode). The +31/+29 [Stash v26] prior belongs to the
STABILITY-SCALED soft target (T1b: keep the projection, spend more only while the decision is volatile),
NOT this one-liner.

**2026-07-03 — T1b STABILITY-SCALED soft target is the win the +31 prior predicted. KEPT (a real Elo).**
Wired the dead `stableIters` READ (search.go already MAINTAINED it via `ReportCompletedIteration`; only
`shouldStopTournamentSearch` never read it) to SYMMETRICALLY scale the soft budget: `stabilitySoftFactor`
= `max(0.85, 1.30 − 0.10·stableIters)` — extend ×1.30 while the root best move is volatile, shrink ×0.85
once it has settled (crossover 1.0 at stableIters 3), applied at the soft-limit check only; the ×1
projection, hard ceiling, and emergency floor are UNTOUCHED. Real-clock 10+0.1 c8 adjudicated SPRT =
**+13.1 penta [+3,+23], pLLR crossed +2.94 at G2225 (peak +2.98, drain to +2.90), H1 ACCEPTED, 0/0
flag-outs / 2233g** (`experiments/2026-07-03-t1b-stability-scaling.md`). This is the
REDISTRIBUTE-not-uniform lever T1a's failure pointed to: keep the load-bearing projection, spend MORE only
on volatile (hard) moves and LESS on settled ones — distinct from the removed one-sided easy-move shrink
(shrink-only → the 25%-clock bug). Confirms the revamp-doc symmetric spec and the CounterGo three-regime
direction. Follow-on = **T1c** (score-drop / eval-direction regimes = the rest of the Stash-v26 +31 /
CounterGo three-regime package; needs a second signal — score tracking — beyond bestmove-stability). The
"every speculative search/eval play is null or negative" headline is now falsified three times over (T4
+9.1, T4-nonpawn +6.4, T1b +13.1) — all standard-technique transplants under the revamp method.

---

## TUNING METHODS — all exhausted **[SPSA rows reopened/superseded by T8, 2026-07-08; texel stays closed]**

- **Texel/linear eval tuning: CLOSED.** Multiple objectives × multiple param sets, all null/dead-by-games. Don't reopen without a NEW (non-linear / structural) basis.
- **Joint-SPSA over search margins: NO-OP** (zero gradient). The coordinated retune single SPRTs can't do — and it found nothing to do. **[INVALID closure (iter-50 kill) — reopened as T8]**
- **Search-param SPSA: CLOSED** (flat). **[reopened / superseded by T8]**

---

## ENDGAME

- "2.5–5× bigger endgame tree" — REFUTED as a mislabeled-position artifact (the anchor was tactically won). Quiet endgames are ~0.85× Blunder (smaller), evals agree. No quiet-eg gap exists.
- Mop-up mating gradient (KR-vs-K converts) — flat self-play SPRT, PARKED (git stash on main).
- Endgame draw-scaling — **LIVE in HEAD** (`eval.go:2684` `drawFactor` + 50-move damp, ported from CounterGo; the "absent / STAGED / never closed" note is STALE).

---

## NNUE — not attempted (correctly)

Gated ≥2800 (Bryan, emphatic). The search-bound proof says it's the wrong tool anyway: a search-bound
engine can't teach an eval network (poor-teacher problem). Pure-Go HCE proves classical 2800-2900
is reachable without it, and the **Go-HCE ceiling is far higher (VERIFIED 2026-07-08, CCRL 40/15 1CPU lists
dated 2026-07-04): the last pure-HCE Counter is 4.1 = 3157 (NNUE began at Counter 5.0, Nov 2023 — NOT 4.x);
Stash 37 (Apr 2025) = 3374, still pure HCE and actively developed → the classical ceiling is demonstrated at
≥3157.** Anchor-label caveat (VERIFIED 2026-07-08, CCRL 40/15 1CPU, list dated 2026-07-04): counter-3.8 CCRL = 3050
(`ratings.json`'s 2994 ≈ Counter 3.7 at 2997); `ratings.json` kept at 2994 for pin-to-pin comparability —
the absolute pin therefore reads ~11 Elo conservative (19.4% pool weight × ~56); do NOT change
`ratings.json` mid-campaign.

---

## INSTRUMENTS BUILT (the measurement scaffolding)

cmd/sprt (self-play SPRT, pentanomial), cmd/gauntlet (absolute CCRL, real-clock -tc), cmd/oracle + `make acpl`
(SF-MultiPV cp-loss eval pre-filter), cmd/texel, cmd/spsa, cmd/zzbias / lossxray / sfpv (eval-bias),
cmd/blunder-finder, scripts/h2h.sh (node-identity + NPS), the gaming-PC SPRT mill (9800X3D), the `sd_*`
EBF-attribution dump (debug on). Hard-won lesson: self-play SPRT gains are transfer-MIRAGES vs the gauntlet;
fixed-nodes/ACPL/MSE/param-movement are FILTERS, never verdicts; a verdict = real-clock GAMES of the actual binary.

---

## META-LESSONS (why the plateau is real)

1. The only +100s ever were correctness fixes. No speculative eval/search play has moved the number.
2. NGN already HAS all the modern machinery — it's at a tight 2666 local optimum where single-axis tweaks regress in BOTH directions.
3. PROVEN search-bound (+185 Elo from 2 plies). Eval has headroom; the binding constraint is search DEPTH/EBF.
4. 2026-06-29 mechanism: depth is blocked by move-ordering precision (FMC 84%, not 90%) gating safe reduction. That is the current spearhead, with FMC as a fast deterministic proxy.
5. The remaining depth work (richer LMR formula, singular double-extension) is delicate (anti-explosion guardrail, search.go:1821) → attended, not fire-and-forget.

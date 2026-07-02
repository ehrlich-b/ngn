# NGN — Everything We've Tried (consolidated ledger)

Last updated 2026-07-01. This is the single inventory of every lever attempted, with verdicts,
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

Strength trajectory (CCRL, real-clock): 2380 → 2449 → 2632 → 2669 → **2666** [2627,2706] (multi-family, 2026-06-28).
The climbs were correctness fixes + the time-mgmt fix. Flat since ~Jun 10 despite weeks of eval/search work.

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
| LMR / NMP single-axis retunes (both directions) | 3 SPRTs / 0 gains — both directions regress (well-tuned local optimum) |
| Joint-SPSA over all 11 margin params (coordinated) | NO-OP: zero gradient at iter 50, params pinned. `2026-06-28-joint-spsa.md` |
| Search-param SPSA (earlier) | CLOSED (flat) |
| J1 extension×LMR reduced-scout base fix | -59 Elo + endgame tree +83% → REVERTED (docs/09 J1 = accepted non-fix) |
| Dynamic NMP | REJECT |
| Multicut (return-beta) | REJECT (the A3 lesson) |
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
| Aux corrhist (C-vs-B, 2026-06-23) | REJECT — and the run was INVALIDATED mid-flight by a ClearHistoryTable cross-game leak (Codex caught it), since fixed |
| Non-pawn corrhist @ weight 6245 | REJECT -9.5 [-20,+1], 1569g (over-correction). `2026-06-28-nonpawn-corrhist.md` |
| Non-pawn corrhist @ weight 3000 (halved) | REJECT -3.8 [-11,+4], 3000g (still non-positive) |
| Minor-piece corrhist | built + suite-green, never game-tested (parked when non-pawn shelved) |

**Closure (2026-06-29):** additive eval-correction is DOWNSTREAM of the search fix. corrhist learns from
`(search - static)`, so a search-bound engine is a poor teacher → learns noise. Matches "conthist pays ≥3200".

**Update (2026-07-02, T4):** the 2026-06-29 closure was measured on a MIS-PLUMBED base (correction never
applied at qsearch stand-pat; learn target corrupted by S7 TT-refinement). Re-plumbed properly, PAWN
corrhist is **+9.1 H1** — so the "poor teacher / downstream of search" framing does NOT close the pawn
table. The NONPAWN retest is the remaining OPEN item, held to a STRICT gate (H1-only keep, NO
provisional-on-cap) because the -9.5/1569g and -3.8/3000g rejections were REAL post-reset games — just on
the old plumbing.

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

---

## TUNING METHODS — all exhausted

- **Texel/linear eval tuning: CLOSED.** Multiple objectives × multiple param sets, all null/dead-by-games. Don't reopen without a NEW (non-linear / structural) basis.
- **Joint-SPSA over search margins: NO-OP** (zero gradient). The coordinated retune single SPRTs can't do — and it found nothing to do.
- **Search-param SPSA: CLOSED** (flat).

---

## ENDGAME

- "2.5–5× bigger endgame tree" — REFUTED as a mislabeled-position artifact (the anchor was tactically won). Quiet endgames are ~0.85× Blunder (smaller), evals agree. No quiet-eg gap exists.
- Mop-up mating gradient (KR-vs-K converts) — flat self-play SPRT, PARKED (git stash on main).
- Endgame draw-scaling absent (scores dead-drawn endings as +1.5) — identified Counter gap, STAGED, never closed.

---

## NNUE — not attempted (correctly)

Gated ≥2800 (Bryan, emphatic). The search-bound proof says it's the wrong tool anyway: a search-bound
engine can't teach an eval network (poor-teacher problem). Counter 3.8 = 2994 pure-Go HCE proves classical
2800-2900 is reachable without it.

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

# NGN Chess Engine — Roadmap

> Reanchor = re-read README.md, this file, CLAUDE.md. This is the canonical strategic home. Deep-rewritten 2026-06-06 (prior 392-line session-log stack collapsed into git history + memory/).

## Where we are (2026-06-06)
- **Strength: ~2540 CCRL, SOFT.** Two clean anchors agree — Blunder 7.2.0→2549 (n=32), 7.4.0→2538 (n=60), up from pre-tune 2447. SOFT because both samples were forfeit-capped (harness storm, since fixed via per-game restart but NOT re-run). We're at/near the GM line, ~even with Blunder 7.4.0 (CCRL 2532). **HARDEN with a fixed-harness 100/anchor rerun (#33).**
- **Goal: 2800 — and HCE CAN reach it, no NNUE required.** Proof: Stockfish 11 ~3400 and **CounterGo (pure-*Go* HCE) ~3373**, both pre-NNUE ⇒ ~600 ELO of HCE headroom above us, same language, no net. NNUE is a SHORTCUT (+300-400), not a requirement; **"NO NNUE (yet)" stands** as Bryan's mandate call.
- HEAD = `83581b7` (3 node-identical speed wins over session-start `ac64ae3`: passersOf + pawn-shield + the SEE short-circuit). `build/ngn` = `build/ngn_base` = HEAD. Working tree clean (no SPRT running). Node-identity baselines unchanged: 1501684/989263/513667.

## Strategic map — CORRECTED 2026-06-06 pm (adversarial review; read before picking work)
**The "no single big lever / +20-at-a-time grind" framing was DEFEATIST and is RETRACTED.** Three siloed adversarial sub-agents (code / reasoning / strategy) showed it was wrong: there ARE large levers — I mistook *"my instruments can't find the lever"* for *"there is no lever."* The two inadequate instruments: (a) one-param-at-a-time short-TC SPRT, which is **structurally incapable of finding the JOINT param optimum** (params interact — this is WHY the 3 reduction sweeps read null, NOT proof they're tuned); (b) the **Texel tuner, broken 3 ways**, which made the eval re-tune (Blunder's single biggest eval lever, +60) look "closed" when it's just un-tooled.

Still-valid dead ends (retired earlier, unchanged): eg-search tree "2.5-5×" REFUTED (won-position artifact) [[project_endgame_tree_diagnostic]]; HCE CAN reach 2800 (CounterGo Go-HCE ~3373, SF11 ~3400 — ~600 ELO HCE headroom).

**Corrected lever ranking (adversarial, with grounded ELO — Agent-3 sourced from Blunder's own testing log + SF/SPSA data):**
 1. **JOINT SPSA over the whole search-param vector — +40-80 in ONE campaign.** The machinery I LACK. Tunes all margins+reductions+LMP+history+aspiration TOGETHER over thousands of games — finds the joint optimum the isolated SPRT can't. First step: expose the `search.go:14-26` consts as UCI `setoption`, then a Θ±cΔ SPSA driver reusing `cmd/sprt`+`internal/`. Validate the result vector on the gauntlet (self-play transfer-mirages). **This is lever #1.**
 2. **Structural per-node speed — REOPENED (my "leaf EXHAUSTED" was WRONG).** Agent-1 pprof-verified misses: **staged move generation ~10-18% NPS** (every node generates+SEE-scores ALL moves before trying the TT move that often cuts first — `search.go:1272/1294`); **lazy qsearch eval ~8-14%** (`qsearchLazyMargin=100000` pins it OFF, eval.go:2861 — full king-safety/threats/mobility at every qleaf; a sound bound or moving terms off the qleaf recovers it); **qsearch stalemate probe ~3-5%** (full movegen+make/unmake every quiet qnode, search.go:1854). ~+20-40 ELO of recoverable speed, mostly node-identical-gateable.
 3. **Eval QUALITY via a FIXED tuner — +30-60 (Blunder's biggest eval lever).** Follow the VALIDATED L1 diagnosis I'd been walking past (86/87 losses = eval ~150-220cp too optimistic, mg→eg). Re-tune isn't "closed-because-optimal" — it's blocked on #21 (3 tuner defects). Fix the tuner → joint re-tune; gate on the EXTERNAL SF/ACPL ruler + gauntlet, NOT self-play SPRT. Don't tune search margins blind while eval is miscalibrated (coupled system).
 - **NNUE — the BIGGEST single lever (+150-250 over a 2500 HCE base), RESERVED BY CHOICE.** Name it honestly: it's off the table because Bryan chose the HCE-in-Go constraint, NOT because it's unnecessary. If the goal ever becomes "2800 soonest by any means," this is the move. Go effort: weeks (SIMD via Go-asm/cgo + a trainer). Bryan's mandate call.

## SEE short-circuit — KEPT (`83581b7`, 2026-06-06 pm)
`seeNonLosingByMVV` (captured≥mover, non-promo, non-EP ⇒ SEE≥0) short-circuits the full swap at the 3 sign-test pruning sites (probcut/badcap/qsearch) = 12.6% of SEE calls. **Node-identity EXACT** (1501684/989263/513667, re-verified twice incl. by an adversarial auditor who proved the theorem against the real SEE impl), interleaved bench **median +0.7%**, no possible regression. I first REVERTED it ("NPS-neutral / fails a bench-faster gate") then waffled it to "your call" — both WRONG: that's the literal T1 KEEP-RULE fold Bryan was furious about. **Node-identical + no measured regression ⇒ KEEP, mechanically.** Done. (The "SEE-reuse" framing from the original plan was a red herring: only 1 of 4 sites was ever reusable, and no site actually reuses — the short-circuit is the right form.)

## Adversarial-review corrections (2026-06-06 pm — supersedes the killed "survey/+20-grind" framing)
The earlier same-day "everything structural is done / 2800 is a +20 grind / margin-tuning is lever #1" survey was DEFEATIST and is retracted (see the corrected Strategic Map above). What the 3 adversarial agents established:
- **Search IS correct + feature-complete** (docs/09, no >20-ELO bug) and **TT is de-padded** and **material+PST/pawns are incremental** — those facts hold. But "therefore nothing structural is left" was the wrong inference: qsearch still runs the FULL non-pawn eval every leaf, movegen is eager-not-staged, and the param space was never JOINTLY tuned.
- **"Reductions are tuned" is OVERSTATED** — 3 *isolated* sweeps came up null, which tests one-knob-in-isolation, NOT the joint optimum. Joint SPSA may still find gains there.
- **Margin-tuning-one-at-a-time = the tractable-knob trap** on a near-zero prior (sibling reductions went 0/3) AND tunes one half of an eval-coupled system. Wrong METHOD; right TARGET only inside a joint SPSA campaign. RFP-90 isolated sweep was launched then KILLED (early −89, and the method is wrong anyway).

## Active queue (corrected path — adversarial review 2026-06-06 pm)
0. **DONE this turn:** SEE short-circuit KEPT (`83581b7`). Speed-batch SPRT + RFP-90 sweep both KILLED (former noise-empty, latter wrong-method). Defeatist survey retracted; map corrected above.
1. **Lever #1 — JOINT SPSA tuner (build the missing instrument).** (a) Expose `search.go:14-26` margins/reductions as UCI `setoption` (1-2h, also enables rebuild-free A/B). (b) SPSA driver (Θ±cΔ perturbation + a/c schedule, reuse `cmd/sprt`+`internal/uci`+`internal/rating`, ~few hundred lines). (c) One joint campaign over the full vector @10+0.1, ~20-40k games. (d) Validate the result vector on the GAUNTLET (not self-play — transfer-mirages). Expected +40-80. This also re-opens the "reductions" question (isolated-null ≠ joint-null).
2. **Lever #2 (parallel, node-identical-gateable can't-hurt speed) — staged move generation.** Yield TT move first; generate+SEE-score captures only if no cutoff; quiets last. ~10-18% NPS, search.go:1272/1294. Then lazy-qsearch-eval (sound bound on `qsearchLazyMargin`) + drop/lazy the qsearch stalemate full-movegen. Gate node-identity where achievable, else short-TC SPRT.
3. **Lever #3 — fix the Texel tuner (#21), then a SOUND joint eval re-tune** targeting the validated L1 mg→eg over-optimism. Gate on the external SF/ACPL ruler + gauntlet. Blunder's biggest eval lever (+60).
4. **Harden the rating (#33):** fixed-harness gauntlet 100/anchor (7.2.0+7.4.0), overnight, after a lever lands.
- **NNUE:** the biggest single lever (+150-250), reserved by Bryan's HCE-in-Go choice. Reopen only on his call.

## The LOOP (serial; one SPRT lane; E-cores; never idle-wait)
1. Prep ONE candidate: edit + `gofmt` + `go build -o build/ngn_<v> .` + gates (faithfulness `TestEvalTraceReproducesEval`, symmetry, pawn-cache, tactical ≥40%, nodecheck as tree-sanity).
2. Launch short-TC SPRT (bg, NO inner `&`/echo — the harness auto-notifies):
   `./build/sprt -base build/ngn_base -new build/ngn_<v> -tc 10+0.1 -openings output/sprt_openings.txt -elo0 -5 -elo1 0 -alpha 0.05 -beta 0.05 -mingames 64 -maxgames 300 > output/sprt_<v>.log 2>&1`
3. WHILE LIVE — no-CPU work ONLY (design next candidate / read / tidy). NO builds, profiler, or 2nd game-process (corrupts the clock).
4. Verdict: **KEEP** (chess-correct + no measured regression, OR LLR ≥ +2.94) ⇒ commit single-sentence + `go build -o build/ngn .` + rebuild `build/ngn_base` + re-lock nodecheck. **REJECT** only on a measured regression (LLR ≤ −2.94) ⇒ shrink or `git checkout`. A consistently-negative trend at low N is a non-KEEP even if the CI clips 0 — kill ~G64, don't burn to maxgames.
5. BETWEEN SPRTs (heavy compute OK): profiler / nodecheck / gauntlet for the next step.

## Gates & discipline (STANDING — do not relax)
- **Short-TC self-play SPRT (10+0.1, real clock, conc forced 1, E-cores) is THE per-change gate.** The GAUNTLET is NEVER the per-change gate — only an occasional absolute CCRL check (Cadence B). [[time_mgmt_ruler]]
- **KEEP RULE:** clear improvement OR correctness + NO measured regression ⇒ KEEP (even within noise). REJECT only on BOTH non-correctness AND a concrete measured regression. [[feedback_confirm_before_dropping_findings]]
- **Node-identity gate** (`scripts/nodecheck.sh`, baselines `1501684 / 989263 / 513667` @ HEAD): eval changes legitimately shift counts; speed changes must reproduce EXACT. Re-lock after every kept change.
- **Contention:** real-clock games and heavy compute NEVER overlap. One SPRT at a time.
- **Commits:** single-sentence, no co-author, no emoji, LOCAL (push only when asked). Never P-cores (`-lowpower true` routes to E-cores).
- **Self-play SPRT gains are partly transfer-mirages** — confirm a batch with a gauntlet before claiming CCRL. [[sprt_instrument]]
- **DO NOT:** re-tune the existing eval basis (closed, regressed 3×); chase the eg-search tree (refuted); mine more eg-passer micro-terms; batch changes; reach for a single silver-bullet lever.

## Open tasks
- **search-param REDUCTIONS** (#34) — **CONCLUDED 2026-06-06: well-tuned, 3 SPRTs / 0 gains.** A (LMR offset 0→0.75, −135 @ G27) + D (LMR divisor 2.0→2.5 LESS, −16 @ G155) ⇒ `/2.0` optimal both directions; B (NMP R=3+depth/6, −23 @ G92) no gain. **DON'T re-test LMR/NMP reductions.** Untested *pruning* (futility/RFP/razor/LMP) is lower-EV; the "untapped lever" premise was too optimistic. [[project_search_param_lane]]
- **#21** — fix the 3 texel-tuner defects (quiet-filter, dedup, train/test split, external SF teacher) BEFORE any future re-tune; the phase-gate headline is already fixed (`a0e326c`).
- **#24** — per-node speed pass (node-identity + bench gated). **LEAF LANE EXHAUSTED 2026-06-06.** Banked ~6.5% (passersOf `6da47d4` + pawn-shield `f06a839`). SEE-reuse premise was 4× optimistic (only 1/4 sites reusable); redirected to a node-identical SEE short-circuit (`seeNonLosingByMVV`, 12.6% of calls) → **NPS-neutral on the interleaved bench → REVERTED** (`output/see_shortcircuit_parked.diff`). The remaining leaf wins are all sub-1%/sub-noise. Profiler map (`/tmp/cpu2.prof`): selectNextMove 10.9% (algorithmic, skip), scoreMovesIntoBuffer 4.8% (hoisted), staticExchangeEvaluation 8.3%cum (short-circuit was neutral), isInCheck 6.3%cum, LoadKey 4.3%, make/unmake 4.2%. **Next speed work is ARCHITECTURAL (the 1.5× gap), not leaf** — pursue via the eval-term cost/benefit audit (survey lever #2), not more micro-opts.
- **#33** — harden the soft-2540 (fixed-harness gauntlet 100/anchor; the harness storm is fixed via per-game restart).

## Parked / shipped this session
- **per-node speed (#24) — SHIPPED 2026-06-06:** passersOf factor-out (`6da47d4`, ~5%) + pawn-shield bit-scan (`f06a839`, ~1.5%), both node-identical (baselines unchanged) — ~6.5% NPS, the day's positive after 3 search-param rejects.
- **safe-checks eval-ADD — REJECTED** (SPRT leaned −77 @ G23, consistently below 0, killed). Reverted; full diff at `output/safechecks_parked.diff` (retry with shrunk weights is possible but low priority — king-attack quadratic likely already prices most king danger).
- **computeSideAtt king-guard — SHIPPED** (`ac64ae3`): fixed a latent panic on king-less malformed FENs that the threats commit introduced. Node-identical for every legal position.

## Tooling
- `./build/sprt` — short-TC self-play SPRT (THE gate). `scripts/nodecheck.sh [bin]` — node-identity. `scripts/h2h.sh` — NGN-vs-Blunder per-node×tree (⚠ DISREGARD its "quiet-end" row — won position). `./build/gauntlet` / `make gauntlet` — absolute CCRL (Cadence B). `cmd/oracle` (label/acpl), `cmd/sfpv`, `cmd/lossxray` — SF-ruler diagnostics (untracked; the transfer-tune toolchain if ever revived). `cmd/pruning-analysis` — eg/mg autopsy.
- Anchors: `opponents/blunder_{720,740,...}` via `make build-anchors`.

## Durable lessons (full history in git log + memory/)
- **L1 — the live eval diagnosis:** 86/87 losses are eval over-optimism ~150-220cp (NGN reads winning while SF has it lost), clustering in the mg→eg transition. The eval-ADD lane targets this; but the basis re-tune is closed and the easy ADDs are mined.
- **follow-the-diagnosis / suspect-the-experiment:** a string of refuted knobs ⇒ the PREMISE is wrong (eg-search proved this twice). Validate the position/ruler BEFORE testing fixes. Eval is the tractable-adjacent-task trap. [[feedback_follow_the_diagnosis]]
- **The gap, decomposed:** ~1.5× slower per node (engineering, recoverable — Blunder is also Go) × tree ~1× mid / ~0.85× quiet-eg (NGN often *smaller*). The old "2× tax / 2.5-5× eg tree" framing was thermal-NPS noise + a mislabeled position. [[project_speed_lane]]
- Bug history → [[project_bugs]]; tuner status → [[texel_pipeline]]; SPRT cautions → [[sprt_instrument]]; CCRL anchoring → [[project_elo_reality]].

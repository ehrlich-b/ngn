# NGN Chess Engine — Roadmap

> Reanchor = re-read README.md, this file, CLAUDE.md. This is the canonical strategic home. Deep-rewritten 2026-06-06 (prior 392-line session-log stack collapsed into git history + memory/).

## Where we are (2026-06-06)
- **Strength: ~2540 CCRL, SOFT.** Two clean anchors agree — Blunder 7.2.0→2549 (n=32), 7.4.0→2538 (n=60), up from pre-tune 2447. SOFT because both samples were forfeit-capped (harness storm, since fixed via per-game restart but NOT re-run). We're at/near the GM line, ~even with Blunder 7.4.0 (CCRL 2532). **HARDEN with a fixed-harness 100/anchor rerun (#33).**
- **Goal: 2800 — and HCE CAN reach it, no NNUE required.** Proof: Stockfish 11 ~3400 and **CounterGo (pure-*Go* HCE) ~3373**, both pre-NNUE ⇒ ~600 ELO of HCE headroom above us, same language, no net. NNUE is a SHORTCUT (+300-400), not a requirement; **"NO NNUE (yet)" stands** as Bryan's mandate call.
- HEAD = `ac64ae3`. `build/ngn` = `build/ngn_base` = clean baseline. Working tree clean.

## Strategic map — CORRECTED 2026-06-06 (read before picking work)
**There is no single big classical lever.** 2800 is a +20-at-a-time grind — the way CounterGo/Blunder climbed. TWO silver-bullet over-rotations were retired THIS session (eg-search → NNUE); stop reaching for one big lever.

Two dead ends, retired:
- **eg-search tree "2.5-5× / pruning collapse" — REFUTED (measurement artifact).** h2h's "quiet-end" anchor (`r7/...R7`) is tactically WON (1.Rxa8 wins a rook), so NGN's 6× tree there was over-searching a won position. On 4 genuinely-quiet balanced endgames NGN's tree is **0.43-1.45× Blunder's (geo-mean ~0.85×, often SMALLER)** with **agreeing evals** — no quiet-eg gap, no eg over-optimism. M2's "+4-7 ply shallower in eg" is likely the same artifact; re-validate on quiet positions before trusting. [[project_endgame_tree_diagnostic]]
- **"2800 needs NNUE" — RETRACTED.** HCE reaches 2800+ (above); NNUE became dominant because it's *better*, but 2800 was crossed by HCE a decade before NNUE existed (SF12, 2020). Blunder is HCE at every version (top ~2674-2700).

**Accessible classical levers, RANKED:**
 1. **Search-param tuning — VERIFIED UNTAPPED (the best lead).** NMP R=3 fixed, LMR = `log(d)·log(m)/2.0` bare (no tuned divisor/offset), futility 200 / RFP 120·d / razor 400 / LMP classic-Ethereal table — all round folklore defaults (search.go:14-26, 74), and **no search-tuning harness exists** (cmd/texel tunes EVAL only). Every strong HCE (SF/Ethereal/CounterGo) grinds these. CAVEAT: it's a GRIND — SPSA is sample-hungry, single-M4 is slow, short-TC self-play has transfer risk. **Cheap honest entry = ONE targeted SPRT on the LMR divisor (`/2.0` → sweep), not a campaign.**
 2. **Per-node speed — ~1.5× recoverable, DETERMINISTIC (can't hurt strength).** Node-identity + bench gated (#24). Easy wins banked; remaining is delicate (incremental-SEE bounced twice on node-identity). The always-progressable fallback — never regresses.
 3. **Eval-ADD — DIMINISHING (the +20 lane, ~80% mined).** Safe-checks just MISSED (leaned −77, killed). Remaining: pawn-storm→attack-units, rook-on-7th. ADD-not-RE-tune; KEEP RULE. Lower priority now.
 - **NNUE — the shortcut (+300-400), RESERVED.** Bryan's mandate call; not without his go.

## Active route
1. **Search-param lever (cheap entry):** ONE targeted LMR-divisor SPRT. Validates or kills the lane for one SPRT's cost.
2. **Per-node speed (#24):** the can't-hurt deterministic lane — run between/around SPRTs.
3. **Eval-ADD (pawn-storm→attack-units, then rook-on-7th):** only if the search lane stalls.
4. **Harden the rating (#33):** fixed-harness gauntlet 100/anchor (7.2.0+7.4.0), overnight slot (game-process — cannot overlap an SPRT).

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
- **search-param tuning** (NEW) — verify the lane via the LMR-divisor SPRT; if it moves, scope the cheap targeted grind (LMR divisor/offset, adaptive NMP R, futility/LMP margins), one axis per SPRT.
- **#21** — fix the 3 texel-tuner defects (quiet-filter, dedup, train/test split, external SF teacher) BEFORE any future re-tune; the phase-gate headline is already fixed (`a0e326c`).
- **#24** — per-node speed pass (node-identity + bench gated); incremental-SEE is the known-delicate candidate (subtle x-ray reveal-timing).
- **#33** — harden the soft-2540 (fixed-harness gauntlet 100/anchor; the harness storm is fixed via per-game restart).

## Parked / shipped this session
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

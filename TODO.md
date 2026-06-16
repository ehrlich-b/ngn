# NGN — Road to 3000 (no NNUE)

> Reanchor = re-read README.md, this file, CLAUDE.md. Canonical strategic home. Implementation
> detail lives in `docs/09` (search spec + joints) and `docs/11` (3000 census); research verdicts
> in the memory files (`project_counter_gaps`, `project_3000_mechanism`, `project_meta_plan_v2`).

## STATUS (2026-06-16)
- **Strength: ~2655 CCRL-Blitz-equiv [2606,2704]** — cloud gauntlet, 256g over 4 Blunder rungs at -tc 120+1 (Jun-14). Corroborated by 2669 [2523,2815] vs Blunder 7.4.0 (24g, Jun-10). Trend **2449 (Jun-2) → 2632 (Jun-8) → 2669 (Jun-10) → 2655 (Jun-14, flat)** — the post-Jun-10 grind banked sound keeps but ~0 measurable absolute elo (the +0.6 Tier-2 cert agrees). Still **single-family (Blunder)**; a multi-family pin (incl. Counter/Fruit at -tc) is owed. SF `UCI_Elo` slider reads ~+400 inflated — disregard.
- **Goal: 3000 CCRL, no NNUE.** Existence proofs (pure-Go / classical HCE): **Counter 3.8 = 2994** (now the top anchor), Demolito 3156, SF11-HCE ~3450. **NNUE floor = 2800** (Bryan emphatic; we are ~2655 ⇒ not on the table).
- **The gap is POSITIONAL EVAL (2026-06-16, 30-loss game autopsy vs Counter — `project_loss_autopsy_eval`).** Of 30 NGN losses (lossxray-classified): **28 are positional (17 SLOW-BLEED + 11 EVAL-BLINDSPOT), exactly 1 is a tactical/depth miss.** Every loss is long (mean 62 moves, ALL end in mate, ZERO quick losses); SF judges NGN clearly worse by move 8-22 in ~22/30, then NGN bleeds out over 100+ plies. NGN's eval is OPTIMISTICALLY BIASED (reads +80 while SF says −320). ⇒ **NGN loses by drifting into inferior MIDDLEGAMES it misjudges, NOT by getting out-searched.** This SUPERSEDES the 2026-06-15 code-diff's "gap = time + endgame + re-search" ranking: the post-Jun-10 grind was 100% search params — the wrong subsystem (why the gauntlet stayed flat + the cert evaporated). Lever = EVAL POSITIONAL JUDGMENT. (Caveat: single opponent / movetime, but the 28-vs-1 signal is strong and converges with the king-safety-optimism + meta-plan findings.) Counter still hits 2994 in ~5,200 lines vs NGN's ~25,600 ⇒ we do things wrong/redundantly, not miss features.

## THE PLAN — EVAL positional judgment first (2026-06-16 game-autopsy reorder)
**The 30-loss game autopsy (STATUS / `project_loss_autopsy_eval`) re-ranked everything: 28/30 losses are POSITIONAL, 1 is tactical ⇒ EVAL judgment is the spearhead; search/time are cleanup.** The post-Jun-10 grind was 100% search params — the wrong subsystem (why the gauntlet stayed flat). Two standing premise corrections from the 2026-06-15 code-diff REINFORCE this: (1) **move-ordering/EBF is NOT the gap** — at d14 NGN's tree is SMALLER than Counter's in tactical (0.33×)/middlegame (0.76×), bigger only in the quiet endgame (1.52×); (2) **~1,500 of eval.go's 3,017 lines are DEAD CODE**, the live eval is ~Counter-sized. Durable in `project_counter_gaps`.

**METHOD SHIFT (load-bearing):** gate eval-judgment fixes on DETERMINISTIC instruments — `cmd/zzbias` (cp self-bias), `lossxray` blindspot-count on a fresh Counter capture, `make acpl` — plus the Tier-3 gauntlet. **Do NOT gate these on the self-play SPRT mill: it is a proven false-positive machine at our game counts** (the +55.7 Tier-1 cert → +0.6; it cannot resolve effects this size and banks noise). The texel WEIGHT lane stays CLOSED (0-for-4); these are STRUCTURAL/correctness eval fixes, not weight tweaks.

Ranked levers (game-autopsy order):
1. **EVAL — kill the self-optimism bias (THE EVAL-BLINDSPOT root, 11/30 losses).** NGN reads equal/better while SF knows it's losing (autopsy L14 +80 vs SF −320). `project_king_safety_lever` localized it to +134cp@phase6-11 (king-safety ignored enemy attacker material); the first fix banked +43.7 but the autopsy shows optimism REMAINS. Audit every term scoring NGN's OWN side without a symmetric enemy weight. zzbias-gated (deterministic −cp = win).
2. **EVAL E4 — passer dynamics (7/30 losses involve passed pawns; 5 of the 11 blindspots).** NGN passers are penalty-only + a flat rank ramp (eval.go:2487); Counter has rule-of-the-square / enemy-king-race / free-path BONUSES (evaluation.go:321-363). ADD the missing chess-correct terms (allowed — a missing term, not a re-tune).
3. **EVAL E3 — threats on weak pawns:** threats never target weak PAWNS and are zeroed in EG; Counter's ThreatForPawn (rook/king attacks an undefended pawn) is EG-weighted.
4. **EVAL E2 — king-safety into the endgame:** kingSafetyWeightEG=84 (eval.go:2795) + a central-king penalty (eval.go:866) fight king ACTIVITY at phase 6-11; Counter king-safety is MG-only (Eg=0). zzbias-gated.
5. ✓ **EVAL E1 — endgame draw-scaling — KEPT +1.1/6195g (b685b7e).** Done; the only landed lever of the arc. End nodecheck re-locked 757720→507458.
6. **SEARCH S1 / TIME F2 — DEPRIORITIZED (only 1/30 losses is a tactical/depth miss; tree already ≤ Counter's).** Valid CLEANUP, not the strength lever. S1 = LMR re-search un-extension (search.go:1879, delicate/J1-adjacent); F2 = extend-on-instability (dead `stableIters`, time.go:256). F1 greedy-clock REJECTED −10.3 (time-spend lever CLOSED; ×1 projection ~tuned).
7. **SPEED (node-identical NPS):** cache working-set shrink (drop followupHistory, re-pack conthist, reconsider the eval cache, lower default Hash) — engineering, not strength-losing; opportunistic.

**IN FLIGHT / RECENT:**
- **F1 time-mgmt — REJECT −10.3/4080g (reverted).** **E1 endgame draw-scaling — KEPT +1.1/6195g (b685b7e).** **ACTIVE NEXT (post-autopsy) = lever #1: the self-optimism-bias eval audit** (the EVAL-BLINDSPOT root) — instrument with `cmd/zzbias` on the autopsy's loss positions, fix terms that score NGN's own side without symmetric enemy weight, gate on zzbias (−cp) + a fresh lossxray blindspot recount + gauntlet. **S1/F2 search/time DEPRIORITIZED by the game autopsy (only 1/30 losses is a tactical/depth miss).**
- **`cutNode` plumbing — DONE (commit 400ebfc, node-identical, nodecheck 293463/130764/757720).** The keystone; consumers below.
- **cut-node LMR — REJECT** −5.8 [−13.1,+1.4]/3988g (runid 260615-133109). Reverted. **IIR (cutNode, depth--) — NEUTRAL** −0.4/3757g, abandoned. Lesson: cutNode REDUCTIONS backfire on NGN's tree (see S1 — the re-search structure is why); don't keep trying them until S1 is fixed.

## OPERATING RULES (non-negotiable)
- **KEEP rule (canon):** improvement OR correctness fix + NO measured regression ⇒ KEEP, full stop — even if small/within noise. "Too small to resolve" is NOT a reject reason; trivial complexity/memory is NOT a cost. REJECT requires BOTH non-correctness AND a concrete measured regression.
- **Three-tier gating:** Tier-1 = per-change cloud SPRT (10+0.1, sign + KEEP rule; the FILTER, magnitudes are noise). Tier-2 = per-batch cert every 4-6 keeps, ≥2-3k games vs the pre-batch anchor (the VERDICT; a fail ⇒ bisect halves serially). Tier-3 = real-clock `gauntlet -tc 120+1` vs Blunder/Counter every ~2-3 batches (absolute pin).
- **Never verdict a mid-run or killed SPRT** — run to the cap or an LLR/penta bound (G68 lesson ×2).
- **`make acpl` before SPRT-ing any EVAL change** (external SF ruler; ACPL up ⇒ suspect, don't SPRT). **`scripts/nodecheck.sh` re-lock** after any kept search/eval change; speed-only changes must reproduce EXACTLY (current baseline 293463/130764/757720).
- **NO NNUE until ≥2800** (`feedback_no_nnue`). **One SPRT lane at a time; never P-cores; `caffeinate -i` for multi-hour local runs. Commits: single sentence, no co-author, local only — only AFTER a verdict** (worktree diff = candidate, HEAD = base).

## THE MILL (cloud SPRT) — how to run a verdict
`scripts/cloudsprt.sh launch -shards 4 -type c7g.4xlarge -ttl 2 -- -tc 10+0.1 -elo0 -3 -elo1 3 -mingames 200 -maxgames 2000` (pentanomial early-stop; ~$0.5-1, ~15-30 min).
- **Launch RAW + backgrounded, then POLL `cloudsprt status`** — it is the DEFINITIVE per-shard phase report (STARTING / PROGRESSING+upload-age / STALE=>LIKELY HUNG / DONE, all regions). **Do NOT judge hung-vs-working from the launch stdout — it has lied twice; `fetch`/`status` (S3) are truth.** Never `| tail` a launch ([[feedback_no_tail_on_long_commands]]).
- `scripts/cloudsprt.sh fetch -runid <id>` ONLY at zero-live (it verdicts + sweeps). `ps` after every fetch; **end every session with `ps` empty** (per-instance `-ttl` is the backstop).
- **HARNESS GUARDRAILS (load-bearing, CLAUDE.md):** never edit the shutdown-guarantee layers; ANY cloudsprt.sh edit needs `smoke` PASS before the next launch; never pool cloud+Mac or mix instance types; AWS AccessDenied/UnauthorizedOperation = a DESIGN boundary (never broaden IAM). **Budget $20/mo** — quote cost before each run ([[feedback_aws_budget]]).
- Harness state (2026-06-15): `status` rewritten (all-regions + upload-age heartbeat), `wait_ssh` heartbeats, launcher start-ssh now setsid+log-only (was false-terminating healthy workers). All smoke-certified. A Go orchestrator is still the eventual right home.

## CLOSED / SUPERSEDED — do not reopen without NEW evidence
- **EVAL WEIGHT re-tuning (texel) — 0-for-4.** threats=no texel signal (tactical term, quiet corpus); unrestricted-mobility=overfit; restricted-mobility=MSE-mirage (ACPL +4.5 worse); fixed-tuner re-tune=no-op. The hand-seed is near ACPL-optimal; texel-MSE pulls AWAY from play-strength. The ONLY eval-tune win ever = the PeSTO core +46 (virgin data). ⇒ eval lever = STRUCTURAL/correctness (E1-E4 above), NOT weight tweaks.
- **EBF "3.0→2.0, +150-250 elo" — MIRAGE/DOWNGRADED.** The 5-agent measurement shows NGN's tree is competitive-to-smaller vs Counter except the endgame. The old "tree is ~3× bloated" reads were single-position artifacts / Blunder-anchored.
- **cutNode REDUCTIONS (cut-node LMR, IIR) — both ~reject/neutral** because NGN's eager LMR re-search (search.go:1879) turns added reductions into re-searches. Fix S1 FIRST, then revisit.
- **Broad pruning EXTENSIONS hurt NGN's sharp search** — W9 aspiration-widen / W6 !isPV-IIR / W2 deeper-quiet-SEE all filter-stopped on sharp-position inflation. The lever is "prune more PRECISELY" (cut-node-conditioned), not "more widely." Clean one-liner pruning extensions are EXHAUSTED.
- **Search-param SPSA — flat** (LMR/NMP reductions well-tuned; 3 SPRTs/0 gains both directions).
- **ADJ (early adjudication default-on) — REJECT** (+45 elo null-corrupting bias; leave `-adjudicate` OFF).
- **statScore-LMR K=2048 — neutral** (needs SPSA-tuned K, not single guesses). **J1 off-PV extension — left dead** (the "fix" bloated the endgame tree +83%; docs/09 J1 = accepted non-fix).

## THE CLIMB SO FAR (kept changes; git is the archive)
- **Correctness fixes = every big win:** half-searched-move-on-interrupt (blunder rate 9.7→3.2%), SEE garbage +161, NMP-was-off-at-cut-nodes +138 (C3), IID/singular re-entrancy, D2 plies-vs-moves time bug +32.5 (Weiss-twin), king-safety enemy-attacker-material.
- **Search keeps:** root PVS+LMR +29 (5a90a9e), A1 LMR-threshold / qmvv qsearch-ordering / A2a improving-RFP (+62 self-play night), W1 futility d3→8, W11 LMP re-bracket + d8-row, singular-neg-ext, NMP-depth-R, O4 killers-by-ply, A9 recapture-ext removal (simplification).
- **Eval keeps:** PeSTO core +46, C1 mobility per-count un-freeze +22.5, king-safety bias fix.
- **Tier-2 cert #1 PASS** (anchor dfde55f→7b53744; the +55.7 Tier-1 self-play SUM certified at +0.6 [-7.6,+8.8]/3877g ⇒ Tier-1 magnitudes are noise, trust the SIGN, the cert owns the truth).
- **This session (2026-06-15):** harness status fix (aea64f7), cutNode plumbing (400ebfc), launcher false-negative fix (402ed19).

## INSTRUMENTS & REFERENCE
- **Absolute rating:** `cmd/gauntlet` real-clock -tc vs Blunder/Counter ladder; cloud via `cloudsprt gauntlet`/`gfetch` (~30 min, 4 boxes). The gold standard. smoke/elo-assess are RELATIVE only.
- **Per-change:** `cmd/sprt` self-play SPRT (the mill). `make acpl` external SF ruler (eval). `scripts/nodecheck.sh` node-identity. `scripts/ebfprobe.py` tree-shape filter. `cmd/zzbias`/`lossxray`/`sfpv` eval-bias. `cmd/blunder-finder` loss diagnosis.
- **Anchors:** `scripts/build-anchors.sh` builds the Blunder ladder + Counter 3.8 (b172b99) native + linux/arm64. Counter source for diffing: /tmp/CounterGo.
- **Specs:** docs/09 (search invariants + J1-J15 joints), docs/11 (3000 census). `output/` is gitignored ⇒ record runs/verdicts here or in memory at verdict time.

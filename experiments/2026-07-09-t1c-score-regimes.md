# 2026-07-09 T1c — soft-budget score-drop / eval-direction scaling (time-management, real-clock gate)

Pre-registered BEFORE launch (CLAUDE.md work-loop step 5). **PREP ONLY at time of writing** — the patch
is in the working tree UNCOMMITTED and NOT launched; diff backed up at `output/t1c.patch`. One isolated
behavior change (the score-drop regime only), candidate vs its immediate base (current HEAD `7f85167`,
engine byte-identical to `a195717` == `ngn_t4c.exe`), **real-clock ONLY** (time/TC changes gate on
real-clock games, never a proxy — CLAUDE.md instrument-match rule). Awaiting team-lead GO before launch.

```yaml
id: 2026-07-09-t1c-score-regimes
date: 2026-07-09
change_class: search/eval heuristic (time-management) — real-clock-ONLY gate
hypothesis: >
  adding a SECOND, orthogonal time signal beyond T1b's best-move-stability — the eval-DIRECTION
  regime of the Stash-v26 / CounterGo three-regime package — extends the soft budget when the root
  score is FALLING across completed iterations (the position is going wrong), so NGN thinks longer on
  exactly the slow-bleed/horizon-optimism moves the loss autopsy flagged (28/30 losses positional
  slow-bleeds). Composed multiplicatively with T1b's stability factor, extend-only, bounded, projection
  and hard ceiling untouched.
base_commit: 7f85167 (current HEAD; engine byte-identical to a195717 == ngn_t4c.exe — git diff a195717..HEAD -- engine/ is empty, and a clean-HEAD windows cross-compile reproduced 40783f04… byte-for-byte)
candidate_commit: 7f85167 + output/t1c.patch (engine/time.go + engine/time_test.go + engine/search.go only)
base_binary: ngn_t4c.exe sha256 40783f0444ed47cf50bf9ca0dfa5d7635536e1a1519fe47c902498665141a1f8 (already staged on box; == current HEAD engine — clean-HEAD rebuild in this env reproduced this exact hash, so base and candidate differ only by output/t1c.patch)
candidate_binary: ngn_t1c.exe sha256 0dc9d15aad819d6a2b2f513be1aa6174d8b5e6265951cb1ed73f9d5269b925da (cross-compiled 7f85167 + output/t1c.patch, GOOS=windows GOARCH=amd64 GOAMD64=v3, go1.26.2, no ldflags/trimpath — same toolchain/flags that reproduced the base byte-for-byte)
patch_sha256: f3e6cfd1cea75b3b5351bd95a353e9257f8315bad46daac21d7a7fc72fc07aed (output/t1c.patch)
harness_commit: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-aware validated mill harness, HEAD 13560dc build — untouched)
mechanism: >
  engine/time.go shouldStopTournamentSearch soft-limit: soft := softTime * stabilitySoftFactor() *
  scoreDropSoftFactor() (T1b's stabilitySoftFactor term UNCHANGED; scoreDropSoftFactor is the new
  second factor). scoreDropSoftFactor() is a clamped linear ramp on scoreDrop (the cp the root score
  fell since the previous completed iteration, maintained by ReportCompletedIteration): 1.0 for drops
  <= scoreDropTrigger (50cp), ramping to scoreDropSoftMax (1.25) at scoreDropFull (150cp), extend-only.
  scoreDrop is threaded through ReportCompletedIteration(bestMoveChanged, windowHeldFirstTry, score),
  gated to 0 on iteration 1 (no previous score) and whenever either endpoint is a mate score. The ×1
  next-iteration projection, the hard ceiling (<=30% bank), and the emergency floor are UNTOUCHED.
node_identity: >
  NOT node-identical for real-clock play (behavioral time-allocation change by design) — hence the
  real-clock gate. BUT fixed-depth node counts are UNMOVED: scripts/nodecheck.sh on the candidate
  reproduced the a195717 baselines EXACTLY (kiwipete d12 230036, mid d12 181982, end d16 587557). The
  score tracking in ReportCompletedIteration is pure TimeManager bookkeeping and scoreDropSoftFactor is
  read only under Tournament time control (ShouldStopSearch returns false for FixedDepth), so the search
  tree is unchanged at fixed depth. nodecheck.sh itself was NOT edited.
regression_test: >
  engine/time_test.go TestScoreDropSoftStop (new): (1) score-tracking — iteration 1 has no previous
  score so drop == 0; 40 -> -80 is a +120cp drop; -80 -> 20 is a -100cp (improving) drop; a mate score
  in either endpoint gates the drop to 0. (2) factor mapping — 1.0 at/below the 50cp trigger, caps at
  1.25 past the 150cp full point, strictly between mid-range. (3) factor application — with stability
  held neutral (stableIters=3 -> stabilitySoftFactor 1.0), a stable-eval search STOPS at the base soft
  budget (4000ms) while a full-drop search EXTENDS past it; and the hard ceiling still stops regardless
  of the drop. RED->GREEN proven by removing only the `* scoreDropSoftFactor()` multiply: the "dropping
  search must extend at 4000ms" assertion FAILS pre-patch (time_test.go:290) and PASSES post-patch.
  TestTournamentSoftStop updated for the 3-arg signature (score 0) plus an explicit tm.scoreDrop=0 to
  isolate the T1b assertions (score factor 1.0) — T1b's own assertions are unchanged and still pass.
  Full go test -short [-race] ./engine + -short ./... GREEN.
openings: canonical sprt_openings (5000 lines) sha256 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
```

## Hypothesis and donor prior

T1b banked the best-move-**stability** sub-signal of the time-management package (+13.1 penta H1). The
Stash v26 "real time management system (bestmove type / eval direction / bestmove stability)" =
**+31/+29 LTC** [S] was the WHOLE package; the CounterGo 2994 peer runs a three-regime `difficulty`
manager (score drop >50cp → 2.0; best move changed → max(1.5, d); stable → max(0.95, 0.9·d)). T1b
covered the second and third regimes (best-move change extends via stableIters=0 → factor 1.30, stable
shrinks to 0.85). **T1c completes the package by adding the first regime — the eval-DIRECTION / score-drop
signal — the only one T1b did not touch.**

**Calibration (expectation rule, honest):** the +31 headline was the whole package and T1b already banked
+13.1, so T1c is the residual — **mid-single-digit to low-double at best**, plausibly smaller. Ethereal's
score-drop scaling component alone is ~+2.5 [E]. Expect a small gain if positive; this is not a +31 lever.

## Mechanism (the single change — score-drop regime only)

`engine/time.go`:

- **New `scoreDropSoftFactor()` + constants** (after `stabilitySoftFactor`): a clamped linear ramp on
  `scoreDrop` — `1.0` for drops ≤ `scoreDropTrigger` (50cp, the aspiration-window noise floor: the
  window is 50cp so smaller iteration-to-iteration moves are noise), ramping to `scoreDropSoftMax` (1.25)
  at `scoreDropFull` (150cp). **Extend-only** (returns ≥ 1.0 always) — the shrink side belongs to T1b's
  stability floor; adding a score-based shrink would be a second behavior change.
- **Score threading through `ReportCompletedIteration`** — signature gains a `score int` param
  (`iterationBestScore` at the sole caller, search.go:899). It computes `scoreDrop = prevIterScore −
  score` (positive == worsening), then records `score` as the new baseline. **Edge cases, explicit:**
  `hasPrevScore` guards iteration 1 (undefined delta → drop 0); either endpoint being a mate score
  (`|score| ≥ MATE_IN_MAX`) gates the drop to 0 (cp arithmetic across mate scores is meaningless). The
  stableIters update (T1b's term) is unchanged and runs after.
- **Soft-limit application** in `shouldStopTournamentSearch`: `soft := softTime * stabilitySoftFactor()
  * scoreDropSoftFactor()` — T1b's factor is UNCHANGED, the score factor is a second multiplicative term.
  The `lastIterationTime==0` first-iteration case and the `elapsed+lastIterationTime > soft` projection
  are UNTOUCHED.
- **New TimeManager fields** `prevIterScore`, `scoreDrop`, `hasPrevScore`, reset per move in
  `SetTimeControl` alongside `stableIters`.

## Design decisions (donor-shaped, bounded)

1. **Multiplicative composition with T1b (sanctioned).** Stash (the +31 donor) multiplies its sub-scales;
   the task authorizes "a second multiplicative factor." T1b's stability term is byte-for-byte unchanged.
2. **Extend-only.** CounterGo's shrink is the "stable" regime (T1b's job); the score regime only extends.
   This makes the key value-add explicit: a position where the best move stays put but its eval is
   crashing (a slow bleed) is shrunk by stability alone but now EXTENDED by the score signal — exactly
   the horizon-optimism loss class. Combined worst case (settled + crashing) = 0.85 × 1.25 = 1.06.
3. **Bounded / flag-safe.** Both factors are capped (stability ≤ 1.30, score ≤ 1.25); combined max
   1.30 × 1.25 = 1.625·soft — still far below the hard ceiling (min(4·soft, 0.3·bank)), which is checked
   FIRST and is untouched. With the ×1 projection the effective stop is ≈ 0.67·1.625·soft ≈ 1.09·soft on
   the rare crashing+volatile move — above the base soft budget (the intended redistribution) but nowhere
   near a flag. T1a proved UNIFORM full-soft spend costs −7.8; T1c is targeted, not uniform.
4. **Conservative first-candidate constants (peer-anchored, real-clock-tunable).** Trigger 50cp is
   CounterGo's threshold and the aspiration-window width; cap 1.25 sits between Ethereal-conservative and
   CounterGo's aggressive 2.0. Named constants, trivially exposable as SPSA dims later (T8). If the SPRT
   regresses, the failure signature picks the next candidate: flat/small loss with clean clocks ⇒ trim
   the cap (1.25→1.15) or raise the trigger; a clock-usage spike or flag-outs ⇒ this lever is closed.

**Known scope limit (donor-minimal, intentional).** `scoreDrop` is a SINGLE-iteration delta
(`prevIterScore − score`, exactly one completed iteration back), matching the CounterGo/Stash donor form
which recomputes against the immediately preceding iteration. Consequence: a *slow* bleed — e.g. the eval
sliding 30cp/iteration across several depths — never crosses the 50cp single-step trigger, so it does NOT
fire, even though the cumulative drift is large. Catching cumulative multi-iteration drift would need a
reference window (best-since-N, or a smoothed baseline) and is deliberately OUT of scope here — the
window and the trigger/cap/full constants are future SPSA dims (T8), not part of this verdict. T1c tests
only the donor-minimal single-step regime; that keeps this a clean one-signal experiment.

## Decision rule (predeclared, real-clock ONLY, standard batch-certification)

Real-clock 10+0.1 c8 SPRT vs `ngn_t4c.exe` (= current HEAD engine), elo0=−3 elo1=+3, alpha=beta=0.05
(pLLR bounds ±2.94), **adjudication ON** (validated non-distorting by the 2026-07-03 M5-enable A/A), on
the A/A-validated c8 config, maxgames 8000 mingames 300. Verbatim:

- **pLLR ≥ +2.94** → KEEP outright (H1 accept).
- **pLLR ≤ −2.94** → REJECT.
- **Capped-positive** at the 8000g cap (penta point est ≥ +1, pLLR > 0, no regression signal) →
  PROVISIONAL keep under batch certification (batch #2 slot).
- **Capped-nonpositive** → SHELVE with reopen condition.
- **Any candidate flag-out spike above the A/A baseline (>2 either side)** → HALT and investigate before
  verdicting.

## Harness / command / machine

Adjudication flags are the M5-validated standard set (`-resignscore 900 -resignplies 5 -drawscore 10
-drawplies 10 -drawminplies 80`; M5-enable A/A PASS 2026-07-03: base-vs-base penta −1.1 [−13,+11], 0/0
flag-outs, 77.6% adjudicated, 961 vs 932 g/hr, `experiments/2026-07-03-m5-adjudication-aa.md`).

```yaml
command: sprt.exe -new .\ngn_t1c.exe -base .\ngn_t4c.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
go_version: go1.26.2 (local cross-compile)
goarch_goamd64: windows/amd64 GOAMD64=v3
tc: 10+0.1 (seconds; bullet)
concurrency: 8
adjudication: ON (-resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80)
maxgames: 8000
mingames: 300
aa_preflight: >
  the 2026-07-03 M5-enable A/A (m5aa_out.txt, ngn_base4 vs ngn_base4 at THIS exact TC/concurrency/machine
  with adjudication ON) — penta -1.1 [-13,+11], 0/0 flag-outs, DONE_EXIT_0. Config (TC/concurrency/
  adjudication) is unchanged from the KEPT T1b/T4c runs on this box, so the M5 A/A is a valid preflight;
  a fresh A/A is not required (same as T1b/T4c cited it).
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
```

Command matches the KEPT T1b/T4c batches exactly except the run name and the two binary names. `.\` on
exe paths is the Go ErrDot dodge; the openings path is bare. Expected duration ~2.5-4h at c8 (crossing
early) up to ~7h at the full 8000g cap.

## Baselines (pre-launch, this env)

- `go test -short ./engine -count=1` — GREEN (4.4s).
- `go test -short -race ./engine -count=1` — GREEN (21.2s).
- `go test -short ./... -count=1` — GREEN (engine + cmd/sprt + internal/uci all ok).
- Red→green verified (remove the `* scoreDropSoftFactor()` multiply → TestScoreDropSoftStop fails at the
  extend assertion; restored).
- `scripts/nodecheck.sh build/ngn` (candidate) = **230036 / 181982 / 587557**, EXACTLY the a195717
  baselines — fixed-depth tree UNMOVED, nodecheck.sh unedited.
- Diff is exactly `engine/time.go` + `engine/time_test.go` + `engine/search.go` (158 insertions / 18
  deletions).

## Status

**LAUNCHED 2026-07-09 (box 6:29:54 PM) — real-clock c8 SPRT live on the box, awaiting verdict.** Patch
reviewed + GO'd by team-lead (within all constraints: T1b term byte-identical, extend-only, bounded
1.625× worst case under a 4× ceiling checked first, score tracking precedes the volatility early-return,
mate/iter-1 gated, nodecheck unmoved). Still UNCOMMITTED (`output/t1c.patch`, engine/time.go +
engine/time_test.go + engine/search.go; NOT committed until its verdict per the speculative-change rule).
This run is watched for completion to apply the decision rule and append the Verdict section.

## Launch (2026-07-09)

Launched via `scripts/boxsprt.sh launch t1c` on the 9800X3D LAN box (192.168.4.108, native Windows) →
`t1c_out.txt`.

```yaml
launched: 2026-07-09, box time 6:29:54 PM (all 17 procs StartTime 6:29:54 PM)
output_file: t1c_out.txt (box C:\Users\ehrli\ngn\sprt\)
command: sprt.exe -new .\ngn_t1c.exe -base .\ngn_t4c.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
harness_commit: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-aware validated mill harness, remote-verified) + scripts/boxsprt.sh (unchanged during the run)
base_binary_sha256: 40783f0444ed47cf50bf9ca0dfa5d7635536e1a1519fe47c902498665141a1f8 (ngn_t4c.exe, remote-verified 40783F04…; == current HEAD engine)
candidate_binary_sha256: 0dc9d15aad819d6a2b2f513be1aa6174d8b5e6265951cb1ed73f9d5269b925da (ngn_t1c.exe, remote-verified 0DC9D15A…)
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: ON (resign>=900cp/5p, draw<=10cp/10p past move 40 [80 plies])
eta: ~2.5-4h at c8 if it crosses early (T1b crossed at 2h18m/2233g); ~7h to the 8000g cap
```

**Liftoff evidence (clean):**
- Remote SHA-256s CONFIRMED == local before launch: ngn_t1c.exe 0DC9D15A…, ngn_t4c.exe 40783F04…,
  sprt.exe 30C33E05…. Box `ps` was empty pre-launch (idle).
- `ps` post-launch: **1 sprt + 8 ngn_t4c + 8 ngn_t1c** (17 procs, c8×2+harness), all StartTime 6:29:54 PM,
  CPU accumulating (~5-6 CPU each = actively searching).
- Header parsed correctly: `mode: real clock 10s+0.1s (concurrency 8) | openings: 5000 (x2 colors)`;
  `H0: elo<=-3.0  H1: elo>=3.0  (alpha=0.05 beta=0.05 -> LLR bounds [-2.94, 2.94])`; adjudication line
  `resign>=900cp/5p  draw<=10cp/10p>=80p`.
- First 7 games flowing, **adjudication firing** (adj-draw G1/G2, draw-rule G3, adj-win G4-G7) — **0
  flag-outs / 0 time-losses / 0 illegal / 0 no-move**. Early Elo (+0.0 at G7, 2W-3D-2L) is small-sample
  noise, not a signal.

Poll: `scripts/boxsprt.sh tail t1c` / `scripts/boxsprt.sh ps`. This run applies the predeclared decision
rule on completion.

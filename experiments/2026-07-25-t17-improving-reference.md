# 2026-07-25 T17 — `improving` reads a never-written reference (defect fix)

Pre-registered BEFORE launch (CLAUDE.md work-loop step 5). Candidate vs its immediate base, which is also the
newly-certified Batch-3 base. **Batch-3 keep candidate #1.**

## The defect (proven from code + measured, 2026-07-24 independent review)

`improving := !inCheck && ply >= 2 && staticEval > info.StaticEvalStack[ply-2]` (search.go:1419) had two bad
references:

**(a) Slot 0 was never written.** `StaticEvalStack[ply]` is assigned only inside `alphaBetaPV`, which is only
ever entered at ply >= 1 — both root loops call it with ply 1, and the ply-0 `alphaBeta` wrapper is dead code.
So **every ply-2 node compared its eval against a zeroed slot**, computing *"is my eval positive"* rather than
*"is my eval trending up"*. Measured directly: slot 0 held **0** after a full depth-8 search while slots 1/2/3
held 653/419/271.

**(b) `-INFINITY` was used as a sentinel and then compared as a value.** An in-check node at ply < 2 stores
`staticEval = -INFINITY`, so a node reading that slot computed `realEval > -2147483647` = **always true**, and
the sentinel propagated down the odd-ply chain through consecutive checks. Measured: **1068 sentinel
references in the endgame position, 100% of them spuriously `improving = true`.**

Measured frequency of *outcome-changing* improving decisions (temporary counters, since reverted):
kiwipete d11 **1798/85390 = 2.1%**, mid d11 **879/36799 = 2.4%**, end d15 **471/310886 = 0.15%**.

**Impact is leveraged, not proportional.** A wrong `improving` mis-sets FOUR pruning knobs at once — RFP
margin `RFM*(depth-imp)` (improving ⇒ prunes MORE), futility margin `FM*(depth+imp)` (prunes LESS), the LMP
threshold row (prunes LESS), and LMR `if !improving { reduction++ }` (reduces LESS) — and it does so at ply-2
and ply-3 nodes that each root an enormous subtree. A 2% decision-level error is therefore not a 2% tree
error, which the nodecheck below confirms.

## The fix (3 hunks, ~40 lines with comments)

1. `const unknownStaticEval = -INFINITY` — names the sentinel so it stops looking like a number.
2. `improvingVsPrevTurn(info, ply, staticEval)` — walks back 2 plies, then 4 (Stockfish's fallback) when the
   reference is unknown, and returns **false** when nothing usable is on the stack. Replaces the raw
   comparison at the call site.
3. `seedRootStaticEval(pos, info)` — writes slot 0 with the root's corrected static eval (raw eval + all three
   correction tables, mirroring `alphaBetaPV`'s non-check branch), or the sentinel when the root is in check.
   Called from **both** root entry points (`searchIterativeDeepeningUnsafe`, `searchFixedUnsafe`).

The in-check store at search.go:1393 now writes `unknownStaticEval` instead of a bare `-INFINITY` — a pure
rename, no behavior change on its own.

## Regression test + negative control

`TestImprovingReferenceIsWritten` (engine/reentrancy_test.go) asserts: slot 0 is non-zero and non-sentinel
after a real depth-6 search; a sentinel reference does **not** read as improving at ply 2 or ply 3; a known
reference still compares correctly in both directions; and the 4-ply fallback fires when the 2-ply reference
is unknown.

**Negative control run (the durable lock against T7's silent-no-op failure mode):** with only the two
`seedRootStaticEval` calls commented out, the test FAILS with
`StaticEvalStack[0] is 0 after a depth-6 search`. The test therefore genuinely discriminates the fix from its
absence, verified rather than assumed. Fix restored and re-verified green afterward.

## Behavioral-delta gate (the >= ~1% rule earned from T7)

| Position | Base (T5 lock) | T17 | Delta |
|---|---|---|---|
| kiwipete d12 | 346662 | 247063 | **-28.7%** |
| mid d12 | 149587 | 208825 | **+39.6%** |
| end d16 | 765656 | 709959 | **-7.3%** |

**Passes the gate by a wide margin — this is nothing like T7's 0.001%.** The mixed direction is the expected
signature, not a red flag: a spuriously-true `improving` pushes RFP toward *more* pruning while pushing
futility/LMP/LMR toward *less*, so correcting it moves the tree in opposite directions depending on which
knob dominates in a given position. Nodecheck baselines will need re-locking **only if this is kept**.

```yaml
id: 2026-07-25-t17-improving-reference
date: 2026-07-25
change_class: search heuristic (defect fix that changes search behavior — NOT a "correctness-flavored" free pass; CLAUDE.md is explicit that no such class exists, so this takes the full real-clock games gate)
hypothesis: >
  repairing the two bad references in the improving heuristic — seeding slot 0 with the root static eval, and
  treating the in-check sentinel as unknown rather than as a value — makes RFP/futility/LMP/LMR margins fire
  on a true "is my eval rising" signal instead of a corrupted one, and is worth >= +3 Elo (H1) rather than
  <= -3 (H0). Same defect class as T1b (+13.1), which was a variable WRITTEN but never READ; this is one READ
  but never WRITTEN. Honest expectation is nonetheless SMALLER than T1b: it changes ~2% of improving
  decisions, so this is a cheap SPRT, not a headline swing.
base_commit: 734a8ee            # == certified Batch-3 base 172e9e1 engine == T5 keep 5b93966 engine
candidate_commit: 734a8ee + t17.patch (UNCOMMITTED per policy; shelf copy output/t17.patch)
base_binary_sha256: 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721   # on-box ngn_t5.exe
candidate_binary_sha256: c928604155baec2c0133e5670edeaa7e475bd62cf5551d3ea8ba7d9f331578c9   # on-box ngn_t17.exe
build: GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -o output/ngn_t17.exe main.go ; go1.26.2, no ldflags/trimpath
harness_commit: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-aware validated mill, box-staged, untouched); launcher scripts/boxsprt.sh untouched
command: sprt.exe -new .\ngn_t17.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
go_version: go1.26.2 (local cross-compile, darwin/arm64 host)
goarch_goamd64: windows/amd64 GOAMD64=v3
tc: 10+0.1 (seconds; bullet)
concurrency: 8
openings: canonical sprt_openings (5000 lines)
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: STANDARD-ON (-resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80; M5-validated non-distorting)
maxgames: 8000
mingames: 300
aa_preflight: standing 2026-07-03 M5-enable adjudication-ON A/A (experiments/2026-07-03-m5-adjudication-aa.md) — same TC / concurrency / adjudication flags / machine / openings config, penta -1.1 [-13,+11], pLLR -0.18, 0/0 flag-outs, 961 g/hr, DONE_EXIT_0. Same basis every T-item and both certifications on this box use.
tests_run: >
  go test -short ./engine -count=1 (ok 5.5s); go test -short -race ./engine -count=1 (ok 24.6s);
  go test -short ./... -count=1 (all ok); go build ./... + go vet ./engine clean.
decision_rule: >
  SPRT [-3,+3] on pentanomial pLLR, bounds +/-2.94, maxgames cap 8000.
  pLLR >= +2.94 -> KEEP, commit to main as Batch-3 keep #1, re-lock nodecheck baselines to the T17 numbers.
  pLLR <= -2.94 -> SHELVE and revert (keep output/t17.patch).
  Capped with point estimate >= +1 and pLLR > 0 -> PROVISIONAL keep into Batch 3 (batch-certification path).
  Capped nonpositive -> SHELVE and revert.
  NOTE the asymmetry this fix creates: a SHELVE verdict does NOT restore the defect's legitimacy. If T17 is
  shelved, the reference bug is still a bug; the finding is then that the search had ADAPTED to it (the
  surrounding margins were hand-tuned on top of the corrupted signal), which makes the fix a prerequisite for
  T8 rather than a standalone gain. Record that explicitly rather than filing it as "improving is fine".
games_or_pairs: (pending)
result: (pending)
flags_errors: (pending)
verdict: (pending)
next_action: (pending)
```

## Why T17 runs before T8 (dependency, not a re-rank)

T17's own queue entry called it "a cheap SPRT, not a queue-jumper", and that judgement stands — its expected
value has **not** been revised upward. It runs first for a dependency reason: **T8 SPSA tunes RFP margin,
futility margin, LMP thresholds and LMR reduction — precisely the four knobs this defect mis-sets.** Tuning
those constants on top of a proven-broken input would converge them toward values that *compensate* for the
bug, and applying the fix afterward would invalidate the tune. Fix the input, then tune it.

## Launch — CONFIRMED LIVE 2026-07-25 10:39:35 EDT

| Check | Result |
|---|---|
| `boxsprt.sh ps` before launch | empty (Batch-2 cert fully drained) |
| candidate on-box hash | `C9286041…78C9` == local build — full match |
| base on-box hash | `0A8F65B7…C721` == certified Batch-3 base engine — full match |
| topology | **1 `sprt` + 8 `ngn_t17` + 8 `ngn_t5` = 17 processes**, all StartTime 10:39:35, CPU accruing evenly (1.77–2.47s) |

Header echo as printed by the box:

```text
new=.\ngn_t17.exe  base=.\ngn_t5.exe
mode: real clock 10s+0.1s (concurrency 8) | openings: 5000 (x2 colors) | concurrency: 8
H0: elo<=-3.0   H1: elo>=3.0   (alpha=0.05 beta=0.05 -> LLR bounds [-2.94, 2.94])
adjudication: resign>=900cp/5p  draw<=10cp/10p>=80p
```

`H0: elo<=-3.0  H1: elo>=3.0` confirms the **per-change T-item bounds**, not the certification `[0,+6]`.

Post-launch hygiene: engine diff extracted to `output/t17.patch` (2 files: `engine/search.go`,
`engine/reentrancy_test.go`, 151 lines) and the working tree reverted to HEAD, per the no-speculative-commit
rule. Verified truly reverted by rebuilding and re-running nodecheck: **346662 / 149587 / 765656 = exact
baseline match**, so the local tree carries none of the candidate's behavior while the run is live.

Poll with `scripts/boxsprt.sh tail t17` / `scripts/boxsprt.sh ps`. Expected duration 1.5-8.5h depending on
where it lands; at the 961 g/hr box rate the 8000g cap is ~8h20m.

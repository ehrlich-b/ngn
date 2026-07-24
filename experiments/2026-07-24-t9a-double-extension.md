# 2026-07-24 T9a — singular double extension (+2 plies on an overwhelmingly forced move)

Pre-registered BEFORE launch (CLAUDE.md work-loop step 5). Isolated single-change per-change gate: candidate =
current main HEAD engine (`5acd68b`, engine byte-identical to the T5-keep `5b93966` — T7 reverted, verified
`git diff 5b93966 HEAD -- engine/` empty) + `output/t9a.patch`; base = that same HEAD engine unchanged, reusing
the on-box `ngn_t5.exe` (the exact T5-keep binary). Real-clock 10+0.1 c8 on the LAN 9800X3D box. Highest-ranked
unblocked queue item after the T7 shelve: the biggest remaining measured prior in the whole T-queue.

Queue rationale (Elo-per-confirmation-hour): T9a **+11.2 / +10.1 LTC** [S #115], +11.9 [Sim] beats T12 NMP
cutnode-only +9.7 [S #230], T11a TT replacement +7.5 [S #16], T9b multicut +5.8, and T10a killer-reset +2.5 [E].
T9 is flagged ATTENDED in the revamp doc because the `:1821`/`:1832` guardrails are load-bearing — that flag is
discharged here by (a) a pre-launch nodecheck explosion gate, (b) a per-path double-extension cap, and (c) a
live early-health flag-out watch at launch, with a HALT rule on any candidate flag-out above the A/A baseline.

## Mechanism (one paragraph)

NGN already runs a full singular framework: at `depth >= SINGULAR_DEPTH` with a TT move it re-searches the
parent with that move excluded at `singularBeta = ttEval - (SINGULAR_MARGIN*depth)/32`, extends by one ply when
the verification fails low (`singularScore < singularBeta`), and — since W-SE2 — shaves a ply on non-PV cut
nodes when the verification fails high with `ttEval >= beta` (search.go ~1868-1885). What is missing is the
standard second arm: when the verification does not merely fail low but fails low **by a wide margin**, every
alternative reply is at least `DOUBLE_EXT_MARGIN` worse, so the move is not just singular but overwhelmingly
forced and is worth a second ply. T9a adds exactly that: `if !isPV && singularScore < singularBeta -
DOUBLE_EXT_MARGIN` then `nextDepth++` a second time. Two guards bound it. (1) Non-PV only — a PV node is
already searched to full depth and re-extending it is how the historical seldepth-111 explosion vectors start.
(2) A per-root-to-leaf-path cap `DOUBLE_EXT_PATH_CAP = 5`, matching the measured prior's shape ([S #115] is
"capped at 5 per path"), carried in a new `info.DoubleExtCount[ply]` stack written for the child before each
recursion (same live-ancestor-only discipline as `MoveStack`, so no unwind restore) — a forcing sequence
therefore cannot compound double extensions without bound. The pre-existing per-node ceiling
`nextDepth > depth+1 -> depth+1` had to be raised to `depth+2` **for a double extension only**: leaving it
would silently erase the second ply on any singular move that also gives check (there `nextDepth` is already
`depth+1`), which would make the change a partial no-op on precisely the forcing nodes it targets — the T7
failure mode. The per-path `EXTENSION_BUDGET = 24` clamp below it is untouched and remains the outer
anti-explosion bound. `DOUBLE_EXT_MARGIN = 24` is registered as SPSA dim `DoubleExtMargin` [4,128] for T8.

```yaml
id: 2026-07-24-t9a-double-extension
date: 2026-07-24
change_class: search/eval heuristic (search behavior — extension depth; real-clock games gate)
hypothesis: >
  extending an overwhelmingly-singular move by 2 plies instead of 1 (non-PV only, capped at 5 per
  root-to-leaf path) nets positive Elo at real clock by spending depth where the tree is genuinely forced.
  Measured prior: Stash #115 double extension +11.2 / +10.1 LTC (capped at 5 per path), +11.9 [Sim] — the
  biggest remaining measured prior in the T-queue. H1: candidate >= +3 Elo over base; H0: candidate <= -3.
base_commit: 5acd68b          # main HEAD; engine byte-identical to T5-keep 5b93966 (T7 reverted)
candidate_commit: 5acd68b + output/t9a.patch   # patch uncommitted per policy (no commit before verdict)
base_binary_sha256: 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721      # on-box ngn_t5.exe (reused T5-keep binary, hash-verified on box)
candidate_binary_sha256: 79a940b7bf28215794c2adabfde1dd3c19d26bf976a6de9bb0aede1a6d4a36dc  # output/ngn_t9a.exe (reproducible build, hash-verified on box)
harness_commit: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-aware validated mill, box-staged, untouched); launcher scripts/boxsprt.sh untouched
command: sprt.exe -new .\ngn_t9a.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
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
aa_preflight: standing 2026-07-03 M5-enable adjudication-ON A/A (experiments/2026-07-03-m5-adjudication-aa.md) — same TC (10+0.1) / concurrency (8) / adjudication-flag set / machine (LAN 9800X3D) / openings config, penta -1.1 [-13,+11], pLLR -0.18, 0/0 flag-outs, 77.6% adjudicated, 961 g/hr, DONE_EXIT_0; the SAME convention adopted by the KEPT T1b/T4c/T1e/T5 and SHELVED T13/T1c/T16/T10b/T7 runs, so no fresh A/A required.
decision_rule: >
  SPRT [-3,+3] on pentanomial pLLR, bounds +/-2.94, maxgames cap 8000; pLLR >= +2.94 -> KEEP (Batch-2 keep per
  CLAUDE.md batch-certification path); pLLR <= -2.94 -> SHELVE and revert; capped at 8000g with point est >= +1,
  pLLR > 0, and 0 excess flag-outs -> Batch-2 PROVISIONAL keep; capped-nonpositive -> shelve. HALT-AND-
  INVESTIGATE (do not verdict) on any candidate flag-out above the A/A baseline of 0 — this is an extension
  change, so a time-loss cluster is the explosion signature, not noise.
games_or_pairs: (pending)
result: (pending)
flags_errors: (pending)
verdict: (pending)
next_action: (pending)
```

## Pre-launch behavioral-delta gate (the T7 lesson, applied)

T7 burned 8h19m measuring a candidate whose fixed-depth nodecheck moved 0.001%. The gate adopted from that
shelve — **a pruning/extension candidate must move nodecheck by >= ~1% on at least one canonical position
before it earns box time** — is applied here BEFORE launch, and passes on all three positions by 10-30x the
threshold. Counts verified deterministic (two identical runs).

| position | depth | base (5b93966) | T9a       | delta    |
|----------|-------|----------------|-----------|----------|
| kiwipete | d12   | 346662         | 393979    | **+13.6%** |
| mid      | d12   | 149587         | 168792    | **+12.8%** |
| end      | d16   | 765656         | 1009455   | **+31.8%** |

Direction and magnitude are both as expected for a working double extension: the tree grows (extra plies cost
nodes) by ~13-32%, NOT by the 5-20x that a runaway extension would show. This is the explosion gate discharging
the ATTENDED flag: a genuine guardrail failure would not sit at +13%. Baselines are NOT re-locked — that happens
only on a keep verdict. The counterpart risk this trades into is real-clock depth loss (more nodes per ply at
fixed time), which is exactly what the SPRT is measuring.

## Mechanism-fires check (new regression test)

`TestSingularDoubleExtensionsFire` (engine/reentrancy_test.go, alongside the existing
`TestSingularExtensionsFire`) asserts the double-extension arm actually fires, is bounded by the singular
counter, and is gated by the singular toggle. Kiwipete d10: **44 double extensions of 135 singular extensions,
0 cap clamps** — the mechanism is live and `DOUBLE_EXT_PATH_CAP` is a safety net rather than a binding
constraint at this depth. This test is the durable lock against the T7 no-op failure mode: a refactor that
re-clamps `nextDepth` to `depth+1`, or a margin tuned out of reach, turns the feature into a silent no-op that
would otherwise only show up as an expensive null 8 hours later.

New counters `SingularDoubleExts` / `DoubleExtCapClamps` are plain diagnostics, never read by search decisions.

## Tests (patched tree, all GREEN)

- `go test -short ./engine -count=1` -> ok (4.455s)
- `go test -short -race ./engine -count=1` -> ok (25.886s)
- `go test -short ./... -count=1` -> ok (engine, internal/uci)
- `go test -short ./engine -run TestSingular -count=1 -v` -> both singular tests PASS (counters logged above)

## Binaries

Cross-compiled with the record-convention command (Makefile `build` Windows line):
`GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -o output/ngn_t9a.exe main.go`, go1.26.2, no ldflags/trimpath,
from clean HEAD `5acd68b` + `output/t9a.patch`. Tree restored after build; patch stays uncommitted.

- **Candidate** `output/ngn_t9a.exe` sha256 `79a940b7bf28215794c2adabfde1dd3c19d26bf976a6de9bb0aede1a6d4a36dc`
  — reproducible (a second identical build reproduced the sha exactly); on-box `Get-FileHash` returns
  `79A940B7BF28215794C2ADABFDE1DD3C19D26BF976A6DE9BB0AEDE1A6D4A36DC` (full match).
- **Base** reuses the on-box `ngn_t5.exe` sha256 `0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721`
  — the exact T5-keep engine binary already staged and hash-verified on the box (on-box re-verified at stage
  time this session); no rebuild.

Patch: `output/t9a.patch` (engine/search.go + engine/reentrancy_test.go, 153 lines).

## Bounds — [-3,+3] per-change gate (NOT the [0,+6] cert shape)

Isolated per-change SPRT: `-elo0 -3 -elo1 3` (H0: elo<=-3, H1: elo>=3), same shape as the KEPT
T1b/T4c/T1e/T5 and SHELVED T13/T1c/T16/T10b/T7 runs. `alpha=beta=0.05` yields pLLR bounds +/-2.94.

## Launch

(pending — recorded on confirmation)

CONFIRMED LIVE 2026-07-24 19:37:47 EDT via `scripts/boxsprt.sh launch t9a` (command exactly as the `command:`
field above). Box was idle immediately before launch (`ps` empty). Topology verified: 1 `sprt.exe` (PID 14736)
+ 8 `ngn_t9a` (PIDs 5872, 11504, 11572, 28276, 29416, 30152, 30288, 31792) + 8 `ngn_t5` (PIDs 11052, 13152,
13848, 14248, 21456, 24824, 26000, 31412), all started 19:37:47 EDT — the sole live workers on the box.
Out-file: `C:\Users\ehrli\ngn\sprt\t9a_out.txt` (fetch via `scripts/boxsprt.sh fetch t9a`).

On-box hashes verified at stage time: candidate `ngn_t9a.exe`
`79A940B7BF28215794C2ADABFDE1DD3C19D26BF976A6DE9BB0AEDE1A6D4A36DC`, base `ngn_t5.exe`
`0A8F65B75E59C376052F83B14787B0AF2EBA65552CEDFEE6F70CE576891CA721` — both full-hash matches. Header echoes the
predeclared binaries/args/bounds exactly (verbatim from the out-file):

```text
new=.\ngn_t9a.exe  base=.\ngn_t5.exe
mode: real clock 10s+0.1s (concurrency 8) | openings: 5000 (x2 colors) | concurrency: 8
H0: elo<=-3.0   H1: elo>=3.0   (alpha=0.05 beta=0.05 -> LLR bounds [-2.94, 2.94])
adjudication: resign>=900cp/5p  draw<=10cp/10p>=80p  (opt-in; A/A-gate before trusting a verdict)
```

Early health at G1->G9: games flowing, all four adjudication paths firing (draw-rule / adj-draw / adj-win /
max-moves), **0 flag-outs, 0 forfeits, 0 panics, 0 illegal moves, 0 crashes** — the flag-out watch that
discharges the ATTENDED flag. (The G5 `pLLR +2.16` is single-pair early noise at a ±hundreds CI, the same
artifact T7's G4 +4.32 produced; the `-mingames 300` floor means the stop rule is not armed there.)

```text
G1     0W  1D  0L   50.0%  elo   +0.0 [-495,+495]  LLR  +0.00  pLLR  +0.00  draw-rule
G5     1W  4D  0L   60.0%  elo  +70.4 [-209,+350]  LLR  +0.09  pLLR  +2.16  adj-win
G9     1W  7D  1L   50.0%  elo   +0.0 [-213,+213]  LLR  +0.00  pLLR  +0.00  max-moves
```

Poll with `scripts/boxsprt.sh tail t9a` / `scripts/boxsprt.sh ps`. The run drives to the 8000g cap or a
+/-2.94 pLLR crossing; apply the predeclared `decision_rule` on completion, and **HALT rather than verdict if
candidate flag-outs exceed the A/A baseline of 0** (extension changes fail as time losses, not as noise).

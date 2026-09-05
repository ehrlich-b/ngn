# Cached-mate termination correction on accepted T8

**Selected and locally validated; no native game launch or deployment.**
Versioned candidate/harness assets and startup verifier are now staged in the
native `sprt` directory, with matching remote hashes. No test engines or game
runs were started by staging. The ongoing rating pin retains its frozen inputs.
The [draw-conversion audit](2026-09-05-draw-conversion-audit.md) reproduces two
current-engine choices that allow immediate threefold while claiming mate13/15
at depth1. This candidate replaces unconditional mate stopping with Counter3.8's
completed-depth requirement: mate distance plus five plies. It changes no
evaluation weights, TT cutoff, history bookkeeping, qsearch move classes or
other search parameters. It is not a general solution to history-dependent TT
bounds. Quiet promotions remain a separate candidate.

## Source and validated behavior

Root compared all158 tracked Go/module files against main71163ab: only
`engine/search.go` differs. The playing base is accepted5cf301a. One added
`engine/mate_history_test.go` verifies the historical fixture and its draw;
the old/new carried-state UCI probes provide actual failing/passing behavior.
The isolated directory is `output/mate-history-20260905/`.

Both reproduced bad moves change under the guard:

- Game46: `e5e8` at depth1/mate15 becomes `e5f4` at depth10/mate3.
- Game118: `e2f2` at depth1/mate13 becomes `d1f1` at depth10/mate3.

Root independently warmed the final candidate through each game's NGN turns,
then continued against Stockfish with400k NGN /100k SF node limits. Both end in
actual checkmate after five plies. Full histories persist and every continuation
move/final board is independently checked. This verifies the concrete mechanism;
two selected continuations are not an Elo estimate.

The original reproduction used Hash64. A follow-up at the live default
**Hash128** reproduces game118's `e2f2` false mate (depth1/mate14), but game46's
baseline choice changes to `e5e3` and does not reproduce its particular immediate
draw. The guard chooses `e5d4` and `d1f1` at depths10/12. Root independently
continued both guarded games with Hash128 and observed actual checkmates after
five plies. Evidence: `default-hash128/results.json`,
`default-hash128/root-conversion-proof.json/log`, and its hash manifest.
Thus one identified immediate-draw witness persists at the actual native default;
do not claim both Hash64 failure choices reproduce at every hash size.

The agent's initial prose interpretation of the Hash128 Stockfish responses
reversed UCI perspective. `mate -2` after NGN's move means the opponent to move
is losing, and does not invalidate the guard. Raw transcripts were preserved;
root's complete played continuations provide the outcome proof.

Full short/race suites and vet pass. Root inspected the logs, checked the source
diff and final hashes, and verified all14 T8 defaults plus the three existing
EP/castling/fifty-move UCI regressions and clean shutdown. The seven guarded
carried-state probes avoid all seven particular archived draw moves.

Frozen local candidate artifacts:

| Artifact | SHA-256 |
|---|---|
|Patch including fixture test|`b26fdd92a5aec0b9c37a8e74ab9c7762403076aee666d6cec556100299d23def`|
|Candidate search.go|`1e504c68b18872a40ab88fe44b7a6e1aa42286a6709d0222bb090ff0c7d90d3e`|
|Source Go/module hash manifest|`c5e54b31d28dc2da69ccca3f69f55c8c230b4065bebdc073d7f901d112ed3edc`|
|Mac `build/ngn_20260905_mateguard`|`d23dc283618cae5950c7ea4ed2d7f86c51239f623aadc48daeead6130b741b15`|
|Windows `build/ngn_20260905_mateguard.exe`|`2c29f19652ce0bfa2d068af093e83c4eeca4f5f1907bc5dba7475162439a3c33`|
|Linux `build/ngn_20260905_mateguard_linux`|`0c067627404ab67a5f4a57fef63be024fd84e9248423286ef3dfc103041d266e`|

Go1.26.2, CGO0, Mac arm64 and Windows/Linux amd64/v3. The Go VCS build stamp
reflects the containing checkout; use the explicit source hashes/diff as
candidate provenance. Root evidence is in `root-evidence-sha256.json`,
`root-conversion-proof.json/log` and `root-protocol.json/log`.

## Native game instrument and prerequisite

The live `r0905acceptedpin` completes first with its unchanged input files.
Root reviews its400 games and the additional terminal-outcome audit before
touching frozen source or starting any other Windows/WSL engine compute.
This candidate is the next correctness priority; quiet promotions wait and
must be rebuilt against the immediate accepted base if this correction passes.

The independently proved harness terminal-precedence fix must be used by both
sides. Root built, but has not staged or run, the new native harness from
`output/harness-boundary-20260905/src`:

- `build/sprt_20260905_terminal.exe`:
  `62e6ffd96d57558938d45eaf0a123ee57dff1a7dec596a768738a0680b262e2a`.
- Future external `build/gauntlet_20260905_terminal.exe`:
  `499946e16f9758f2dd52d9760e1752565d98438b221c24bb451e46cd8a58a247`.
- Harness production file SHA-256:
  `2dee9d00fbc0b207d5fe9233530af850a39a0fcca5bab0434aa60788a8c8ad42`.
  Full short/race/vet and boundary regressions pass. No playing-engine source
  change is bundled into that harness archive.
- Immediate baseline `ngn_20260905_t8qcap.exe`:
  `43b2a2c93d9a773b0a08d8017d611edb7b39d63efe071cb7366c1ac4c11f92fc`.
- Existing5000-opening book:
  `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.

Before launch, verify corrected harness/source, both exact engine binaries,
book, unused run name, Windows/WSL idle state, native UCI/default/regression
smokes and clean quits. Preserve all run logs and errors. No new automatic
handoff controller has been created; root verifies these prerequisites.

Staged verifier `verify_mateguard_startup.ps1`, SHA-256
`f7aaf3126a92a253c366485a6543b8003d0ba13b9ffcb23be342d7a151ebf9df`,
rejects active Windows/WSL chess workers before starting any process. It checks
five asset hashes, the selected unused run name, Hash128 plus14 T8 defaults,
both engines' three UCI regressions and clean exits. It then runs the native
terminal-boundary regression executable, SHA-256
`99cbbb9a258f41963689fa4460212ed09886b5745e4f94ebee72a6b9a36085b7`.
`-Phase aa` checks the A/A name; `-Phase candidate` checks the candidate name
after A/A. Native PowerShell parsing passed; actual startup execution remains
pending the live pin's completion. Staging details and the no-launch staging
script are under `output/mate-history-20260905/native-preflight/`.

**Score adjudication is disabled for both runs.** Resigning a game on large
evaluation would hide the exact failure being tested: the base announces mate
and subsequently draws. Games must reach actual terminal/draw rules or the
explicit200-ply cap. Clock, concurrency and cap are identical for both sides.

## Fixed A/A followed by fixed candidate gate

Native9800X3D,10+0.1 real clocks, c8, default scheduling, lowpower=false.
Run **`r0905terminalaa`**:200 games/100 pairs with identical accepted T8 binaries.
Require zero operational errors/flags and paired95%CI containing0. A failed
A/A holds the candidate for investigation; no favorable-repeat sampling.

```sh
BOXSPRT_BIN=sprt_20260905_terminal.exe scripts/boxsprt.sh launch r0905terminalaa -new '.\ngn_20260905_t8qcap.exe' -base '.\ngn_20260905_t8qcap.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -maxmoves 200 -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 200 -mingames 201 -resignscore 0 -drawscore 0
```

After passed A/A and reverified idle state/assets, run **`r0905mateguard`**:
400 games/200 pairs, candidate versus the immediate accepted T8 baseline.

```sh
BOXSPRT_BIN=sprt_20260905_terminal.exe scripts/boxsprt.sh launch r0905mateguard -new '.\ngn_20260905_mateguard.exe' -base '.\ngn_20260905_t8qcap.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -maxmoves 200 -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 400 -mingames 401 -resignscore 0 -drawscore 0
```

The fixed cap prevents SPRT early stopping. Correctness acceptance requires the
proven witnesses/checks, all400 completed games, one DONE_EXIT_0, verified
cumulative W/D/L/pentamomial accounting and zero flags, illegal/missing moves,
crashes, disconnects or watchdogs. A paired95% interval wholly below0 rejects
this implementation pending investigation. Otherwise an inconclusive result
may support the proved correction, with no H1 or Elo-gain claim. No cap extension
or pooling; a strength claim requires its own completed positive evidence.

Any later external goal confirmation uses a separately frozen accepted binary
and corrected measurement instrument. Do not add a self-play estimate to2726
or relabel the old pin to manufacture2800.

# Owned NNUE state review and continuation — September 28

This is an independent review of the September 27–28 night claim at research
commit `1d127e3`, followed by a plan for the user's request to improve strength,
measure the current engine against competitors, and prepare the validated
configuration for use by the next morning. Historical README/TODO paragraphs
are evidence about their recorded builds, not the current configuration.

## Verified result

The central claim survives an independent recount of the copied PGNs. M11 is
247 wins, 386 draws, 167 losses from WDL25's perspective: 440/800 points,
55.0%, +34.860 Elo. All 400 pairs reverse colors; the independently computed
pentanomial is `[11,60,185,126,18]`. Stored paired-bootstrap 95% interval is
[+20.000,+49.406]. A separate paired normal calculation gives a nearly
identical score interval [0.52926,0.57074]. Successive 100-pair blocks score
55.25%, 52.5%, 56.75%, and 55.5%; the advantage is not confined to one block.

The locally copied M11 PGN, audit, final EPD, frozen opening PGN/prefixes and
network match their recorded SHA-256s. Match/audit stderr are empty, exit codes
are zero, and terminal reasons consist entirely of mates and legal draws.
All ten copied completed gates M1/M1b/M2/M3/E1/M5/M8/M6/M4/M11 likewise have
matching PGN/audit hashes, game counts, scores, pairings and pentanomials. The
four new network files match their conversion receipts. This review recounts
results and verifies saved audits; it does not repeat the independent
Stockfish legal replay on the Mac.

The exact winner is `wdl25-e10.nnue`, 4,744,976 bytes, SHA-256
`1ec8fc1737ddfdd5f8b6ff0b4e26778085fb563f7c5e82c1fc5d3ea454b6ec29`, with
`EvalBackend ngn-k4-768-v1`, `K4EvalScale 60`, Threads 1, Hash 128, OwnBook
false, Move Overhead 100 and GOMAXPROCS 1. Both M11 roles use binary
`75b0afdb8cf56ebb16daabb41f19979a9881b3d0211e1faf0773d80f14846279`.
Borrowed Rodent V1.2 is the project's previously adopted evaluator, not the
native Rodent engine. The result does not establish superiority to all
available pretrained networks or to the strongest Go engine.

## Corrections and limits

- Counter 5.5's +177.22 [+151,+205] external result belongs to **A1 at scale
  60**, not WDL25. It is 223W/142D/35L = 294/400 points, 73.5%. WDL25's
  external strength remains to be measured. Adding +20.9 or +34.9 to +177
  would not substitute for that match.
- NNUE training progressed from 20M freshly SF18-labeled positions to **2.374B
  archive records** carrying calibrated archive scores. Ten-epoch training
  makes 26.84B presentations, about 11.3 passes. Record count is not a proof
  that every record is a distinct original position or that whole original
  games are disjoint. Encoded-chain/input holdouts are developmental controls.
- Owned means random-initialized project-trained weights and an owned
  training/export/evaluator pipeline. The graph is the familiar mirrored
  K4/768/SCReLU/eight-output-head design; Bullet/CUDA and teacher data are
  external dependencies. Architecture novelty is not part of this result.
- Most night tests reuse opening pairs 4341–4740. Rules fixed before each run
  prevent score-based extensions; this does not make repeated development on
  the same opening set an untouched generalization test. Fresh openings,
  longer clocks and current multicore measurements are still needed.
- K8 [+4.8, CI -10.0 to +19.6], W1024 [+2.6, -11.7 to +17.4], and stacked
  [+10.9, -3.0 to +24.8] did not earn promotion. These intervals do not prove
  that capacity is irrelevant; they rule out adopting these particular runs
  under the fixed gate. A several-Elo effect remains below their precision.
- Scale 60's E1 interval only narrowly excludes zero. It is a supported
  working setting, not a proven optimum. Consecutive-ply score magnitudes are
  a diagnostic, not identical-position calibration. The game gate provides
  the useful evidence.
- S1 is a 1.1623x speedup (**16.23% throughput increase, 13.97% less elapsed
  time**), not 16.2% less elapsed time. Default-scale integration identity
  holds, and load-time int16 bounds plus kernel/reference tests support the
  arithmetic change.
- The lightweight quickmatch instrument checks input hashes and performs
  independent legal/terminal PGN audit. Its copied night artifacts contain
  no fresh A/A control, and the script does not run one. It also omits the
  original candidate-match runner's full protocol/process/continuous-load
  evidence. These are omissions relative to CLAUDE.md's normal gate contract,
  not evidence that a game was illegal. Restore those controls for new gates
  instead of quietly calling the instruments equivalent.
- Absolute 3300, strongest Go engine, longer-clock transfer and current owned
  NNUE SMP strength remain unmeasured. The old README's 3000–3100 orientation
  refers to much older Counter-backed source, not today's strongest setup.

## Current engineering position

The classical 2800 target has historical audited evidence. Subsequent work
established real Lazy SMP, exact imported evaluator compatibility, large AVX2
speed gains, and the adopted completed-winner history/LMR package (H1's
independent confirmation +56.96 [+38.37,+75.88]). H1 is present in the owned
research tree. The staged-picker, shallow SMP staggering, helper-result and
TT publication candidates were shelved; they are not pending freebies.
SF18 BIG integration/reference work exists, but its strong HCE comparison is
not a current owned-vs-BIG promotion result.

The new weights beat the strongest previously adopted borrowed evaluator in
NGN and the prior owned configuration beats the exact external Counter release.
This makes the engine a credible competitive project. The important remaining
uncertainties concern opponent breadth, clock/width transfer, useful multicore
search, search/eval calibration, and delivery of the measured configuration.

In response to the user's request for a present absolute range: a reasonable
working judgment for the strongest WDL25 single-thread research configuration
is about **3500**, with a broad plausible **3300–3600** range on a CCRL-like
engine scale. This is not a measured confidence interval or an official list
entry. Counter's author lists 5.5 at 3333 CCRL 40/15; A1's short-clock +177
would imply roughly 3510 if it transferred. WDL25's current direct external
measurement, longer-clock transfer and opponent breadth remain missing.
Do not transfer the much tighter within-engine +35 interval into absolute
precision. The older installed classical build's historical audited estimate
remains 2884 [2834,2934] on its frozen calibration.
Source: [Counter's published strength](https://github.com/ChizhovVadim/CounterGo#strength).

The 23 completed research commits have now been pushed to the verified private
tracked branch. The primary
checkout remains the separate, shelved staged-picker branch. Default startup
is HCE; the winning net and scale are opt-in. Live September 28 installed
hashes match the recorded historical classical installs:
WSL `0c067627...`; Windows and repin `2c29f196...`. Thus an ordinary installed
launch is not the winning research configuration. `make build-release` also
does not request GOAMD64=v3, while the accepted fast kernels require it.

One concrete delivery defect was reproduced: K4EvalScale lived in a
process-global atomic and was absent from evaluator identity. UCI
joins a search before changing it, but the model identity is unchanged, so
`prepareTTGeneration` can retain TT scores/bounds and histories from the old
scale. Independent search engines in one process also share that setting.
The frozen night games configure each fresh process once and are not invalidated
by this defect. Failure-before regressions showed engine A changing engine B's
score from 1062 to 637, and scale changes retaining the cached TT score 888.
The fix stores scale per engine and freezes it into worker/TT evaluator identity.
Same-value/invalid requests preserve valid state; inactive K4 options preserve
HCE state. Required engine/repository short and race suites pass on WSL.
Configured-60 depth-12 searches match the accepted binary on all 88 fixtures
over three alternating rounds: identical depths, 20,524,751 total nodes, scores
and best moves. Median speed ratio is 1.0036, consistent with unchanged speed.
Explicit startup flags advertise scale 60 and OwnBook false for a separate
candidate launcher. Ordinary defaults and historical installations stay held.
See the [verification receipt](2026-09-28-owned-scale-verification.json) and
[identity receipt](2026-09-28-owned-scale-identity.json).
The earlier trainer's result-weight mode was missing from structured attempt/
checkpoint receipts, and its completion line said result_weight=0 for
WDL25. Executable hashes distinguish the existing runs. New variants now
record the actual target schedule explicitly.

## Live host capacity and maintenance

At 21:25 UTC September 28, Windows C: had **0.23 GiB free** out of 1906.74
GiB. Ubuntu's virtual filesystem reports 589 GiB available while its VHDX is
430.08 GiB physically long and Linux uses about 367 GiB. Linux free space is
not the authoritative Windows capacity. The selected e20 checkpoint at
candidate-2375680 exists; the unchanged 76.0GB A1 corpus exists.

Retained night checkpoint directories total approximately 39 GiB and the
original A1 scatter directory is 83 GiB. No deletion is authorized by this
review. In particular, files touched in the past 24 hours stay untouched.
Compaction may recover tens of GiB; it cannot create the approximately 350GB
headroom required by the proposed 6.4B-record scatter/gather experiment.

The user approved a temporary maintenance interruption. Trim, WSL shutdown
and read-only diskpart compaction succeeded at 21:54 UTC. VHDX length fell from
430.078 to 380.702 GiB, recovering 49.376 GiB from that file. Windows C: had
53.365 GiB free after service restoration (other Windows activity means this
is not exactly the VHDX delta). No reboot was required. Docker and the hopper
are enabled/active; both original worker tasks restarted in successor containers,
and the relay and keepalive are running. The 72-hour keepalive limit remains.
The current SSH token has administrator membership, unlike the old night
report. Hyper-V Optimize-VHD was unavailable; diskpart was used. No training
data or checkpoints were deleted in this continuation. Subsequent small-write
jobs require a physical Windows capacity floor; the 350GB lane remains blocked.

Microsoft documents that VHDs grow without automatically shrinking after
file deletion and that diskpart compaction requires a detached/read-only VHD.
Sparse VHD automatic reclamation is experimental; the existing run observed
its potential-data-corruption refusal and did not force --allow-unsafe.
Use an explicit offline maintenance window, preserve/restart services, trim,
compact and verify. A reboot is normally unnecessary. Any automated offline
compaction must coordinate the keepalive and skip active jobs; a scheduled
WSL shutdown is not harmless maintenance on this shared worker.

Sources: https://learn.microsoft.com/en-us/windows/wsl/disk-space ;
https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/compact-vdisk ;
https://learn.microsoft.com/en-us/windows/wsl/systemd ;
https://learn.microsoft.com/en-us/windows/wsl/wsl-config ;
https://github.com/microsoft/WSL/issues/13075 .

## Bounded continuation by morning

Working deadline assumption: 08:00 America/New_York, September 29. Do not spend
available model tokens on idle polling; remote jobs get an exact handle,
completion/failure receipts, bounded host-side waiters and infrequent checks.
All engine/build/test/training execution stays on WSL.

1. Obtain the maintenance window, compact and verify capacity/services. Do not
   launch write-heavy work until a Windows-side capacity guard passes.
2. Make the accepted configuration reproducible and usable: isolate K4 scale
   per evaluator identity, provide explicit startup scale for a launcher,
   run focused failure-before regressions and required WSL short/race suites,
   then prove configured-60 fixed-search identity. Prepare Linux/Windows v3
   binaries, exact net, launchers and a manifest in a separate candidate bundle.
   Respect the existing installation/release hold.
3. Confirm the unchanged WDL25 configuration on fresh openings, with a fresh
   A/A and recorded effective options. Measure a longer-clock external cell
   against the exact existing Counter release. Add a native Go-engine control
   if its exact artifact and protocol boundary can be admitted in time.
4. In the small-write training lane, complete the resumable e20 diagnostic and
   test one predeclared higher-result-weight variant (0 to 0.4) against the
   **current WDL25** owned baseline. Freeze selections/caps before games; a
   validation-loss improvement alone cannot promote a net. Limit checkpoint
   footprint, preserve the resumable parent and enforce physical free space.
5. Give owned NNUE a width-8 smoke/lifecycle check and, if time remains, one
   fixed owned8/owned1 playing cell. Do not claim linear Elo from aggregate NPS.
6. Commit coherent verified source/records; verify private destination before
   a routine push. Report measured gains, nulls, packaged artifacts and blocked
   work separately. The large-data lane waits for substantially more capacity
   or a separately validated pipeline that avoids materializing both copies.

## Frozen queue launched

The [overnight plan](2026-09-28-owned-overnight-plan.json) was written before
training or scored games. WDL40 changes only the linear result-weight endpoint
from 0.25 to 0.4; the final 1,638,400-update checkpoint is selected in advance.
Conversion, full/incremental integer and trainer-probe parity must pass before
its fixed 1,600-game gate against WDL25. Promotion requires a positive paired
95% Elo lower bound. The original e20 executable separately resumes its bound
parent to 3,276,800 updates; completing it is diagnostic, not promotion.

GPU training started at 22:43 UTC in `ngn-owned-training-20260928.service`.
The trainer passed real CUDA smoke/resume checks and rejected simultaneous
target flags and a mismatched parent target. The exact harness/executable and
opening sets are hash-bound in the accompanying build/smoke/opening receipts.

The match queue started at 23:05 UTC in `ngn-owned-matches-20260928.service`:
128-game short-clock A/A, fresh 800-game WDL25/borrowed-V1.2 confirmation,
the 1,600-game WDL40 gate, 80-game long-clock A/A, and 400-game WDL25 versus
the exact native Counter 5.5 release at 60+0.6. Six single-thread pairs use
separate physical cores; training is pinned to cores outside that set. The
host remains shared with the restored hopper, recorded explicitly in the plan.
Both queues enforce a physical C: floor of 25 GiB and the morning deadline;
interrupted cells remain unscored. C: had 46.08 GiB free at match launch.

The final instrument passed two four-game legal/protocol/process/trace smokes,
with 648 and 544 independently checked legal plies. These are procedure checks
and carry no strength estimate. Counter's exact release emits two benign
startup messages to stderr. Its SHA-specific allowance retains raw transcripts
and requires one version and one embedded-weight record per witnessed process.
Eight mutated traces (wrong revision/runtime/hash, duplicate or absent records,
warning severity, illegal PV and messages attributed to NGN) were rejected;
four protocol-reader fixtures preserve raw output and reject foreign, duplicate
or fatal stderr. No illegal-PV exception was introduced for native Rodent.
See [compatibility controls](2026-09-28-owned-counter-controls.json),
[instrument smokes](2026-09-28-owned-instrument-smokes.json), and the
[launched match unit](2026-09-28-owned-match-unit.json).

The first short-clock A/A stopped at 23:07 UTC and remains unscored. Twelve
NGN processes plus supervision approached the service's 6 GiB memory limit:
sampled process-tree RSS peaked at 6,287,364 KiB, only 4,092 KiB below that
ceiling, which also accounts for file-backed cache. Two roles then showed
522/682 ms response latency and each missed the clock by 100 ms. This points
to pressure from the experiment's resource limit, rather than a measured
strength failure. The revised instrument uses a 10 GiB process-tree ceiling,
12 GiB service ceiling, and records cgroup memory pressure events as well as
host capacity. Both procedure smokes and the compatibility controls passed
again. At 23:27 UTC, unit `ngn-owned-matches-20260928-r2.service` restarted the
complete A/A; all engine settings, clocks, openings, selections and game caps
are unchanged. No partial games were pooled. See the
[abort receipt](2026-09-28-owned-aa-abort-001.json) and
[revised launch](2026-09-28-owned-match-unit-r2.json).

The revised A/A completed at 23:39 UTC: 128 games, 64 pairs and 19,285 legal
plies, with every protocol/process/trace/terminal audit passing. Its paired
score interval [0.46875,0.56640625] contains parity. Peak process-tree RSS was
7.441 GiB, exceeding the old 6 GiB ceiling; the revised cgroup recorded zero
pressure-limit or OOM events. The fresh 800-game reference cell then started.
See the [completed A/A](2026-09-28-owned-aa-short-result.json).

WDL40's final selected checkpoint completed conversion and parity at 23:43
UTC. Its net is `41ffe3f372579705f0c199096e0967544aecf130afd0256793c53c3a55379364`.
All 1,638,400 updates and 26,843,545,600 presentations are bound to the frozen
attempt with result-weight endpoint 0.4. Trainer/raw float delta is at most
5.36e-7 in the parity fixtures; scalar/context integer evaluations match,
with mean quantization error 4.33cp and maximum 17.68cp within the predeclared
8/32/64 gates. These are export checks, not game-strength evidence. The final
resumable checkpoint and net plus the complete A/A trace/PGN/control records
were copied off-worker and hash-checked. The original e20 resume then started.
See [WDL40 eligibility](2026-09-28-owned-wdl40-e10-ready.json) and
[parity evidence](2026-09-28-owned-wdl40-parity.json).

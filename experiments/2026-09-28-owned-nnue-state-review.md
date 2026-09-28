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

Research source is 23 commits ahead of its recorded tracking ref. The primary
checkout remains the separate, shelved staged-picker branch. Default startup
is HCE; the winning net and scale are opt-in. Live September 28 installed
hashes match the recorded historical classical installs:
WSL `0c067627...`; Windows and repin `2c29f196...`. Thus an ordinary installed
launch is not the winning research configuration. `make build-release` also
does not request GOAMD64=v3, while the accepted fast kernels require it.

One concrete delivery defect needs a regression test: K4EvalScale currently
lives in a process-global atomic and is absent from evaluator identity. UCI
joins a search before changing it, but the model identity is unchanged, so
`prepareTTGeneration` can retain TT scores/bounds and histories from the old
scale. Independent search engines in one process also share that setting.
The frozen night games configure each fresh process once and are not invalidated
by this defect. Fix it before packaging mutable UCI scale selection; preserve
fixed-60 behavior with exact search identity against the accepted binary.
The trainer's result-weight mode is also missing from structured attempt/
checkpoint receipts, and its completion line still says result_weight=0 for
WDL25. Executable hashes distinguish the existing runs, but new variants should
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

The keepalive task is running, with RestartCount 0 and a 72-hour execution
limit. The NNUE queue is idle. The Arli hopper and three rootless Docker
container scopes are running. WSL shutdown would interrupt them. Approval
for a maintenance window is pending; current SSH token reports administrator
membership, unlike the earlier night report. Hyper-V Optimize-VHD was not
found in the current PowerShell session, so diskpart is the prepared route.

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

# Direct quiet promotions on the Counter baseline

```yaml
id: 2026-09-12-qpromo-gate
date: 2026-09-12
change_class: search/eval heuristic
hypothesis: Including direct quiet promotions in qsearch improves the accepted Counter-backed engine at real clocks.
base_commit: b7ffdc886a56516af673243cae2dbd7597115cc0
candidate_commit: uncommitted base plus production patch 171cfea254797ec0f756ed3503ea295ec4a5175bf88bd486a0cbb2009870bbb7
base_binary_sha256: 86ef7473fa5f53e4fdd800f550f4fe3f182db09056afacbad98cf174d88554bc
candidate_binary_sha256: 3a0300324e3f1727f62d6d64b88e382075a7b6f33381ec76f4bb42468aca55a2
harness_commit: frozen Fastchess and audit sources listed by the run manifest
command: python3 /home/ehrli/repos/ngn-qpromo-gate-20260912/qpromo_gate_driver_v2.py /home/ehrli/repos/ngn-qpromo-gate-20260912/qpromo_gate_manifest_v2_approved.json
machine: Ryzen 7 9800X3D, Windows WSL Ubuntu, physical-core representatives 0/2/4/6
go_version: go1.25.5
goarch_goamd64: linux/amd64/v3, CGO_ENABLED=0, GOMAXPROCS=1 per engine
tc: 10+0.1
concurrency: 4
openings: existing audited 200 unique six-ply development opening prefixes, sequential and reversed colors
openings_sha256: 2257ecfb9af03f1ee0bff68f71ec97dbe363cdb9f535fab1a0ebc5af1abaaf33
aa_preflight: PASS100 games, 50 pairs, identical base binary and match configuration; candidate admitted
decision_rule: complete AA100 with paired95 interval including0; then complete candidate400; adopt only if candidate paired-bootstrap lower95 Elo is greater than0 and every integrity gate passes; otherwise shelve; no score-conditioned extension or rerun
games_or_pairs: AA100/50 then candidate400/200
result: pending
flags_errors: must be zero
verdict: pending
next_action: monitor the frozen WSL run and audit the complete outcome
```

The same behavior source is now accompanied by rating-documentation commit
`6b72088`. No rating games are added in this experiment, and its relative result
will not be added arithmetically to the historical classical rating.

## Scope and prior proof

The [earlier witnesses](2026-09-05-quiet-promotion-witness.md) expose a qsearch
move-class omission, including mirrored quiet-promotion mates. This experiment
tests the playing value of the small direct-generator patch on current source.
It appends Q/R/B/N quiet promotions to the existing captures and widens the
local move/score arrays from 64 to 96. Existing delta/SEE pruning and evaluator
transition policy remain intact. A witness is not itself an Elo verdict, so
this run uses the stricter positive-gain acceptance rule above.

Sol implemented the rebase in `/home/ehrli/repos/ngn-qpromo-candidate-20260912`.
Root reviewed the production diff and regression coverage. Tests include both
colors, capture/quiet promotion mixtures, blocked and diagonally pinned pawns,
all 32 possible quiet-promotion variants, a deterministic legal walk, exact
mate scores, and search-stack restoration. The Counter regression uses active
nonzero pawn and promoted-piece feature weights, checks full-refresh parity on
committed frames, and verifies exact restoration of the starting raw value.

Final source passed short engine tests (7.481 s), engine race tests (39.759 s)
and the full short suite. Both binaries use the same build flags and toolchain.
Root read the final logs. The immutable build/test receipt is:

`/home/ehrli/repos/ngn-qpromo-candidate-20260912/output/qpromo-receipt/qpromo-build-test-receipt-20260912.json`

SHA-256 `aa709d5e9e6e63a1e27bcc7a3fae66e8143ec25f114010cab36f9cdca5141d5a`.
It binds the production patch, test patch, source files, commands and logs.

## Frozen protocol

Run root: `/home/ehrli/repos/ngn-qpromo-gate-20260912/run-001`.

- Driver SHA-256: `917b8550307fc213ce1997ad91d264bca2ca064b090bb4fda2f06d4fd8fc7eb0`.
- Held manifest SHA-256: `a6e5807a1df671ec52e3ee36515e7c589c6a294de9cf1b9629727ed819570fa6`.
- Released manifest SHA-256: `7f74b75abec5a0f3adc18ad0e1c42b1ce9b7a21d403c15a0170600a13296fd17`.
- Canonically copied run manifest SHA-256: `89e3523f49f8237d7bdc021e131263bc1fa52be89333ba789d7fae99d40f976c`.
- Fastchess SHA-256: `e61220f6651599b2a40a7bd1d7535cf59be8e3eaa1f74dd2f6d5dc2fb1c0bc00`.
- Counter model SHA-256: `3488baed71f4d432d028e05b65e2ff13f8c9afe7f91974d1a3c34a0691d6670c`.
- Opening PGN SHA-256: `1ae6a65a0224d7a4ff893a3c999542cd3d05e9a7f0e989fe2deee1cff7b8bf5a`.
- Hash 128 MiB, Threads 1, OwnBook false, Move Overhead 100 ms per engine.
- Match seed 20260912; paired percentile bootstrap seed 2026091201,
  100,000 resamples of complete opening pairs.
- A/A uses the first 50 openings; candidate uses all 200. A/A is a control,
  not candidate selection or tuning, so this overlap is intentional.
- Candidate is engine A in the independent audit; base is engine B.
- No adjudication. Every played move, natural terminal outcome, opening pair,
  protocol trace and process identity is independently checked.
- A 10-second prelaunch CPU sample must show at most one non-owned core of
  activity. During each match, 12 consecutive five-second samples above that
  threshold reject timing evidence. Existing unrelated jobs are preserved.
- Base and candidate binaries are copied and hash-checked before A/A starts.
  Source patches, build receipt and audit tools are frozen alongside them.
- Failed or incomplete runs have no strength verdict. The fixed schedule is
  not extended after its score is observed.

Root and an independent Sol reviewer corrected the initial scoring orientation,
checked the frozen module interfaces and provenance, and approved this exact
driver/held-manifest pair before launch. Both real UCI preflights passed. Root
authorized changing only the manifest's release state and launching once.

The original exact driver and run manifest remain in run 001. The output directory is immutable and
the script refuses to rerun into it. A detached launch was rejected before it
started. After fresh 96–100% idle CPU observations and explicit task/resource
review, the foreground WSL launch was approved and started once. The driver
then passed its own prelaunch CPU check and began A/A. No timing games were
started by the rejected attempt.

## Run 001: operational failure before candidate admission

The supervisor terminated A/A after 77.297 seconds when sampled process-tree
RSS reached 4,196,856 KiB, exceeding its 4,194,304-KiB limit. Seven control games
had finished; no candidate game started. The receipt records
`sampled-process-tree-memory-limit`, no timeout or monitor error, and zero
surviving descendants. These incomplete control games have no verdict and are
not reused.

Eight NGN processes accounted for 4,155,648 KiB at the peak, versus 29,944 KiB
for the Python stage and 9,856 KiB for Fastchess. The cap was undersized for the
resident engine processes; `Hash=128` limits the transposition table, not the
whole Go process. The preserved earlier accepted Counter/HCE run reached
751,108 KiB in an individual engine. Eight such processes alone project to
5.73 GiB, leaving little overhead under a 6-GiB cap. The current host had
13.10 GiB available after cleanup. Non-owned CPU activity stayed below 0.13 core
in all recorded samples.

Root authorized a **memory-only operational amendment to 8,388,608 KiB** and a
fresh full A/A followed by the original fixed candidate schedule. Engine bytes,
model, options, opening schedule, clocks, CPU affinity, seeds and decision rule
remain unchanged. Run 001 and its exact source/manifest remain preserved.
The amendment is based on resource receipts, not the incomplete control score.

## Run 002: memory-only amendment

The [tracked driver](2026-09-12-qpromo-gate.py) now matches v2 exactly; its only
code difference is the memory limit. The [tracked manifest](2026-09-12-qpromo-gate-manifest.json)
is the canonically frozen run-002 copy. Original v1 files remain unchanged on
WSL. The reviewed amendment receipt binds run 001's supervisor, memory samples,
CPU observations and prior accepted-engine memory evidence.

- Driver SHA-256: `9c35fe89ef99a854deba2919e230af4decf6d7f93e55cb10c96444e8c51b2455`.
- Held v2 manifest SHA-256: `68070eb184132e0818a98aa1dac3951c4b67405d7f16fb7874f40d8c95dd0c27`.
- Approved v2 manifest SHA-256: `12fa10771a3b101f7b1613f613eeb230114f417928f9c684ed7dc9594c1e7839`.
- Canonically frozen v2 manifest SHA-256: `0fde74caace183f2c95c2b1c160833b7083c7c5e16c82a3fe4a9619edba09dae`.
- Resource amendment SHA-256: `a672be9ff284171c5dcd79ee69f770fab438d26635bfde029220d3ba20e404c8`.

Root reviewed the one-line driver diff and launched v2 once in the foreground.
It uses a new immutable `/home/ehrli/repos/ngn-qpromo-gate-20260912/run-002`
directory and starts the full A/A schedule again. The engine comparison and
its fixed statistical gate are unchanged.

### Completed control; candidate admitted

Run 002 completed all 100 control games / 50 pairs in 835.755 seconds:
22 wins, 52 draws, 26 losses for base A; penta `[1,18,18,10,3]`.
The predeclared paired-bootstrap interval includes zero. Independent legal
replay passed all 15,196 plies and all natural terminal outcomes; strict
operational audit passed 490,480 trace lines. The supervisor returned zero,
with no termination, timeout, monitor error or surviving descendants. Peak
sampled process-tree RSS was 5,259,180 KiB, below the amended 8-GiB cap.
The frozen runner admitted the original 400-game candidate schedule. Its
result remains pending; the control is not strength evidence for the patch.

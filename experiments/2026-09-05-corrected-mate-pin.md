# Prospective corrected external rating: r0905matepin

Prepared before any game under this run name. The earlier accepted-T8 run
completed with two terminal-result mislabels and remains
[held, with raw results unchanged](2026-09-05-accepted-t8-pin-result.md).
This is a new measurement using the proved harness correction. No old games
are relabeled, rerated or pooled, and no old controller is restarted.

## Selection and prerequisites

The only proposed candidate is the exact mate-stop binary, conditional on root
acceptance of its [predeclared Windows comparison](2026-09-05-mate-stop-windows-recovery.md).
That fixed400-game gate has now completed and passed root's original correctness
rule:91/242/67, paired+20.87 [+3.56,+38.29], zero flags/errors. Its direct
old-fail/new-pass proof and combined WSL short/race/vet/oracle checks also pass.
No candidate switch occurred. Exact source integration and deployment records
remain prerequisites to this rating launch.

Before launch, require candidate acceptance, exact source integration and WSL
short/race/vet plus independent move-generation checks. Preserve rollback and
deploy the accepted version on Windows/WSL. No Mac engine/build/test execution
is permitted. Require all other native and WSL engine/build workers to drain.

| Exact versioned native repin asset | SHA-256 |
|---|---|
|ngn_20260905_mateguard.exe|2c29f19652ce0bfa2d068af093e83c4eeca4f5f1907bc5dba7475162439a3c33|
|gauntlet_20260905_terminal.exe|499946e16f9758f2dd52d9760e1752565d98438b221c24bb451e46cd8a58a247|
|ratings.json|f1ac3a64ee26448f51c82b14312a04335b690ea00d3a56b527e2be7238696d32|
|sprt_openings.txt|974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222|

The candidate is accepted T8 plus only the frozen mate-stop guard; no evaluation
weights or NNUE change. Its source mapping is in the linked comparison record.
The harness is the exact terminal correction integrated at231b686, with the
same playing-engine dependency sources as the already tested harness archive.
Both binaries were built with Go1.26.2, CGO0, Windows amd64/v3.

## Fixed instrument and command

Native Windows9800X3D,120+1 real clocks, concurrency8, default scheduling,
lowpower=false,200-ply cap and no score adjudication. Fixed400 games:80 against
each of the original five anchors,40 paired openings, indices0–39. Preserve
the exact anchor binaries and historical labels: Blunder6.1=2155,7.2=2425,
7.4=2532,8=2674; Counter3.8=2994. Their hashes remain those in the original
[400-game protocol](2026-09-05-2800-pin-plan.md).

```text
BOXREPIN_BIN=gauntlet_20260905_terminal.exe scripts/boxrepin.sh launch r0905matepin ngn_20260905_mateguard.exe 80 'v6.1.0,v7.2.0,v7.4.0,v8.0.0,counter-3.8'
```

Before this single launch, reverify all assets, unused batch/log/tally/PGN names,
Windows/WSL idle state, the exact candidate's Hash128 and14 defaults, its three
canonical regressions, all five anchor handshakes and native harness terminal
tests. Preserve startup output and the actual generated command/process receipt.
The source-only package is `output/corrected-pin-20260905/`; it contains no
launcher or automatic handoff. Runtime startup evidence is pending.

Root reviewed the adapted runner and startup verifier, checked all inherited
anchor/label/book hashes against the original scripts and retained the frozen
rating/legal/terminal implementations byte-for-byte. Corrected a copied hash
typo and an incorrect primary-estimator description before staging. Worker
name fixtures include the actual `stockfish18` executable and truncated WSL
test names via their argv path.

The source-only validation package is staged under
`/home/ehrli/repos/ngn/output/corrected-pin-20260905`; all nine package hashes
pass. Archive SHA-256:
`166859129c273fdea019db11a9e4eefcb63ae068ba9575f947ac71ea86e05ae1`.
Runner SHA-256:
`e6d3e4a73e10034e27bb96197d41d10087cf1e6a7d49f8c1bad7207428c7fe03`.
Native startup verifier SHA-256:
`f3abe688d1043426a8a8fc30aa17da079bcb2ac4036072e00d7dfb87addc169d`;
its PowerShell AST parse passes. Candidate, harness, terminal-test and Linux
candidate artifacts also pass remote hashes. This is staging only: no startup
search, integration test, default replacement or new rating game ran.
Raw evidence is `staging-verification.*` in the local package directory.

## Unchanged goal and integrity gate

The primary estimator remains the original add-half-smoothed inverse-variance
rating. **Its lower95% confidence bound must exceed2800**, both engine families
must remain represented, retained anchors must score strictly above and below
50%, and NGN must have zero operational errors and zero clock losses. Preserve
the original >2% per-anchor operational-error exclusion and disclose exclusions.
Real opponent clock losses remain in the primary result. Whole-opening cluster
bootstrap, seed2800 and10000 resamples, is supplementary only. These are local
historical calibration labels, with unmodeled label uncertainty.

Require exactly400 records, one terminal DONE_EXIT_0, complete log/tally/PGN
agreement, all scheduled openings and colors, pinned artifacts and no missing
games. Run the frozen legal replay on all400 complete histories and the frozen
terminal supplement on every final board, on WSL with the pinned Stockfish18
oracle. Snapshot raw inputs before validation and verify their hashes afterward.
A label conflict holds the result. The validator's success alone does not
declare2800; root applies every requirement above to the completed evidence.

No extension, pooling, favorable-repeat sampling, label adjustment or addition
of self-play gains to an old rating. If the valid fixed sample does not clear
the threshold, the goal remains unverified and development resumes from the
observed weaknesses. This run is not yet launched.

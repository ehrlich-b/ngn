# Durable net jobs

Run from the repository root with Python 3.12+ (stdlib only). Jobs and exports
are immutable, hash-bound JSON documents. `net_job.schema.json` describes the
three job types; `training_result.schema.json` describes the producer's terminal
receipt; `reservation.schema.json` describes resource admission. Paths in
artifacts are absolute. The state root must be outside the source checkout and
shared by all jobs in one experiment.

```
python3 -m scripts.wave2.net_job train.json --root /absolute/experiment/jobs --detach
python3 -m scripts.wave2.net_job await.json --root /absolute/experiment/jobs --detach
python3 -m scripts.wave2.net_job gate.json --root /absolute/experiment/jobs --detach
```

`--detach` returns immediately and starts a supervisor in its own session.
`--watch` stays attached, waiting only on receipt/admission state; without either
flag the command performs one step. A waiting gate holds neither stage lock.
PID is diagnostic only. Read `<root>/<id>/receipt.json` for the durable outcome.
A process crash that leaves RUNNING requires inspection and a new job identity;
neither a PID disappearing nor another invocation restarts that work. SUCCEEDED,
FAILED, BLOCKED and READY are terminal. A completed SHELVED gate is still a
successfully executed job. An incomplete/faulted gate is FAILED.

To construct a train job, bind `inputs.source` to an owned source tar archive,
include the training script/data/config as hash-bound inputs, and supply `argv`.
The training process runs in an extracted source snapshot, under at most two
reserved CPU cores, with no GPU and tensor threads=2. It receives
`NET_JOB_RESULT` and `NET_JOB_INPUT_SHA256`. Only after exporting a valid model
should it atomically publish the explicit SUCCEEDED result described by
`training_result.schema.json`. Every named export binds both its net and the
same source archive. Exit status zero alone never completes training. The
optional hash-bound `prerequisite` artifact must already certify READY or
SUCCEEDED. No credentials are forwarded to proof subprocesses.

Admission is an external, experiment-local contract: write `<root>/reservations.json`
atomically, following `reservation.schema.json`. Its `jobs[id]` entry must name
exactly the job's `input_sha256` (`net_job.identity(job)`), stage, ADMITTED status,
a nonempty ticket and one or two available CPU IDs. Absence or WAITING/REVOKED
means wait; identity mismatch means BLOCKED. The supervisor never edits a queue
or grants itself admission. Admission is captured before launch; later changes
do not stop admitted work. The training and gate stage locks are independent;
per-job locks prevent duplicate execution across concurrent supervisors.

Both await-result and gate jobs select one export by training id, complete job
input SHA256, export name, net SHA256 and source SHA256. Missing or failed
training blocks immediately without games. WAITING/RUNNING training is awaited.
The consumer rehashes only the explicitly selected export from a terminal
successful receipt and records that receipt as its prerequisite.

A gate additionally pins `inputs.base_source`, `inputs.base_net`, `book`, all
`claim_books`, and a retention budget. The baseline net must be exactly the
current H768 NGNN3 digest baked into the runner. Both engines and cmd/sprt are
built from the pinned source archives. The baseline source must include the
strict `-noretries` harness support. Source trees must match; the only exception
is an explicit selected evaluator change: list the exact implementation files
and SHA256 of the canonical `{path: [base_hash, candidate_hash]}` map. Only
`engine/ngnn*.go` and `engine/worker_evaluator.go` implementations are eligible.
Tests, search, startup, embedded defaults and harness changes cannot pass this
exception. Executable hashes, source/net identities, machine, CPUs, clock,
options and frozen book all enter the A/A control key.
The campaign result parser's SHA256 also enters the key, so a corrected fault
policy cannot reuse a control certified under the old parser.

Books require one full FEN per line. Supply the complete claim position inventory
as `claim_books` FEN exports; this is an input obligation, not inferred from
filenames. Positions are compared without halfmove/fullmove counters. FENs
avoid move-order/transposition ambiguity. The development book is copied and
hashed before play and must have no position in that inventory. Claim and
production data are never used by the fixture.

The runner uses campaign_queue's net class and cmd/sprt for exactly 1000 games,
500 reversed-color pairs, at 10+0.1, Threads=1, Hash=64, OwnBook=false,
UseNNUE=true and concurrency=6. It first reuses an evidence-bound matching A/A
control or runs exactly 200 A/A games. `-mingames 1000000` disables SPRT stopping
at this fixed sample. No live score polling, game retry, engine startup retry,
watchdog restart or automatic job rerun is permitted. Both printed and exact
paired lower 95% bounds must exceed zero, with zero operational faults. PGN
round/color/time-control accounting must agree with the summary.

A KEEP immediately writes a result and private adoption manifest, extracts the
selected source, embeds the exact gated bytes, updates DefaultNetSHA256, startup,
build and browser identities, and adds format/dimension and independent oracle
parity tests. It bundles that source before checking the consumer. The candidate
must pass four fresh-process FENs at depth4 with exact default-versus-EvalFile
evaluation, nodes, score, PV and bestmove identity, plus its format's trainer
parity and selected-net oracle fragment. A failed candidate keeps the KEEP result,
bundle and FAILED adoption manifest. Nothing updates the working checkout,
default play or remote services. Required evidence is preserved even above the
budget, with measured size and a keep reason.

Small reproducible proofs (no games, corpus, GPU, download or Elo claim):

```
python3 -m unittest scripts.wave2.test_net_job
python3 experiments/wave2-1104/net-job/run_fixture.py --output /absolute/evidence
```

The first uses owned training/UCI/match process fixtures. The second builds a
real consumer of the committed 24,704-byte H16 owned model, compares the committed
result, and retains source, bundle, logs and identity output outside the checkout.
Its source recipe explicitly includes Go/assembly build inputs and NGNN1 trainer
parity fixtures; original full-size embedded nets are unnecessary for this proof.
Synthetic KEEP results exercise adoption plumbing only. Real fixed-clock gates
and production admission belong to separately authorized runs.

The process tests also build the real `cmd/sprt` harness and invoke it through
`campaign_queue.run`. A reported warmup, post-ucinewgame readiness or in-game
search error must abort strict play and produce an UNSCORED result with faults.
Readiness failure returns engine-error before any position or search is sent.
The parser also rejects logged configuration errors in otherwise complete,
positive transcripts with exit zero; they cannot register an A/A control or
produce a KEEP.

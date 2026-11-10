# Experimental option jobs and defaults

The jobs directory is a separate serial queue, ordered with T1104ContPlies
first. It covers all 14 T1104/W2 switches in TunableSearchParams and the
default-off time switch W2ScoreTime. This source exposes one default-off time
switch; Move Overhead is a reserve in milliseconds with default 100, and the
three time tuning parameters are ordinary SPSA coefficients.

Each recipe pins the repaired queue source 7c2776ff, H768 network and existing
opening book. Both sides use the same engine, Threads=1, Hash=64, OwnBook=false
and explicit zeroes for every switch, except the candidate's selected switch
and any shared prerequisite. W2NullMargin enables W2NullVerify on both sides,
so its sole treatment difference measures the margin with verification active.
SPRT bounds are [0,5], alpha=beta=0.05, clock 10+0.1, minimum 200 and cap 20,000
games. The cap never implies acceptance: H0, INCONCLUSIVE and ERROR stay off.
An accepted margin row enables only W2NullMargin; verification still needs its
own accepted row before both become defaults together.
No recipe is a completed measurement. Prepare build/ngn-7c2776ff from the
declared source before separately activating this queue.

Validate without binaries, locks, services or games:

```
python3 scripts/match_queue.py run --jobs experiments/option-verdict/jobs --dry-run
python3 -m unittest scripts.option_verdict_test -v
```

The consumer reads only schema-valid terminal ledger rows and pinned recipes.
Only H1, with at least one complete pair, no faults and matching recipe hash,
source/binary/net/book/harness provenance and command, authorizes a default of 1. Other queue
families are ignored. Malformed, duplicate, truncated or mismatched rows fail
before output is touched. Generate a proposed source file explicitly:

```
python3 -m scripts.option_verdict --ledger experiments/ledger.jsonl --output /path/to/accepted_option_defaults.go
```

Inspect that generated file before replacing engine/accepted_option_defaults.go
and rebuilding. Its initialization sets both search variables and advertised
defaults; W2ScoreTime sets each new UCI time manager and its advertisement.
Generating defaults does not run matches, edit jobs, install services, commit
source or activate an engine. The committed generated map is empty, matching
the empty production ledger. Fixture ledgers are synthetic and never authorize
production adoption. Tests compile each fixture through a Go overlay and assert
the advertised enabled set equals exactly the fixture's H1 rows, then run the
existing short Go suite against each generated default map.

Ledgers can be transferred between checkouts. The consumer infers the recorded
checkout prefix from the candidate binary's exact recipe path and reconstructs
the complete command under that prefix, including the default build/sprt
harness. Base binary, net and book paths must share that prefix; options,
bounds and flags must still match exactly. Custom harness paths need separate
review and are refused. Provenance fields retain their original values and
schema validation; source/net/book hashes must equal the pinned recipe. Binary
and harness hashes must match independently trusted artifact identities. By
default the consumer hashes this checkout's recipe binaries and build/sprt;
missing artifacts fail closed. Both recipe roles use the same binary path, so
both recorded digests must match the same expected digest.

When artifacts are unavailable locally, supply --artifacts /trusted/artifacts.json
with this format (replace the example digests with actual SHA-256 values):

```json
{"version": 1, "binary_hash": {"build/ngn-7c2776ff": "<engine SHA-256>"}, "harness_hash": "<sprt SHA-256>"}
```

Obtain this manifest independently from trusted build artifacts, never by copying
hashes out of the ledger being checked. It is an explicit trust input and must
travel separately from an untrusted ledger. The synthetic artifacts.json beside
the test ledgers is exclusively a fixture and must never authorize production
adoption. Equal forged base/candidate hashes and changed harness hashes are
refused against either independent source. These checks are not signatures; the
ledger still needs trusted queue custody to authenticate its reported verdict.
No executable or artifact in the recorded checkout is opened or run, and no
binary is executed to compute its digest.

The offline node check compares the explicit original engine checkout to each
switch after an on/off transition on the three canonical nodecheck positions:

```
python3 -m scripts.option_identity --base-carry /path/to/pinned-queue-checkout --base-repo /path/to/original-engine --output /path/to/evidence
```

It uses HCE, one thread, 16 MB hash, cold state, depth 12 and no clock or games.
All 15 switches produce 45 printed comparisons. Node identity is a correctness
check and makes no claim about playing strength.
The optional base-carry argument requires the exact 7c2776ff queue checkout and
creates the original worktree at bc87352d if absent. Existing worktrees must
belong to that carry and be clean at the exact baseline; mismatches fail without
changing or deleting files. This makes the comparison replayable from a fresh
checkout without retaining a previous build directory.

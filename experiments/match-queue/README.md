# Standing match queue

`python3 scripts/match_queue.py run --dry-run` validates both committed jobs,
including net and book hashes, without a lock, engine binary, or game process.
Paths in recipes are relative to the repository; absolute paths are also
supported for prepared artifacts. Time-control bases must be at least one
millisecond; both base and increment milliseconds must fit a signed native
integer, so the harness cannot truncate the base to zero or overflow and use
fixed depth. Non-finite conversions are rejected as well.
Engine source hashes are build provenance
declarations; the runner hashes the actual binaries at execution time. Prepare
the declared source with its normal build process as `build/ngn-bc87352d` and
build `cmd/sprt` as `build/sprt` before executing these recipes. Changing engines
or other recipe fields requires a new job ID after a result has been recorded.

The first job is a fixed 2,000-game H768 A/A (minimum equals maximum). The second
is H768 as candidate against the earlier H512 network, with SPRT bounds [0,6].
Both explicitly set Threads=1, Hash=64, OwnBook=false, EvalFile and UseNNUE=true.
They reuse the committed 4,947-opening book. They are queued controls, not
completed measurements or a claim that either control passes.

`run` stays alive and scans sorted `*.json` files every second. `--once` drains
the current queue and exits. Add jobs with unique IDs by atomically publishing
complete JSON files in the jobs directory. The runner never waits for a parent
process. It opens the existing `session138/cpu0-11.lock` without creating or
truncating it, takes an exclusive flock before reading completion state, and
keeps that lock through every subprocess, ledger append, and report update in
the scan. It releases the lock while idle. Binaries are pinned by their actual
SHA256 in the ledger; net and book hashes are checked immediately before use.
`taskset` keeps games on the shared CPU set (default 0-11).

The harness inherits the lock descriptor so killing the controller cannot free
CPUs while the harness is still alive. The user service stops its entire
control group. The runner invokes strict `-noretries` mode and records a synced
ledger line per finished subprocess, including `ERROR` for operational faults
or missing summaries. An error stops the controller. No result automatically
adopts a candidate. Native H0/H1 verdicts and complete-pair counts are preserved;
orphan games are separately counted. Partial pairs can occur when native SPRT
stops with work in flight.

Successful restarts skip identical completed IDs. Changed recipes under an old
ID, truncated ledgers, and a pre-existing run directory without a ledger row
stop execution. A started but unfinished job needs operator inspection of
`started.json`, `sprt.log`, and running processes; it is never replayed
automatically. Failed jobs remain in the ledger and require a new ID to retry.
The queue accepts only trusted local recipes and binaries. Use immutable
artifact paths while games run. The ledger is authoritative: if report writing
is interrupted, regenerate it without playing games:

```
python3 scripts/match_queue.py validate --ledger experiments/ledger.jsonl
python3 scripts/match_queue.py strength --ledger experiments/ledger.jsonl --output docs/STRENGTH.md
```

The committed ledger is empty because no games have been played. Synthetic
ledger and two-job process fixtures are under `fixtures/`; they never populate
the production ledger. The generator reads only the ledger, uses no wall clock
or runtime files, and emits deterministic Markdown.

`ngn-match-queue.service` is an uninstalled systemd **user** unit for a checkout
at `%h/ngn`. Bryan prepares the binaries, verifies the existing rig CPU lock,
adjusts checkout paths if needed, and installs/starts it on the rig separately.
It needs no root access. It deliberately does not restart failed jobs.
Validate its syntax with `systemd-analyze --user verify
experiments/match-queue/ngn-match-queue.service`. On systems without that tool,
the unit test checks its sections and required directives instead.

Run bounded proofs with `python3 -m unittest scripts.match_queue_test -v`.
Set `MATCH_QUEUE_PROOF_OUTPUT` to an external directory to retain each proof's
runtime ledger, subprocess logs, lock observations and regenerated reports.

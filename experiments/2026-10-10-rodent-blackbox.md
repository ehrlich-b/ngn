# Rodent black-box diagnostic — prepared session47

Current plan experiment2, after the expanded-data net and provenance review.
No Rodent games or probe result has been produced here. Existing Rodent binary
`10dd722e…` only: never inspect its source or weights, rebuild it, or adopt it.
The diagnostic wrapper is isolated on `night/rodent-blackbox`, rebased onto
accepted`9bfbae4`, and uses the existing legal `uci.PlayGame` budget implementation.
It is not a playing-engine candidate. Freeze the eventual accepted engine/net
in a new concrete manifest; do not silently substitute an accepted new net.

The owned wrapper supports exact reversed pairs at a common node limit,
1s/move, or120+1. Its opt-in UCI transport records monotonic and UTC timestamps,
all sent commands/stdout, stderr and normal/abnormal process exits. Its
ordinary nonrecorded transport retains the existing behavior. The recorded
transport enforces silent-process deadlines and drains stdout/stderr before
recording the exit. Failed starts, forced shutdowns and incomplete samples
are retained; no hidden retries, omitted failed pairs or automatic adoption.
Record full moves and legal outcomes using the existing harness rules.

Freeze `cmd/blackboxmatch/positions.txt`:32 unique nonterminal FENs,
12 opening/12 middle/8 endgame, sourced from existing NGN diagnostic positions
and side-to-move variants. Freeze the first100 complete opening lines of
`974e4b5a…` for all cells before results. Both binaries use Hash64/Threads1/
book off; NGN loads the eventual nominated owned net, Rodent its existing
binary default net. All runs use forge4–11/nice10/GOMAXPROCS1, at most2
simultaneous games in the8GB allocation. Record `host_pinning:on` until the
current plan explicitly changes it. Pause SPSA at a durable checkpoint and
record all concurrent workloads. No rig CPU run, pooling or claim.

Order, with one concrete frozen manifest before starting:

1. Same-position throughput: exactly3 alternating-order repetitions of both
   binaries at1s and5s per FEN (384 searches). Log wall time, reported nodes,
   UCI time/depth and legal bestmove. Require natural return within budget
   plus200ms; an overrun stops the diagnostic and preserves its record.
   Freeze the common node budget as the integer median of the96 NGN1s
   reported counts, using `review.py budget`. Startup/warmup is logged
   separately and is excluded from this median. Do not tune the node budget
   after scored play. Nodes are engine-defined, not standardized work units.
2. One32-position node-compliance repetition (`-mode compliance`, same node
   limit both sides). Require legal bestmove, reported nodes>0 and overrun
   no greater than max4096 nodes/1% of budget. This tolerance is declared
   before results; it does not promise exact implementation-level work.
3. Fresh40-game NGN A/A for each equal-node/equal-time cell at the exact forge
   CPU/concurrency/budget and pinned state. Require complete20 reversed pairs,
   zero operational failures and paired95 interval containing0. Then exactly
   200 games/100 reversed pairs per cell using the same100 openings. Run
   cells sequentially, with their own directories and complete trace checks.
4. Fresh40-game120+1 A/A at the same topology, then exactly24 profile games/
   12 reversed pairs. Collect banks, move times and flags from timestamped UCI;
   Use the separately built `ngnclocktrace` NGN binary for this cell, with
   its own tagged-binary A/A. This compile-time diagnostic attaches the
   existing observer to the ordinary UCI/search session and emits exact
   actual nodes, completed-iteration times, first termination and tail
   before bestmove. Default builds inline empty hooks. Require fixed-node
   UCI identity against the actual accepted binary before nomination;
   the clock sample diagnoses policy and is never strength evidence.

Compare cells by100 per-opening differences of the two color-averaged scores.
Student-t95/df99/critical1.9842169515086827. Support a material throughput
contribution only if the equal-node Elo deficit improves by at least10 Elo
and the paired difference's lower95 score bound>0. Otherwise report unresolved.
Persistent equal-node loss keeps evaluation/search first. Clock profiles can
justify a policy experiment, never a keep. All traces must have normal exit0,
no flags/crashes/illegal/no-move outcomes, complete legal pairs, intact pins
and compliant per-move budgets. No sample extension or cell subtraction from
the historical120+1 Rodent match; this is not a causal Elo decomposition.

On the host-pinned forge allocation, these cells may take a day rather than
Astra's3–6-hour c6 rig estimate. This stage waits behind the higher-priority
net and provenance decisions. Record real throughput when nominated; no
competing new behavior gate or automatic queue launch.


Session47 checks complete on actual accepted9bfbae4 at10:46, scope5–7/
2GiB/nice10: full short, full short race, tagged full short race, normal/
tagged vet, three PGO builds, three paired-review controls and12 actual UCI
50000-node accepted/normal/clock identities with exact observer node counts.
All13 source files match the checked receipt at publication. Topic
**30b8d4d** is pushed to privateorigin on `night/rodent-blackbox`; ordinary
playing source stays9bfbae4. No Rodent probes/games or settled net nomination.
[Receipt](2026-10-10-session47-results/blackbox/receipt.json).

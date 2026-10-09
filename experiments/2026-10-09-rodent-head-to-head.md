# Accepted NGN versus Rodent V 1.2 — overnight head-to-head

Bryan's [00:05 EDT priority](/Users/ehrlich/repos/ngn-campaign/steer-rodent.md)
requests as many complete 120+1 pairs as finish by **07:45 EDT October9**, at
least160 games. This is an external comparison, no engine-change acceptance,
campaign claim or publication. Report complete-pair W/D/L, score, penta Elo/95%
interval, flags/crashes and performance relative to the supplied3532 label.
One opponent at one TC supplies a performance comparison, not a rating.

Accepted NGN is runtime5b882e2 /NGNN3 KB8 H512 be34532c, the previously built
session18d baseline6550e98d. Rodent is freshly cloned/built **Rodent_v_1.2**,
commit **b53ffaf670590932957cb63b7b6d871f6f33b7d8**, from its
[official release](https://github.com/nescitus/Rodent-V/releases/tag/Rodent_v_1.2).
Its default768×4KB/8OB networkc35a1abc is match-only. Source, personalities,
net and executable remain in `session19-rodent/opponent-only/`, outside NGN.
No third-party code or net enters NGN or a branch. AVX2 Linux build,
Go1.25.5 /GOAMD64=v3, one build worker on CPU15/nice19.

Current run `/home/ehrli/ngn-data/night-20261007/session19-rodent3/` holds the full
build receipt, advertised UCI options, pre-game manifest and input hashes.
Both engines: Hash64, Threads1, OwnBookfalse; NGN explicitly loads the
accepted net, Rodent explicitly disables UCI_LimitStrength. Standard opening
file974e4b5a, each opening played twice, colors reversed. Six workers on
CPU0–11 /nice19 /GOMAXPROCS1. No score adjudication: normal chess rules,
existing300ms flag grace and usual400-ply draw cap.

The NGN-owned `cmd/pairmatch` reuses the unchanged UCI/rules instrument but
schedules each pair as one work unit. Exactly one JSONL row records each
completed or failed pair, its opening, both outcomes/reasons and move lists.
No repeated openings or SPRT stopping. Each game has a12-minute watchdog;
no new pair is admitted after **07:20**, leaving25minutes to finish admitted
pairs before07:45. A watchdog excludes its incomplete pair and stops admission;
the summary reports only completed pairs and flags operational faults.
Retain all raw receipts and independently recompute the penta before reporting.
If fewer than160 games complete, report the shortfall; no extension/partial claim.

Pause SPSA only at its next25-iteration durable checkpoint, using the existing
inotify/pidfd waiter, save its full state and terminate only its exact process
group. Resume the same unfinished old-base tune after the match. During this
window **SIGSTOP only factorizer bash controller630389**, whose trainer630440
keeps running independently on12,14/GPU; this holds parity/A/A/gate and both
dependent CUDA windows until the match ends. The controller releases it with
SIGCONT after restoring SPSA, even on a match failure. No pinned factorizer
scripts/inputs change and no trainer interruption or automatic net adoption.

Results: `pairs.jsonl`, `summary.json`, `match.log`, `checkpoint-ready.json`,
`tuner-before.jsonl`, `input-check-{before,after}.txt`, `controller.{exit,done}`.
The campaign log records actual PIDs and stage after detached launch.

First two scheduling attempts played zero games: an exited-pidfd open failure
after iteration825 drain, then a global guard rejecting disjoint TT13,15. Their
receipts remain session19-rodent/session19-rodent2. Final667005 waits resumed
SPSA666877 at850, opens its pidfd before TERM, and allows disjoint game jobs.
The original opponent-only build/source is reused via a local symlink, no rebuild
or repeated experiment. Exact latest argv/input hashes are in the session19
[pre-game manifest](2026-10-09-session19-results/rodent-game-manifest.json).

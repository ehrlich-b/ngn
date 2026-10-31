# NGN >3000 claim tooling — 2026-10-06

This implements the [predeclared protocol](2026-10-06-3000-claim-protocol.md).
All new instrument code is NGN-written, using NGN's independent board/move
layer and Go stdlib. Opponents supply measurement binaries only. No earlier
third-party oracle implementation/toolchain is used. The abandoned untracked
`internal/claim/` skeleton was reviewed, retained and substantially hardened.
No candidate training, engine search/evaluation change, screen or confirmation
is part of this work. Net1 is used only for a tooling check; it is not nominated.

Admitted records use NGN-CLAIM-2, a unique cryptographic pre-run attempt ID and
UTC game start/end times. Audit rejects missing/mismatched IDs and any game
predating its receipt, so records from older attempts cannot be mixed unchanged.
The runner creates a fresh game directory and never imports/replays old results.

Mac: editing/git only, `/Users/ehrlich/repos/ngn-claim`, local `claim/tooling`
from `06beca9`. WSL: independent detached worktree
`/home/ehrli/wt/claim-tooling-20261006`. Changes travel as Git bundles through
SSH; nothing is pushed. The other worker's checkout/worktrees and T80 directory
are untouched. Every remote command inherits `taskset -c 13,15 nice -n 19`.
Go commands use Go 1.25.5, GOAMD64=v3, GOMAXPROCS=2, GOTOOLCHAIN=local,
GOPROXY=off, `-p 2`; engine children use GOMAXPROCS=1. The private Go cache and
all outputs are under `/home/ehrli/ngn-data/claim-tooling-20261006/`.

## Instrument contract

`cmd/claimtool` provides `identity`, `prepare`, `run`, `audit`, `analyze`,
`recheck`. Its executable digest freezes all collector/audit/analysis code.
Identity JSON binds absolute binary/net paths, hashes, source commit and clean
checkout, exact build argv and manifest/data/training/export/parity artifacts.
Preflight checks embedded Go/VCS/GOAMD64 metadata and excludes external module
dependencies. All persistent advertised options are explicitly frozen, including
Move Overhead 100. EvalFile precedes UseNNUE. Unknown options are never sent.
Candidate static eval and effective Threads=1 receipts are required.

Receipt creation is exclusive, fsynced and read-only. `receipt-seal.json` binds
its digest; every subsequent command checks the seal. The receipt pins Counter
3.8/5.5 against protocol digests, Maelstrom and adjacent/advertised network
files, full owned opening file and canonical first-80 slice, and checks every
one of those openings legally with complete history. Pre/post hash checks cover
all binary/net/tool/opening/provenance/dependency artifacts.

`cmd/claimbridge` is an NGN-written Windows stdio bridge. Linux affinity does
not constrain Windows children. The bridge pins itself to Windows CPUs 13/15
and Idle priority before starting Counter, then verifies the child's affinity
0xa000 and priority 0x40 before forwarding UCI input. A Windows job object
kills the child if the bridge is killed or closed. The bridge digest is frozen.

The fixed runner has no score stop or score adjudication. A record for every
game includes opening index/initial FEN/history, reversed-color pair ID,
candidate color, every move/STM/role, both clock banks before/after, elapsed
nanoseconds and exact go/bestmove transcript indices, terminal result/reason,
engine hashes/options/launch/scheduling, complete timestamped stdin/stdout/stderr
and exit status. Operational failures preserve invalid records and fail the
whole run. The collector does not claim success until independent replay passes.

Audit independently replays opening and played moves, tracks exact normalized
FEN repetition keys (including only legal EP) and recomputes halfmove clocks.
It checks legal membership, STM, real clock/increment arithmetic, every terminal
and its priority over the cap, UCI lifecycle and complete raw search coverage.
It enumerates every declared game/color/index/pair, derives W/D/L from White
results and reconciles the collector summary. Missing/extra/duplicate records,
illegal openings/moves, missing moves, engine errors, nonzero exits, clock
losses, no-move/forfeit/watchdog records and terminal mislabels invalidate the
entire sample; exclusions are always zero.

Analysis replays and hash-checks the persisted audit, independently recounts
pairs/results, and implements the protocol's displayed primary formula verbatim:

```text
p_i = (W_i + 0.5*D_i + 0.5)/(N_i + 1)
R_i = label_i + 400*log10(p_i/(1-p_i))
v_i = (400/ln(10))^2/(N_i*p_i*(1-p_i))
weight_i = 1/v_i
R_pool = sum(weight_i*R_i)/sum(weight_i)
SE_pool = 1/sqrt(sum(weight_i))
CI95 = [R_pool - 1.96*SE_pool, R_pool + 1.96*SE_pool]
```

Every anchor's W/D/L/N/pairs/cap draws, raw score, p, label, performance,
variance and weight are reported. Eligibility requires strict raw bracketing,
strict lower bound >3000, exact fixed pool/sample/pairs and a clean audit.
A screen cannot create a claim; a tooling check cannot pass either gate.
Supplemental bootstrap uses whole opening vectors jointly across anchors,
retains both colors, and uses 10,000 draws and seed 3000. Confirmation receipt
creation recomputes the admitted screen analysis and requires unchanged
candidate, tool, bridge, anchors and options; a forged eligibility flag fails.

Scripts `scripts/claim/screen.sh` and `confirmation.sh` take `--candidate ID.json`
containing engine binary/net paths and hashes, plus tool/build/source/bridge
inputs. Live execution writes the receipt first. `--dry-run` checks metadata,
hashes and all 80 openings, and writes a fixed plan without engines/games;
it cannot masquerade as a completed pre-run receipt. Screen stays 10+0.1/c8,
80 per anchor/indices 0–39; confirmation stays 120+1/c8,160/0–79.
Both need separate user authorization before `--execute`. Neither game launch
is authorized or performed here. `identity-check` is a zero-game mode that
checks all anchors and bounded A/A probes; its receipt cannot launch games.

## Questions and conservative readings

- Does 400 plies include the forced opening? Implemented as total opening plus
  played plies from initial FEN; mate/stalemate/rule terminals always run first.
- Are threefold/50-move draws claim-based? Implemented as automatic at three
  occurrences/100 halfmoves, retaining full initial history. Mate/stalemate win
  precedence. Standard dead-material sets are automatic; KNN versus K and
  opposite-color bishops stay live.
- What constitutes a usable opening line? Blank/comment lines are skipped;
  malformed/illegal lines reject the input rather than being silently replaced.
  The canonical slice trims surrounding whitespace and uses LF with final LF;
  its own digest is recorded alongside the untouched full file digest.
- What is the bounded A/A control? Same candidate/options/declared clocks and
  total concurrency, two 50ms stop-response probes per lane, maximum 500ms
  response and maximum 250ms paired timing difference. This is clock plumbing,
  not an A/A game-strength/parity verdict. A one-minute host-load bound of 16
  and the probes must pass; this operational choice changes no statistics.
- Bootstrap interval construction and screen count are unspecified. Use a
  supplemental percentile interval with type-7 interpolation; screen resamples
  its actual 40 indices. Confirmation uses exactly 80,10,000,seed3000.
- October 5/6 orientation uses pair variance, while the current protocol's
  displayed formula specifies binomial v_i. The displayed formula above is
  implemented exactly; orientation data/estimators are not imported.

## WSL validation and commands

Validation results and exact executable/run receipts are recorded below after
completion. Historical oracle packages/commands are omitted from broad baseline
runs to obey this task's ban; no external-oracle environment is configured.

Build/receipt setup uses the same SSH route as requested, stripping NULs:

```sh
ssh -o BatchMode=yes -i ~/.ssh/id_ed25519 ehrli@192.168.4.108 \
  'wsl.exe --exec bash -l' <<'WSL' | tr -d '\000'
exec taskset -c 13,15 nice -n 19 bash -l <<'PINNED'
# Exact command blocks below run inside this pinned shell.
PINNED
WSL
```

The installed Go VCS detector did not stamp binaries in the linked worktree,
even with explicit `-buildvcs=true`; preflight rejected them before engines or
games. The identical committed source is therefore built in its own standalone
Git worktree `/home/ehrli/wt/claim-build-20261006` (local shared clone; no network
remote involved). This gives embedded `vcs.revision` and `vcs.modified=false`.
The rejected `frozen/` and `frozen-v2/` artifacts remain preserved unadmitted.
A further preflight rejected its own `timeout` wrapper as another match, before
engines/games. The guard now excludes only its process ancestry, retains every
foreign matching process, and has positive/negative controls. It also rechecks
live host load immediately before the game directory is created.

All new tests run on WSL. Baseline commands:

```sh
export GOAMD64=v3 GOMAXPROCS=2 GOTOOLCHAIN=local GOPROXY=off
export GOCACHE=/home/ehrli/ngn-data/claim-tooling-20261006/go-cache
unset NGN_MOVEGEN_ORACLE
ROOT=/home/ehrli/ngn-data/claim-tooling-20261006
cd /home/ehrli/wt/claim-build-20261006
/usr/local/go/bin/go test -p 2 -short ./internal/claim ./cmd/claimtool -count=1 -v
/usr/local/go/bin/go list ./... > "$ROOT/packages-all.txt"
rg -v '/(scripts/counter55-oracle|internal/oracle|cmd/(gametest|sfpv))($|/)' \
  "$ROOT/packages-all.txt" > "$ROOT/packages-tested.txt"
xargs /usr/local/go/bin/go test -p 2 -short -count=1 < "$ROOT/packages-tested.txt"
xargs /usr/local/go/bin/go test -p 2 -short -race -count=1 < "$ROOT/packages-tested.txt"
/usr/local/go/bin/go vet -p 2 ./internal/claim ./cmd/claimtool
```

Fixture assertions that passed include:

| Fixture | Required result |
|---|---|
| Fool's mate, valid result | Admit `0-1/checkmate` |
| Same mate labelled draw/cap | Reject terminal mismatch |
| Repetition with first occurrence in opening | Detect third occurrence using full history |
| Total ply 399 versus 400 | Live at 399, cap draw at 400 |
| Mate/stalemate/repetition/fifty/dead material at cap | Natural terminal outranks cap |
| Halfmove 99→100; capture/pawn reset | Draw at 100; resets remain live |
| Pinned versus legal EP; changed STM/castling | Correct distinct repetition keys |
| 120+1, 50ms move | Moving bank 120,950,000,000 ns; opposite bank unchanged |
| Missing/extra move/search/transcript; wrong STM/used time/increment | Reject |
| No-move/watchdog/clock-loss/panic/error/terminal relabel | Reject |
| Windows bridge receipt absent/changed mask | Reject |
| Changed receipt clock/coverage/options/labels/clean flags; altered seal | Reject |
| Missing/wrong attempt ID; game predating receipt; missing/backward end time | Reject |
| Declared NGNN1/2/3 versus silent HCE | Verify exact backend; reject fallback |
| Four games 2W/1D/1L, label 3000 | raw .625, p .6, R 3070.4365036222725 |
| Weights .25/1, R 3000/3050 | pool 3040, SE .8944271909999159, CI [3038.2469227056403,3041.7530772943597] |
| Lower CI exactly 3000; all raw scores .5/one-sided | Fail eligibility |
| Joint resample [1,1,0] | Duplicate complete anchor/color vectors |
| 80 constant split-color pairs, 10,000 draws/seed3000 | Bootstrap endpoints both 9670/3 |

Provenance for net1 is frozen into small copies under `frozen/provenance/`:
gen0/gen0b generation manifests, snapshot and prefix hashes, record counts,
training config/history, export/results, parity and selected-net receipt. The
manifest binds all those copies and their original paths. No training data
shard or checkpoint is copied, trained, changed or imported as game evidence.

The live NGNN2 probe initially found an overly broad NGNN1 expectation in the
receipt validator. This was corrected to match the actual declared NGNN1/2/3
format, with explicit negative controls for silent HCE fallback. The final
zero-game identity probe verified Maelstrom v3.3.0 digest
`66ea969530d63f5904114079f518ba618b0deb4e269eb0c95d7496960379ea53`;
that exact executable is now pinned in admission code, with labels unchanged.
No adjacent or advertised Maelstrom network dependency was present.

An initial four-game pipeline passed under the earlier record schema. Final
audit review added pre-run attempt binding and absolute game times; the final
validation uses a fresh four-game attempt and does not pool the two. There are
eight tooling games total, all net1 versus Counter 5.5 at 1+0.02/concurrency1,
opening indices0–1 with both colors, no adjudication/score stopping and cap400.
The first sample remains under `tooling-check/`; the final admitted schema2
sample is under `tooling-check-final/`. Neither is a screen or strength sample.

## Admitted final validation

Final source/tool commit: `c5b4ffe569c36a86105d8770407ececfebaf28ef`.
The clean standalone WSL worktree remains at this commit, preserving the exact
build identity after the report commit. No engine/trainer source changed from
`06beca9`; all changes are new tooling/tests/scripts and this report.

Final outputs are `frozen-v7/`, `dry-screen-final-code/`,
`dry-confirmation-final-code/`, `identity-check-final/`, and
`tooling-check-final/` under `/home/ehrli/ngn-data/claim-tooling-20261006/`.
`identity-check-final/receipt.json` contains all three handshakes, exact options,
Windows scheduling proof, evaluator probes, 80-opening legality/slice receipt
and 16 A/A clock probes at 120+1/concurrency8 (maximum response 74.646 ms;
one-minute host load 2.41). Its post-probe and post-tooling
hash receipts pass. The original opening SHA is unchanged; frozen slice SHA:
`6e42b15c5c57c17905d28c1c0cb7e83d558f8092bd5fc672c421ccf25e419919`.

The final tooling receipt was written at `2026-10-07T02:45:23.930281813Z`,
before games at `02:45:24.151812033Z`–`02:45:38.236905566Z`.
Attempt ID: `1caa113f3faf16b4c357e270bd361e4d`.
All four final games/two intact color pairs passed: **0W/0D/4L**, all checkmates,
436 played plies plus 24 legally replayed opening plies. No cap draws, errors,
flags, voids, exclusions or missing records. The first four-game tooling sample
had 404 played plies/24 opening plies, likewise four checkmates and zero anomalies.
These two samples are separate, eight games total, and are never pooled.
Analysis recomputed the exact primary formula and a 2-index/10,000-draw/seed3000
supplement solely to exercise the pipeline; screen/claim eligibility are false.
Confirmation still requires its fixed 80 indices and full 160-per-anchor sample.
No nominated candidate, screen, confirmation, >3000 claim or official rating is
admitted by this work.

| Final artifact | SHA-256 |
|---|---|
| frozen-v7/ngn | `24dd06fa3c5076ee251cbe0622fe5f42bc99ff0b5b4d4893eaca133c61cfffab` |
| frozen-v7/claimtool | `677689450bfc898d876aecb0496e8f24698edc1e45d6b686e8abf1abd03807cf` |
| frozen-v7/claimbridge.exe | `2de8a0287d4ecf81f070bc3d3e3521b2cfeb1b2700cad829bda9286ec81c06a5` |
| tooling-check-final/receipt.json | `e7f3f54b3913e376b6b778b4d296bed9153f22991bd13852f3014378b35f84b0` |
| tooling-check-final/audit.json | `1d2139e4382875bdcb0e6508c50026b4744ac97cae825b9bf2fb882a54179196` |
| tooling-check-final/analysis.json | `28cffb277d0a5e84991a97584d594855e5ca49a5c735f62133a3e6980160302a` |
| tooling-check-final/post-run-hashes.json | `f6267a39366452a1cc1d67caf82de57fdae66e035449be115e444ff5eefb872f` |

Final-source validation logs: `narrow-tests-complete.log`,
`short-tests-complete.log`, `race-tests-complete.log`, `vet-complete.log`.
All pass. Later formatting changes only whitespace. Linux engine/claimtool and
Windows/amd64 bridge builds pass with stamped commit/clean tree/Go1.25.5/v3.

Additional saved-record negative audits, without new games:

| Changed copy | Actual rejection |
|---|---|
| mate labelled draw/cap | terminal mismatch: replay 1-0/checkmate, record draw/ply-cap |
| missing game | missing 1 games |
| duplicate record | unexpected/missing-record file: duplicate.json |
| edited collector W/D/L | collector W/D/L reconciliation failed |
| wrong receipt attempt ID | game belongs to another pre-run attempt |

Copies and rejection logs remain under `negative-*`. Original game/receipt
files were not edited. Existing binaries and all frozen inputs rechecked
unchanged after the games. The final process check found no match or own frozen
binary process; every engine/bridge session exited zero. There is no detached
job, server or monitor from this task. New retained disk use is 378 MiB of
receipts/binaries/private cache plus 24/25 MiB for the two owned source trees,
well below 5 GB. Nothing outside these own directories was deleted.

Exact successful final build and run commands (inside the pinned SSH shell):

```sh
export GOAMD64=v3 GOMAXPROCS=2 GOTOOLCHAIN=local GOPROXY=off
export GOCACHE=/home/ehrli/ngn-data/claim-tooling-20261006/go-cache
export NGN_CLAIM_DEADLINE_UTC=2026-10-07T11:00:00Z
ROOT=/home/ehrli/ngn-data/claim-tooling-20261006
WT=/home/ehrli/wt/claim-build-20261006
cd "$WT"
/usr/local/go/bin/go build -buildvcs=true -p 2 -o "$ROOT/frozen-v7/ngn" .
/usr/local/go/bin/go build -buildvcs=true -p 2 -o "$ROOT/frozen-v7/claimtool" ./cmd/claimtool
GOOS=windows GOARCH=amd64 /usr/local/go/bin/go build -buildvcs=true -p 2 \
  -o "$ROOT/frozen-v7/claimbridge.exe" ./cmd/claimbridge
TOOL="$ROOT/frozen-v7/claimtool"
"$TOOL" identity --engine "$ROOT/frozen-v7/ngn" \
  --engine-sha 24dd06fa3c5076ee251cbe0622fe5f42bc99ff0b5b4d4893eaca133c61cfffab \
  --net "$ROOT/frozen-v7/net1.nnue" \
  --net-sha 6a203ffe128541ef422f9344f2906bffbebea6d93931bb1f50e27f44bf2beff6 \
  --source "$WT" --build-argv "$ROOT/frozen-v7/ngn-build.json" \
  --provenance "$ROOT/frozen/provenance.json" \
  --out "$ROOT/frozen-v7/net1-tooling-identity.json"
for kind in screen confirmation; do
  bash "$WT/scripts/claim/$kind.sh" \
    --candidate "$ROOT/frozen-v7/net1-tooling-identity.json" --source "$WT" \
    --out "$ROOT/dry-$kind-final-code" --tool "$TOOL" \
    --windows-bridge "$ROOT/frozen-v7/claimbridge.exe" \
    --tool-build-argv "$ROOT/frozen-v7/claimtool-build.json" --dry-run
done
pgrep -af 'sprt|cutechess|fastchess' || true
timeout 120s "$TOOL" prepare --mode identity-check \
  --candidate "$ROOT/frozen-v7/net1-tooling-identity.json" \
  --out "$ROOT/identity-check-final" --source "$WT" \
  --windows-bridge "$ROOT/frozen-v7/claimbridge.exe" \
  --tool-build-argv "$ROOT/frozen-v7/claimtool-build.json"
"$TOOL" recheck --receipt "$ROOT/identity-check-final/receipt.json" \
  --out "$ROOT/identity-check-final/post-probe-hashes.json"
timeout 60s "$TOOL" prepare --mode tooling-check \
  --candidate "$ROOT/frozen-v7/net1-tooling-identity.json" \
  --out "$ROOT/tooling-check-final" --source "$WT" \
  --windows-bridge "$ROOT/frozen-v7/claimbridge.exe" \
  --tool-build-argv "$ROOT/frozen-v7/claimtool-build.json"
pgrep -af 'sprt|cutechess|fastchess' || true
timeout 180s "$TOOL" run --receipt "$ROOT/tooling-check-final/receipt.json" \
  --games "$ROOT/tooling-check-final/games" --out "$ROOT/tooling-check-final/run-status.json"
"$TOOL" audit --receipt "$ROOT/tooling-check-final/receipt.json" \
  --games "$ROOT/tooling-check-final/games" --out "$ROOT/tooling-check-final/audit.json"
"$TOOL" analyze --receipt "$ROOT/tooling-check-final/receipt.json" \
  --games "$ROOT/tooling-check-final/games" --audit "$ROOT/tooling-check-final/audit.json" \
  --out "$ROOT/tooling-check-final/analysis.json"
"$TOOL" recheck --receipt "$ROOT/tooling-check-final/receipt.json" \
  --out "$ROOT/tooling-check-final/post-run-hashes.json"
"$TOOL" recheck --receipt "$ROOT/identity-check-final/receipt.json" \
  --out "$ROOT/identity-check-final/post-tooling-hashes.json"
```

Receipt/result files are exclusive and read-only; these output paths already
exist. The commands are a run receipt, not instructions to repeat games. Future launches take a new frozen candidate
identity and fresh output directory. Confirmation additionally requires the
matching admitted screen receipt/analysis and separate user authorization.

## Prospective v2 run configuration — 2026-10-07

The [v2 protocol](2026-10-07-3000-claim-protocol-v2.md) adds the official Linux
Viridithas 10.0.0 / historical CCRL Blitz 1CPU 3557 anchor, pinning its embedded
network through binary SHA-256
`35b32c4acc3216fd9d1084bebad90452cee2e8725af4bdd0f175cc42c4e3def3`.
The original three versions, labels and digests are unchanged. Screen and
confirmation plans now contain four anchors and still require exactly 80 and
160 games per anchor, respectively. All decision statistics and legal/terminal
rules remain identical; the whole-opening bootstrap includes all four anchors.

Restricted preparation remains CPU13,15/nice19. Once the T80 worker exits,
live receipts and games require the exclusive rig CPU0–15/nice19; the Windows
bridge proves job affinity0xffff and Idle priority before any UCI input reaches
Counter3.8. The process guard additionally rejects training/datagen. The old
Counter-only tooling mode stays Counter-only. Two explicitly unscored
Viridithas stability modes declare four games each, 10+0.1 and 120+1, indices0–1
both colors, with the same full replay and terminal audit. They never become
screen/claim evidence. The execution lock and all new artifacts are under
`/home/ehrli/ngn-data/claim-run-20261007/`.

Owned claim fixtures passed on restricted WSL before committing this revision,
including new-anchor substitution rejection, fixed counts, strict boundaries,
joint bootstrap, legal/terminal and Windows scheduling controls. Preparation
log: `preparation-tests.log`. Clean Go1.25.5/v3 builds and both four-anchor
dry runs passed at `d4110d5`; plans are `dry-screen-pool/dry-run.json` and
`dry-confirmation-pool/dry-run.json`. Their counts are 40/80 indices, eight
total game workers and labels 2994/3317/3359/3557. Full baseline checks and
free-rig stability checks precede any screen launch. Bryan's
October6 goal/overnight instructions and October7 task supply the screen and
confirmation authorization recorded in v2; the older report's lack of run
authorization describes the historical tooling-only task.

The sequential `scripts/claim/run-queue.sh` admits confirmation only after the
screen analysis and post-run hash recheck pass. The claimtool independently
replays that screen again when admitting confirmation. Any operational error
ends the queue and writes an immutable explicit NO CLAIM disposition; a failed
statistical screen ends successfully without launching confirmation. There is
no job deadline, score stop, replay, extension or sample pooling. The queue's
shell syntax passed on restricted WSL before admission.

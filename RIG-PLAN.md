# One conditional rig candidate: P1-R1

**Blocked on implementation, not authorized for immediate launch.** The pinned
source contains the admitted P1-R1 contract, not its patch. The next forge2 lane
must supply one immutable candidate commit and passing mechanism receipts. No
candidate SHA, binary hash, timed result, or rig availability is invented here.
This bringup makes no engine change and never contacts the rig.

Use P1-R1 because accepted-H1 profiling found 6.91% ordinary generation/scoring
cost. This is a measured opportunity, not an expected Elo number. Do not test
already accepted H1 as though it were pending, or revive the rejected June
TT-first/global-fallback picker. Source of the rule:
`experiments/2026-09-19-p1-staged-picker-profile/prototype-plan-v1.md`.

Before shipping a candidate, forge2 must prove exactly-once generation and
legality/lifecycle coverage, pass short and race suites, and pass BOTH frozen
screens: cold/warm fixed-depth geometric-mean node ratio <=1.05, no cell >1.15;
ten interleaved six-position 400,000-node blocks with paired aggregate median
candidate/base time ratio <=0.97. No second architecture or quiet-only generator
rescue is admitted. The candidate must differ from the immediate accepted
`8ba878cb03a7eb81ca465b3d562953d0f45e81c7` engine only by P1-R1. If a later
accepted engine becomes the base, freeze a new plan before building either role.

## Protocol

Only opponents: P1-R1 NGN versus accepted-H1 NGN; same exact Rodent V1.2 network.
Both: Threads=1, GOMAXPROCS=1, GOAMD64=v3, CGO_ENABLED=0, Hash=128 MiB,
OwnBook=false, Move Overhead=100 ms, **10+0.1**, concurrency=4,
physical affinity masks **0,2,4,6**, reversed colors, six opening plies, no
ponder, tablebases, contempt change, or score adjudication. Maximum eight active
engine workers. Inspect the rig's physical topology before freezing those masks;
if they are not four distinct suitable physical cores, amend the plan before
scores. Shared-host telemetry remains visible and other workloads stay running.

Fresh A/A: base/base, 50 pairs/100 games, paired-bootstrap 95% interval must
contain zero. Pilot: candidate/base, fixed 64 pairs/128 games, canonical lines
1–64. Reject if upper95 Elo <0; also reject a non-positive point estimate unless
the independently recorded mechanism receipts establish an advantage beyond
the admission thresholds. The conservative executable below shelves every
non-positive pilot (no such additional evidence exists in this lane).
Otherwise confirm on **independent lines 201–400**, 200 pairs/400 games.
Accept only if confirmation's whole-opening-pair bootstrap **lower95 Elo >0**
and every legal, terminal, process, supervision, trace and telemetry gate passes.
Use 100,000 resamples, seeds 2026102301/02/03 for A/A/pilot/confirmation.
Never pool pilot with confirmation, extend at the cap, select another candidate,
or tune after scores. Any operational failure invalidates the run; no adoption.

Expected games: 628 maximum. Historical E0 took about 69 minutes for 500 games
on the old 9800X3D; budget **1.5–2 hours plus build/audits**, with hard phase caps
30/30/120 minutes (three hours total, plus audits). This is a planning estimate,
not a measurement of the restored rig. No absolute 3300/3600, SMP-strength,
installation or release claim follows from this match.

## Coordinator commands (Mac controls; all computation runs in rig WSL)

After the future forge2 candidate passes, transfer its committed object via a
bundle using the coordinator's ordinary transport. Do not push from forge2.
Save the next bash block as `rig-p1-r1.sh` on the Mac. Set the one required value
to that full candidate SHA, then send the script to WSL; do not run engines on
the Mac. These are **future coordinator commands**, not actions taken here:

```bash
: "${P1_CANDIDATE:?set the full SHA from the passed P1-R1 forge2 lane}"
[[ "$P1_CANDIDATE" =~ ^[0-9a-f]{40}$ ]]
{ printf 'export P1_CANDIDATE=%q\n' "$P1_CANDIDATE"; cat rig-p1-r1.sh; } |
  ssh -T ehrli@192.168.4.108 'wsl -d Ubuntu -e bash -s'
```

```bash
set -euo pipefail
: "${P1_CANDIDATE:?missing passed forge2 candidate commit}"
[[ "$P1_CANDIDATE" =~ ^[0-9a-f]{40}$ ]]
BASE=8ba878cb03a7eb81ca465b3d562953d0f45e81c7
SOURCE=/home/ehrli/repos/ngn
RUN=/home/ehrli/ngn-rig-p1-r1-20261023
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GOMAXPROCS=8 GOAMD64=v3 CGO_ENABLED=0
test "$(git -C "$SOURCE" rev-parse "$P1_CANDIDATE^{commit}")" = "$P1_CANDIDATE"
test "$BASE" != "$P1_CANDIDATE"
mkdir "$RUN"
git -C "$SOURCE" worktree add --detach "$RUN/base-src" "$BASE"
git -C "$SOURCE" worktree add --detach "$RUN/candidate-src" "$P1_CANDIDATE"
nice -n 10 go build -C "$RUN/base-src" -p=2 -trimpath -o "$RUN/ngn-base" .
nice -n 10 go build -C "$RUN/candidate-src" -p=2 -trimpath -o "$RUN/ngn-p1-r1" .
nice -n 10 go test -C "$RUN/candidate-src" -p=2 -short ./... -count=1 > "$RUN/short.log" 2>&1
nice -n 10 go test -C "$RUN/candidate-src" -p=2 -short -race ./engine -count=1 > "$RUN/race.log" 2>&1
go version > "$RUN/toolchain.txt"
go env GOOS GOARCH GOAMD64 CGO_ENABLED >> "$RUN/toolchain.txt"
lscpu > "$RUN/topology.txt"
sha256sum "$RUN/ngn-base" "$RUN/ngn-p1-r1" > "$RUN/binaries.sha256"
git -C "$SOURCE" diff "$BASE" "$P1_CANDIDATE" -- engine > "$RUN/candidate.patch"
```

Stop here until the coordinator verifies that patch, the forge2 mechanism
receipts, v3 ISA support, physical topology, and the rig build hashes. Freeze
these with the following script and artifact inventory before observing scores.
The script deliberately requires the historically hash-pinned harness, opening
and model files already recorded on the rig. Missing/mismatched files are a
blocker; do not download replacements, substitute nets, change paths inside an
old manifest, or silently relax an audit.

After that review, save the following block as `rig-p1-r1-match.sh` on the Mac
and send it with
`ssh -T ehrli@192.168.4.108 'wsl -d Ubuntu -e bash -s' < rig-p1-r1-match.sh`.
It initializes its own run path and build environment in the new WSL shell. It copies
frozen tools into a new run and reuses the accepted E0 supervision, per-role
UCI preflight, process witness, independent legal/terminal audit, trace audit,
and paired-bootstrap functions. It does **not** replay an old E0/H1 manifest.
Shell syntax, frozen inputs and the Python phase handoff are checked here with
local stand-ins for match operations; rig A/A and operational audits are required
to validate actual execution before candidate games.

```bash
set -euo pipefail
RUN=/home/ehrli/ngn-rig-p1-r1-20261023
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GOAMD64=v3 CGO_ENABLED=0
nice -n 10 env GOMAXPROCS=1 python3 - "$RUN" <<'PY'
import importlib.util, json, shutil, sys
from pathlib import Path
r = Path(sys.argv[1]).resolve()
src = r / 'base-src'
path = src / 'experiments/2026-09-19-e0-v12-anand-strength/driver.py'
spec = importlib.util.spec_from_file_location('e0', path)
e0 = importlib.util.module_from_spec(spec); spec.loader.exec_module(e0)
manifest = json.loads((src / 'experiments/2026-09-19-h1-history-policy/manifest-release-v1.json').read_text())
tools = r / 'tools'; tools.mkdir()
sources = manifest['frozen_sources']
assert e0.sha256(path) == sources['e0_driver.py']['sha256']
names = ['role_exec.py', 'run_match_stage.py', 'process_supervisor.py',
         'trace_audit.py', 'audit_fastchess_match.py', 'uci_preflight.py',
         'common.py', 'fastchess', 'stockfish', 'v12_model',
         'pilot_openings', 'pilot_prefixes', 'confirmation_openings', 'confirmation_prefixes']
# All inputs are checked before launching any engine.
for name in names:
    source = e0.checked_file(sources[name], name)
    target = tools / name
    e0.copy_frozen(source, target, sources[name]['sha256'], executable=name.endswith('.py') or name in ['fastchess', 'stockfish'])
assert e0.sha256(tools / 'v12_model') == 'c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053'
protocol = {'tc':'10+0.1', 'concurrency':4, 'physical_cpu_mask':['0','2','4','6'], 'seed':20261023}
e0.atomic_json(r / 'frozen-p1-inputs.json', {
    'base': '8ba878cb03a7eb81ca465b3d562953d0f45e81c7',
    'candidate': __import__('subprocess').check_output(['git','-C',str(r/'candidate-src'),'rev-parse','HEAD'], text=True).strip(),
    'protocol':protocol, 'bootstrap_seeds':[2026102301,2026102302,2026102303],
    'bootstrap_replicates':100000, 'phase_pairs':[50,64,200],
    'rule':'clean AA containing zero; positive pilot; independent confirmation lower95>0; no extension',
    'hashes':{str(p.relative_to(r)):e0.sha256(p) for p in [path,r/'ngn-base',r/'ngn-p1-r1',*sorted(tools.iterdir())]}})
e0.baseline_cpu(10, r / 'prelaunch-cpu.json')
for index, (label, pairs, seconds) in enumerate([('aa',50,1800),('pilot',64,1800),('confirmation',200,7200)]):
    phase = r / label; phase.mkdir()
    family = 'confirmation' if label == 'confirmation' else 'pilot'
    openings = tools / (family + '_openings')
    prefixes = tools / (family + '_prefixes')
    roles = []
    for role_id, binary in [('a','ngn-base' if label == 'aa' else 'ngn-p1-r1'),('b','ngn-base')]:
        cwd = phase / role_id; cwd.mkdir()
        engine = cwd / 'ngn'; shutil.copy2(r / binary, engine)
        launcher = cwd / 'role_exec.py'; shutil.copy2(tools / 'role_exec.py', launcher)
        e0.role_config(cwd / 'role-config.json', engine, e0.sha256(engine), cwd)
        role = {'id':role_id, 'name':label+'-'+role_id, 'engine':engine,
                'sha256':e0.sha256(engine), 'cwd':cwd, 'launcher':launcher,
                'model':tools/'v12_model', 'backend':'rodent-v1.2-default'}
        e0.preflight(tools, role, phase)
        roles.append(role)
    command = e0.phase_command(tools, phase, roles, openings, pairs, protocol)
    e0.atomic_json(phase / 'exact-command.json', command)
    e0.supervise(tools, command, phase, roles, protocol['physical_cpu_mask'], seconds)
    (phase / 'match.exit').write_text('0\n', encoding='utf-8')
    result = e0.audit(tools, tools, phase, roles, openings, prefixes, pairs*2, pairs)
    estimate = e0.paired_bootstrap(result['pair_audit'], 2026102301+index, 100000)
    e0.atomic_json(phase / 'estimate.json', estimate)
    if label == 'aa':
        assert estimate['lower95_elo'] <= 0 <= estimate['upper95_elo'], 'A/A INVALID'
    elif label == 'pilot' and (estimate['upper95_elo'] < 0 or estimate['elo'] <= 0):
        e0.atomic_json(r / 'decision.json', {'verdict':'SHELVE','phase':label,'estimate':estimate})
        break
    elif label == 'confirmation':
        e0.atomic_json(r / 'decision.json', {'verdict':'ACCEPT' if estimate['lower95_elo'] > 0 else 'SHELVE','estimate':estimate})
e0.atomic_json(r / 'terminal-inventory.json', {
    str(p.relative_to(r)):e0.sha256(p) for p in sorted(r.rglob('*'))
    if p.is_file() and 'base-src' not in p.parts and 'candidate-src' not in p.parts})
PY
```

Retain every failed and completed receipt. A failure before `decision.json`
means INVALID/BLOCKED, never ACCEPT. Do not rerun after viewing a score. The
coordinator must retain a hash of the exact script itself with the frozen run
inputs, verify no surviving children, and review the final inventory and all
phase receipts. Historical rig paths and binaries have not been revalidated by
forge2; that and the missing P1 implementation are explicit remaining work.

# Clean v3 candidate-engine preparation — campaign session 10

Session starts **14:55:31 EDT**, deadline **16:35:31 EDT, October 8**.
Accepted runtime remains **5b882e2 / be34532c**. The longer H512 gate,
operational admission and factorizer CUDA control remain pending. No candidate
net is nominated, no claim protocol is activated and no games are queued.

The prepared common net-gate engine is an archive build without the clean Git
metadata required by claimtool. The original v3 development engine predates
the accepted refresh/output/delta speed changes. Build a clean engine containing
exactly the accepted engine inputs now, while preserving the original v3 tool,
source and development control used by queued admission.

## Prospective preparation contract

New root: `/home/ehrli/ngn-data/claim-v3-20261008/nominee-session10c/`.
Fetch a local Git bundle into a private checkout, detach at the recorded exact
commit and require a completely clean tree. Reconcile the complete set of
**230** compilation/test/fixture/profile inputs against the accepted delta
receipt. Require accepted PGO `616b1ebe`, exact inherited full short/race/vet
log hashes, unchanged original v3 tool `4cdf39aa` and its clean source
`376671f842aad7841cd2c743c834c61f702eb0ae`.

On WSL CPUs **13,15**, nice **19**, Go **1.25.5**, amd64/v3, build NGN and a
private speedprobe with `-buildvcs=true -p 2`, automatic accepted PGO for NGN
and explicit `-pgo=<clean source>/default.pgo` for speedprobe, stdlib dependencies
only. Require build metadata to identify the recorded commit and
`vcs.modified=false`. No engine source changes, no timing measurement and no
tuner interruption. GPU remains with the existing trainer; hopper is untouched.
Require host C: free >=25 GiB. Preparation has a single 30-minute bound;
preserve failure without automatic retry.

Before accepting these preparation artifacts:

- The new speedprobe must reproduce all **36** saved accepted-delta records:
  NGNN3/NGNN2 twelve iterative/fixed depth-14 cases each, HCE twelve depth-11
  cases. Compare nodes, score, depth, best move and PV, excluding time/NPS.
  Hash-bind the saved raw records and existing speed receipt; do not replay or
  extend the historical timing sample.
- The actual new NGN executable must agree with the frozen common net-gate
  executable on **18** fresh UCI cases: all six owned speedprobe positions,
  NGNN3/NGNN2 at depth 14 and HCE at depth 11, Hash 64, Threads 1,
  GOMAXPROCS=1, book off, Move Overhead 100. Require exact final nodes, score,
  depth, best move and PV. Save handshakes, selected commands and raw streams.
- Use the unchanged original v3 tool to bind a **development** identity with
  `be34532c` and its already frozen provenance. Check four fresh dry modes:
  screen, stability-screen, stability-confirmation and tooling-check. Require
  schema 3, threshold 3600 and exact declared clocks/counts/anchors. Demonstrate
  rejection of changed engine/net/provenance digests, source commit, Go version
  and build output. No dry invocation may create a live receipt or games.
- Require clean source and unchanged pre/post compilation, artifact and
  provenance hashes. The build receipt records exact argv, metadata, inherited
  test hashes and all new identity/dry-control hashes.

The detached [controller](2026-10-08-v3-nominee-build.sh) writes terminal markers
and campaign `status.jsonl`; the [verification helper](2026-10-08-v3-nominee-check.py)
runs only on WSL. Launch once after committing this manifest/controller:

```bash
cd /home/ehrli/ngn-data/claim-v3-20261008/nominee-session10c
setsid nohup timeout 1800 bash controller.sh source.bundle "$(cat source-commit.txt)" \
  < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
```

## Later nomination boundary

This engine can support the reviewed longer net or subsequent accepted nets
without rebuilding, provided no later engine change has been accepted. Its
development identity never constitutes final nomination. After the complete
longer-net verdict and normal operational admission/CUDA completion, review
the then-immediate accepted engine/net, freeze that net and its actual
manifest/data/training/export/parity provenance, and write a new final identity.
Do not reuse the development identity or old provenance for a different net.

The original tool remains rooted at `tooling/source` and keeps its original
build argv. A live screen must receive that tool source separately from the
candidate source. Commit/push the complete prospective v3 attempt manifest
and all final pins before any game. Fresh screen = 320 games at 10+0.1;
only a clean, bracketed screen with pooled lower95 >3600 permits the same
candidate's fresh 640-game 120+1 confirmation. No historical games are pooled.
Both need every independent audit and unchanged hashes, zero exclusions,
strictly bracketed raw anchor scores and primary lower95 >3600. Prefer the
prepared factorizer trial before a strength screen if the immediate accepted
net remains short of the strict screen threshold. No live screen/claim or
automatic nomination is chained by this preparation.

## Preserved initial preparation failure

Initial wrapper **584273** / controller **584276**, root `nominee-session10/`,
started **15:02:35 EDT** and stopped **15:02:37 EDT**, exit one, before any
build or engine execution. The new input-set selector counted **227** source
files but omitted the three `testdata/ngnn*/random*_evals.json` parity fixtures
included in the accepted **230**-input receipt. The exact set difference proves
the cause. Correct the selector to include all tracked `testdata/` inputs;
preserve every original file/log and use a fresh `nominee-session10b/` root.
No engine/trainer behavior or evidence rule changes, and no running job was
interrupted. The corrected preparation performs the complete declared checks
once; it reuses no partial search/build result from the failed directory.

The second wrapper **584384**, `nominee-session10b/`, started **15:05:41 EDT**
and stopped **15:05:55 EDT**, exit one. All 230 input checks and both builds
passed; the artifact check correctly rejected missing PGO metadata on the
speedprobe. Go's automatic lookup uses the main package directory, so the root
profile applies automatically to NGN but not `cmd/speedprobe`. Add the same
explicit accepted profile used by the historical probe to this probe build
and its recorded argv. No search, game or dry plan ran. Preserve both initial
roots; the final fresh preparation root is `nominee-session10c/`.

## Completed preparation and review

Final wrapper **585978**, clean source
**1d635447f833dd0c73e2eef057c8ecbb14edfc2e**, started **15:08:08 EDT** and
completed **15:08:40 EDT**, exit zero. Engine SHA-256
**737c18bee49408ce8884655f7c2b1a0b322b7637c35974e75e1e737638525b48**;
private speedprobe **4ca3d409**; development identity **4ae66de2**. Both
build metadata checks pass, including the accepted profile, clean VCS revision,
Go 1.25.5, Linux amd64/v3 and no third-party module dependencies.

All **230** engine/test/fixture/profile inputs match the accepted tested tree,
and inherited engine/full short/race/vet logs match their completed hashes.
All **36** fresh probe records reproduce the accepted raw delta identities;
all **18** UCI pairs match the frozen common engine exactly. Four positive
v3 dry modes and six changed-identity rejection controls pass. The original
v3 tool/source and queued operational control retain their original pins.

Independent review recomputed every UCI final depth/nodes/score/PV/bestmove
from raw stdout, compared all saved/fresh probe records, checked the four
plans and every completion artifact, and reconciled all 230 local source
inputs. [Build receipt](2026-10-08-session10-results/build-receipt.json),
[source/test binding](2026-10-08-session10-results/source-receipt.json),
[identities](2026-10-08-session10-results/identity-receipt.json),
[dry controls](2026-10-08-session10-results/dry-receipt.json),
[launch custody](2026-10-08-session10-results/launch-observed.json) and
[final review](2026-10-08-session10-results/final-review.json).

The completed snapshot initially included `frozen/ngn_crashes.log`. Its
contents are only initialization lines, but later engine starts append to it.
Preserve the original build receipt and `frozen-inputs.sha256` as historical
completion snapshots; the reusable **114**-entry
[`candidate-inputs.sha256`](2026-10-08-session10-results/candidate-inputs.sha256),
SHA-256 **9cebb8a2**, removes exactly that runtime log and rechecks every other
input. The helper now excludes this log in future preparations; its selection
is verified against the corrected list without rebuilding or replaying search.
Successful box scripts/manifests and both failed preparation roots remain
unchanged. The common gate engine's generated log is outside its existing
frozen list too; none of its bound inputs changed.

No new accepted playing behavior, net, search vector, absolute strength result
or live claim attempt. At review **15:12 EDT**, longer H512 has **55/116** epochs;
SPSA has durable **375 / 4,500 games** at **15:01:04 EDT**. GPU snapshot **94% /
4,197 MiB**, host C: **146.5 GiB free**; existing PIDs and hopper are unchanged.

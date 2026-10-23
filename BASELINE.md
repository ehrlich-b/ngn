# forge2 baseline

Source `8ba878cb03a7eb81ca465b3d562953d0f45e81c7`; Intel Ultra 9 285H,
16 logical CPUs, Linux/KVM, Go 1.27.3. All execution used nice +10 and at most
eight configured engine workers. Build/test/vet used GOMAXPROCS=8, GOFLAGS=-p=8.
Measured binaries: GOAMD64=v3, CGO_ENABLED=0, `go build -trimpath`.

`go build ./...`: **build exit 0**. `go test ./...`: **test exit 1**:
`TestConcurrencyStress/ConcurrentMoveGeneration` reported
`crash_stress_test.go:266: No moves in goroutine N` for every N=0–9; the engine package
then timed out after 10m0s in `TestNumericalLimits/Depth_100` (9m37s).
No fixes. `go test -short ./... -count=1`: exit 0. Internal vet: **vet exit 0**,
empty output; pipefail preserved failures through `tail -5`.
Exact failures: `../evidence/test.log`; other logs: `build.log`, `test-short.log`,
`vet.log` in that directory.

The unchanged `cmd/benchmark` ran all 24 positions at each width, 1 second
requested per position, default **HCE**, Hash128, OwnBook=false. Mean of reported
position NPS (not total nodes/wall time):

| Threads | Mean NPS | Benchmark exit |
|---:|---:|---:|
| 1 | 1854963 | 1: two historical-baseline regressions |
| 2 | 3426422 | 1: two historical-baseline regressions |
| 4 | 6284108 | 0 |
| 8 | 9522963 | 0 |

Evidence: `../evidence/legacy-threads{1,2,4,8}.{json,log,uci.log}` and
`measurement-summary.json`. This is one shared-host pass; the full test suite
overlapped, and the fixed-node probes overlapped early measurements. It is a
diagnostic baseline, not a repeatable speed verdict or Elo evidence.

Six frozen V1.2-benchmark positions, executed here with HCE, each reached
**exactly 400000 nodes** at every configured width:

| Position | Move | Depth | Last completed score cp |
|---|---|---:|---:|
| start_white | e2e4 | 14 | 29 |
| start_black | e7e5 | 14 | 37 |
| kiwipete_white | c5b6 | 11 | -83 |
| kiwipete_black | b4c3 | 12 | 402 |
| asymmetric_white | c3d5 | 14 | 558 |
| asymmetric_black | d5a5 | 15 | 257 |

Node limits deliberately force **effective Threads1**. All 24 outcomes match;
these runs provide no fixed-node SMP scaling measurement.
See `../evidence/fixed-results-verified.json` and raw `fixed-*.uci.log`.
`searchsnapshot` additionally passed all 11 fixtures, cold/warm, with identical
JSON across GOMAXPROCS 1/2/4/8; `snapshot-threads*.json`, `snapshot-compare-*.json`.

Accepted competitive engine: H1 policy11, production `30e4905`, validation
`383e757`; this checkout has the identical engine tree. Recorded production
binary SHA256:
`19ddf423ff6addffadd4b830834ca2d3483b47c64b7530689141337b8a39a518`.
Accepted Rodent V1.2 net, 4744768 bytes:
`c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053`.
[Engine validation](experiments/2026-09-19-h1-history-policy/integration-v1.json),
[network acceptance](experiments/2026-09-19-e0-v12-anand-strength/result.json).
That external net is absent here; no NNUE baseline is claimed. Fresh HCE binary
SHA256: `b70318ee5f522622cc44eb9f1f6351953758eb9bce762807c45df642ddb87988`.
Historical installed `53e4d1b` used HCE, no net, Linux SHA256
`0c067627404ab67a5f4a57fef63be024fd84e9248423286ef3dfc103041d266e`;
[deployment](experiments/2026-09-04-deployment.md).

Adaptations: external input/output staging, a UCI option-injection wrapper, and
an external make build check avoided the legacy benchmark's deployment recipe.
Its original regression baseline and failures were preserved. Commands and
wrapper source: `../evidence/measure.py`, `fixed.py`, `legacy/Makefile`.

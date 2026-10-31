# Tiny WSL generation-loop validation

All builds, tests, training and chess execution occurred in the isolated WSL
worktree `/home/ehrli/wt/genloop-20261006`; the Mac was used for editing/Git.
Engine/runner source `a3349ed`, pipeline source `c878fca`, and final resume
regressions `7c421b1`. Go 1.25.5, GOAMD64=v3, GOMAXPROCS=2, Go build/test
`-p 2`, GOTOOLCHAIN=local, GOPROXY=off; CPU training used two threads. The
existing venv supplied only Python/NumPy/PyTorch. No third-party engine or
trainer sources, data or weights were read or used as implementation inputs.

Receipts remain under
`/home/ehrli/ngn-data/genloop-validation-20261006-c878fca/`: generation manifest,
exact-command events/logs, immutable shards/sidecars, split, checkpoint, exports,
parity inputs/logs, `datastat.json`, status snapshots and `summary.json`.
The receipt tree occupies 1,929,077 bytes. Earlier build/test logs and the
initial venv-path failure are retained in `genloop-validation-20261006-a3349ed/`.
The symlink bug was fixed and covered by a regression test before this rerun.

| Check | Result |
|---|---|
| Stdlib orchestration suite, final source | 13 tests passed; manifest/resume, immutable inputs, locks, closed shards, target/disk priority, holdout, pair reconciliation and side-effect-free dry runs |
| Trainer suites, final source | 24 tests passed, including exact resumed weights and recovery of corrupted final history without changing checkpoint bytes |
| Go packages | `cmd/sprt`, `internal/uci`, `internal/datagen`, `internal/ngnp` passed; NGN, datagen, sprt and datastat built |
| Datagen | 2 workers, 5,000 nodes, 60.153 seconds; 43,583 positions in 2 closed shards, both sidecars/records/checksums validated by datastat |
| Train | 1 epoch, H=32/NB=1, batch 4,096, CPU; 21,796 training records and 21,787 held-out records in disjoint whole shards |
| Loss | Training 0.0640756121; validation 0.0600501838; six optimizer updates |
| Exports | NGNN1 and equivalent one-bucket NGNN2; exact reference/engine agreement on 20 held-out positions each |
| Gate | 8 games / 4 complete color pairs at 10+0.1, concurrency 2, Hash 64, Threads 1, OwnBook=false; NGNN2 versus same-binary HCE |
| Gate result | 0W/0D/8L, pentanomial `[4,0,0,0,0]`; paired Elo -800, CI [-800,-598.0408157], using the runner's clipped tails/variance floor |
| Operational checks | Zero time forfeits, illegal/no-move errors or watchdog voids; all recorded datagen worker PIDs gone |
| Repeat completed stages | All three verified hashes and returned without new workloads |
| Real disk refusal | An intentionally unreachable 99,999 GiB threshold refused before spawning the driver or creating shards |

This is a tooling smoke test. Eight losses from a six-update network do not
establish strength, and its clipped Elo estimate is not an absolute rating.
No large generation, multi-hour job, detached workload, schedule, expensive
claim run or third-party opponent game was launched for this task.

The full NGN opening file digest matched the historical receipt
`974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
Gate used its first four lines, digest
`8d1f45c12789131cba6f7dc3270f75b730f4f305c7c22939cbd19651f0e8f728`.
The completed generation manifest digest, unchanged by status/idempotence:
`fa86c9df49707f3d8890b9d0b132131300c6ab13d14c9dea37697e371f247015`.
Last datagen disk sample: C: 13,908,434,944 bytes, WSL 611,380,293,632 bytes,
both above the default 6 GiB threshold.

| Artifact | SHA-256 |
|---|---|
| NGN binary | `ec2f92b058bdcf1b372bcbdb5428f745da8514316596c5383957b51bd5ea758a` |
| Datagen binary | `e38cd0d1cc43d5a71a830c5a44029abe2a4d84d203257f0222849f7ef30c1ac3` |
| SPRT binary | `03a3265f5e79dc0d74c6978f0452f399fd1ec8f190c54980c1c3a1e23a3f6e8d` |
| NGNN1 export | `b24fba38c3ac5976c3887db36279647dd15c65b79cc8ac1470a0e68ce8af081d` |
| NGNN2 export | `63c83efc6b49aee1ec31a96b38abac7d52ffcc1909463758254def5a18bde5fe` |

The final resume changes add upstream-shard verification on completed-stage
reuse and atomic checkpoint-history recovery; final WSL suites exercised both.
They do not change the recorded engine/net bytes or match behavior.

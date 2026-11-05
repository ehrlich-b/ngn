# Quiescence capture history candidate

`W2QCapture` is a UCI spin option (0..1), shipped at 0. At 1, non-TT
captures add `clamp(captureHistoryScore(move)/8, -512, 512)` to the existing
MVV/LVA score. Quiet promotions retain their old score; TT remains first.
The existing incremental selection loop, move generation, pruning and learning
are unchanged. Equal scores retain the existing deterministic selection behavior.

From the repository root:

```sh
nice -n 10 python3 experiments/wave2-1104/qcapture/check.py focused
nice -n 10 python3 experiments/wave2-1104/qcapture/check.py parity
nice -n 10 python3 experiments/wave2-1104/qcapture/check.py policy
nice -n 10 python3 experiments/wave2-1104/qcapture/check.py provenance
```

`focused` runs only `GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run
'^TestWave2QCapture' -timeout 15s -v`. The real qsearch witness uses evaluator
transitions and a controlled static value to force a capture cutoff after two
nodes; it checks the chosen move, TT record, board/hash/repetition restoration,
frame/evaluator depth, incremental accumulator against the independent scalar
oracle, and unchanged capture history. Other cases cover clamp/truncation,
quiet/capture promotions, both en passant colors, option advertisement and setting.

`parity` extracts only the starting head's engine, internal packages and go.mod
into external scratch, builds the same probe against each source, and explicitly
loads `engine/nets/ngn-h768-2dcb6d93.nnue` in both. Each FEN runs in a fresh
process, explicitly configured Threads1 Hash16, with a 10 CPU-second limit.
It compares exact depth4 iterative nodes, score and bestmove with the committed
`result.json`. These are correctness observations, not speed samples.
`record` initially generated that result from actual execution after staging the
product files. Source digests and engine subtree hashes avoid self-reference in
the committed result. Replay validates these against the current committed tree.

The runner strips credential/token/secret/password variables, uses shared Go
caches, GOMAXPROCS=2 and tensor threads=2, hides GPUs, and records native wait4
elapsed/CPU/peak-RSS data and subprocess logs under external `evidence/`.
It leaves scratch for the retention policy. No network or model download is used.

Verdict: **DEFERRED**. Default play stays unchanged. The completed rig strength
gate has not been run; fixture success makes no Elo claim.

The short-suite replay guard blocks only the exact hostname `forge` (exit 77).
Ordinary Go tests, including neural kernel tests, are permitted on `forge2-vm`
and the rig. The guard never matches a hostname prefix. Its dispatcher regression
check simulates both blocked and permitted hosts without executing their suites:

```sh
PYTHONDONTWRITEBYTECODE=1 nice -n 10 python3 -m unittest discover -s experiments/wave2-1104/qcapture -p test_check.py -v
```

Run all four required gates directly from the repository root with GOMAXPROCS=2
and the shared Go caches configured:

```sh
nice -n 10 go vet -p 2 ./...
nice -n 10 go test -p 2 -short -count=1 ./...
nice -n 10 git diff --check 8ba878cb03a7eb81ca465b3d562953d0f45e81c7 HEAD
git diff --check 3517a353a99bbe627a505ccc5463aadce047431b HEAD
```

The optional `check.py exec go test -p 2 -short -count=1 ./...` wrapper runs
the same short suite on permitted hosts and preserves the exact-host restriction.
No privileged operations, kernel modules or block devices are involved. The
completed rig strength gate remains separately outstanding; keep the default off
and the experiment verdict DEFERRED until that gate is completed.

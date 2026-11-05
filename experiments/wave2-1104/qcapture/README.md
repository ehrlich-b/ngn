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

The brief prohibits kernel tests on forge hosts. The exact required short-suite
gate includes neural kernel tests. The `exec go test -p 2 -short -count=1 ./...`
entry refuses that gate on a host whose name begins with `forge`, exits 77, and
prints the command to run on a suitable CPU host. This is a blocked gate, not a
passing or reduced-suite substitute. Vet and whitespace gates run normally.

The short-suite gate remains BLOCKED on forge. To finish verification, move the
committed candidate and the external readiness inputs to a suitable non-forge
CPU host with the same repository path and existing H768 net. Preserve the
brief, all acceptance IDs, all four gates, and the default-off option. Provision
the documented shared caches and lane-local temporary directory before replay.
Then run from the repository root:

```sh
nice -n 10 python3 experiments/wave2-1104/qcapture/check.py exec go test -p 2 -short -count=1 ./...
nice -n 10 python3 /work/lanes/readiness.py run ..
```

The first command must execute the complete required short suite. The second
regenerates every acceptance and gate log against the same committed HEAD; a
prior failed record or results copied from a different HEAD cannot certify it.
Keep generated readiness records and logs outside the checkout. Do not disable
kernel tests, narrow the gate, or change the hostname to bypass the guard.
Until that replay passes, retain BLOCKED readiness and the DEFERRED experiment
verdict; the completed rig strength gate remains separately outstanding.

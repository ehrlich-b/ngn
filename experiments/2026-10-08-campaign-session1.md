# 3600 campaign session 1 — 2026-10-08

Session starts 10:18:46 EDT; deadline 11:58:46 EDT. Base `be34532c` remains accepted.
All engines, tests, builds, matches and training run on WSL. The Mac edits and coordinates only.

## Screen startup repair and prospective attempt

The original screen failed before games. Its `preflight-failure.json` identifies Counter 3.8,
with stderr `claimbridge.exe: Invalid argument`; NGN net loading was not the failing step.
Two detached bridge probes reproduce failure with `/run/WSL/2_interop` (rc 1), then pass
`uciok`/`readyok` with the existing keepalive endpoint `/run/WSL/552_interop` (rc 0, verified
Windows job affinity `0xffff`, Idle priority). Probe receipt on WSL:
`night-20261007/session1/interop-probes.json`, SHA-256
`6588b1c64043fa34c5480db35c470a43cce05351a514670a977000ecb1cb3f1f`.
The launcher now uses that endpoint and permits a fresh output directory. It refuses the old
SIGSTOP-trainer shortcut: claimtool rejects stopped trainers too. No engine, bridge or claimtool
implementation changes. `bash -n` passes; the real bridge probe proves the corrected startup.

Predeclared attempt: `screen-kb8h512-session1/`, no prior games pooled or replayed. Same frozen
v2 claim engine `8fb36cfb` (source `9fe2cb6`, before the release clock-stop correction),
claimtool `a4960f59`, anchors, options, first-40 reversed opening pairs and complete audits as
the original screen, only net `be34532c`. This measures that frozen source/net combination;
it is development orientation, not a 3600 claim or release-binary measurement. 10+0.1,
8 games total, CPU 0–15 / nice 19, GOMAXPROCS=1; Windows job `0xffff` / Idle.
SPSA stops for the screen. No trainer starts until screen termination. Preserve failures as
invalid; do not extend the sample. Report pooled interval, bracket and all integrity audits.
No 120+1 confirmation is queued; the 3600 protocol needs its stronger top anchor first.

## Longer H512 candidate — manifest frozen before training/games

```yaml
id: 2026-10-08-kb8h512-long
date: 2026-10-08
change_class: search/eval heuristic
hypothesis: Doubling the fixed H512 training duration improves real-clock strength.
base_commit: be74e25 (release engine); trainer 36a36575479224051d066022a22952806a1c1cf8
candidate_commit: Same engine and trainer; only total epochs 58 -> 116.
base_binary_sha256: b0b36c9d6b8da8c7aeadda30667a12277bc58f62b5ff236681bcb4b84af4e0b8
candidate_binary_sha256: b0b36c9d6b8da8c7aeadda30667a12277bc58f62b5ff236681bcb4b84af4e0b8
base_net_sha256: be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a
candidate_net_sha256: Produced after training; frozen by parity receipt before games.
harness_commit: Campaign session1 controller; copied script hashes in session1/frozen-inputs.sha256.
command: See the exact shell command below and campaign-session1.sh arrays.
machine: Ryzen 9800X3D; RTX 5080; WSL2 Linux 6.6.87.2
go_version: go1.25.5
goarch_goamd64: amd64/v3
tc: 10+0.1
concurrency: 8
affinity: CPU 0-11; GOMAXPROCS=1 per engine (same as night gates)
openings: net1-gates-20261006/openings.txt
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
aa_preflight: Fresh same-net 40 games; require CI containing zero and zero operational anomalies.
decision_rule: Accept only completed 400 games / 200 pairs with penta 95% CI lower > 0 and zero flags/crashes/illegal/no-move results; otherwise shelve, no extension.
games_or_pairs: 400 games / 200 reversed pairs
result: pending
flags_errors: pending
verdict: pending; no automatic promotion
next_action: Inspect parity, A/A and final gate receipts; update accepted line only after review.
```

Training from scratch, same seed/data/holdout/model/optimizer recipe as the immediate accepted
KB8 H512 net. The only recipe change is `--epochs 116`; the cosine schedule consequently spans
116 epochs. This is a longer complete schedule, not a continuation with a changed cosine cap.
Selected checkpoint = lowest predeclared validation loss across the complete 116 epochs;
validation is not Elo evidence. Fixed 29.696B exposures; ~10.5 hours on the GPU.

```bash
cd /home/ehrli/wt/t80-20261006
taskset -c 12,14 nice -n 19 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python -m trainer.train \
  --data /home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf \
  --data-format archive --archive-holdout-modulus 1024 --archive-split-seed 20261006 \
  --out /home/ehrli/ngn-data/night-20261007/kb8h512-long \
  --format 3 --kb-map default --hidden 512 --buckets 8 --batch 16384 --lr .001 \
  --epochs 116 --positions-per-epoch 256000000 --wdl .75 --k 400 --decay cosine \
  --min-lr-ratio .1 --threads 2 --precision bf16 --seed 20261006 --keep-best-only
```

Controller [2026-10-08-campaign-session1.sh](2026-10-08-campaign-session1.sh): stop SPSA,
screen, resume SPSA, train, stop SPSA, exact 20-case integer parity and byte-identical reexport,
A/A40, fixed gate400, resume SPSA. Failure cleanup resumes paused SPSA. SPSA has no signal
handler; pause terminates its campaign process group and resume uses its last JSONL checkpoint.
Uncheckpointed iterations are not counted as retained tuning evidence. GPU is reserved for
the exclusive screen first, then training starts automatically. `hopper` is never touched.
Results and PIDs: `/home/ehrli/ngn-data/night-20261007/session1/`; status in campaign `status.jsonl`.

## Results and handoff

Detached controller PID **522940**, launched 10:24:37 EDT. Fresh screen runner PID **523430**,
started 10:25:10 EDT; completed preflight and entered games. Original failed attempt remains
untouched. Estimated screen finish ~10:55 EDT, then training ~10.5 hours and the ~30-minute gate.
Three prior complete logs exercise the new gate parser: passing A/A, neutral rejected H1024,
and accepted H512. Candidate configuration and copied scripts are hash-frozen before launch.

Queued follow-ups are independent preparation, with no engine behavior changes: match-only Berserk 12
source build controller **533425** waits until the screen ends; PGO revalidation controller **534228**
waits for the screen and anchor build, then temporarily pauses SPSA for its fixed probe. The first PGO
launcher produced no status or work; only the observed live controller 534228 is retained. Neither
follow-up changes the accepted line automatically. The 155 actual WSL PGO source inputs match the
profile parent by Git blob digest. See [anchor/protocol preparation](2026-10-08-3600-protocol-preparation.md)
and [the speed manifest](2026-10-08-pgo-kb8.md).

Final batch check at 10:39:54 EDT: controller/screen/build-wait/PGO-wait PIDs all live, screen sample
still incomplete, GPU reserved idle until the screen ends, hopper 1168 untouched, C: free **151.7 GiB**.
No candidate verdict or 3600 claim is drawn from this unfinished sample. Exact frozen-input manifest
`session1/frozen-inputs.sha256` hashes to
`c4336244d0fcf0a3f56286a5a64aae80941d0d5f4363502c1be548c8b253e42b`.

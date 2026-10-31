# Overnight net chain — 2026-10-07/08

Goal (user, stop-hook): NGN at or above 3600 on the v2 claim protocol (120+1, 640 games, 4 anchors), no
copied code or nets. Read as the v2 pass rule retargeted: pooled lower 95% bound > 3600 with scores
bracketing 50%. The v2 anchors top out at Viridithas 10 (3557), so a 3600 claim needs a stronger top anchor.
Accepted base: 0.3.0, net `c6c12796` (NGNN2 H512, 8 output buckets), claim 3461.56 [3429.68, 3493.43].

## Candidate 1: king-bucket inputs (NGNN3 KB8 H512)

One change vs `c6c12796`'s recipe ([T80 record](2026-10-06-t80-net.md), H512 section): `--format 3
--kb-map default`, NGN's own 8-bucket king map with horizontal mirroring (`trainer/king_buckets.py`,
`engine/ngnn3.go`; king buckets are a published Stockfish idea, not Rodent's 4-bucket layout). Same data
(T80 archive `270f65bd…`), seed 20261006, 58 × 256M, batch 16384, BF16, AdamW .001, wd .01, lambda .75,
K 400, cosine to .1, `--keep-best-only`, `taskset -c 12,14 nice -n 19`, trainer source `36a36575`
(`/home/ehrli/wt/t80-20261006`). Launched 2026-10-07 21:43 EDT, PID 509606, root
`/home/ehrli/ngn-data/night-20261007/` (`kb8h512.cmd`, `kb8h512.log`). ~323 s/epoch (GPU 82%).
Early validation loss (lambda .75): epoch 10 .009183 vs `c6c12796` .009357 at epoch 10 (.009233 at 20).

## Controller

[2026-10-08-night-chain.sh](2026-10-08-night-chain.sh), status lines in `status.jsonl`:

1. Wait for PID 509606; require the epoch-58 history line and `best.nnue`.
2. Parity: the recipe-trials `parity()` helper (20 exact trainer/engine integer cases via
   `TestNGNN3TrainerParity`, byte-identical reexport). Dry run on a copy of the epoch-13 best: 20/20, AVX2,
   reexport identical.
3. Start candidate 2 on the GPU immediately: same command with `--hidden 1024` (`kb8h1024/`), one change vs
   candidate 1. If candidate 1 fails its gate, candidate 2 is stopped and replaced by NGNN2 H1024 (manual).
4. A/A then gate, same binary both sides: 0.3.0 release build `pub-ngn-linux` `b0b36c9d…` (byte-identical
   to a rebuild of the public tree), runner `sprt` `9b0002f8…`, openings `974e4b5a…`, 10+0.1, concurrency 8,
   `taskset -c 0-11`, GOMAXPROCS=1, Hash 64, Threads 1, OwnBook false, UseNNUE true, `-mingames 1000000
   -maxmoves 400 -gametimeout 180 -lowpower=false`, no adjudication. A/A: 40 games, `c6c12796` both sides.
   Gate: 400 games / 200 reversed pairs, candidate vs `c6c12796`. 4-game smoke with the epoch-13 net:
   2-0-2, 0 flag-outs.

Decision rule (predeclared): accept iff the pentanomial 95% CI lower bound > 0 with zero flag-outs,
crashes, illegal/no-move results. Accepted nets get the v2 10+0.1 320-game 4-anchor screen (0.3.0
screened 3386.78). No 120+1 games tonight.

Machine: 9800X3D, WSL2 Linux 6.6.87.2, RTX 5080, Go 1.25.5 linux/amd64 GOAMD64=v3, PyTorch 2.7.0+cu128.

### Result: candidate 1 ACCEPTED (2026-10-08 03:45 EDT)

Training finished 03:15 EDT, best epoch 56, net `be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a`
(checkpoint `d7dabf70…`), validation loss .008520 at epoch 52 (`c6c12796` final .008935). Parity 20/20,
reexport identical. A/A (40 games, `c6c12796` both sides): penta -34.9 [-113, +40], 0 flag-outs.
Gate (400 games, `gate-kb8h512.log`, 27m23s): W-D-L 117-207-76, 55.1%, pentanomial [4, 32, 94, 59, 11],
**penta Elo +35.7, 95% CI [+15, +56]**, flag-outs 0/0, no crashes, illegal or no-move results (ends: 193
checkmate, 204 draw-rule, 2 max-moves, 1 stalemate). Rule met; `be34532c` is the new immediate base.

The screen could not run while `kb8h1024` trains: `claimtool` refuses any live screen while a
`trainer.train` process exists, stopped or not (`internal/claim/receipt.go` `NoOtherMatch`). The refused
attempt is kept as `screen-kb8h512-refused/`. The screen moves to after `kb8h1024` finishes (controller 2).

## Controller 2

[2026-10-08-night-chain2.sh](2026-10-08-night-chain2.sh) (`e538f1fc…`), started 03:54 EDT: wait for the `kb8h1024`
trainer; SIGINT the tuner; parity; A/A (`be34532c` both sides, 40 games); gate `kb8h1024` vs `be34532c`, 400
games, same command and rule as above; v2 screen of `be34532c`; v2 screen of `kb8h1024` if its gate CI lower
bound > 0 with 0 flag-outs; resume the tuner with `-resume`. No trainer is started, so the screens can run.

## Speed: profile-guided build (branch `night/nnspeed`)

`default.pgo` (`616b1ebe…`) = CPU profile of `cmd/speedprobe -mode timed -duration 10s` with `c6c12796`
(six positions, CPU 2). Profile top: alphaBetaPV 11.5% flat, selectNextMove 11.4%, CachedEval.LoadData 9.0%,
ngnn1MoveRows 8.1%, ngnn1OutputSmall 5.3%. PGO build vs plain build: identical speedprobe identity output at
depth 14 (all six positions); interleaved timed NPS, 10 rounds, CPU 4: 1,869,678 → 1,929,608, ratio
**1.0323 [1.0140, 1.0506]**. Pure speed change; keep.

## Screen script

[2026-10-08-night-screen.sh](2026-10-08-night-screen.sh): builds a claimtool identity for the night net with the
frozen claim engine `8fb36cfb…` (source `9fe2cb6`, NGNN3-capable), claimtool, claimbridge, anchors and
first-80 openings, then runs the unchanged `scripts/claim/screen.sh`. Any running trainer is SIGSTOPped for
the screen and resumed after. Interop socket `/run/WSL/2_interop`. Dry run (`--dry-run`) on the epoch-13
kb8h512 net: rc 0.

## Joint search SPSA under NNUE (background, resumable)

The 14 `TunableSearchParams` were last tuned under HCE. `cmd/spsa` from release/0.3.0 `6488235`
(`67cf935d…`, selftest PASS), engine = 0.3.0 release binary `b0b36c9d…` (embedded `c6c12796`), 10+0.1,
`-iters 2000 -pairs 6` (24,000 games total), openings `974e4b5a…`, seed 20261008, `taskset -c 0-11`,
GOMAXPROCS=1, out `/home/ehrli/ngn-data/night-20261007/spsa/spsa.jsonl` (checkpoint every 25 iterations;
continue with `-resume`). Started 2026-10-07 23:09 EDT under `timeout --signal=INT` to stop at 03:00 EDT,
before the gate. An aborted 90-second start with `-iters 12000` is kept as `spsa-aborted-12000.log`.
Evidence rule (CLAUDE.md): only the converged vector after >= 20k games, verified by games.

That run stopped at 03:00 EDT after 175 iterations (2,100 games, 0 forfeits) and is abandoned: the base net
changed. Restarted from the 0.3.0 defaults against the new base at 03:54 EDT, same command except the engine
`kb8bin/ngn-kb8` (`f09b76b7…`): the 0.3.0 source with `be34532c` embedded instead of `c6c12796`
(`engine/default_net.go` name and digest only). Identity vs `pub-ngn-linux` + `EvalFile=be34532c`: depth 13 on
three positions (startpos, an Italian middlegame, a rook endgame), identical nodes, score, PV and bestmove.
Out `spsa/kb8/spsa.jsonl`; controller 2 pauses and resumes it around the games.

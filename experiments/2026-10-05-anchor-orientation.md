# Pilot-net anchor orientation — 2026-10-05

This is a **10+0.1 orientation with the NGN-produced H256 pilot net**, not a
calibrated CCRL rating or evidence of reaching 3000. All builds, tests and
engine execution occurred on WSL; the Mac was used for editing. No third-party
source was read, downloaded, built, linked or used as NGN scaffolding.

## Inventory

Read-only filename/binary-header/hash searches covered `/home/ehrli` and
`C:\Users\ehrli`, excluding package caches, container layers and dependency
trees. Identical preserved copies are collapsed below. Paths are actual
present binaries, not the now-missing paths in `opponents/ratings.json`.
NGN binaries that happened to contain opponent names are excluded.

Path abbreviations: `H=/home/ehrli`, `W=C:\Users\ehrli`,
`B=H/ngn-original-nnue-night-20261001/hce-fourcore-20261002-2/artifacts`,
`P=H/repos/ngn-next/output/go-opponent-preflight-20260906`,
`E=H/repos/ngn-external-opponents-v1/output/external-fixed-comparisons-20260908-attempt4`.

| Engine/version | Present path(s) | Recorded CCRL Blitz label | Record |
|---|---|---:|---|
| Blunder 5.0.0 | `B/Blunderv5.0.0` | 2080 | [ratings](../opponents/ratings.json) |
| Blunder 6.1.0 | `B/Blunderv6.1.0`; `W\ngn\repin\anchors\blunder_610.exe` | 2155 | [ratings](../opponents/ratings.json) |
| Blunder 7.2.0 | `B/Blunderv7.2.0`; `W\ngn\repin\anchors\blunder_720.exe` | 2425 | [ratings](../opponents/ratings.json) |
| Blunder 7.4.0 | `B/Blunderv7.4.0`; `W\ngn\repin\anchors\blunder_740.exe`; `W\blunder_740.exe` | 2532 | [ratings](../opponents/ratings.json) |
| Blunder 8.0.0 | `B/Blunderv8.0.0`; `W\ngn\repin\anchors\blunder_800.exe`; `W\blunder_800.exe` | 2674 | [ratings](../opponents/ratings.json) |
| Blunder 8.5.5 | `P/blunder-8.5.5-avx2` | unknown | [release inventory](2026-09-05-go-opponent-inventory.md) |
| Counter 3.8, pinned historical binary; UCI says `Counter dev` | `W\ngn\repin\anchors\counter_38.exe` | 2994 | [ratings](../opponents/ratings.json), [exact binary digest](2026-09-04-strength-check.md) |
| Counter 5.5 | `P/counter-5.5-linux-amd64` | 3359 (September 5); 3360 (September 30) | [rating reassessment](2026-09-12-rating-reassessment.md), [later catalog](2026-09-30-owned-top3-go-goal.json) |
| Zahak 10.0 | `H/nnue-owned-top3-20260930/opponents/zahak-10/zahak-linux-amd64-10.0-avx`; `P/zahak-linux-amd64-10.0-avx` | 3333, 1CPU; 3440 is an 8CPU label | [1CPU catalog](2026-09-30-owned-top3-go-goal.json), [8CPU snapshot](2026-09-06-rodent-v1.1-release-confirmation.md) |
| Maelstrom 3.3.0 | `H/nnue-owned-maelstrom-20260930-v2/maelstrom-3.3.0` | 3317, 1CPU | [catalog](2026-09-30-owned-top3-go-goal.json), [version receipt](2026-09-30-owned-maelstrom-result.json) |
| GoChess (no UCI version), pin `2c0974e390c85ea1e6f6b08862f38ba748c917d3` | `H/nnue-owned-top3-20260930/opponents/gochess/gochess-pinned` | unknown | [catalog](2026-09-30-owned-top3-go-goal.json); adjacent `build-receipt.json` identifies binary/network |
| Rodent V1.1 Anand/testers | `E/rodent-v1.1-anand-testers-t1-30p0d3/inputs/opponent/engine` | 3522; CCRL personality unspecified | [release confirmation](2026-09-06-rodent-v1.1-release-confirmation.md) |
| Rodent V1.2 non-Tal/testers | `E/rodent-v1.2-nontal-testers-t1-30p0d3/inputs/opponent/engine` | unknown | [release confirmation](2026-09-06-rodent-v1.1-release-confirmation.md) |
| Stockfish 18 | `H/nnue-owned-top3-20260930/gochess-120p1-v3/matches/smoke/stockfish` | unknown | Saved UCI inventory receipt |

Blunder 8.0.0 also has preserved AVX2/AVX512/POPCNT variants under
`H/ngn-original-nnue-night-20261001/hce-reference-artifacts-20261002-1/v8.0.0/`.
Additional Stockfish copies are present in earlier match `tools/stockfish`
and `matches/*/stockfish` directories. UCI probes confirmed the displayed
versions of Blunder 8/8.5.5, Counter 5.5, Maelstrom, Rodent and Stockfish.
Fruit 2.1, Counter 4.x, Weiss, Ethereal, Chess-3 and Combusken executables were not found.
Fruit's recorded 2685 label is therefore unavailable for this run. Counter
4.1's historical 3157 reference is 40/15, not a verified Blitz anchor here.

**Useful bracket:** Counter 3.8 (2994) is the only located rated executable
inside 2800–3200. Counter 5.5 (3359/3360), Maelstrom 3.3.0 (3317) and Zahak
10.0 (3333, 1CPU) provide upper anchors; Blunder 8.0.0 (2674) supplies a lower
check. Counter 3.8 plus Maelstrom and Counter 5.5 are the best immediately
available candidates for a later >3000 measurement. Rodent is farther away;
GoChess needs a rated identity and explicit network admission first.

## Reproduction

- Engine/runner source: `880b6ff9d0fd2cbf0bfae529f48288219d9aa4e5` on `indep/main`.
- Own detached WSL worktree: `H/wt/anchor-orientation-20261005`.
- Build: Go 1.25.5, `GOAMD64=v3 GOMAXPROCS=2 GOTOOLCHAIN=local GOPROXY=off`;
  `go build -p 2 -o "$RUN/ngn" .` and `go build -p 2 -o "$RUN/sprt" ./cmd/sprt`.
- NGN SHA-256: `ae3fd7813a3764acb2d79d71a009ac11f4c567907a5bc21c8591392dbbd14c53`.
- SPRT runner SHA-256: `0325a59396d24d40d0b0c78fb1fab84bc32674757bf6c5f759845046bbde142b`.
- Net: `H/ngn-data/nets/pilot-h256/pilot-h256.nnue`, SHA-256
  `3c6973541ea70ccbbdc82e1eb282e7f967c300be99f7e9ed513a49da286d77e5`.
- The requested `H/repos/ngn/output/sprt_openings.txt` was absent. Used its
  preserved NGN copy at `H/repos/ngn/output/corrected-pin-20260905/results-r0905matepin/inputs/sprt_openings.txt`.
  SHA-256 `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`
  matches [the historical opening receipt](2026-09-05-corrected-mate-pin.md).
- NGN UCI options, in order: `EvalFile=<pilot path>`, `UseNNUE=true`, `Hash=64`,
  `Threads=1`, `OwnBook=false`. Diagnostic probe confirmed backend `ngnn1`
  and effective Threads 1 despite the default UCI version suffix `dev-hce`.
- Opponents: Hash 64, Threads 1 where advertised; Blunder `UseBook=false`.
  Counter has no book option. Counter 3.8's Windows SHA-256
  `d260174182ea10c5b902a8d25e217166f5b5b54dc4fa64c220b498acce6dec11` matches
  its historical receipt; the binary does not advertise its release version.
  Blunder 8.0.0 SHA-256 `f38056868f24bfa73550b69c30eb38b9c8667d87984e3e6d4203716dfcd1aa7f`;
  Counter 5.5 SHA-256 `6c48fb52934d49d3774633e32f0b4fb4796b1e2c63925f24167ef0f0c0d761c8`.
- `go test -p 2 ./cmd/sprt ./internal/uci` passed on WSL. No tooling change.

Each bounded runner job used:

```text
sprt -new <ngn> -base <anchor> -tc 10+0.1 -concurrency 1
  -maxgames <2*opening-count> -mingames 1000000 -maxmoves 400
  -gametimeout 180 -openings <consecutive opening slice> -lowpower=false
  -newoption EvalFile=<pilot> -newoption UseNNUE=true
  -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false
  -baseoption Hash=64 [ -baseoption Threads=1 | -baseoption UseBook=false ]
```

Four foreground game slots; one search thread per engine. Each opening is
played in both colors against every anchor, with equal admitted games. SPRT
score stopping and score adjudication are disabled. Draw rules and a 400-ply
cap remain. A 3600-second wall budget and runtime-only shrinking of final
balanced blocks determine the sample size. Fruit's absence reduces the pool
to the three available requested/upper anchors; no new opponent was fetched.

## Results

**Pooled orientation: 2557, nominal match-only 95% CI [2509, 2605].**
Finished **420 games**, 140 per anchor, over the same first 70 opening pairs.
Wall time **58m10s** (3489.996 seconds), 2026-10-05 20:21:07–21:19:17 EDT
(2026-10-06 00:21:07–01:19:17 UTC). There were **zero time forfeits, watchdog
voids, illegal moves or no-move results**; 395 checkmates and 25 rule draws,
with no 400-ply-cap draws. All paired blocks completed successfully.

| Anchor / label | NGN W/D/L | Raw score | Smoothed p | Performance | Pair-based SE | Pool weight |
|---|---:|---:|---:|---:|---:|---:|
| Blunder 8.0 / 2674 | 42/14/84 | 35.00% | 0.351064 | 2567.27 | 27.44 | 80.12% |
| Counter 3.8 / 2994 | 3/10/127 | 5.71% | 0.060284 | 2516.88 | 57.44 | 18.29% |
| Counter 5.5 / 3359 | 0/1/139 | 0.36% | 0.007092 | 2500.55 | 194.50 | 1.60% |

Pair counts `[LL,LD,{LW,DD},WD,WW]`: Blunder `[24,8,28,6,4]`, Counter 3.8
`[59,7,3,1,0]`, Counter 5.5 `[69,1,0,0,0]`. Counter 5.5 is saturated in
this pilot comparison; Blunder supplies most of the pooled information.
The run offers no support for a >3000 claim for this build/net.

For each anchor, `p=(W+D/2+0.5)/(N+1)` and
`R=label+400*log10(p/(1-p))`, without tail clipping. Uncertainty uses color
pairs: if the five pair counts are `c[k]`, their per-game scores are `k/4`.
With `m=N/2+0.5`, `v_p=((sum(c[k]*(k/4)^2)+0.25)/m-p^2)/m`. The added half
pair is split equally between scores 0 and 1, giving exactly the requested
score smoothing and finite tail uncertainty. Delta-method Elo variance is
`v_R=(400/(ln(10)*p*(1-p)))^2*v_p`. The inverse-variance pool is
`sum(R/v_R)/sum(1/v_R)`, with nominal 95% CI `pool +/- 1.96/sqrt(sum(1/v_R))`.
This conditional interval excludes anchor-rating error, shared-opening
covariance across anchors, clock-transfer error and family non-transitivity.

Raw receipts, all binary hashes, options, exact commands, pair counts and
game-ending reasons: `H/ngn-data/anchor-orientation-20261005/` (`preflight.json`,
`inventory.json`, `inventory-uci.json`, `result.json`, `progress.json`, own
`run.py`/`probe.py`/`inventory.py`, per-job logs and opening slices).
An independent WSL recount verified every log digest, W/D/L total, reason,
pair count, opening coverage, variance and pooled interval. Final engine/runner
hashes were unchanged; no match processes remained. Artifacts occupy about
7.7 MB. `result.json` SHA-256:
`26ab2b010e96cd94869fdeeaa0459307e0d4f2e46d010bd4b5651335909bc1e4`.

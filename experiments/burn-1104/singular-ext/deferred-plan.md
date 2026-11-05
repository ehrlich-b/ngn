# Fixed-node pre-screen: DEFERRED

The implementation and tests are pinned to commit `ea962b51f21e09c0f918a983158d186a70b57aaa`, whose parent is `faaf3741eaf0e2add133ffe6574a3c24db415907`; the candidate patch is included here. This record adds evidence only. The default for T1104SingularDouble remains 0. No other engine source was copied. The net is `engine/nets/ngn-h512-81215229.nnue` (SHA-256 `812152293dad04184599b2f9ab82779168e6e134b221de136d8f8daae86d9f22`).

The option-off identity probe built both executables from `cmd/speedprobe`: the base used a Go overlay of `engine/search.go` from the parent commit and the new binary used the candidate tree. `identity.json` records binary and net hashes and all 12 HCE and 12 NNUE depth-12 comparisons; `identity.log` is the command output. Command (output directed outside the checkout during execution):

```sh
nice -n 10 python3 scripts/speedcheck.py build/sp-base build/sp-new --net engine/nets/ngn-h512-81215229.nnue --cpu 3 --depth 12 --identity-only --out ../evidence/identity.json
```

`targeted.log` records `nice -n 10 go test -p 2 -short ./engine -run SingularDouble -v`. The smoke used the same candidate binary for both players, the option enabled only for new, built-in openings, 40,000 nodes per move, and two games. Command:

```sh
nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104SingularDouble=1 -base build/ngn -nodes 40000 -concurrency 2 -elo0 0 -elo1 5 -mingames 2 -maxgames 2 -maxmoves 300 -pgn ../evidence/smoke.pgn
```

The smoke binary SHA-256 was `1e57d57921ee9de097747309129ce4ba8466ba3be49dc0dac5a2568aebe83b50`; the harness SHA-256 was `17199ada58a9151232e25ef535117f70808cdb41d7b3215bfc7bf98665f5b051`. `smoke.log` and `smoke.pgn` preserve the run. Harness statistics printed for two games are not a strength estimate or a nomination verdict.

After Friday 2026-11-06 18:27 America/New_York, execute the original full command from the implementation-pinned source, with the option as the only difference. Preserve its generated book, PGN and raw output. Stop only at the tool's own verdict or maxgames; do not peek and stop early:

```sh
mkdir -p scratch; GOAMD64=v3 go build -trimpath -o build/ngn .; go build -o build/sprt ./cmd/sprt; build/sprt -lowpower=false -new build/ngn -base build/ngn -genbook 2000 -genplies 6 -balancecp 100 -balancedepth 8 -seed 1104 -out scratch/book.txt; nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104SingularDouble=1 -base build/ngn -nodes 40000 -openings scratch/book.txt -concurrency 4 -elo0 0 -elo1 5 -mingames 1000 -maxgames 4000 -maxmoves 300 -pgn scratch/games.pgn | tee evidence/prescreen.txt
```

NOMINATE if the tool prints H1 or the Penta Elo lower bound is above 0; otherwise SHELVED. No parameter retries. Record option, verdict, penta Elo, interval, games, W/D/L and pentanomial counts in `evidence/nomination.json`. On SHELVED remove the engine change from the branch and retain `evidence/candidate.patch`. The 40,000-node filter is not a 10+0.1 result or an Elo claim. Full execution and verdict remain DEFERRED; the coordinator owns the follow-up.

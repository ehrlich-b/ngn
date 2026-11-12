# NGN

NGN is a UCI chess engine written in Go by Bryan Ehrlich, with its own
NNUE trainer in PyTorch. The default network is embedded in the executable.
This is the development repository; it includes research tools and records.

Releases and downloads live in
[ngn-releases](https://github.com/ehrlich-b/ngn-releases).
Development commits may differ from the published releases.

## Strength

**Official rating: CCRL 40/15.** NGN 0.3.0 (1 CPU) is rated **3411 ±37** from
133 games on the list computed November 7, 2026
([engine page](https://computerchess.org.uk/ccrl/4040/cgi/engine_details.cgi?print=Details&each_game=0&eng=NGN%200.3.0%2064-bit)).
NGN is not yet on the CCRL Blitz list.

**Our estimate for the development version, early November 2026: about 3580
CCRL Blitz (8 threads).** This is our own measurement, not an official rating.
A 320-game gauntlet at 10+0.1 with 8 threads against Koivisto 6.0, akimbo 1.0.0,
Black Marlin 9.0 and Berserk 12, using their CCRL Blitz 8CPU ratings as fixed
anchors, measured **3582 [3542, 3623]**. A longer 120+1 run against three of
those anchors stopped early at 368 games, at 3629 [3593, 3665]. This version
is not released yet.

For scale: CCRL 40/15 ratings run lower than Blitz ratings (the top of each
list, Stockfish 19, is 3650 on 40/15 and 3803 on Blitz). The same kind of local
calibration put 0.3.0 at 3462 against CCRL Blitz 1CPU anchors, and CCRL 40/15
rated it 3411.

Measurement methods and the match ledger are described in
[docs/STRENGTH.md](docs/STRENGTH.md). The dated protocols, opponent anchors,
results and limitations are kept in [experiments/](experiments/).

## Build, test and run

Use Go 1.21.2 or newer. These builds target AMD64 CPUs with AVX2 support;
use `GOAMD64=v1` for a portable AMD64 build. Keep `default.pgo` in the root.

```sh
GOAMD64=v3 go build -trimpath -ldflags "-X main.releaseVersion=0.4.0" -o ngn .
GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -trimpath -ldflags "-X main.releaseVersion=0.4.0" -o ngn.exe .
go vet ./...
go test -short ./...
./ngn
```

Use a UCI-compatible chess GUI, or type these commands into the engine:

```text
uci
isready
position startpos
go depth 8
quit
```

`Hash` sets hash memory in MB; `Threads` sets the search thread count.
Pondering is not supported. Use `UseNNUE=false` for classical evaluation.
For reproducible binary comparisons across commits, build clean source archives
with the same Go toolchain, environment and `default.pgo`; Git checkouts embed
revision metadata by default.

## Repository map

- `engine/`: board, move generation, evaluation, search and UCI handling.
- `internal/`: shared tools for match records, measurement and other support.
- `cmd/`: analysis, data generation, training conversion and match commands.
- `trainer/`: NGN's NNUE training and export tools.
- `web/`: browser chess interface.
- `experiments/`: dated protocols, results and preserved evidence.
- `docs/`: architecture, measurement, provenance and historical notes.
- `testdata/` and `scripts/`: fixtures and development helpers.

## License and credits

NGN is GPL-3.0-only; see [LICENSE](LICENSE) and
[LICENSE-GRANT.txt](LICENSE-GRANT.txt). Credits and historical origins are
recorded in [NOTICE](NOTICE) and [docs/THIRD_PARTY.md](docs/THIRD_PARTY.md).
Training-data terms are recorded in [LICENSES/](LICENSES/).
NGN is developed with AI assistance.

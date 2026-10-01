# NGN

This private recovery candidate defaults to the classical HCE evaluator with
evaluator switching locked. It has no Rodent/K4 runtime evaluator or automatic
model loading. The old `owned` build profile is retired and fails closed.

It is an experimental Go chess engine licensed under GPL-3.0-only. See
[NOTICE](NOTICE), [third-party origins](docs/THIRD_PARTY.md), and the
[recovery evidence](experiments/2026-10-01-attribution-first-runtime-cleanup/report.md).
The engine still includes credited Zahak-derived board/position/hash/cache code
and PeSTO-derived classical tables. This candidate is not claimed to be wholly
original and has no new verified rating. Public historical releases are retained.

A separately created and tested neural evaluator is available as a private
standalone experiment; it has no strength validation and is not selected here.

Build on the authorized WSL host with `go build -o build/ngn main.go`. Existing
owned-release packaging targets are historical and cannot build this candidate.

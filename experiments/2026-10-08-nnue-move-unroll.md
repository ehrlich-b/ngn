# NNUE move-row loop experiment — campaign session 3

Session starts **11:24:19 EDT**, hard stop **13:04:19 EDT**. This is a pure
speed experiment against immediate accepted source `8d9289c` with PGO profile
`616b1ebe` and accepted NGNN3 KB8 H512 net `be34532c`. Longer-net training and
its frozen gate remain under controller 522940; nothing changes those inputs.

Hypothesis: the AVX2 quiet-move accumulator kernel pays avoidable loop overhead
for eight-neuron blocks. Process sixteen neurons per iteration, scheduling two
independent halves before stores. Widths are already validated as multiples of
sixteen; the full int16 row range, wrapping int32 arithmetic, unaligned accesses
and exact source/destination alias contract remain unchanged. Only
`ngnn1MoveRows` changes. Implementation is derived from NGN's own integer
contract, with no third-party implementation or weights.

Prospective acceptance rule, fixed before measurement:

- Existing exhaustive row differential tests must pass for every width 16–2048,
  extreme and random arithmetic, guards, misalignment and exact aliasing.
- Required engine short/race, full short and engine vet checks must pass on WSL.
- All twelve iterative/fixed depth-14 records must match exactly in nodes,
  depth, score, best move and PV for each of accepted `be34532c` and embedded
  NGNN2 `c6c12796`. Twelve HCE depth-11 records must also match.
- Warm both probes once, then exactly **20** interleaved AB/BA timing blocks,
  six positions × three seconds per side, CPU 4, GOMAXPROCS=1, nice 19.
  Use the geometric mean of block aggregate NPS ratios and a two-sided 95%
  t interval on log ratios, df=19. Accept only if lower ratio >1 and both
  order strata have ratio >1. Otherwise shelve with no extension.
- Microbenchmarks are diagnostic only, never a substitute for search timing.
  Preserve full outputs, before/after input hashes, exact build/run commands
  and a result receipt. A timeout or incomplete sample gives no verdict.

Build/tests use CPU 13,15 / nice 19, GOAMD64=v3, Go 1.25.5, a private cache
and separate source snapshots under `night-20261007/session3/`. Both probes
explicitly use the same accepted PGO profile. No candidate source is committed
before its verdict. The campaign tuner is drained and resumed from its latest
retained checkpoint around identities/timing; GPU training continues on 12,14.
The existing refresh cache is already implemented and is not changed.

**SHELVED**, reviewed **11:51 EDT**. Controller **544503** completed **11:49:04**;
all required checks and **36/36 search identities** passed. Twenty complete
blocks give geometric NPS ratio **0.999722 [0.991891, 1.007615]**, AB/BA strata
**0.998637 / 1.000808**. This fails the fixed lower-ratio and both-order rule.
The assembly edit was restored; no extension and no candidate installed.

Diagnostic H512 move microbenchmarks improved from a median **28.33 ns** to
**25.59 ns**, demonstrating why microbenchmarks cannot replace whole-search
evidence. Accepted-net profile: move kernel **6.32% flat**, refresh cache **7.36%
cumulative**, output kernel **4.56% flat**. The next separate speed experiment
can reduce redundant refresh-cache traversals, rather than extending this sample.

Exact [build receipt](2026-10-08-session3-results/move-build-receipt.json),
[source/test receipt](2026-10-08-session3-results/move-source-tests-receipt.json),
[complete timing receipt](2026-10-08-session3-results/move-timing-receipt.json),
[profile](2026-10-08-session3-results/profile-top.txt). **227** compilation/profile
inputs differ only in the proposed assembly file. Source archive SHA-256
`c5e54276a9044eb0ea18f49df6c35b7ecdd045cfa4777d14a511e63b3ecd4065`,
patch `a0986129f5d76e6acca479fda06ec8081e278572625b7c9d88dcb6624abcba75`.
The exact rejected patch, raw records and binaries remain on WSL under
`night-20261007/session3/`; no speculative engine code is committed.

The first profile-only launch, PID **544408**, never began: no profile output or
engine existed. Its empty original log is preserved. The verified controller
544503 performed the single profile/build/test/timing sequence. Frozen input
hash checks pass before/after; SPSA resumed as **552797** from its retained
checkpoint. Trainer and frozen longer-net gate inputs were untouched.

# N1 richer-model feasibility: select Stockfish 18 BIG/dual

Status: **N1a and N1b complete; SF18 BIG/dual remains open for N1c.** This
does not integrate a new evaluator into search, change the default, authorize a
match, or establish any Elo gain.

The immediate question after the rejected P1 staged-picker candidate was which
single richer pretrained evaluator deserved the next architectural lane.
Berserk 14 and Stockfish 18 BIG/dual were the two predeclared alternatives. The
decision is now evidence-bound:

- Berserk 14's pinned source names `berserk-9b84c340af7e.nn`. The exact release
  asset is 25,201,924 bytes and hashes to
  `9b84c340af7e45f6e07f0046235ccb327f4ae0840c8ee2c4b97b99121e5c5084`.
  Its engine source is GPLv3-or-later, but the separate
  `jhonnold/berserk-networks` repository contains model blobs without a license
  or model-provenance statement. That fails this project's pre-incorporation
  gate. It is not a technical rejection of the architecture and can reopen if
  the model owner supplies an explicit artifact license/provenance binding.
- Stockfish's official network archive pins `nn-c288c895ea92.nnue` at
  revision `dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4` with the full SHA-256
  `c288c895ea924429ea9092e3f36b2b3c1f00f2a3a4c759ff7e57e79e3b43e4a7`
  and size 108,919,594 bytes. The archive states that all uploaded networks are
  under CC0. NGN's existing strict BIG loader already passed its official-file,
  independent tensor-spot, focused, race, vet and build gates. This clears the
  provenance and loader prerequisites without reopening them.

Primary bindings:
[Berserk 14 makefile](https://github.com/jhonnold/berserk/blob/8ae895a6151695be4a50d4fb65b0c131659c513a/src/makefile),
[Berserk network repository](https://github.com/jhonnold/berserk-networks),
[Stockfish network archive](https://github.com/official-stockfish/networks/tree/dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4),
[archive license](https://github.com/official-stockfish/networks/blob/dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4/LICENSE), and
[pinned SF18 source](https://github.com/official-stockfish/Stockfish/tree/cb3d4ee9b47d0c5aae855b12379378ea1439675c).

## Slice N1a: selected-head propagation

The existing SMALL implementation was deliberately a diagnostic oracle: every
call propagated all eight material heads and retained wide intermediate traces.
That would make any production comparison artificially slow. N1a adds a
separate trace-free `EvaluateSelected` path for both full refresh and the
worker-private incremental context. It propagates exactly one material-selected
head; `EvaluateAll` remains unchanged as the oracle. Nothing calls the new path
from engine search.

Correctness passed at three independent levels:

- synthetic all-bucket, both-side, defined-wrapping and real-transition tests;
- all 145 pinned full-refresh upstream Stockfish fixtures; and
- all 703 pinned upstream incremental rows, including real/null transitions,
  refresh boundaries, stack growth and unwind.

The configured package race gate and repository-wide short suite passed on WSL
using Go 1.25.5, `GOAMD64=v3`, bounded `GOMAXPROCS=2`, and the exact official
SMALL model. The source hashes and commands are retained in `result-v1.json`.

On one pinned Ryzen 7 9800X3D core, ten one-second blocks measured:

| Path | All-stack median | Selected median | Ratio | Reduction |
| --- | ---: | ---: | ---: | ---: |
| Full refresh + output | 30,825 ns | 5,223.5 ns | 0.16946 | 83.05% |
| Existing incremental state + output | 21,619 ns | 1,420 ns | 0.06568 | 93.43% |

This proves that the all-head diagnostic path is not a valid production-cost
proxy and removes that known confounder before BIG work. It does not predict
BIG cost: BIG has a 1,024-byte transformed input and FullThreats state, while
SMALL has a 128-byte transformed input and no threat features.

## N1b gate definition

The exact eager dual numeric footprint remains 137,844,032 shared model bytes
before object overhead and at least 8,896 bytes per live ply per worker. At 128
frames that is 1,138,688 numeric bytes per worker. This is feasible in memory,
but blindly copying the full dual frame at every edge would repeat a design that
the richer model makes much more expensive. Stockfish's demand-driven
accumulator stack is therefore part of the cost model, not optional polish.

N1b is one bounded, non-search-integrated BIG reference slice:

1. Freeze legal move sequences before looking at timing. Reuse the accepted
   SF18 incremental action corpus and add representative game prefixes only if
   needed to cover occupied slider targets.
2. Generate an independent oracle from pinned Stockfish for both perspectives:
   active FullThreats indices, exact before/after added and removed sets,
   refresh decisions, base/threat accumulators, PSQT, transformed input and the
   material-selected head.
3. Implement only scalar exact BIG full refresh plus a simple sorted-set threat
   diff. Measure active threats, changed rows, refresh incidence, enumeration,
   1,024-lane update, transform and selected-head costs separately. Do not port
   Stockfish's fused/double update machinery in this slice.
4. Report the observed distribution and an explicit optimized-cost ceiling.
   No search/UCI wiring, games or default change may occur until this result is
   reviewed. Failure of exact parity or an unbounded need for donor machinery
   shelves the route; lower NPS by itself does not.

## Slice N1b: exact BIG reference and cost boundary

N1b implements the bounded scalar reference without crossing the search
boundary. The BIG package now has separate HalfKAv2 and FullThreats accumulator
state, allocation-free active-threat enumeration, a simple sorted-set
transition diff, exact 1,024-lane base/threat row updates, transform, and the
material-selected output head. It does not have a worker context, accumulator
stack, SIMD, UCI option, search adapter, default change, or game authorization.

The pinned Stockfish diagnostic was built independently and run on the already
frozen core/growth legal corpus plus 12 frozen 64-ply real-game prefixes. Every
one of 1,203 rows matches for both perspectives across active threats,
before/after threat sets, refresh decisions, base/threat accumulators and PSQT,
transformed input, raw selected outputs, and divided components. Each upstream
incremental row is also compared with a fresh Stockfish stack/cache before it
is admitted. Exact model/oracle/source hashes and commands are in
`result-v2.json`.

The representative 780-row corpus prevents the 128 sparse king-coverage cases
from distorting the cost model:

| Distribution | Median | P95 | Maximum | Mean |
| --- | ---: | ---: | ---: | ---: |
| Active threats / perspective | 36 | 48 | 54 | 33.18 |
| Changed threat rows / perspective-transition | 7 | 13 | 82 | 7.24 |
| Changed threat rows / nonrefresh move, both perspectives | 14 | 26 | 36 | 14.10 |
| Base rows / perspective | 2 | 3 | 4 | 2.27 |

Only 6/1,536 perspective-transitions cross the FullThreats king-file boundary;
47/1,536 move their own king and require a base refresh. The large threat-diff
maxima are those refresh cases, not unbounded ordinary updates.

Ten one-second blocks on one pinned Ryzen 7 9800X3D core measured these scalar
medians, all with zero allocations:

| Kernel | Median ns/op |
| --- | ---: |
| FullThreats enumeration, one perspective | 2,786.5 |
| Sorted threat diff, one perspective | 126.3 |
| One threat row, 1,024 lanes + PSQT | 536.95 |
| One base row, 1,024 lanes + PSQT | 451.85 |
| Transform | 5,519.5 |
| Material-selected head | 6,073.5 |
| Complete two-perspective full refresh | 57,791 |
| Complete full refresh + selected output | 69,694 |

Charging observed changed rows, both enumerations/diffs, and conservatively
replacing every own-king move with an entire two-perspective refresh gives a
measured scalar-kernel ceiling of 15.15 us median and 17.93 us mean for
maintenance. Adding transform/head gives 26.74 us median and 29.53 us mean;
the P95 is 69.38 us because own-king moves make up about 6% of moves. This
is a bounded reference ceiling, not an acceptable eager production cost and not
a search-NPS prediction. Stack bookkeeping and evaluated-node incidence remain
for N1c.

N1b therefore passes rather than shelves the route: exactness is complete,
feature/update counts are bounded, and refresh incidence is low. The next gate
is N1c only: implement a non-search-integrated demand-driven BIG context that
stores dirty transitions, computes from a usable donor only when evaluated,
does not copy 8,896-byte frames on every edge, and matches the same independent
oracle. It must measure evaluated-node cost before any UCI/search/default/game
work.

The goal remains active. N1b establishes feasibility, not playing strength or
a 3300 rating.

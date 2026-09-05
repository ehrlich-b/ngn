# Classical model v2: implementation and first grouped corpus

Status: offline implementation integrated; no evaluation weights changed and no
new playing default installed. This addresses the [classical comparison's first
target](2026-09-04-classical-evaluation-gap.md), not a claim of Elo improvement.
The active goal remains at least 2800 without NNUE; the latest absolute pin is
2726 [2668,2784]. The separate frozen qsearch repair is in its own game gate.

## Model and fidelity

`ngn-texel-v2-936` preserves the 798-entry legacy prefix and appends 66 MG
mobility cells, 66 EG cells, and six threat coefficients. Two legacy mobility
scalars remain explicitly inactive reservations: 936 stored, 934 active.
No weights are transplanted from another engine. The existing live evaluator
`engine/eval.go` and its coefficients are byte-identical to the pre-expansion
source; extraction helpers live in offline tuning files.

The JSON export includes version, exact ordered names, integer values, stored
and active counts, and inactive reservations. `scripts/apply_tuned.py` validates
all metadata and all 936 entries before changing either source file. Go-source
input remains available for legacy coordinate descent. Source writes are staged
but two file replacements are sequential, not a two-file atomic transaction.
`ApplyTexelModel` clears derived caches and requires fresh/rebuilt board
accumulators; it is an offline/startup API, never safe during an active search.

Proofs completed:

- Every appended coefficient changes the actual live evaluator and matches a
  previously captured feature trace under perturbation: 138/138, no zero skips.
  Mobility-bin coverage uses constructed evaluator boards (including pawn
  blockers on back ranks), explicitly not a reachable-game training corpus.
- Existing diverse legal walks test frozen-trace parity under changes to the
  complete parameter vector; numerical gradients cover the new coordinates.
- `scripts/verify_texel_model.py` copies sources to a temporary directory,
  changes every one of 936 values, imports, rebuilds the actual engine and tuner,
  and requires the compiled dump to equal the full expected vector. Invalid
  versions/names/counts/reservations, missing values, booleans, NaN, out-of-range
  integers, extra and duplicate JSON fields reject before source changes.
- Root-built initial model preserves nodes, score, complete PV and best move on
  the three canonical searches versus frozen qcap: 463172 / 138436 / 858070.
  Evidence: `output/recovery-2026-09-04/texel-v2-identity.json` and UCI logs.
- An independent read-only agent review found no model-order/activation/import
  blocker; it noted the sequential source-file replacement limitation above.

## Grouped data contract

`cmd/texelcorpus` records full startpos UCI move lists, canonical opening IDs
(placement/turn/rights/EP), absolute opening indices, nodes/maxplies/seed,
generator executable SHA/source, filter ID/source, terminal reasons and White
outcomes. Only legally completed games carry labels; max-ply truncations remain
unresolved and contribute no rows. Mate takes precedence over clock draws.
Sampling uses a seed derived from the configured seed and canonical opening,
so sharding and duplicate book indices cannot change a played game's samples.

The loader replays every game, checks samples against exact replayed FENs, rejects
terminal continuations, false/truncated max-ply records and terminal samples,
deduplicates identical games, and partitions whole opening groups 80/10/10 by
fixed hash. It keeps at most 32 board-distinct samples per game. Distinct games'
observations remain within a partition; cross-partition board collisions are
kept only in test, then validation, then train. File order cannot choose outcomes
or row order. Reports retain input checksums, exclusions, completed groups,
partition row W/D/L and training phase/new-feature support.

The loader deliberately does not rerun weight-dependent quiet classification:
the generator's exact source/filter identity is its provenance. It validates
legal replay and labels, not whether a newer evaluator would make the same
quietness decision. Existing anonymous flat datasets cannot prove game-disjoint
validation; the old loader also now rejects nonfinite/out-of-range labels.

`texel -mode audit -corpus ... -report ...` measures corpus structure and training
feature support without fitting. `-mode gradient -corpus ...` calibrates K on
train only, uses validation for checkpoint selection, and measures baseline and
fitted integer models on the test partition only after selection ends. A
regression changes test labels and proves the fitted model, K and validation
loss stay identical. Test-set results may reject this one fit, never select
hyperparameters or another checkpoint from it.

## Predeclared first corpus and fit

Before generation: reserve original opening lines 0–999 for confirmation games.
Canonical audit finds 1000 unique reserved groups and zero overlap with lines
1000–4999. Remove four duplicate training groups, leaving **3996** unique
training openings in original order. Frozen training book:
`output/recovery-2026-09-04/classical-v2-training-openings.txt`, SHA-256
`4f7e93fb77514d27c4e565ac1ab4d54fb37511aa54700473a91d7f50688e1c05`.
Original book SHA-256:
`974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
Original-index mapping and exclusions: `classical-v2-book-audit.json` in that
output directory. No new opening source or opening cycling is used.

1. Freeze a committed generator source and binary hash. Smoke exactly 32 games,
   first 32 filtered opening indices, 8000 nodes/move, max400 plies, seed20260905,
   GOMAXPROCS=1 on the Mac. Require no operational failure, valid full replay,
   exact recorded settings and nonzero completed games/samples. Exclude this
   smoke file from the final corpus; the full run regenerates these openings.
2. If smoke passes, generate the entire 3996-opening range once: four independent
   Mac processes, GOMAXPROCS=1 each, ranges [0,999), [999,1998), [1998,2997),
   [2997,3996), same frozen binary/settings. No Windows/WSL compute contention
   with timed games. Every process ends after its fixed range, no recycling.
3. Load all four shards with split seed `ngn-classical-v2`. Require all replay
   checks, at least 60% legally completed games, at least 1500 train and 100 each
   validation/test completed opening groups, and 20000 train rows. Each split
   must contain all three outcomes, with no outcome exceeding 90% of its rows.
   Report unsupported mobility/threat coordinates rather than asserting full
   empirical support; unsupported weights stay at their starting values.
   Failure stops this fit and prompts diagnosis, not favorable-seed resampling.
4. After the qsearch repair verdict and corpus acceptance, run one joint fit:
   300 maximum epochs, LR1, patience30, L2=1e-7, baseline-calibrated fixed K.
   Freeze the selected rounded model and compare both integer validation and
   final-test MSE with their baselines at the same K. Both must improve to
   advance; a lower float training loss is insufficient. Do not refit based on
   the final-test result. Record exact model/report/binary hashes.
5. A model that passes offline gates still needs a separately predeclared paired
   real-clock comparison against the then-accepted immediate base, capped at
   1–2 confirmation hours. No default changes or speculative-weight commits
   before that game verdict. No predicted or added-up Elo gains.

The generator includes the committed qsearch correction currently under game
test. Those are legally played outcomes regardless of that implementation's
eventual strength verdict. Fitting/playing integration waits for that verdict;
generation does not establish or substitute for its acceptance.

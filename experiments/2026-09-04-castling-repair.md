# Internal castling legality can produce a losing UCI fallback

Status: proved defect, isolated repair in progress. This takes precedence over
the unfinished T8 heuristic confirmation, which was cancelled without a verdict.
Reviewed/played source: `71399c0`; main's later changes are offline tuner/docs only.

## Mechanism evidence from an actual loss

Fresh gauntlet GAME170, ply16:

```text
r1bqk2r/pppnnpp1/3p3p/4p3/3P1PPb/1P2P2P/P1P1N1B1/RNBQK2R w KQkq - 3 9
```

The black bishop on h4 checks the white king along g3–f2–e1. Independent Stockfish
perft lists exactly **Ng3, Kf1, Kd2**. Castling is illegal. The released engine
produces this result from either the FEN or the complete game history:

```text
info depth 1 ... score cp 3  ... pv e1g1
info depth 2 ... score cp -17 ... pv e1g1
info depth 3 ... score cp -3 ... pv e1g1
bestmove e2g3
```

After Ng3, NGN itself finds **...Bxg3+**, winning the knight, at depth 1. At depth
3 it scores Black +439. The internal root search optimizes an illegal castle;
the existing UCI legality guard substitutes the first legal fallback, Ng3.
Therefore legal emitted moves and a clean game replay do not establish internal
search legality. The search's chosen best move and PV need their own checks.

In the targeted diagnostic, 400k nodes and each of ten individually disabled
search heuristics still produced Ng3. Complete SF depth14 prefers Kf1 (root −105)
and scores Ng3's continuation around −461. This is a demonstrated search-rule
failure, not evidence that a new evaluation model is required.

Raw full-history/FEN/independent perft transcript:
`output/recovery-2026-09-04/illegal-castle-repro.txt`.
Targeted ablations and exact prefixes: `targeted-ablation.json`.

## Repair scope and validation requirements

Trace the shared and duplicated castle checks. The iterative root's fast
make/final-king-check path does not prove starting/transit square safety.
The in-check qsearch and offline quiet-search consumers need the same audit.
The main alpha-beta loop already has special castling checks; prefer consistent
semantics over another independent partial implementation.

The isolated Terra checkout is `output/castle-repair-20260904`. Root reviews and
controls integration, game tests and deployment. Cover both colors/wings,
starting check, attacked transit squares, final-square safety, valid castles,
repeated searches and caller-state preservation. Test the actual internal best
move/PV before UCI fallback, and the full-history case above. Run short/race
suites and the independent move-generation oracle as relevant.

No speculative vector or new weights are included. A source proof and regression
suite establish legality; a bounded real-clock comparison with the frozen release
checks cost and playing behavior. Record its exact manifest before launch. The
existing accepted release stays deployed until the new repair is validated.

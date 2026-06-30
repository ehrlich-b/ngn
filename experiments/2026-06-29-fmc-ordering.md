# 2026-06-29 FMC ordering campaign — Task 2 baseline: WHICH class causes the ordering misses

Codex's brief (the night's plan) added a correction to the tree-shape lever: before touching the
history formula, **attribute the non-first cutoffs by move class** so we don't fire another blind
search tweak. If the missed cutoffs are capture-heavy → history weighting is useless (need capture
ordering/SEE); if quiet-history-heavy → the history formula is the right lever.

## Instrument (Task 1, committed c20c3fc, node-identical)

`sd_miss` dump (`debug on`): among NON-first beta cutoffs (the ordering misses, = bcut − fmc),
the class of the cutting move (what *deserved* to be ordered first), split move2 / move3 / move4+.
Class 0..5 = TT / capture / promo / killer / counter / quiet-history. Never read by search
(node-identity proven: HEAD vs instrumented byte-identical on all 3 nodecheck positions; gate
baselines refreshed e16f512 — they had been stale since the 2026-06-05 pre-reset lock).
Self-consistency verified: the six class totals sum exactly to bcut − fmc on every position.

## Data — `ebfprobe.py -diag` on the 6 canonical positions (deterministic, base==cand)

### 400K nodes

| position   | FMC%  | misses | cap% | quiet%* | move4+ qh%** | ttlist% |
|------------|-------|--------|------|---------|--------------|---------|
| QGD mg     | 84.9  | 5487   | 19   | 81      | 75           | 28      |
| closed mg  | 83.4  | 6113   | 34   | 66      | 58           | 27      |
| lateMg     | 83.7  | 6147   | 28   | 72      | 84           | 27      |
| najdorf mg | 83.6  | 6181   | 29   | 71      | 68           | 30      |
| rook eg    | 83.6  | 7784   | 5    | 95      | 99           | 38      |
| pawn eg    | 89.6  | 8003   | 3    | 97      | 100          | 93      |

\* quiet% = killer + counter + quiet-history share of all misses.
\** move4+ qh% = quiet-history share of the *expensive* misses (the cut landed on move 4 or later,
   i.e. ≥3 child searches wasted). This is the ordering-failure signal that actually costs nodes.

### 1600K nodes (depth confirmation)

Same shape, depth-stable. FMC 82.9 / 82.3 / 83.5 / 83.2 / 84.7 / 88.4. move4+ qh% =
76 / 54 / 79 / 64 / 99 / 99. The precision gap and its quiet-history character both persist at depth.

## Read — decisive, and it resolves codex's disambiguation

1. **The expensive ordering misses are overwhelmingly quiet moves the history table under-ranked.**
   move4+ misses are 54–99% quiet-history across every phase and both node budgets. When NGN wastes
   3+ child searches before cutting, it is almost always a quiet move that no heuristic (TT, capture,
   killer, counter) flagged and the general history score ranked too low.

2. **Captures are NOT the problem.** Capture share of misses is 3–5% in endgames, 19–34% in
   middlegames, and is concentrated at move2 (cheap — 1 wasted node, behind a non-cutting TT/cap).
   → codex candidate #5 (capture-history / MVV) is **ruled out** by the data.

3. **Killer/counter are working.** A large share of move2 misses are killers — that is the killer
   heuristic doing its job (the killer is *supposed* to be tried early, just behind captures). Not a
   failure mode; not the lever.

4. **TT-move availability is low in the middlegame (~27–30%).** When there is no hash move (70% of mg
   nodes), ordering rests entirely on captures + killers + history — so history quality governs those
   nodes directly. codex candidate #6 (TT retention) is a real secondary lever; pursue after history.

## Decision → Task 3 candidate order (one at a time, FMC-proxied, then ONE SPRT)

Pursue the quiet-history-quality levers, in codex's order:
1. **history update amplitude** (bonus shape — NOT historyMax, which desyncs the killer/counter bands
   per moveorder.go:103).
2. **continuation/followup weighting** — GetHistoryScore sums main + cont + followup EQUAL
   (moveorder.go:168); strong engines lean on continuation history. Most aligned with the data
   (move4+ quiet misses are position-specific → continuation signal should rank them).
3. **low-depth first-quiet anti-pollution.**
4. butterfly `[color][from][to]` index (main history is coarse `[piece][to]`) — only if 1–3 fail.

Proxy pass bar (codex): FMC improves ≥~1 point on ≥3/4 middlegames, b_all does not inflate, depth
does not drop materially. Keep nothing on proxy alone — proxy-pass → manifest → one real-clock SPRT
vs immediate base on the box.

Reproduce: `python3 scripts/ebfprobe.py -diag -nodes 400000 BIN BIN`, read `sd_miss` / `sd_order`.

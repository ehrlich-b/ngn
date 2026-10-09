# NGN exchange-value candidate — plan item0b(c)

Independent topic `night/see-values`, `output/campaign-see-values/`, sourcea49ebfc,
immediate accepted runtime5b882e2/be34532c. One ordering/SEE value table changes:
P100, N300, B340, R500, Q950, K32767. The prior320/330/525/1000 quartet is removed
from `Piece.weight`, used by SEE, qsearch capture gain/MVV-LVA and promotion
ordering. Network, HCE/PST evaluation, search/time algorithms, data and Hash
remain unchanged. No TT, board or time candidate is stacked.

These are declared NGN priors, not learned/tuned or claimed optimal values:
the pawn defines100, three-pawn knight, modest bishop premium, five-pawn rook,
and queen below two rooks. The explicit bishop/knight gap discourages throwing
a bishop away for a knight in exchange pruning. Only actual paired games can
decide whether this is useful. Piece-value API tests reflect the changed policy;
full semantic engine tests must pass without weakening their SEE/move contracts.

Detached code checks in `session19-see/` wait for the two CUDA windows, then
use the newly free12,14 while Rodent occupies0–11 and TT games occupy13,15.
Require engine/full short/race, vet and accepted-PGO builds. No source commit,
games or promotion is queued. After TT/board provenance work and factorizer
review, replay the tiny table change onto the actual immediate accepted source/net.
Write the exact manifest before **10+0.1 A/A40 then400 paired games** by the
standard rule required by plan0b(c): penta lower95>0, complete200 pairs, zero
flags/crashes/illegal/no-move results, intact pins; otherwise shelve without
extension. Preserve the old-base tune and require its own converged-vector games.

Session21 retains the initial stale-arithmetic failures, corrects all crafted
SEE expectations without changing FEN/move semantics, and rebases the table
onto39eb128 on night/see-provenance-gate. Corrected full checks are675045,
held for disk recovery. Future provenance games use Bryan's updated upper95>0
rule, provisional with later combined certification; the standard lower95
rule still applies to Elo candidates. [Current record](2026-10-09-session21.md).

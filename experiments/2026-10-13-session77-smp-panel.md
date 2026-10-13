# Session77 prospective Hash256 worker panel

Diagnostic source: accepted f1d0a03/a4d90b41/final14 plus campaign-only hooks,
never a playing binary. Twelve full-history frames deterministically selected
from the frozen54, sorted by outcome/color/move-band/material/id and taking
floor(i*54/12), i0–11. Selection frozen before any new searches; their entire
original opening pairs already excluded from the next game's4947 book.

Main panel: Hash256, Threads1/4, movetime250/2000ms with zero overhead,
six balanced AB/BA blocks; 288 searches. Four distinct physical rig cores,
GOMAXPROCS=width, cold engine/history/TT per cell, no training or games overlap.
Record process CPU time, exact joined nodes, worker starts/finishes and completed
iteration move/depth/score/timing. Repetition history replay and root restoration
must pass. Setup includes evaluator resets; TT allocation is before move time.

Separate sampled diagnostic: one balanced block (48 searches), private bounded
buffers sample alpha-beta (hash, remaining depth, worker) each4096 completed
nodes; no qsearch-key sampling. This is a biased overlap proxy, not duplicate-work
fraction. Do not pool its timings with unsampled panel. Overflow reported.
Eligible disagreements: after join, votes from completed workers whose completed
depth is at least primary's, replace only with strictly larger group than the
primary's group, at least2 votes. Ties retain primary; deepest/lowest-index
representative. Count opportunities; do not call agreement a solved position.
Prospective admission: at least5/144 Threads4 cells and at least2 distinct frames
have eligible replacement, with no legality/lifecycle failures. Below that, hold
consensus games and move to helper root tie diversity. This cheap rule buys games,
never promotes source or claims Elo.

Held-out24 elementary owned mate fixtures (three geometries times eight square
symmetries), 250/2000ms, three repeats, widths1/4. Answers are exhaustively checked
legal mate-in-one moves independently of search/eval; report misses/reversals.
These have custom-FEN initial history; the realistic12-frame panel preserves full
game histories. They are correctness sentinels, not a general tactical solve set.

Run only after SEE's reviewed terminal. Clean SEE shelf permits unchanged-source
panel automatically; SEE winner/fault holds for manual immediate-base review.
Spare13/15 may compile/check this diagnostic while SEE plays, with no timing
claims. The final panel waits natively for completion and needs four physical
cores. No foreign source/net, new training, engine adoption, game verdict or claim.

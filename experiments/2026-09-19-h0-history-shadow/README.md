# H0 passive history-shadow diagnostic

Status: workload selection frozen; observer contract and execution pending.

H0 is the mechanism screen for the bounded Counter-inspired quiet-history
package in the September 19 Sol plan. It will retain the accepted V1.2
evaluator and the current search tree while maintaining a write-only shadow of
the proposed completed-node labels and response rate. No shadow value may feed
move ordering, pruning, reduction, evaluation, TT storage, or time control.

The workload is selected before observer results: source games 4–11 inclusive
from the immutable accepted Counter match PGN, excluding the three games used
by earlier profiling. Selection is sequential with no result or position
filtering; the extracted artifact intentionally omits results. Each game must
have at least 64 plies. Searches will occur after plies 16, 32, 48, and 64,
retaining the same searcher, TT, actual history, shadow history, evaluator, and
repetition-bearing position inside each game, with `NewGame` between games.

The source PGN SHA-256 is
`35bdd00b28f585224e6c04392bdbf79cd4472c9ea7549b84acfc4379b07847d2`.
`extract_prefixes.py` is the frozen result-blind extractor.


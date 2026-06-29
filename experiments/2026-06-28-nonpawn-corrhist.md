# 2026-06-28 Non-pawn correction history — real-clock self-play SPRT (STAGED)

Structural ADDITIVE lever (not a single-knob retune, so not a local-optimum wall the
way the LMR divisor was). NGN has only `pawnCorrectionHistory`; strong engines run
several. This adds a second corrector keyed on each side's NON-pawn piece placement,
summed with the pawn correction at the static-eval site. Targets the proven
eval-optimism / horizon-error loss pattern (`project_loss_autopsy_eval`,
`project_search_bound_proven`): it nudges the optimistic leaf eval toward what search
actually finds, which is the cure for horizon errors in a search-bound engine.

```yaml
id:            2026-06-28-nonpawn-corrhist
date:          2026-06-28
change_class:  search/eval heuristic (eval correction) — gate = real-clock self-play SPRT
hypothesis:    a second (non-pawn) corrector reduces optimism at the leaves ⇒ NEW > BASE
base_commit:   2dadef8 (build/win/ngn_base.exe on box)
candidate:     working tree (moveorder.go nonPawnCorrectionHistory + search.go wiring) -> build/win/ngn_corr.exe (staged on box ~\ngn\sprt\)
command:       sprt.exe -new .\ngn_corr.exe -base .\ngn_base.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 0 -elo1 6 -maxgames 3000 -mingames 400
local_checks:  go build OK; go test -short ./engine GREEN (eval symmetry holds — correction is in search, not Evaluate); depth-14 search sane.
decision_rule: KEEP if real-clock self-play shows NEW >= BASE non-regressing. If it REGRESSES, first suspect over-correction (pawn+nonpawn each ±49cp, summed ±98) — halve the nonpawn weight (6245/131072 -> ~3000/131072) and retry before shelving. Then commit-or-revert.
status:        STAGED — box is busy with the joint-SPSA; run this the moment the box frees (and decide base = HEAD vs SPSA-tuned params depending on the SPSA verdict).
result:        PENDING
verdict:       PENDING
```

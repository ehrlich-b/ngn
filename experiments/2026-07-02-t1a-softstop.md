# 2026-07-02 T1a — soft-stop drops the next-iteration projection (time-management)

Pre-registered BEFORE launch (CLAUDE.md work-loop step 5). One isolated change, candidate vs its
immediate base (post-iir3 HEAD), **real-clock ONLY** (time/TC changes gate on real-clock games, never
a proxy). Speculative until its verdict: NOT committed; the diff is backed up at `output/t1a.patch`.

```yaml
id: 2026-07-02-t1a-softstop
date: 2026-07-02
change_class: search/eval heuristic (time-management) — real-clock-ONLY gate
hypothesis: firing the soft stop at elapsed>=soft (vs the prior ×1 next-iter projection) spends more of the soft bank -> more depth, with no added flag risk
base_commit: 6959a14 (post-iir3 HEAD; includes iir3 provisional #1)
candidate_commit: 6959a14 + output/t1a.patch (engine/time.go + engine/time_test.go only)
base_binary_sha256: 85703c188be79b728909433cb91a2b673efe3b03a6bc6fabc8fce59e1cd58aa5 (ngn_base2.exe, built from 464c33e = post-iir3 engine)
candidate_binary_sha256: 7193250d355093bd1f75567bd7f3e19e4ddf0b8debdf4899415ad2b71922ae33 (ngn_t1a.exe, 464c33e + T1a)
harness_commit: sprt.exe sha256 e9d904b878f4cb373bcc55abc325763d24f81f75248e9cb9d1540c6b93d3a668 (box, unchanged)
command: sprt.exe -new ngn_t1a.exe -base ngn_base2.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
go_version: go1.26.2
goarch_goamd64: windows/amd64 GOAMD64=v3
tc: 10+0.1 (seconds; bullet)
concurrency: 8
openings: canonical sprt_openings (5000 lines)
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
aa_preflight: M2 c8 A/A LAUNCHED 2026-07-01 ~23:07 (aa_c8_out.txt, ngn_base2 vs ngn_base2, 1600g, adjudication OFF, via scripts/boxsprt.sh); healthy start confirmed; final penta/flag-out result pending completion
decision_rule: H0 elo<=-3 / H1 elo>=+3 (alpha=beta=0.05 -> pLLR bounds +-2.94); 8000 max / 300 min; adjudication OFF. pLLR>=+2.94 KEEP; <=-2.94 REJECT; capped-positive (point>=+1, pLLR>0, no regression) PROVISIONAL keep #2 (batch cert); capped-nonpositive SHELVE; candidate flag-outs above the A/A baseline (0) -> HALT + investigate
games_or_pairs: TBD
result: TBD
flags_errors: TBD
verdict: TBD (NOT launched; launch order + A/A pass call come from the team lead)
next_action: cross-compile ngn_base2/ngn_t1a (Phase C), run the M2 c8 A/A, then (on the team lead's call) launch this verdict
```

## Hypothesis
NGN under-spends its soft clock. The tournament soft-stop fired on the ×1 next-iteration projection
(`elapsed + lastIterationTime > soft`), which trips once `elapsed` reaches `soft − lastIterationTime`
— roughly 2/3 of the soft budget — leaving the bank underused on a depth-starved engine. Firing only
when the soft budget is actually spent (`elapsed >= soft`) buys ~1 more iteration's worth of clock per
move (more depth) WITHOUT touching the hard ceiling or emergency floor, so it cannot raise flag risk.
Class prior: Stash v26 time-management package +31/+29 Elo. This is the 2-week-overdue staged 1-liner
(TRIED-LEDGER "Time management" residual: Counter spends ~98% of clock vs NGN ~70-80%).

## Mechanism (the single change)
`engine/time.go` `shouldStopTournamentSearch`: the soft-limit return changes from
`elapsed + tm.lastIterationTime > soft` (with the `lastIterationTime==0` special case) to
`return elapsed >= tm.softTime`. The hard ceiling (`elapsed >= hardTime`, capped at ≤30% of the usable
bank), the emergency floor (`baseTime − elapsed <= emergencyTime`), and the "finish ≥1 iteration" guard
are UNTOUCHED — they remain the flag protection. `lastIterationTime` is still maintained (its write path
is unchanged) but is no longer read by the soft stop; a KEEP would remove the now-unused field as
follow-up cleanup, a REJECT reverts this single hunk.

`engine/time_test.go` `TestTournamentSoftStop` updated to the new semantics — red→green: the old code
stopped at `elapsed=3700ms` (3700+500 > soft 4000) via the projection; T1a must NOT stop before
`soft=4000ms` and DOES stop at/after it. Full `go test -short [-race] ./engine` + `go test -short ./...`
GREEN with the change.

## Change isolation
- Base = post-iir3 HEAD `6959a14`. Binary `ngn_base2.exe` (cross-compiled from clean 6959a14).
- Candidate = base + T1a (`output/t1a.patch`; engine/time.go + engine/time_test.go only — verified via
  `git status`/`git diff --stat`, 20 insertions / 11 deletions across the two files).
- Binary SHA-256s recorded in Phase C after cross-compile; the diff is the soft-stop condition only.

## Status
Implemented + tested + backed up (2026-07-02). The T1a VERDICT is NOT launched; its required A/A
preflight (the M2 c8 run) IS launched and healthy. Binaries cross-compiled and hash-verified on the box.
The T1a launch pass/fail call and ordering come from the team lead (command above).

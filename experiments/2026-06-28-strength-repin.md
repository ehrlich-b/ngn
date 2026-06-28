# 2026-06-28 strength re-pin (post-correctness-reset)

First absolute-strength measurement of the engine after the 2026-06-28 correctness
reset + queue (items 2-7). Multi-family gauntlet on the LAN 9800X3D. Predeclared
before launch.

```yaml
id:                 2026-06-28-strength-repin
date:               2026-06-28
change_class:       measurement (absolute re-pin, not a keep/reject candidate test)
hypothesis:         after the reset + correctness queue (root-rep, EP-hash, promotion-safety, UCI-lifecycle, SearchFixed), where does NGN sit on the CCRL scale, and did the fixes move it vs the Jun-14 single-family Blunder pin (2655)?
base_commit:        n/a (re-pin of HEAD; prior reference is the Jun-14 pin at 2655)
candidate_commit:   2dadef8336cac93b3fae522929b7a571af9762ab
candidate_binary_sha256: 4e4a04321c8421cce1d5983225565b3aa4466628662ec4a51b9c38b4450d9089  # build/win/ngn.exe (GOOS=windows GOARCH=amd64 GOAMD64=v3)
harness_commit:     2dadef8 (cmd/gauntlet, cross-compiled gauntlet.exe)
command:            gauntlet.exe -ngn .\ngn.exe -anchors ratings.json -tc 120+1 -games 80 -concurrency 8 -lowpower=false -openings sprt_openings.txt -tally-out repin_tally.json -pgn repin.pgn -no-record
machine:            LAN 9800X3D (AMD Zen5 V-Cache, 8c/16t), native Windows
go_version:         go1.26.2
goarch_goamd64:     windows/amd64 v3
tc:                 120+1 (real clock, SECONDS — CCRL Blitz proxy; drops the movetime optimism bias)
concurrency:        8 (16 engine procs on 16 threads ~= 1 SMT thread/engine; both sides equally slowed, so the relative result/rating is preserved — the standard oversubscribed-gauntlet assumption)
openings:           sprt_openings (corpus_manifest.md)
openings_sha256:    974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
aa_preflight:       n/a for a gauntlet (NGN-vs-anchors, not self-play); the box pipeline was validated by two 2-game smokes (NGN 1W-1D vs Blunder 7.4.0, 0-2 vs Counter 3.8) confirming handshake/clock/scoring on the box before launch
anchors:            Blunder v6.1.0 (2155), v7.2.0 (2425), v7.4.0 (2532), v8.0.0 (2674) [continuity ladder]; CounterGo 3.8 b172b99 (2994) [cross-family top anchor]. 5 anchors x 80 games = 400 games. Fruit (C++) omitted on Windows.
decision_rule:      report the pooled CCRL rating + 95% CI and per-anchor crosstable; this is a MEASUREMENT (no keep/reject). Multi-family (Blunder + Counter) so it is NOT single-family-confounded like the Jun-14 pin. Compare central estimate vs 2655.
games_or_pairs:     400 (pending)
result:             PENDING
flags_errors:       PENDING
verdict:            PENDING
next_action:        PENDING
```

# After-move checking classification

`wave2GivesCheck(*Bitboard, Move)` is an internal query for a correctly encoded
pseudo-legal NGN move. It adjusts local occupancy and the mover's piece masks,
then tests attacks on the opposing king with NGN's attack tables. Own-king
legality is a separate check. Search and the evasion generator are unchanged.

The fixed `corpus.json` has 46 positions and 110 named check/control witnesses.
It includes both colors, direct/discovered/double checks, both castles,
vacated king squares, en-passant rank/diagonal discoveries, direct en passant,
all four quiet/capture promotions, preserved blockers and illegal own-king
exposure. Every generated pseudo-legal move is compared with complete
make/isInCheck/unmake on a separate position. Caller mailbox, piece masks,
material phase, side, flags, clock, raw cold/warm hash, repetition history and
the full HCE cache/generation are compared before and after. A second test
uses sorted legal moves and xorshift seed 1104 for at most 128 plies.

From the repository root, run:

```sh
nice -n 10 python3 experiments/wave2-1104/check-classify/check.py fragment
nice -n 10 python3 experiments/wave2-1104/check-classify/check.py mutation
nice -n 10 python3 experiments/wave2-1104/check-classify/check.py artifacts
```

The fragment runs `GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run
'^TestWave2GivesCheck' -timeout 15s -v` under nice. The runner limits CPU
affinity to two allowed CPUs, sets tensor threads to two, removes credential
environment variables, uses the shared Go caches and keeps temporary files
outside the checkout. No speed, strength or Elo measurement is made.

`legacy_check.go.txt` retains NGN's removed `moveGivesCheck` from
`a6b9995b38b80814fd56668050b8c2c4ef700176:engine/movegen.go`, with only the
signature and renamed board accessors adapted. A Go overlay substitutes it
for the new predicate, without changing the source tree. The mutation command
requires test assertion failures on the discovered/EP witnesses and rejects
compilation errors as evidence. The current baseline has the qsearch comment
but no definition of the old function, so this historical fixture supplies it.

Execution evidence stays outside the checkout. By default, the runner writes
to `../evidence/check-classify/`; set `CHECK_CLASSIFY_OUTPUT` to another external
directory if needed. It generates native wait4 resource/exit receipts and logs
for each command. After the candidate is committed, `artifacts` verifies the
fragment and mutation receipts against HEAD and their logs, then emits
`result.json` in that output directory. This honors the requirement to keep
execution evidence external; the corpus, command and expected witness results
remain committed here. Fixture success does not authorize a search change.

# Exported integer validation fixture

This owned synthetic arithmetic control has 32 NGNP1 records and an H32
NGNN3 export with eight king input contexts and eight material output buckets.
It uses QA=255, QB=64 and score scale=400. The sigmoid objective is lambda=0.75,
K=400. It needs NumPy and the standard library; torch is never imported.

Run from the repository root using the existing NumPy-enabled interpreter:

```bash
python3 -m unittest trainer.tests.test_quantized_validation
experiments/wave2-1104/integer-validation/run.sh
python3 -m experiments.wave2-1104.integer-validation.check_fixture
```

`run.sh` invokes the diagnostic with `--max-records 32` and writes JSON to
stdout. `report.json` is that actual output; reproduce it outside the checkout
with `run.sh --out /path/to/report.json`. `check_fixture` compares a fresh
evaluation against the committed report. Its deliberately incorrect signed
division control must exit nonzero:

```bash
python3 -m experiments.wave2-1104.integer-validation.check_fixture --mutate-division
```

Reproduce the fixture inputs into a separate directory:

```bash
python3 -m experiments.wave2-1104.integer-validation.build_fixture --out /path/to/fixture
```

`cases.json` contains explicit hand-oracle arithmetic and labels for every
record. The first two hidden channels start at -3 and 300, clipping to 0 and
255. Channel 2 depends on the correctly oriented/mirrored own king row and
king bucket: White/Black values are 32/51 or 82/65. The distinguished White
pawn, when present, makes channel 3 saturate high for White and low for Black;
other pieces have zero feature rows. Mirrored variants move that pawn as well
as both kings. All remaining features and channels are inactive.

The output dot product is `-255^2 - 64*us_q^2 + 32*them_q^2`, plus
`(7 if White STM else -3)*255^2` when the pawn is present. Biases are
`[-18,1000,2000,3000,4000,5000,1100000,-1100000]`. The two divisions truncate
towards zero; output cp clamps at +/-24999. The first case has dot=-47329,
first division=-185, bias=-18 and integer cp=-4. Flooring only its first
division instead gives cp=-5. Buckets 6 and 7 force the final score clamp.
Piece counts reach both material boundaries of each bucket (2..5 through
30..32); both STM values and both mirrored/unmirrored contexts occur.

`float-cp.jsonl` contains the exact real-valued pre-export forward cp for
these parameters, calculated with rational arithmetic from the same dot and
bias. Every fourth prediction is intentionally unknown: 32 integer records,
24 paired float records, eight missing. The tests independently compute every
count, error sum, loss sum, mean and difference, overall and per bucket; they
also check lambda=0 and lambda=1, byte reproducibility, a cap with a failing
lookahead iterator, short prediction streams, malformed inputs, and a CLI
execution with a torch import blocker. Floor mutations of either/both
divisions and nearest-rounding substitutions must change the report.

This diagnostic does not alter training metrics, the holdout partition,
checkpoint selection or default play. These synthetic positions are plumbing
controls rather than legal-game or strength evidence. Never select on claim
games. Full H768 corpus evaluation and any subsequent play decision await a
completed rig job; this report makes no Elo claim.

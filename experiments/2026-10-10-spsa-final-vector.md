# Frozen SPSA endpoint and final-vector transfer — session54

The legacy run completes its predeclared iter2000/24,000 durable optimizer
games at21:30:06 EDT October10, controllerexit0 onforge. Preserve the exact
final rounded vector; do not select intermediate values or extend the endpoint.
The optimizer spans rig andforge and recorded seeded RNG resets at resume;
this history is optimization work, never a pooled strength sample. The old
frozen tuning package embedsbe34532c; the current playing package is
9bfbae4/a4d90b41. Parameter movement/plus_pct is no playing verdict.

| Option | Current default | Final value |
|---|---:|---:|
| NullMoveR |3|4|
| NullMoveMinDepth |2|2|
| LMRFullDepth |3|3|
| LMRMoveThreshold |2|2|
| FutilityMargin |103|104|
| FutilityMaxDepth |8|8|
| ReverseFutilityMargin |88|63|
| DeltaMargin |96|202|
| SingularDepth |6|7|
| SingularMargin |48|57|
| ExtensionBudget |26|29|
| AspInit |13|17|
| AspMult |154|163|
| LMPBase |4|3|

Before nomination, verify the final values through the current UCI handler
into all14 actual search variables, record executable reads, and run exactly
six owned positions atdepth14/nodes1M with current defaults and with the full
vector on the existing accepted PGO binary/net. This is12 bounded searches,
not a new endpoint sweep or strength result. Require advertised ranges,
exact override values, at least one changed search witness, normal exits and
no error/panic/illegal output. Useforge5–7/2GiB/swap0/nice10 in the existing
NGN allocation4–11/8GiB, host_pinningON; SPSA is terminal, reader untouched.
Copy the existing9bfbae4 source archive for the handler test and add only a
temporary test. No engine behavior patch, new binary selection or broad
repeat of completed suites. Rig TT games/training remain separate.

After higher-priority TT/factorizer/admission decisions settle the actual
playing package, manually nominate that package and predeclare one rig gate.
Changed source/net requires a fresh combination check; never silently reuse
this9bfbae4/a4d90b41 nomination. Candidate and base use the same checked
binary/net; the candidate gets the frozen14 overrides only. Hash64/Threads1/
OwnBookfalse/UseNNUEtrue/GOMAXPROCS1, rig0–11/c6/nice19/3GiB/swap0,
unchanged9b0002f8 harness and974e4b5a openings. Own A/A40 at10+0.1 must
contain0 in both printed and exact paired95 intervals with zero faults.
Then exactly400 paired10+0.1 games/200 reversed pairs. Keep only printed
AND exact paired lower95>0, zero flag-outs/crashes/illegal/no-move results,
complete sample and intact inputs; otherwise shelve without rescue games,
positive-sign adoption or pooling. A/A deadline1h, gate4h. This fixed400
rule follows the original live-audit admission contract; no batch alternative.
No games launch automatically, tuning adoption, claim or publication.

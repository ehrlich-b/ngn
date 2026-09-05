# Mate-stop correction: Windows continuation after user-stopped Mac run

Prospective protocol, before any new Windows A/A or candidate game. The user
explicitly prohibited all NGN execution on the Mac because it is on battery.
All Mac workers, collector and automatic resumption processes were killed.
The incomplete Mac candidate run has no verdict and is not pooled or used to
select this sample. This hardware restriction supersedes the earlier unlaunched
Windows-plan prohibition; the move is required by the user, not selected from a
favorable partial score. No partial candidate strength estimate was calculated
for this decision. All raw Mac evidence is retained.

The correctness candidate and its proved witnesses are unchanged from
[the original mate-stop record](2026-09-05-mate-stop-confirmation.md). The
current playing base remains5cf301a. There is no NNUE or evaluation change.

## Prerequisites and exact assets

Finish the already-running Windows absolute pin `r0905acceptedpin` first,
preserve all400 games, and complete its original report plus legal/terminal
validation on WSL. A failed pin-integrity audit holds the goal rating claim;
root must inspect it before moving on. No competing Windows/WSL engine work
is allowed while that timed pin is active. The Mac is for editing and lightweight
inspection only, including after it is plugged in unless the user changes the
restriction explicitly.

Then run the already-staged `verify_mateguard_startup.ps1 -Phase aa` in the
native sprt directory. It checks unused run name, Windows/WSL idle state,
asset hashes, both NGNs' Hash128 and14 T8 defaults, three canonical UCI
regressions and clean quits, plus native corrected-harness terminal tests.
No earlier A/A with different platform or score adjudication substitutes.

| Asset in C:\Users\ehrli\ngn\sprt | SHA-256 |
|---|---|
|ngn_20260905_t8qcap.exe|43b2a2c93d9a773b0a08d8017d611edb7b39d63efe071cb7366c1ac4c11f92fc|
|ngn_20260905_mateguard.exe|2c29f19652ce0bfa2d068af093e83c4eeca4f5f1907bc5dba7475162439a3c33|
|sprt_20260905_terminal.exe|62e6ffd96d57558938d45eaf0a123ee57dff1a7dec596a768738a0680b262e2a|
|terminal_rules_20260905.test.exe|99cbbb9a258f41963689fa4460212ed09886b5745e4f94ebee72a6b9a36085b7|
|verify_mateguard_startup.ps1|f7aaf3126a92a253c366485a6543b8003d0ba13b9ffcb23be342d7a151ebf9df|
|sprt_openings.txt|974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222|

## Fixed control and candidate

Native9800X3D,10+0.1 real clocks, concurrency8, default scheduling,
lowpower=false,200-ply cap. Score adjudication stays disabled to expose
false-mate conversion failures. Both runs are fixed samples with no SPRT
stopping, no extension and no pooling.

Run `r0905terminalaa`,200 games/100 pairs of identical accepted T8:

```sh
BOXSPRT_BIN=sprt_20260905_terminal.exe scripts/boxsprt.sh launch r0905terminalaa -new '.\ngn_20260905_t8qcap.exe' -base '.\ngn_20260905_t8qcap.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -maxmoves 200 -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 200 -mingames 201 -resignscore 0 -drawscore 0
```

Require DONE_EXIT_0, exactly200 games, correct cumulative WDL/pentamomial,
zero flags/operational errors, and paired95%CI containing0. A failed control
holds the candidate; no favorable-repeat sampling.

After a passed control and `verify_mateguard_startup.ps1 -Phase candidate`,
run `r0905mateguard`,400 games/200 pairs against immediate accepted T8:

```sh
BOXSPRT_BIN=sprt_20260905_terminal.exe scripts/boxsprt.sh launch r0905mateguard -new '.\ngn_20260905_mateguard.exe' -base '.\ngn_20260905_t8qcap.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -maxmoves 200 -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 400 -mingames 401 -resignscore 0 -drawscore 0
```

Correctness acceptance requires the original mechanism/witness checks, all400
completed games, one terminal DONE_EXIT_0, reconciled WDL/pentamomial, zero
flags, crashes, illegal/missing moves, disconnects or watchdogs. A paired95%
interval wholly below0 rejects this implementation pending investigation.
Otherwise an inconclusive result may support the proved correction, without
an H1 or Elo-gain claim. Preserve all output and errors. This log has no moves
or pair identities; do not claim independent PGN replay from its pair totals.

No native game has been launched under this recovery protocol yet. Root must
record the exact startup/launch evidence and final interpretation. Any further
absolute-rating claim requires the original goal's valid external evidence;
self-play estimates cannot be added to an old rating to claim2800.

## Completion auditor prepared before launch

`output/remote-pin-20260905/audit_windows_mategate.py`, SHA-256
`48c0e63b9b0d5d8f5aed31afadc4a57d5f6175fda456eef410f195282752f1a1`,
was reviewed against the corrected native harness and launcher. Root reproduced
17 lightweight log-fixture checks: two valid synthetic formats and15 malformed
cases, including native LF/CRLF framing, unexpected stderr, missing/corrupt
game rows, footer, flags, secondary summaries and wrong sample/verdict.
These fixtures transform only the completed Mac A/A as source text; they are
explicitly synthetic Windows format and provide no native playing evidence.
No engine or build runs during these parser checks.

The parser requires exact protocol identities, every cumulative WDL/score row,
fixed completion, reconciled penta/paired CI and zero operational failures.
Secondary trinomial summaries are required as numeric lines but do not decide
the verdict. It refuses Python optimization that would disable assertions.
Reproducible fixtures and review are retained beside the parser.

The file is copied to native repin as `root_audit_windows_mategate_20260905.py`;
reverify its hash before use. On WSL, completed native logs can be audited with:

```text
python3 /mnt/c/Users/ehrli/ngn/repin/root_audit_windows_mategate_20260905.py /mnt/c/Users/ehrli/ngn/sprt/r0905terminalaa_out.txt ngn_20260905_t8qcap.exe -3 3 200 200
python3 /mnt/c/Users/ehrli/ngn/repin/root_audit_windows_mategate_20260905.py /mnt/c/Users/ehrli/ngn/sprt/r0905mateguard_out.txt ngn_20260905_mateguard.exe -3 3 400 400
```

Parser completion alone does not accept either gate: root applies the A/A
CI-containing-zero rule or the candidate's predeclared correctness rule above.

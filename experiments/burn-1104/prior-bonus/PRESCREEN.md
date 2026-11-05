# T1104 prior bonus: rig handoff

Status: DEFERRED. This candidate is nominated for the rig's pre-screen;
composite strength confirmation remains pending its result. No strength claim
is made from the short harness smoke. There are no parameter retries.

The implementation is original NGN code from the fail-low prior-move idea in
the task specification. It reuses NGN policy 11's capped depth-squared response.
Colored piece indices reward the previous mover, with the continuation context
from the move before it. The parent's ordering TT move is saved separately and
restored across singular re-entry; restricted singular results do not learn.

The default-off identity check used speedprobes built from the starting commit
6d3168b8c5f15532cc7a352d05465a77ab50f361 and this candidate, at depth 10 on CPU 3,
for both HCE and the existing 81215229 network. identity.json is its raw output.
Identity builds use -trimpath -buildvcs=false so committing the evidence cannot
change the binary hashes via embedded Git metadata. Builds and live evidence
are placed outside the checkout. The requested command
therefore used ../artifacts/sp-base, ../artifacts/sp-new and
--out ../evidence/identity.json; all search flags are unchanged.

The smoke used the same engine binary on both sides, with only the option
changed. It is capped at 1,000 nodes per move and 40 plies to test the harness
in seconds; its two max-move draws have no statistical meaning. Command:

```sh
../artifacts/sprt -lowpower=false -new ../artifacts/ngn -newoption T1104PriorBonus=1 -base ../artifacts/ngn -baseoption T1104PriorBonus=0 -nodes 1000 -openings experiments/burn-1104/prior-bonus/smoke-openings.txt -concurrency 1 -elo0 0 -elo1 5 -mingames 2 -maxgames 2 -maxmoves 40 -pgn ../evidence/smoke.pgn
```

The requirement below is preserved verbatim for the later rig run. Its local
paths describe rig execution; no pre-screen was executed on this host.

Commit experiments/burn-1104/prior-bonus/ with identity.json, the log of one two-game fixed-node harness smoke, nomination.json with verdict DEFERRED, and PRESCREEN.md holding this requirement verbatim for a later rig deep-run: Pre-screen with the option as the only difference: 'mkdir -p scratch; GOAMD64=v3 go build -trimpath -o build/ngn .; go build -o build/sprt ./cmd/sprt; build/sprt -lowpower=false -new build/ngn -base build/ngn -genbook 2000 -genplies 6 -balancecp 100 -balancedepth 8 -seed 1104 -out scratch/book.txt; nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104PriorBonus=1 -base build/ngn -nodes 40000 -openings scratch/book.txt -concurrency 4 -elo0 0 -elo1 5 -mingames 1000 -maxgames 4000 -maxmoves 300 -pgn scratch/games.pgn | tee evidence/prescreen.txt'. Stop at the tool's own verdict or at maxgames, never on a peek; this takes about 2 hours. It is a filter at 40,000 nodes per move, not a 10+0.1 result and not an Elo claim. Verdict: NOMINATE if the tool prints H1 or the Penta Elo lower bound is above 0; otherwise SHELVED. Either is a complete result, with no parameter retries. Write evidence/nomination.json with option, verdict, penta Elo, interval, games, W/D/L and pentanomial counts. On SHELVED the branch carries no engine change and the patch stays at evidence/candidate.patch. Registered gates pass with the option off. Do not run the pre-screen here. A review reads the commit, so these files must be in it; write KEEP and final.md before finishing.

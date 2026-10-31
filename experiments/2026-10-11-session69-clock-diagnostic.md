# One fixed clock-fault diagnostic

Session68's fresh forge SEE A/A40 failed at 20:17:19 EDT: one baseline
time-forfeit, 10/15/15, penta [2,6,8,3,1]. No candidate games ran. This
is operationally UNSCORED, not a SEE loss. The failed log has no moves,
banks or per-move elapsed times. Forge rebooted later at 20:37:45;
that reboot cannot explain a fault twenty minutes earlier. Its current
16-vCPU state differs from the recorded 12-vCPU state, and other work
now appears on CPUs4–11. Do not repeat that SEE control. A distinct singular nomination must
receive its own once-only control on the newly declared post-reboot
topology; this diagnostic is never used to admit it.

Prepare a tool-only topic from checked6341225, retaining its async UCI
reader, child exit receipts, fixed-count admission and per-ply ledger.
Add initial-clock/increment flags; defaults remain120+1. Verify both
default120+1 and requested10+0.1 wire commands and bank arithmetic in
the subprocess fixture, then relevant short/race, full short/race, vet
and PGO build on forge4–11. Engine/search/net code is unchanged.

One prospective diagnostic uses exactly24 target games/12 reversed
pairs, first12 standard openings, baselinef1d0a03/4f8f631f on both sides,
owneda4d90b41, compiledfinal14, factorizer off, explicitEvalFile,
Hash64/Threads1/OwnBookfalse. Forge4–11/c2/nice10/3GiB/swap0,
GOMAXPROCS1, GOAMD64v3. Record current guest topology/background;
physical host pinning is unverified after the reboot. No other NGN job
is live on forge. Native20-minute diagnostic cap, no retry or extension.
The existing runner ends admission on the first operational fault and
preserves the offending move's ledger, position command and clean-up
receipts. Incomplete output is diagnostic evidence only.

Question: does the baseline exceed its supplied bank plus300ms, and
if so at which position, command and elapsed time? Reconcile each
recorded wire command, bank deduction and applied increment. Report
faults and minimum observed bank margin. No statistical/Elo verdict,
no SEE retry, no automatic gate or adoption. A clean24-game sample does
not establish the cause of the old fault or reopen forge to verdicts.
Rodent's separately frozen rig instrument and sample remain untouched.

Run directory: /work/lanes/ngn-s69-clock-diagnostic. The controller
records checks, binary/source/input hashes and the concrete manifest
before its single launch, then status.jsonl and controller.exit/done.

# Forge TT A/A failure review

The preserved40-game60+0.6 control has12W/22D/6L,
penta[0,1,12,7,0], paired+52.5116[+9.7276,+96.9457]. Printed and exact
paired intervals exclude0, so the declared control fails. Candidate games=0;
TT remains unresolved. This session does not retry or pool that sample.
[Original review](../../2026-10-10-session48-results/tt/aa-failure-review.json),
[saved log](tt-aa.log), [manifest](../../2026-10-10-session47-results/tt/game-manifest.json).

Both sides use identical binary path/net/options; the manifest has
GOMAXPROCS1, shared4–11/c2/nice10 and host pinningON. No flags, crashes,
illegal/no-move results or input mismatch were reported. Reversed colors
and pair accounting are symmetric by construction in NGN's harness
(`cmd/sprt/main.go:239`, `internal/uci/uci.go:310`, clock accounting at482–522).
No role-specific CPU mask is assigned. Bounded forge CPU checks and residual
preparation overlapped the control's start; the original run record preserves
that workload. This observation does not establish directional bias.

A40 control can reject by sampling noise; one failed interval does not prove
a harness or host defect. These logs contain no per-worker process/core or
per-move timing trace, so they cannot distinguish noise from a scheduling
mechanism. No measured cause is established. Preserve the failed sample,
require a passing new A/A at each new verdict's exact TC/topology/starting
pin state, and stop before candidate games if it fails. The fresh rig SEE
A/A is a separate instrument; its pass does not validate forge.

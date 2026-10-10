# Null accumulator frames — session33 prospective speed candidate

Independent topic `night/null-frame`, accepted playing base **f8686be** and
NGNN3 KB8/H512 **be34532c**, unchanged PGO616b1ebe. Published idea from
Fable L6, implemented from NGN's own accumulator ownership contract. No
foreign engine code or weights. Data expansion is capacity-blocked; existing
time/search candidates are prepared and await strength lanes.

Null moves currently copy both accumulators despite changing no features.
Store a logical frame's read slice and physical slot separately. Null plies
share the parent's frame; real plies always write their own depth's storage.
Growth relocates all logical views by their recorded slot. Reset overwrites
the root; each subsequent push overwrites its logical frame. Search ply,
network arithmetic, king refreshes and undo behavior stay unchanged.

Before launch: require a named old-fail/new-pass ownership test covering
NGNN1/2/3, H16/512/2048, full int16 weights, two stack reallocations with a
non-root owner, real descendants, sibling reuse, null unwind and root reset.
Existing all-format accumulator/oracle/special-move/worker/reload controls,
AVX2 and purego, full short/race/vet must pass. All heavy work stays on WSL.

The detached [controller](2026-10-09-null-frame-controller.sh) waits once
natively for VNNI512 speed PID26077 to finish, preserving its fixed sample.
Then tests/builds use12,14/nice19 and explicit identical PGO on both probes.
Require36 exact identities (NGNN3/2 twelve each depth14, HCE twelve depth11),
then warm both and run **exactly40 balanced AB/BA blocks**, six positions ×
three seconds per binary on14/GOMAXPROCS1/nice19. The
[checker](2026-10-09-null-frame-check.py) requires geometric NPS lower95>1
with Student-t df39 and both order strata>1. Otherwise shelve and restore
only this topic's runtime/test patch; no sample extension or automatic keep.
This is pure speed, no Elo or net-strength verdict.

Box `session33-null-frame/`, terminal `controller.{exit,done}`,
`build-receipt.json`, `timing/receipt.json`. Freeze source archive, patch,
manifest, checker and controller before launch. C: admission27GiB provides
2GiB above the25GiB floor for this bounded code-only job. The shared hopper
and all existing SPSA/key jobs stay untouched.

# Fixed-count Rodent instrument preparation

Topic `night/rodent-fixed-count`, based on accepted f1d0a03. `pairmatch
-games 1000` admits exactly500 distinct opening/color-reversed pairs at
the existing120+1 clock. Optional `-until` preserves its25-minute pair
admission reserve. Incomplete counts/deadline/faults exit nonzero, with raw
append-only/fsynced pairs retained. No resumption, replacement games or
extension. Operational results stop further admission; startup retries,
abnormal child exits and either side's flag-outs invalidate the sample.

The next instrument version implements the Astra review's required
asynchronous stdout reader, real ready/move/analysis deadlines and timed
stop, serialized writes, child reaping, bounded stderr and exit diagnostics.
Reset failure cannot score an already-terminal board. Worker diagnostics
are in the final summary; inspect stderr/read/exit evidence before a verdict.
The existing live SPRT/data-control binaries remain frozen and unchanged.

Forge4–11/3GiB/swap0/nice10 checks PASS: complete module short/race,
vet/build; affected-package short/race after adding startup-failure
accounting. Witnesses include silent output, EOF, ignored stop/quit,
exit7/stderr/reaping, option rejection, startup retry, concurrent fixed
admission, CLI exact10/5 pairs, deadline-incomplete and operational-fault
nonzero exits. Initial source staging omitted tracked testdata; that failure
is retained separately and resolved by restoring the fixtures.
[Receipts](2026-10-11-pairmatch-fixed-results/).

Prepared only, with no Rodent games or strength claim. After tune transfer
and the ready data gate, freeze the best engine/net/options and nominate
this exact instrument with its own120+1 A/A40 on rig0–11/c6. The separately
predeclared Rodent sample is1000/500, paired lower95>0, complete pairs,
zero faults/intact controls; 3532 is a fixed opponent label. A missed sample
ends the attempt. No automatic campaign claim or publication.

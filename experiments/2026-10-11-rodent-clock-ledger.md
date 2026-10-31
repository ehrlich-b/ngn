# Per-ply clock evidence for the Rodent confirmation

Separate checked/pushed topic **6341225/night/rodent-clock-ledger** extends
the tested d24bc75 fixed-count instrument. Engine/search/eval source stays
f1d0a03. Existing live SPRT binaries remain frozen. No Rodent or new A/A
games were launched here; final frozen-package/instrument A/A40 is owed.

Opt-in GameConfig.ClockOut records the actual tournament go command,
wall milliseconds charged, color/ply/move, both banks before/after,
increment applied and flag status. Pairmatch enables it and serializes
the ledger with each raw game. Opening plies remain free. The existing
clock calculation and300ms grace are preserved; flags receive no increment.
Other shared-UCI users keep tracing disabled.

Rig13,15/2GiB/swap0/nice19 checks PASS: four real subprocess cases cover
both engine POVs, both banks, opening exclusion and forced flags; CLI
10games/5pairs verifies clock evidence on the raw JSON wire. Affected
short/race, vet and explicit accepted-PGO build pass. Exact binary
**e6c2a22f97716db6b13e39d8636757801b3ec6dc39ac0154f0d34933e2455cb6**
is at rig session66-clock-ledger/pairmatch; formatted checked source is
under that lane's source/. No unchanged engine suite repeated.

The original combined checker exits1 only after all Go stages pass:
system Python lacks chess. That receipt/log is preserved under
python-system-failure. The same two legal color/opening controls and ten
falsifying clock controls pass on the existing training venv with
python-chess1.11.2; no Go stage repeats. Future launch/check scripts use
that explicit interpreter. [Reviewed evidence](2026-10-11-session66-results/clock-ledger/review.json).

[Prepared confirmation launcher](2026-10-11-rodent-confirmation.md) imports
the existing independent legal-replay/recount module, reconciles the exact
fixed count and child diagnostics, then audits every recorded command and
bank. It is prepared and syntax/clock-fixture checked, with no final
package nomination or automatic launch. Own120+1 control precedes1000/500
Rodent games; relative positive lower95 only, fixed3532 label, no3600 claim.

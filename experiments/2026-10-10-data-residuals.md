# Expanded-data source/phase residual diagnostic — session47

Prepared before seeing the selected expanded-data network or residuals.
Keep its original full-holdout selection and fixed400-game verdict unchanged.
Freeze64 uniformly spaced8192-record byte windows per original T80/T76/T74
source (512K records/source), using source counts from the running recipe's
existing split metadata. Rig supplies raw bytes only; all conversion,
content-hash partitioning and evaluation run on forge. T74 reads its existing
forge archive. No new source, score map, exposure weight, corpus audit or
training parameter is introduced. Sample metadata/receipts stay with results
at `/work/lanes/ngn-s47/residuals`; raw archives never enter Git.

Use the same board hash/seed20261006/modulus1024, selecting its existing
holdout remainder0 before canonical conversion. Report every occupied
source/output-phase cell (the existing8 material-count heads), record count,
teacher sigmoid(score/400), result probability, original.75 blended target,
and each integer net's mean residual/Brier loss. Mark cells below32 records.
These clustered windows sample the selection holdout; they supply bounded
diagnostics rather than independent population intervals or playing evidence.
Repeated records stay explicit; no residual chooses a checkpoint or source
weight, reopens a failed gate, or extends its sample.

Before candidate evaluation require the completed58/reexport20-parity training
receipt and selected net hash. Compare againstbe34532c, with the existing
owned int64 reference. A same-net control requires exact equality of every
metric in every populated cell. Its synthetic receipt is markedcontrol-only,
its report is renamed, and no candidate `residuals.json` is left by that control.
Real evaluation is manual after finalization/review and never launches games.
[Implementation](2026-10-10-data-residuals.py).


Preparation/control complete10:49/10:53 on forge5–7/1GiB/nice10:
existing holdout samples contain t76=499/t74=528/t80=501 records,1,528 total
in24 populated source/output-phase cells. Exact same-net control passes;
no real candidate report exists. CPU preparation/control briefly overlaps
TT A/A's start, recorded as actual workload without changing its pinned
manifest. [Sample receipt](2026-10-10-session47-results/residuals/sample-receipt.json),
[control](2026-10-10-session47-results/residuals/control-success.json).

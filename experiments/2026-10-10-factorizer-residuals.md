# Expanded-data factorizer residual diagnostic — session55

After full58/reexport/20-parity review, compare selected factorizer1885d7f9
with the accepted expanded-data neta4d90b41 on the existing frozen1528
content-holdout records. Reuse the original64x8192 windows/source and
canonical samples, owned int64 evaluator, teacherK400 and.75 blend.
No resampling, new controls, checkpoint selection or recipe change.

Run onforge5–7/1GiB/swap0/nice10, host pinningON. The original diagnostic
implementation changes only its pinned comparator frombe34532c to
a4d90b41; save its exact copy and all source/output-phase rows in the
session55 result lane. Its already-completed same-net control is reused.
Report weighted source MSE/residuals and small cells as bounded selection-
holdout diagnostics only. No population interval, Elo, source weight,
training nomination or game-rule change follows these results.

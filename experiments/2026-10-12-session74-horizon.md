# Session74 horizon nomination on final14

Prospective replacement for the old, never-played 60+0.6 horizon nomination.
Accepted base f1d0a035f8b4f0b7883fed0a859e5769d076089f / a4d90b41 / compiled
final14. Candidate bc59c3f on night/s74-horizon changes only the estimated
remaining-move floor10 to25. Explicit moves-to-go, increment share, phase
estimates above the floor, allocation-before-game-state-update, overhead,
soft/hard stop and iteration admission remain the existing policy.

Current-source four regression tests, engine/full short/race, vet and explicit
PGO build pass on rig13,15. Reuse these checks; no SEE/admission/score callback
change is included. The package remains unadopted. The earlier horizon source
and nomination are historical, with no games to extend or pool.

The full-history panel and its fixed admission rule are frozen before any
traces: 54 outcome/color/move/material cells, observed120+1 banks, persistent
prior-own-turn allocator phase/move count, cold TT/history. All normal soft
ratios must be below1, median below.75, aggregate actual tournament elapsed
ratio below.90, and static/q/rootSEE outputs identical. This admits a games
experiment only, never strength or a closed-loop bank-retention claim.

Detached native controller follows the panel terminal. It starts games only
if SEE's independent rig400 is SHELVED, keeping immediate basef1d0a03, the
panel exits0 and admits horizon, and capacity is available. SEE eligible or
an operational/control failure holds for manual actual-base review. This is
prospective scheduling, not a sample extension or score-based alteration.

Rig0–11/c6/nice19/GOMAXPROCS1,3GiB/swap0; Hash64/Threads1/bookoff/
UseNNUEtrue/explicita4d90b41/MoveOverhead100; GOAMD64v3/PGO616b1ebe.
Use the frozen4947-opening book excluding all53 diagnostic opening pairs,
SHA256f2fed55a1645dba85a9ef3e047eb5183b5b6342878c6dc8483e6768973470acc.
Instrument unchangedSPRT9b0002f8, TC120+1, maxmoves400/gametimeout1800.
Exact argv and binary hashes are in the lane game-manifest.json before waiting.

Fresh own A/A40 at the exact clock/concurrency/affinity. Both printed and
unrounded paired intervals must contain0, complete20 pairs, zero faults and
intact inputs. A pass alone chains fixed400/200 pairs. KEEP iff both printed
and unrounded penta lower95>0, zero flag-outs/crashes/illegal/no-move/watchdog
results, full sample and intact inputs. CI overlaps0: shelf; upper95<0:
reject; fault/control failure: unscored. No extension, retry, pooling, extra
horizons or automatic adoption. Existing parser unchanged. Control3h/gate12h
limits are operational, not verdicts. Expected total7–9h at120+1.

Lane /home/ehrli/ngn-data/night-20261007/session74-horizon-gate. Read status,
controller.exit/done, aa-summary.json and gate-summary.json at terminal.
Capacity requires no trainer/sprt/datagen/pairmatch, C:>=25.5GiB and
nonavailable memory plus3GiB<=12GiB. GPU diagnostic is finished before this
A/A; no later GPU job is admitted opportunistically inside the scored sample.
No claim, public publication or playing-package adoption follows automatically.

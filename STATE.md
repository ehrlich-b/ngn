# NGN state at the pinned checkout

The historical classical goal is complete: `53e4d1b` measured **2884
[2834,2934]**, independently audited over 400 native Windows games. This is
separate from subsequent WSL research and is not an official rating.
[Historical result](experiments/2026-09-05-2800-result.md).

The supplied source ends at `8ba878c` (October 14 README URL change). Its engine
tree is `67150c8e7811a2ca76cb94250d17f6503a6e248f`. This snapshot contains no
October 17 astra-ngn3600 or s39–s46 experiment records. It cannot establish their
candidate identities, results, or pending work. Its documented active goal is
3300; neither that goal nor 3600 is verified. Do not add internal Elo estimates
to a published opponent rating.
[Live queue](TODO.md), [goal/evidence contract](experiments/2026-09-19-sol-fix-plan.md).

The competitive evaluator is **Rodent V1.2 default**, with accepted exact AVX2
output, update and refresh kernels. E0 adopted it over optimized Anand:
145W/188D/67L, **+68.63 [45.42,92.46]** over 400 games, Threads1, 10+0.1.
[E0 result](experiments/2026-09-19-e0-v12-anand-strength/result.json).

The last completed search-policy screen is **H1 policy 11**, accepted after its
independent 400-game confirmation: **+56.96 [38.37,75.88]** against policy 00.
The production change is `30e4905`, validated through `383e757`; production no
longer needs experiment build tags. TODO's top block still calls H1 pending;
the later completed record and integrated source supersede that stale status.
[H1 result](experiments/2026-09-19-h1-history-policy/result-v1.json),
[production validation](experiments/2026-09-19-h1-history-policy/integration-v1.json).

The next admitted candidate is **P1-R1 reference-style staged move picking**:
ordinary generation/scoring consumed 6.91% of sampled search CPU. Its frozen
contract requires exactly-once TT/tactical/quiet/losing-capture stages, bounded
node inflation, at least 3% paired speed improvement, then pilot/confirmation
games. **No implementation or candidate binary exists here.** It is the best
supported next strength opportunity in this snapshot, without a defensible
numerical expected Elo estimate.
[Admission result](experiments/2026-09-19-p1-staged-picker-profile/result-v1.json),
[candidate contract](experiments/2026-09-19-p1-staged-picker-profile/prototype-plan-v1.md).

C1 clock admission, lazy accumulators, shallow SMP staggering, helper result
selection, and sequence-based TT publication were shelved. Do not revive them
or extend their capped runs. N1 is one later richer-network feasibility study;
owned training remains deferred. The installation/release hold remains:
recorded installed engines are classical `53e4d1b`, HCE is the source default,
and D0/Q0 require separate release and external-evidence decisions.
[Current dispositions](TODO.md), [D0/Q0](experiments/2026-09-19-sol-fix-plan.md).

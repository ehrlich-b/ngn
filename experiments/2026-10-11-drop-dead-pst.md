# Remove unused board PST sums — session61

Prospective exact-speed candidate on immediate accepteded34480/a4d90b41,
separate from SEE geometry, singular learning and admission. Root's09:00
plan permits this lower-priority exact-speed work on genuinely spare capacity.
Both correctness repairs are prepared first; all builds/tests/timing use
rig13,15 after their checks. No forge competition with conversion or A/A.

Source evidence: production writes accMG/accEG only inBitboard.addPiece,
Clear,Move,recomputeAccumulator; no production reader. HCE evaluateRaw
rebuilds frommailbox; currentsearch IIR readsaccPhase. Remove only the
unused fields/arithmetic,retain live material phase and its lifecycle.
Keep all piece-square tables/evaluation and phase policy unchanged.
Tests retire assertions/dump bytes for the removed sums and retain independent
occupancy/mailbox/copy/recompute,material phase,make/unmake/EP/castle/
promotion,HCE immutable-model lifecycle and NNUE behavior controls.

Require full short/race suites,vet,explicit accepted-PGO build,one private
board invariant walk,then36 exact identity cases:NGNN3(currenta4d90b41)
andNGNN2embeddedc6c12796 atdepth14,HCE atdepth11,cold and persistent warm
prefixes. Every deterministic result,node count,score and PV must match.

Then fixed40 balanced alternating timing blocks oncurrentNGNN3,oneCPU15,
GOMAXPROCS1,nice19,6fixtures×3seconds per binary/block;two discarded warmups.
KEEP iff paired-log-NPS ratio lower95>1 and both order strata>1,all36
identities and cleanoperations/intact binaries/net. Shelve otherwise;
no extension or counting NPS as Elo. Use existing speedprobe and original
null-frame speed scorer,two-sided Studentt(df39)=2.0226909200367606.
TimingHash16 records workload mechanism; no absolute/strength claim.

Rig scope13,15/CPUQuota200%/2GiB/swap0; timing15 only; admission0–11/c6,
GPU/loading12,14 retainspriority,hopper1170 untouched.C:about73GiB.
Write manifest,pins,build receipt beforetiming. Detached job can finish
after thissession; terminalsource/resultreview precedes any integration.

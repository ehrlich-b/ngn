# T74 on forge and exact remote archive reads — session41

The rig's T74 continuation was capacity-blocked at70GiB. Forge now reports
464GB free under `/work`, so continue the **same**194-per-mille sample,
seed2026100903, conflict exclusion, source-scale fit and calibrated score map
there. Use only `/work/lanes/ngn-data-volume-20261010`, CPUs8–11, total NGN
memory below5GiB alongside the frozen SPSA scope. The external existing archive
decoder remains outside this repository and runtime; transfer its checked binary,
not third-party source or nets. Preserve the rig's existing data and25GiB floor.

New original trainer IO on isolated `night/remote-archive` reads exact bounded
byte ranges from the mapped T74 file. A small descriptor pins bytes/SHA-256;
the LAN server serves only manifest-listed immutable campaign files and checks
the pin on each request. No local T74 corpus copy, new loss, feature, optimizer,
architecture, sampling or batching policy is introduced. Local/remote synthetic
controls require byte-identical train and holdout partitions and every batch,
including shuffled orders, tails and cached validation. Run the full trainer
unit suite on rig12,14 and a real cross-host range/throughput check before use.
IO code may be committed only after these correctness controls pass.

Transfer and conversion are detached. Conversion emits its source/gather/scale
receipts and a served-file descriptor, then coverage scans T80/T76 locally and
T74 by range using the existing fixed bottom65536 occupied-board estimator.
Review source scales, sentinel drops and estimated distinct ratio against the
2–3× target before unchanged58×256M KB8 H512 data-only training. No automatic
training or adoption. Freeze actual immediate source/net and write training
and game manifests after the architecture verdict. Net strength still requires
20 exact parity cases, byte-identical reexport, fresh A/A40 and400 paired10+0.1
games with lower95>0 and zero faults. No existing completed sample is replayed.

**IO accepted:** original sourcec439acd committed/pushed on
`night/remote-archive`, integrated as09d53c7 onnight/night.61 trainer tests pass:
the first staged full suite passed59/60 but lacked its existing committed NGNN3
fixture; preserved that failure, staged the fixture and recovered only its
failed test. All four final remote controls pass, including atomic descriptor
publication waking a blocking request. No model/optimizer/export changes.
Four exact8MiB cross-host reads and irregular duplicate indexes pass; throughput
**65.96,75.18,77.46,80.90MiB/s**, above the declared25MiB/s admission threshold.
[Receipts](2026-10-10-session41-results/remote-archive/check-receipt.json),
[cross-host](2026-10-10-session41-results/remote-archive/cross-host-receipt.json).

Transfer6694 completes05:56:42/exit0, leaving both rig inputs intact. Forge
controller117502 starts the unchanged scatter at05:56:42. Its own lane had
464GB free at admission; at06:02,437GB remains. Scope2GiB plus frozen SPSA2.5GiB
and read server128MiB stays below NGN's5GiB total. Upload server117501/8766 is
stopped after the completed transfer; immutable reader117631/8767 remains.
No agent runs onforge and all jobs stay on8–11. Native rig coverage7300 waits
on one HTTP request backed by inotify, then runs on12,14/1500MiB with the exact
existing coverage estimator. At current3.4M decoded rows/s, scatter roughly
06:40, mapped output roughly06:50, coverage roughly07:10–07:40. These are ETAs,
not receipts. Conversion status lives inforge `status.jsonl`; coverage appends
the rig campaign status only when complete. Private token/URL descriptors stay
on the hosts and out of Git. No trainer is automatically launched.

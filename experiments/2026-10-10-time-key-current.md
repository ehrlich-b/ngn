# Isolated time candidates on the accepted key base — session39

Rebase the existing admission,25-move horizon and completed-root-choice
patches separately onto actual accepted `6c0fc26/be34532c`. Their code and
named mechanism tests are unchanged; only the playing key source and the
already accepted test synchronization repair differ from their previous
ac658e2 checks. No stacked policies, new trajectories, optimizer changes,
runtime adoption or games.

Forge lane `/work/lanes/ngn-s39-time-key/`, wrapper109399, CPUs8-11/nice10,
CPUQuota400%/MemoryMax2.5GiB/MemorySwapMax0. Starts after native completion
of the separate architecture integration scope, keeping combined active
check+tuner caps at5GiB. Go1.25.5/GOAMD64v3, original616b1ebe PGO, shared
NGN caches under `/work/cache/go`. The other coordinator's work is untouched.

Each candidate gets its existing narrow controls, full short/race/vet and
explicit-PGO build on this source. Append-only `status.jsonl`, separate
`{admission,horizon,partial}-receipt.json`, logs and controller terminal files
record the results. Runtime topics remain uncommitted/unadopted; the exact
patches already committed in experiments preserve the work.

Strength order stays admission,horizon,completed-root choice after
provenance/architecture/final-vector priorities. Each needs fresh actual-base
60+0.6 A/A40 and paired SPRT[0,+6]/alpha.beta.05/min400/cap4000, completed
H1 AND final paired lower95>0/zero faults/intact inputs. H0 or cap shelves
without extension. If an accepted runtime or net changes before its gate,
freeze that actual base and apply this patch alone; no evidence transfer to
another source package or10+0.1 time verdict.

Admission completes03:15:45 and horizon03:20:10, both full checks and
source review PASS on6c0fc26. Partial-root starts03:20:10 and remains active
at03:22. The three Mac topic worktrees have advanced to6c0fc26; their exact
mechanism patches are preserved uncommitted. Review only partial's later
terminal and receipt before its future games.

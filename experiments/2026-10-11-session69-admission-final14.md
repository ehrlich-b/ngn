# Existing admission change on final14

Rebase only the already prepared immediate-iteration-admission candidate
onto actual playingf1d0a03/a4d90b41/final14 in the new isolated worktree
/private/tmp/ngn-s69-admission-final14, topicnight/time-admission-final14.
Keep the older topics and their administratively stopped sample intact.
This prepares an existing queued change, not a new heuristic or policy.

engine/time.go is unchanged between old checkeded34480 andf1d0a03.
Retain the exact four regression controls: immediate soft admission,
immediate hard deadline, active-search1024-call throttle, early admission.
The sole runtime difference remains checking the clock at every depth
admission boundary. No allocation order, horizon, soft-abort policy,
overhead, SEE, singular correction or new parameter is stacked.

Check this necessary changed-source combination once on forge4–11,
explicittaskset/nice10/GOMAXPROCS2/3GiB/swap0: four focused controls,
engine short/race, full short/race, vet and accepted-PGO build with
Go1.25.5/GOAMD64v3. Reuse baseline4f8f631f and owneda4d90b41 when a
future actual-base game manifest is nominated; no baseline rebuild,
training, parity or clock-trajectory rerun. Controller only checks/builds
and appends status.jsonl, then exits. It cannot launch games.

Session68's SEE nomination ended unscored after a baseline flag. The
separate24-game clock diagnostic did not reproduce that fault and does not
serve as a control. Singular has its own distinct post-reboot control/gate
and remains ahead of admission in the
games queue. If the accepted source changes, necessary combination checks
on that new base remain required; do not relabel this artifact.

Future manually nominated admission games retain the original ownA/A40
at60+0.6 then pairedSPRT[0,+6],alpha/beta.05,min400/cap4000. H1 AND
positive printed/unrounded lower95 with zero faults and intact inputs
keeps; H0/cap shelves, operational failure unscored. A winning actual
package gets a separately declared120+1 transfer. No old sample pooling,
extension, automatic nomination or adoption. This is not a new strength
verdict. Leave speculative runtime edits uncommitted until their verdict;
tracked patch and receipts preserve the exact candidate.

Run directory /work/lanes/ngn-s69-admission-final14.

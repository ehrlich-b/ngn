# Deep non-PV check-extension cap — session46

Fable L5a permits a cap on the unconditional check extension after the
scout-depth verdict. Scout's completed fixed400 gate is SHELVED,
+8.69[-9.77,+27.20]; its patch stays absent. This independent trial uses
immediate accepted56b35ef/be34532c on isolated night/check-extension-cap.
No third-party source was opened or copied.

Only the check-extension condition changes: retain depth2–6 and all PV
checks, but decrement normally for non-PV checks at depth7 or greater.
Six is NGN's untuned cutoff for this one trial. Preserve move ordering,
checking-move reduction exemption, passed-pawn/singular extensions,
per-path budget, evaluation, training and time policy. This is a search
heuristic; a smaller tree or passing tests cannot accept it for play.

The live witness forces a checking rook TT move with an exact depth-(d-1)
child entry. An extended child searches and replaces that entry at depthd;
a capped child uses the existing entry. Depth7 must fail on unchanged
source and pass on candidate. Check depth2/6/12, PV depth7, the original
path-budget clamp and stopped-child unwind. Then full short/race suites,
vet and explicit accepted-PGO build on forge8–11/nice10/2GiB/swap0.
Unchanged SPSA2.5GiB and reader128MiB retain their scopes; no forge games.

This code lane stays behind data, capture decision, time management,
SPSA and the already prepared priority search topics. Before strength
admission, nominate the then-current accepted source/net and apply only
this patch; source-combination checks are required if that source changed.
Write exact source/binary/net/opening/harness/profile hashes, argv,
machine/environment/concurrent work into the manifest before games.
Fresh rig A/A40 then paired10+0.1 SPRT[0,+6],alpha/beta.05,min400/cap4000;
GOMAXPROCS1,Threads1,Hash64,book-off,NNUE-on,PGO616b1ebe,
openings974e4b5a,unchanged9b0002f8 harness. A/A requires printed and
unrounded paired intervals containing0 and zero faults. Keep only
completedH1 AND both lower95>0 AND zero flags/crashes/illegal/no-move
results AND intact inputs. H0/cap shelves without extension; an
interruption has no verdict. No automatic games, stacking or adoption.

Forge293195 completes08:39:53 EDT/exit0. The unchanged depth7 search fails
at child depth7 versus6; candidate boundary/PV/budget/stop controls and
full short/race/vet/PGO pass. Local checked source hashes reconcile. The
exact patch remains isolated on56b35ef; current playing source isde148b9,
so actual-source combination checks precede its eventual rig game gate.
[Receipt](2026-10-10-session46-results/check-cap/receipt.json).

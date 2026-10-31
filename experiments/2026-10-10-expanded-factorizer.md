# Expanded-data king factorizer — session51

Prospective separate follow-up after the data-only400-game gate passes:
9bfbae4/a4d90b41 is now the immediate playing base. The current plan permits
retesting factorizer as its own change on accepted expanded data. The earlier
T80-only factorizer gate remains shelved and is never pooled or extended.

Use the exact reviewed trainer source fromsession44-expanded-data/source,
original numerical recipe and exact T80/T76/T74 mapped corpus. Derive the
command from its frozen training-manifest, changing only output directory
and adding `--king-factorizer`. Fresh initialization/seed20261006, KB8/H512/
NB8,58x256M, batch16384, AdamW LR.001/WD.01, WDL.75/K400, BF16, cosine to.1,
two feeder threads, full content-holdout minimum-loss selection unchanged.
NGN's already-tested shared-plus-residual factorizer is folded before a
single quantization into ordinary NGNN3; engine/inference stays unchanged.
Reuse completed CPU/CUDA factorizer controls and original corpus pins;
no repeat full-corpus hashing, smoke or training. The immutable remote
archive verifies its established content pin at use. No source weighting,
warm start, checkpoint lottery or schedule extension.

Rig stage `/home/ehrli/ngn-data/night-20261007/session51-expanded-factorizer`.
Detach under AllowedCPUs12,14/CPUQuota200%/MemoryMax4G/MemorySwapMax0/
nice19. Use the existing trainer-only disk-reserve watchdog,25GiB mandatory
floor; no automatic retry/resume. Record exact argv/source/files/corpus/
base/net/environment/workloads before training. Expected6–8h. TT isolation
runs0–11/3GiB, hopper unchanged, forge SPSA/reader and fixed-position
diagnostics on their separate machine. WSL target below12GiB.

Require58 complete epochs/14.848B presentations and exactly the original
config except king_factorizer/output. Finish training with a terminal
receipt only. Then transfer checkpoint/history/config/net and20 parity
records to forge for byte-identical folded reexport, finite/nonzero shared
weights, full selection/recipe checks and20 exact integer Go parity cases.
The already-prepared finalizer does this without restarting training.

After a session reviews the full receipt, nominate the actual accepted
engine/net (now9bfbae4/a4d90b41) and freeze one rig game topology/manifest.
Fresh A/A40 at10+0.1 must contain0 in printed/unrounded95 intervals with
zero faults. Exactly400 games/200 reversed pairs against immediate base,
Hash64/Threads1/OwnBookfalse/UseNNUEtrue/GOMAXPROCS1, same974e4b5a openings/
9b0002f8 harness. Keep only printed AND unrounded paired lower95>0 with
zero flags/crashes/illegal/no-move outcomes and intact inputs; otherwise
shelve without extension. No automatic games/adoption, claim, embedded-net
change or publication. Training is independent of the rig diagnostic queue;
games follow already-nominated higher-priority TT/Rodent work when ready.

Session55: full58/selected58, folded reexport and20 exact integer parity
reviewed PASS; net1885d7f9. Prepared rigstage `session55-factorizer-games`,
syntax/pins PASS, actual nomination/games still absent. After TT terminal
review select the immediate source/binary, update its nomination template
and workloads, and run changed-source combination parity if TT is kept.
Then manually launch the prepared runner under0–11/c6/3GiB/swap0/nice19.
Original A/A40/fixed400/lower95>0 rule remains unchanged.
[Review](2026-10-10-session55-results/factorizer/review.json),
[stage](2026-10-10-session55-results/factorizer/stage.json),
[runner](2026-10-10-expanded-factorizer-games.py).

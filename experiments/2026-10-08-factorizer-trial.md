# Prospective factorizer trial preparation — campaign session 8

Session starts **14:07:33 EDT**, deadline **15:47:33 EDT, October 8**.
The immediate accepted runtime is **5b882e2 / be34532c**. The longer H512
net has no verdict yet. This session prepares a trial; it launches no real
trainer, datagen, strength match, screen, claim or publication.

## Hypothesis and single-change boundary

The optional king-bucket factorizer shares gradients across contexts during
training and folds into the existing NGNN3 export. Test whether this optimization
parameterization improves strength. No inference or search policy changes.
The [CPU preparation](2026-10-08-king-factorizer.md) passes; its mandatory CUDA
check still owns the window after v3 admission. Tooling tests provide no Elo.

Two prospective schedules are prepared. **Exactly one may be activated**, after
reviewing the longer net's complete parity/A/A40/gate400 and the admission/CUDA
receipts. A winning longer net supplies the immediate base and its **116**-epoch
recipe; a clean completed non-winning gate retains `be34532c` and its **58**-epoch
recipe. An incomplete, failed, invalid or unreviewed result activates neither.
The controller does not choose or promote a base automatically.

Add only `--king-factorizer` to that base's exact from-scratch recipe:

```text
--data /home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf
--data-format archive --archive-holdout-modulus 1024 --archive-split-seed 20261006
--format 3 --kb-map default --hidden 512 --buckets 8 --batch 16384 --lr .001
--epochs {58 or 116, matching the reviewed base} --positions-per-epoch 256000000
--wdl .75 --k 400 --decay cosine --min-lr-ratio .1 --threads 2 --precision bf16
--seed 20261006 --keep-best-only --king-factorizer
```

The output path alone changes administratively. Keep AdamW decay .01, identical
zero-initialized shared rows/RNG stream, original data/labels, split, default
king map, batch order, precision and checkpoint selection. T80 corpus SHA-256
`270f65bdf9f5c948160cae5330030a60f99e94383de2a290b565ae7aad7cde26`,
**75,977,281,056 bytes / 2,374,290,033 records**. Full in-place digest verification
precedes training; no corpus copy. Lowest validation loss across the entire
declared schedule selects `best.pt`; no interim checkpoint can enter games.

## Artifact custody and fixed strength rule

Preparation root `/home/ehrli/ngn-data/night-20261007/session9b/` uses clean source
archive **d4144602c6f0ca5b3cc3575e60a00c937ed75651**. Reconcile all **230** accepted
Go/assembly/test/fixture/profile inputs with the completed delta receipt and all
**25** trainer/fixture inputs with the completed factorizer receipt. No source
or binary used by a live job is modified. Build a private common engine for both
sides and a private Go parity executable; reuse the frozen campaign SPRT and
opening artifacts. Record full binary/source/profile hashes and exact build argv.
Accepted PGO `616b1ebe` is explicit. Go 1.25.5, GOAMD64=v3, Ryzen 9800X3D /
RTX 5080 / WSL2 Linux 6.6.87.2. Preparation CPUs **13,15**, nice **19**, GPU excluded.
Run the existing 64-case folded-net integer parity control on the new test binary.
Required engine suites are inherited only after exact compilation-input equality
to their tested tree; no engine source change is introduced here.

After activation: train on **12,14**, nice **19**, existing PyTorch 2.7.0+cu128;
24-hour timeout, no automatic resume/retry. Validate complete contiguous history,
the selected epoch/step/history, unchanged recipe/split, finite tensors and
nonzero shared rows. Require byte-identical reexport and **20 exact** independent
integer-reference/Go cases with the full H512 net before games. Freeze the new
net/checkpoint/parity hashes before the first game.

Fresh base/base **A/A40**, then exactly **400 games / 200 reversed pairs** at
**10+0.1**, concurrency **8**, CPUs **0–11**, nice **19**, GOMAXPROCS=1 per engine.
Both sides use the same private engine; only EvalFile differs in the gate.
Hash=64, Threads=1, UseNNUE=true, OwnBook=false. SPRT flags:
`-mingames 1000000 -maxmoves 400 -gametimeout 180 -lowpower=false`.
Opening corpus SHA-256 `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`;
first 20/200 openings, both colors, no prior game pooling or adjudication.
A/A must bracket zero. Accept only the complete gate with **penta lower 95% >0**,
zero flags/crashes/illegal/no-move results and intact pre/post hashes. Otherwise
shelve without extension. Incomplete/invalid samples have no verdict. Raw logs,
counts, pair histogram and the independent recomputed penta formula must reconcile.
Use the printed integer lower bound >0 as in existing campaign gates, additionally
requiring the recomputed unrounded lower bound >0. No automatic promotion.

The controller preserves the current tuner during training. Before games it
waits natively for its next durable 25-iteration checkpoint (<=90 minutes),
verifies the exact campaign wrapper/group, drains only that group and restores
an unfinished tune on success/failure. A completed tune is never restarted.
Old-base tuning remains separate if the longer net wins. Hopper stays untouched.
Require C: >=25 GiB and no trainer/datagen/claim/match/speed probe before new
training or games. Admission and CUDA must finish first. No strength screen or
120+1 claim follows this gate automatically.

## Activation contract for the next session

The [preparation controller](2026-10-08-factorizer-prepare.sh) freezes and validates
tooling only, reusing the verified first preparation's builds. The
[trial controller](2026-10-08-factorizer-trial.sh) is **not queued**;
it refuses to run without a reviewed `nomination.json`, successful dependencies,
an activation commit and frozen activation inputs. The [helper](2026-10-08-factorizer-trial.py)
accepts no arbitrary source, data, schedule, flags or network. Both choices are
bound to the reviewed immediate base.

After reviewing all actual raw evidence, write a compact activation record in
`experiments/` containing the selected base/recipe, preparation build hashes,
admission/CUDA review, fixed rule and exact command. Commit and push it before
launch. `nomination.json` contains:

```json
{
  "reviewed": true,
  "review_record": "experiments/<committed activation record>.md",
  "review_commit": "<full activation commit>",
  "longer_verdict": "accepted or shelved",
  "base": "kb8h512-long or kb8h512",
  "base_net_sha256": "<exact selected best.nnue digest>",
  "evidence_sha256": {
    "aa": "<session1/aa.log>", "gate": "<session1/gate.log>",
    "parity": "<session1/parity.json>", "post_gate": "<session1/post-gate-input-check.log>",
    "admission": "<admission-session3/admission-receipt.json>",
    "cuda": "<session8b/cuda-receipt.json>"
  }
}
```

The commit records the nomination payload except its eventual self-referential
commit field; fill that field afterward, write the same full hash to
`activation-commit.txt`, and bind those two files plus the exact copied activation
record in `activation-inputs.sha256`. Validate with `bash trial.sh --dry-run`.
Then launch once on WSL:

```bash
cd /home/ehrli/ngn-data/night-20261007/session9b
setsid nohup bash trial.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
```

Record the observed PID and startup event. Results: `train.log`, `candidate/`,
`parity.{json,log,err}`, `aa{.log,-summary.json}`, `gate{.log,-summary.json}`,
`controller.{exit,done}`, pre/post hash checks and campaign `status.jsonl`.
Do not reuse this directory after a failed/partial activated trial.

## Preparation result

Initial preparation **578197**, `session9/`, completed with exit zero at
**14:19:02 EDT**: three historical gate controls, eight malformed-gate rejections,
six recipe rejections, twenty nomination rejections and 64 exact Go integer cases.
Review then found that the independent parser's unexercised extreme-score guard
used a different clamp from the pinned instrument. Correct that guard, add eight
boundary controls and ten semantic-dependency rejection cases, and freeze the
amended preparation in a fresh `session9b/`. Preserve the complete original root;
no real training or games occurred.

Final controller **580297** completed **14:24:39 EDT**, exit zero. Three historical
complete logs pass, alongside **8** malformed-game-log rejections, **6** changed-
recipe rejections, **30** nomination/dependency/binary rejections and **8** score-
boundary controls. Both 58/116 schedule plans pass synthetic nomination checks.
The actual controller's dry run refuses the missing nomination, producing no
training output, candidate directory or started marker. All **2,016** frozen
inputs reconcile; local helper/controller bytes equal the tested box copies.
`bash -n` and `git diff --check` pass. No real trial is queued.

Private common engine **8ad1c326**, Go parity executable **4544a3be**, SPRT
**9b0002f8**, PGO **616b1ebe**, source archive **f3dffd84**. All 230 engine and
25 trainer inputs match their accepted tested trees. The amended preparation
reuses the original builds and 64-case parity evidence after exact binary/log
hash checks; no build, search, timing sample or game is replayed. Exact
[build receipt](2026-10-08-session8-results/factorizer-trial-build.json),
[controls](2026-10-08-session8-results/factorizer-trial-controls.json) and
[final review](2026-10-08-session8-results/factorizer-trial-review.json).
The engine is a net-gate artifact; a future v3 claim still requires its separately
frozen clean Git source/build/provenance and prospective protocol.

Launch custody: initial amended launcher **580251** never entered its script;
it created no source directory, status event, exit marker or log output. Preserve
`session9b/prepare-launch1.{pid,log}`. One replacement **580297** was observed
through native inotify/pidfd startup before returning the SSH session; its
`launch-observed.json` binds the actual start. This replaces a launch that never
began, without replaying any evidence job. Original `session9/` remains intact.

Frozen trial helper SHA-256 **ed10e851**, controller **400b4354**, amended
preparation **82e032eb**, prospective box manifest **090b699c**, input list
**fcc92e9c**. Result annotations here are separate from the immutable prospective
box manifest. Existing training, tuner and admission/CUDA sources/binaries/nets
remain untouched. Final box snapshot **14:25 EDT**: longer net **45/116** epochs;
tuner durable **325 / 3,900 games**; GPU **71% / 4,193 MiB**; C: **146.5 GiB free**.

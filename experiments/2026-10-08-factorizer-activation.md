# Factorizer trial activation — session 16

Session starts **20:40:07 EDT October 8**, deadline **22:20:07 EDT**.
Activate exactly the prepared **58-epoch** trial in `session9b/`, against
immediate accepted runtime **5b882e2 / be34532c**. Only training parameterization
changes: add `--king-factorizer`. No engine, inference format, data, seed,
optimizer, batch order, holdout, schedule or checkpoint-selection change.
The accepted runtime remains unchanged until a completed eligible gate is reviewed.

## Reviewed prerequisites

Longer H512 training completed **19:57:57 EDT**, 116 contiguous full epochs,
29.696B exposures; lowest validation loss selected epoch **113**. Net
`61dbd7ef0ec57585093f2cedb9dc68c9b5073528230a74f1d7b125679c8562f1`,
checkpoint `1222215d36af9e8350e120346ee0b5b53dca6cc0c55450f39c330aebb49a8447`.
Byte-identical reexport and 20 exact integer cases pass. A/A40 is **9/22/9**,
penta **0 [-60,+60]**, no flags. Gate ended **20:28:43 EDT**: **87/205/108**,
penta histogram **[8,49,104,34,5]**, **−18.3 [−38,+1]**; independently
recomputed interval **[−38.0096,+1.3779]**. All 400 raw game increments and
reasons reconcile, zero flags/crashes/illegal/no-move results, all 14 post-gate
pins OK. **SHELVED** by its predeclared rule, no extension. Retain `be34532c`.
[Machine-readable review](2026-10-08-session16-results/prerequisite-review.json).

V3 operational admission completed **20:36:54 EDT**: all **12** games admitted,
zero exclusions, fresh bounded clock checks and unchanged post-run hashes.
Counter 5.5 1+0.02 **3/0/1**; Berserk 12 at 10+0.1 **0/2/2**, at 120+1
**1/1/2**. Raw game outcomes/errors, advertised selected options, audit/analysis
and receipt hashes reconcile. These games are operational evidence only;
all screen/claim eligibility flags are false. Result root
`/home/ehrli/ngn-data/claim-v3-20261008/admission-session3/`.

Factorizer CUDA **577875** completed **20:37:01 EDT**, zero exit, one actual
FP32/BF16 forward/gradient/fused-AdamW/export test, **3.134 s**, no skips.
NGNN4 CUDA **622811** completed **20:37:07 EDT**, zero exit, its one actual
CUDA test, **2.183 s**, no skips. Both log digests equal their receipts.
PyTorch **2.7.0+cu128**, CUDA **12.8**, RTX 5080; CPU 13,15 / nice 19.
No short window remains pending before this new trainer.

## Frozen trial and decision

Use the already tested [preparation and activation contract](2026-10-08-factorizer-trial.md)
unchanged. Source **d4144602c6f0ca5b3cc3575e60a00c937ed75651**, common engine
`8ad1c326ad4837da018b2154058f2f9865809cea2cceb2b3c977a4cf79605a2b`,
parity executable `4544a3bef1570cb1d9f8e7a571b070db9fe463addfb17bcfd5da62482dffcaef`,
SPRT `9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a`,
PGO `616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0`.
Original exact build commands are in the preparation receipt; no rebuild.
Base net `be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a`.

The [nomination payload](2026-10-08-session16-results/nomination-payload.json)
is committed before launch. Fill its `review_commit` with this activation commit,
write the same commit to `activation-commit.txt`, and freeze it with the nomination
and exact copied record in `activation-inputs.sha256`. Run the existing dry plan,
then launch once:

```bash
cd /home/ehrli/ngn-data/night-20261007/session9b
bash trial.sh --dry-run
setsid nohup bash trial.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
```

The validated plan selects the exact 58-epoch from-scratch command declared in
the preparation record. Training uses CPU 12,14 / nice 19, BF16 GPU; no automatic
retry/resume. T80 in-place digest check precedes training. Full contiguous history,
selection/recipe/split checks, byte-identical folded export and 20 exact integer
cases precede games. Fresh A/A40 then **400 games / 200 reversed pairs**, **10+0.1**,
concurrency 8, CPU 0–11 / nice 19 / GOMAXPROCS=1, Hash64/Threads1/OwnBook=false.
Opening digest `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
Accept only complete penta printed and recomputed **lower95 >0**, zero operational
failures and intact pins; otherwise shelve without extension. No promotion,
strength screen, claim, or next trainer follows automatically.

SPSA stays active during training. The controller waits for its next durable
checkpoint before draining/restoring the exact campaign group for games; its
unfinished vector supplies no strength evidence. Hopper is untouched. C: free
at startup **142.8 GiB**, above the 25-GiB floor. No data deletion or publication.

Results stay in `session9b/{train.log,candidate/,parity.*,aa*,gate*,controller.*}`
and campaign `status.jsonl`. The campaign handoff records the actual launch PID
and ETA after startup; this prospective record remains frozen.

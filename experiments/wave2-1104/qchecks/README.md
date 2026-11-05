# One horizon layer of quiet checks

`W2QChecks` is a UCI spin option, range 0–1, default 0. With value 1, a
non-check qsearch node at qDepth 0 keeps its captures and all promotions in the
existing incremental MVV/LVA/TT order, then searches legal quiet non-promotion
checks in generator order. The passed exact after-move predicate detects direct,
discovered and castling checks. Board-only own-king legality and the existing
castling legality test reject illegal checks. Every class is disjoint. A 256-move
buffer holds the complete class; the 96-score buffer covers only the unchanged
capture/promotion prefix. No added check receives capture-only delta/SEE pruning.
The admission/search counters participate in the existing additive worker merge.
In-check evasions and deeper capture/promotion search retain their existing policy.

Enhanced qDepth-0 TT bounds have a separate key. Their lower bounds cannot leak
into deeper capture-only qsearch or option-off qsearch. Ordinary TT moves remain
ordering hints at enhanced entries. This also prevents a capture-only exact or
upper bound from suppressing an available first-layer check.
Changing the option through UCI stops and joins any active search, then clears
the owning searcher's TT when the parsed/clamped value changes. This also
invalidates main-search bounds derived from the previous horizon policy.
Repeated values and invalid inputs preserve the warm table. The transition
witness checks both directions against a fresh table through actual UCI handling.

From the repository root:

```sh
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py policy
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py fragment
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py off
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py artifacts
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py identity
```

The fragment is exactly `GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run
'^TestWave2(QChecks|GivesCheck)' -timeout 15s -v`, under nice. It retains the
classifier's corpus/walk and adds ten small qsearch fixtures. Independent full
make/check/unmake classifies root moves, while evaluator transition observation
and production admission/search counters verify the real branch, uniqueness,
noisy prefix order (including a TT hint), both checking castles, a castle through
check rejection, promotion exclusion, illegal own-king exposure, a nonchecking
control, qDepth 1 exclusion and unchanged in-check evasions. Qg7# is compared
against an exhaustive tiny no-pruning oracle with zero static leaves. Raising
alpha above the delta margin still finds mate. The same root includes SEE-losing
Qg8+, and every admitted check is observed exactly once. Cancellation after a
check push and at four node limits restores position/hash/phase/history, live
repetition/frame depth and HCE/H768 evaluator depth/root accumulator.

`off-cases.json` declares four depth-4, single-thread, Hash=16 cases using the
embedded H768 network pinned in `option-manifest.json`. `expected-results.json`
contains machine-derived fixture assertions (best move, score, nodes and PV)
from the required classifier candidate, not a timing or strength result. The
baseline was created by cloning the required local mirror at its exact commit,
then running this same driver against that checkout:

```sh
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py baseline ../dependency-check-classify
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py freeze
```

`off` builds the committed candidate and requires exact parity with all four
assertions. Each case is a separate engine process with a 10 CPU-second hard
limit and a 30-second wall timeout; there are no speed samples. The runner uses
at most two CPUs, GOMAXPROCS=2, Go -p 1 (registered suite gates use -p 2), tensor
threads=2, shared caches, external scratch, a sanitized environment and no GPU.
The full short suite includes kernel tests, so the G2 runner refuses it on
forge hosts (including `forge2-vm`) with exit 75. That gate requires a suitable
non-forge host; fragment success does not satisfy it.

If readiness rejects G2 for this host restriction, keep the candidate blocked.
On a suitable non-forge host, restore the committed candidate and retained
readiness plan, then run from the repository root:

```sh
nice -n 10 python3 experiments/wave2-1104/qchecks/check.py run G2 -- nice -n 10 go test -p 2 -short -count=1 ./...
```

Regenerate the complete readiness evidence there against that same committed
HEAD before accepting the candidate. Keep G2 in the plan; the named fragment
and earlier short-suite receipts cannot replace its current-HEAD execution.

Execution transcripts, native wait4 exit/resource receipts and the generated
candidate result live outside the checkout in `../evidence/qchecks/`. The
committed result file is an expected-value fixture; this preserves reviewable
assertions while honoring external execution-evidence retention.
`pending-timed-gate.json` remains pending. Fixture success makes no Elo claim;
no default-play enablement or real-clock rig run is included.

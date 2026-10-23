# N1: bounded Stockfish 18 BIG/dual feasibility

Decision: select **SF18 BIG/dual only** for the richer-network study. Admit a
future standalone oracle/compatibility slice only after the provenance receipt
below is available. Production search integration remains blocked. This study
establishes reproducible operation and memory bounds; it does not establish
official-model arithmetic parity, evaluator latency or playing strength.

## Ordering and existing work

At source base `8ba878cb03a7eb81ca465b3d562953d0f45e81c7`, P1's recorded
disposition is **ADMIT prototype**, not an accepted production picker: the
[frozen profile](../2026-09-19-p1-staged-picker-profile/README.md) reports
600/8,680 ms = 6.91%, exceeding its 3% gate. Its
[P1-R1 contract](../2026-09-19-p1-staged-picker-profile/prototype-plan-v1.md)
predeclares node-inflation, timing and game gates. Neither that record nor this
study supplies a P1-R1 outcome. N1 follows that admission disposition and the
recorded V1.2/post-V3 findings; it does not restart P1 or claim it passed.

The [Sol plan](../2026-09-19-sol-fix-plan.md#10-deferred-architectural-branches--avoid-another-indefinite-project)
asks for exactly this feasibility boundary. The
[September BIG/dual plan](../2026-09-06-sf18-big-dual-plan.md) remains the detailed
format reference. Its implementation-status paragraph is historical: this
checkout already has `nnue/sf18big/format.go` and parser tests, but no BIG
feature extractor, evaluator or incremental context. SMALL's `EvaluateAll`
computes eight heads and is a correctness reference, not a selected-head path.
The existing plan therefore does not already complete N1's trace-cost study.

## Identity and license gate

[provenance.json](provenance.json) pins source release
`cb3d4ee9b47d0c5aae855b12379378ea1439675c` and network archive
`dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4`, together with exact filenames,
full SHA-256 values, byte lengths and local assertion sources:

| Model | Bytes | Full SHA-256 |
| --- | ---: | --- |
| BIG `nn-c288c895ea92.nnue` | 108,919,594 | `c288c895ea924429ea9092e3f36b2b3c1f00f2a3a4c759ff7e57e79e3b43e4a7` |
| SMALL `nn-37f18f62d772.nnue` | 3,519,630 | `37f18f62d772f3107e1d6aaca3898c130c3c86f2ab63e6555fbbca20635a899d` |

Source license: GPL-3.0-or-later. Historical network archive claim: CC0-1.0.
Those are separate provenance chains. The old plan records a BIG LFS identity
match and an external source-review receipt, but that receipt and the actual
license text/pointers are absent here. This offline study has not reverified
them. A filename, test constant or historical receipt hash cannot establish
license permission for an unseen blob. **Do not incorporate or redistribute
either model until the archive license and exact LFS oid/size are bound to
the actual bytes.** No model or upstream source was downloaded for this study.

The next receipt must include exact source/archive HEADs and clean status,
license-text hashes, both LFS pointers, official endpoint redirects, actual
byte hashes/lengths, compiler/flags/build command, binary hash, corpus hash,
instrumentation hashes, raw transcripts and stderr. Preserve GPL notices on
derived source. No network substitution, mirrored filename inference or
SMALL-only receipt can close BIG provenance.

Pinned primary references, recorded here for the future receipt rather than
claimed as newly fetched evidence:

- [release source](https://github.com/official-stockfish/Stockfish/tree/cb3d4ee9b47d0c5aae855b12379378ea1439675c), especially `src/evaluate.cpp`,
  `src/nnue/nnue_architecture.h`, `nnue_feature_transformer.h`,
  `nnue_accumulator.cpp` and `features/full_threats.{h,cpp}`;
- [network archive](https://github.com/official-stockfish/networks/tree/dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4).

## Independent arithmetic oracle contract

The donor oracle must call actual pinned `NetworkBig::evaluate` and
`trace_evaluate`, actual incremental push/update/pop and `Eval::evaluate`.
Extend the existing diagnostic-only
[SMALL instrumentation](../../nnue/sf18small/testdata/upstream_oracle/README.md)
and [incremental harness](../../nnue/sf18small/testdata/upstream_incremental_oracle/README.md).
Do not generate expected outputs from the Go port, or use a handwritten scalar
reimplementation as the donor oracle. Run a scalar/general-64 donor build with
embedding disabled; retain the actual production intermediates alongside
widened range checks and canonicalized tensor/lane order.

Compare at every root, legal push, eligible null, pop and complete unwind:

1. HalfKAv2 base and exact FullThreats active indices for both perspectives,
   including friendly occupied targets, blocked sliders, exclusions and the
   two feature families' different king orientations.
2. All 1,024 BIG base and threat int16 lanes separately, both sets of eight
   int32 PSQTs, and all SMALL lanes/PSQTs. BIG biases/base weights are not
   doubled; SMALL uses its existing doubled-load contract.
3. BIG base+threat **int16 wrap before clamp [0,255]**, paired multiplication
   and division by 512; SMALL clamp [0,254]. Compare the full transformed
   buffers, not just final scores. The 255-versus-254 witness is explicit in
   [arithmetic.json](arithmetic.json).
4. Selected material stack `(pieces-1)/4`, FC0's 16 int32 outputs, 15 squared
   and 15 clipped activations, two zero padding inputs, FC1's 32 outputs,
   clipped FC1, FC2, FC0[15]'s wrapped `*9600/8128` skip and final positional
   value. Preserve signed shifts, width narrowing and division toward zero.
   Compare each of eight heads in diagnostic mode as a second check.
5. BIG PSQT `(baseSTM-baseNTM+threatSTM-threatNTM)/2/16` and positional `/16`
   separately. Check subtraction/addition ranges with widened shadows;
   official-network intermediate overflow rejects the compatibility receipt.
   Explicit Go wrapping for adversarial tensors is a separate contract from
   portable C++ behavior outside the donor's supported arithmetic range.
6. Selector: material values 208/781/825/1276/2538; SMALL first iff
   `abs(simple)>962`; blend `(125*PSQT+131*positional)/128`; BIG fallback iff
   SMALL was first and `abs(blend)<277`. Compare boundary values ±962/963 and
   ±276/277, negative divisions, first and final network, components and blend.
7. Full donor adapter: actual upstream complexity, caller-supplied optimism,
   material, rule-50 and tablebase clamp. Compare optimism zero and nonzero,
   rule-50 boundaries and in-check rejection. Keep raw components, selector
   result and full adapted score distinct. A future NGN adapter must resolve
   ±25,000 versus donor ±31,506 and apply rule-50 attenuation exactly once.

The local Python checker uses literal synthetic arithmetic witnesses, native
integer narrowing independent of the Go evaluator, and exact integer division.
These check the study's arithmetic specification; **no donor oracle or real
network evaluation ran here**. Threshold synthetic rows also do not establish
how often fallback occurs on legal positions with the official SMALL network.

## Reproducible legal-trace cost bound

From the repository root:

```sh
nice -n 10 python3 experiments/2026-10-23-sf18-dual-feasibility/check.py
```

This emits machine-generated summary and per-checkpoint evidence to stdout.
The runner invokes only an ordinary Go diagnostic, without weights, engine
search, external engines, training data or GPUs. Redirect evidence outside
the checkout. The tool's wall time includes compilation and JSON I/O and must
not be reported as network evaluation latency.

Inputs are the unchanged SMALL incremental `sequences.json` (143 cases,
703 checkpoints) and this directory's eight additional cases (54 checkpoints).
There are **151 cases, 757 checkpoints and 303 forward pushes**. Moves are
matched against NGN legal generation, then made; nulls require a non-check
position. Unwind verifies board, side, hash, castling, en-passant and halfmove
state. Coverage includes quiet/capture, both en-passants, both colors'
promotions/capture-promotions, both castles, all king squares/mirror halves,
a 132-ply growth path, all eight material stacks and a blocker-rich opening.
The extra material fixtures are constructed positions, not representative
search traffic; results have no workload-frequency interpretation.

Base columns are counted under the existing SMALL context policy: refresh
the moving king's perspective, otherwise subtract/add changed piece-square
columns. Both widths use that policy for this estimate. FullThreats uses a
conservative **occupied-attack-edge superset**, including all friendly/enemy
occupied targets and kings, with sliders stopped by the first blocker.
Keys contain both endpoint pieces and squares. Within a king file half, use
the old/new typed-edge symmetric difference; on a half crossing, refresh that
perspective. Excluding release-specific piece-pair features can only reduce
these column counts under the pinned plan's attack-edge semantics. The tool
does not assert exact FullThreats indices or implement `DirtyThreats`. This
superset bound must be validated against the actual donor in the next slice.

| Forward-push quantity | Sum over 303 pushes | Maximum per push |
| --- | ---: | ---: |
| Base columns across perspectives | 1,310 | 36 |
| Threat columns upper bound | 434 | 34 |
| Dual base lane additions, columns × (1024+128) | 1,509,120 | 41,472 |
| BIG threat lane additions upper bound, columns × 1024 | 444,416 | 34,816 |
| PSQT additions upper bound, (2×base+threat) × 8 | 24,432 | 848 |

Roots add full-refresh work; saved-frame pops need no numeric updates. Null
frames likewise need no column updates, but an eager frame-copy policy copies
8,896 numeric bytes on every push, **2,695,488 bytes** across this trace.
Bias initialization, zeroing, board snapshots, edge enumeration/set diff,
activation/clamp operations, allocator overhead and metadata copies are
additional costs. The simple geometry enumerator scans up to 32×32 pairs and
at most six interior squares per slider pair per board; its map allocations
are diagnostic costs, not a proposed hot path. No Finny-cache savings are
credited. These are trace-specific counts, not global maximum bounds.

Selected-head dense cost is **17,440 MACs for BIG**, **3,104 for SMALL**:
`16*width + 32*32 + 32`, including padded FC1 lanes. Pairwise transformer
products add 1,024 or 128 respectively. The first selector chooses BIG at
726 checkpoints and SMALL at 31. Without weights, SMALL fallback is unknown:
quote **[3,104,20,544] MACs** and **[128,1,152] pair products** on those rows.
Querying each checkpoint once would cost **[12,757,664,13,298,304] dense
MACs**, excluding accumulator maintenance and scalar adapter work. Pops do
not themselves query an evaluator; their head costs are hypothetical queries
at the restored position. The eight-head oracle path costs 139,520 BIG or
24,832 SMALL dense MACs after a shared transform. It must not stand in for
selected-head production timing. Fallback must preserve both networks' state
even after long runs selecting only SMALL.

## Memory and ownership

Shared immutable tensors total **137,844,032 bytes**: BIG 131,329,312 plus
SMALL 6,514,720, independently recomputed from the loader shapes by `check.py`.
That excludes descriptions, objects, allocator overhead and temporary file
buffers. Streaming load avoids a second full file buffer. Transactional model
replacement can retain old and candidate tensors simultaneously:
**275,688,064 bytes** before old workers retire.

Numeric frame payload is SMALL 576 + BIG base 4,160 + BIG threats 4,160 =
**8,896 bytes**. At 128 frames this is 1,138,688 bytes/worker; at 247 it is
2,197,312. The inherited optional Finny-cache estimate is 327,680 bytes/worker
for both base networks and needs donor `sizeof` confirmation; no threat cache
is assumed. Eight workers at 247 frames plus those caches need **20,199,936
bytes** above shared tensors, TT/history, board state, edge scratch and Go
overhead. Dynamic growth and simultaneous old/new worker pools raise peaks;
measure actual RSS/allocations before integrating, and enforce a documented
depth bound rather than silently allocating an unbounded stack.

One immutable composite model identity binds both hashes/sizes, adapter revision
and HCE/PST generation. Each worker exclusively owns stacks, board facts,
transition records, diff scratch and caches. Only model tensors are shared.
No mutable globals, shared accumulator pools or copying a live context.
Prevalidate transitions for both networks, publish both child frames atomically,
and leave both depths unchanged on failure. Model replacement prebuilds new
workers, then changes identity and invalidates TT/history; failures preserve
the old evaluator. A null can alias numeric payload only with immutable frame
ownership; a later real push must not mutate its parent. Pop restores both
contexts, including after SMALL-to-BIG fallback.

## Bounded next slices and stop conditions

1. **Receipt only.** Recover and verify the pinned provenance and license
   artifacts. Stop if either exact blob or its license binding is unavailable.
   Do not change the existing strict BIG loader or choose another model as a
   workaround. The official-file tests remain opt-in and were not run here.
2. **Standalone scalar BIG oracle.** After the receipt, one portable reference
   implementation and one diagnostic upstream build; no SIMD, caches, generic
   framework or search hooks. Freeze the current 757 checkpoints plus at most
   256 static coverage-gap rows, total ≤1,024, before observing outputs. Require
   every base king bucket/mirror, legal attacker-target family, blocker,
   friendly target and all eight stacks. Compare all intermediates above,
   then incremental/full-refresh identity after every push/pop/null. Stop on
   any discrepancy or uncovered required category; allow one narrow arithmetic
   or feature repair, then rerun the same corpus once. No corpus replacement.
3. **Standalone dual and selected-head cost.** One coordinator with raw,
   selector and donor-adapter APIs; compare the same all-head oracle to one
   selected head, including fallback and error rollback. Freeze boundary
   vectors, two model identities and one/eight-worker ownership tests. Once
   exact parity passes, run five fixed passes on the frozen legal traces,
   separately reporting features/diff, column updates, head evaluation, frame
   copying, cold setup, allocations, peak RSS and model-swap peak. Keep both
   initial selector paths and fallback as separate strata. Stop if correctness,
   ownership or resource bounds fail; do not optimize around a mismatch.

Only after those three slices can a separate integration proposal wire a
concrete dual backend into the existing worker lifecycle and named opt-in UCI
selection. It must settle score scale/clamp/rule-50 policy, check stop/reset,
move/null/unwind and transactional failure behavior, and pass full short and
race suites before any search experiment. A fresh result-unseen experiment
would be needed for strength. This study authorizes no production integration,
owned training, bundled weights, games, deployment or strength claim. Runtime
feasibility and exact license permission remain unresolved, with explicit
conditions for closing each.

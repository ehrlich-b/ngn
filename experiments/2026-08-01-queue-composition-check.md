# 2026-08-01 Queue composition pre-check — T12 x T19 is STRONGLY NON-ADDITIVE

**Zero box cost. No code changed; tree reverted, nodecheck re-verified at 346662/149587/765656.**
Run while T12 held the box at G2514.

## Why

T12 is trending toward a **capped-positive** finish (elo ~+3, pLLR ~+0.8 and flattening at G2514, ~967 g/hr,
cap due ~04:55). Under the batch-certification path a capped-positive with point estimate **>= +1** enters
main as a **PROVISIONAL keep**. If that happens, **T19 and T15 — both built against `ngn_t5` — must be
rebuilt on the new tree**, and any textual or semantic conflict would be discovered at ~5am with the box
idle. This resolves that in advance.

## Result 1 — both compose cleanly, textually and semantically

| composition | applies | builds | `-short` | `-race` |
|---|---|---|---|---|
| T12 + T19 | clean | yes | green | green |
| T12 + T15 | clean | yes | green | — |

**No rebuild risk.** If T12 keeps, T19 and T15 rebuild from the new HEAD mechanically.

## Result 2 — the finding worth keeping: T12 x T19 is strongly NON-ADDITIVE

Fixed-depth node counts vs the locked `ngn_t5` baseline (346662 / 149587 / 765656), all from fresh builds:

| variant | kiwipete d12 | mid d12 | end d16 |
|---|---|---|---|
| T12 alone | 451141 (**+30.1%**) | 136100 (-9.0%) | 796906 (+4.1%) |
| T19 alone | 410704 (**+18.5%**) | 236690 (**+58.2%**) | 516337 (-32.6%) |
| **T12 + T19** | **295507 (-14.8%)** | **112109 (-25.1%)** | **667703 (-12.8%)** |
| T12 + T15 | 332261 (-4.2%) | 164728 (+10.1%) | 796906 (+4.1%) |

**Each change alone GROWS kiwipete (+30.1%, +18.5%). Together they SHRINK it by 14.8%** — and the composition
shrinks all three positions while neither alone does. This is not a small deviation from additivity; the
composed tree is ~35-45 percentage points below where naive addition would put it.

**T12 + T15 is by contrast close to additive** (T12 +30.1/-9.0/+4.1 and T15 +3.8/+11.8/-12.9 compose to
-4.2/+10.1/+4.1 — the endgame is *identical* to T12 alone, consistent with T15 being nearly inert in the
endgame as its own gate found). So the non-additivity is specific to the **T12 x T19** pair, not a general
property of stacking.

**Plausible mechanism (recorded, not load-bearing):** both changes alter the **node-type distribution**
rather than just node counts. T12 forces all-nodes to expand fully, which produces more cut-node children;
T19's reduction behaviour is keyed on the ttPv mark, whose population shifts once the node-type mix changes.
Two changes that each perturb the same distribution have no reason to compose additively — which is exactly
the confound recorded for T12's `nmtry` counter ("absolute or rate counters cannot witness a restriction that
changes tree composition").

## Consequence — binding if both keep

**If T12 and T19 both keep individually, their composition may NOT be assumed additive and requires an
explicit interaction check** before both are carried forward together. CLAUDE.md already bars stacking
("Do not stack changes unless the experiment is explicitly about an interaction"); this supplies a concrete,
measured reason it matters for this specific pair rather than a general caution.

**The batch-certification path is the natural place this gets caught**: Batch 3's powered
[0,+6] SPRT vs the pre-batch base measures the composite, so a non-additive loss would show up as the batch
under-shooting its claim. Note both prior batches **over**-shot (Batch 1 +43.2 vs +34.3 claimed, Batch 2
+35.1 vs +26.8), so an under-shoot in Batch 3 should be read against this measurement rather than treated as
a surprise.

**This does NOT change the run order.** T12's verdict is its own; T19 runs next against whatever base is
current at that time; the interaction question only arises if both keep.

```yaml
id: 2026-08-01-queue-composition-check
date: 2026-08-01
change_class: pre-check / measurement (no change made)
result: >
  T12+T19 and T12+T15 both apply and build clean with green suites, so a T12 keep carries no rebuild risk.
  T12 x T19 is strongly non-additive: each alone grows kiwipete (+30.1%, +18.5%) but composed shrinks it
  14.8%, and shrinks all three positions. T12 x T15 is near-additive.
box_cost: zero
binding_consequence: >
  If T12 and T19 both keep, their composition requires an explicit interaction check and may not be assumed
  additive. Batch 3 certification is where a non-additive loss would surface.
```

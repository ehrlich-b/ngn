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

---

## ADDENDUM 2026-08-02 — the THREE-WAY stack is far more non-additive than the pair

Measured after T12 was kept (so the baseline is now **T12 = 451141 / 136100 / 796906**), while pre-building
both possible T15 launch binaries so the next verdict tick stays mechanical.

| variant | kiwipete d12 | mid d12 | end d16 |
|---|---|---|---|
| **T12 (the new base)** | 451141 | **136100** | 796906 |
| T12 + T15 | 332261 (-26.4%) | **164728 (+21.0%)** | 796906 (**identical**) |
| T12 + T19 | 295507 (-34.5%) | **112109 (-17.6%)** | 667703 (-16.2%) |
| **T12 + T19 + T15** | 316808 (-29.8%) | **355965 (+161.5%)** | 667710 (-16.2%) |

**The middlegame is the story.** If T15's effect composed anything like additively on top of T12+T19, the
three-way stack would sit near `112109 x (164728/136100) ~= 136000`. **It measures 355965 — 2.6x that, and
3.2x the T12+T19 tree it is built on.**

So the pairwise finding understates the problem: **T12 x T19 was strongly non-additive, and adding T15 on top
is worse still.** Each change is individually modest in the middlegame (T15 alone +21%, T19 alone reduces
it), yet stacked they produce a 2.6x middlegame tree explosion.

**Two details that sharpen it rather than soften it:**

- **The endgame is essentially untouched by T15 in both stacks** — 796906 identical on T12+T15, and
  667710 vs 667703 on the three-way (a 7-node difference). That reproduces T15's own gate finding that it is
  nearly inert in the endgame, so the instrument is behaving and the middlegame number is not an artifact.
- **kiwipete does NOT explode** (-29.8%), so this is specific to the quiet middlegame position, not a
  general blow-up.

## Consequence — this is now a live risk for Batch 3, not a hypothetical

Batch 3 currently holds T12 (+2.2 provisional). If **T19 and T15 both keep**, the certified composite is
exactly this three-way stack, and its middlegame tree is 2.6x what additivity predicts. **A tree that size at
fixed real clock is paid for in depth**, which is how T9a and T18b lost.

**Binding, and it strengthens the pairwise rule already recorded:** if two or more of {T12, T19, T15} keep,
**their composition must be interaction-checked before the batch is certified, not after.** Batches 1 and 2
both OVER-shot their claims (+43.2 vs +34.3, +35.1 vs +26.8); **if Batch 3 UNDER-shoots, this measurement is
the first place to look**, and the bisect rule in the ledger should start from the middlegame interaction
rather than from the largest claimed contributor.

**Run order is unchanged** — T19 is live against T12, T15 follows against whichever tree wins. Both T15
launch binaries are already built, pushed and hash-verified:

- **`ngn_t15on12.exe` `3db313b5334159a4ff53133d829a333a575d6c1a4ac07d1f606882b7224d12f4`** — if T19 shelves.
- **`ngn_t15on1219.exe` `5819e33c757e87654407dbfef1286063ec6d97e58a420d7ea5bacc2b96545925`** — if T19 keeps.

Both build clean with green `-short` suites; engine tree reverted and nodecheck re-verified at
451141/136100/796906.

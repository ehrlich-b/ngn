# Fresh 2600 check after the correctness repair

Predeclared before launch. This runs only after the repaired harness A/A and
replacement 400-game correctness non-regression check pass. The original repair
failed its game gate; the exact speed optimization and fresh preflight are
documented in [the replacement manifest](2026-09-04-fast-correctness.md). The user asked to get the engine
past 2600 and deploy it on WSL; the old 2704 estimate used the faulty harness,
so a fresh absolute measurement is warranted. Expected cost ~90 minutes for
200 games, based on July's 400-game run taking 3h06m.

## Instrument

- Candidate and source patch: [optimized repair manifest](2026-09-04-fast-correctness.md).
  `ngn_20260904_fast.exe`, SHA-256
  `7b5af06a4a0ff02a0d9c51aa5eb041dd8ddf9935bbe4691d65dd76146a515828`.
- Gauntlet rebuilt with the corrected shared rules/parser. `gauntlet_20260904.exe`,
  SHA-256 `5d1b6ca586420c04bffed729f5e48c775105c2fb04b2c267a15bd6b6a6769b9a`.
  go1.26.2, GOOS=windows, GOARCH=amd64, GOAMD64=v3.
- Same 9800X3D/native Windows box, concurrency 8, default scheduling, **120+1
  real clocks**. No movetime discount applies despite the old footer text.
- Five historical anchors, **40 games each**, colors reversed per opening.
  Same pinned 5000-line openings SHA-256
  `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
- On-box `repin/ratings.json` SHA-256
  `f1ac3a64ee26448f51c82b14312a04335b690ea00d3a56b527e2be7238696d32`.
  Historical labels retained for continuity; these are local calibration labels,
  not a claim to a current official CCRL listing. The Counter label is known to
  be conservative; it is deliberately not adjusted in this run.

| Anchor | Label | Binary SHA-256 |
|---|---:|---|
| Blunder 6.1.0 | 2155 | `d1bfa4cbc435e0a6d18e2e1fa301df8556bea05a2c6b1f38bf714049860040b4` |
| Blunder 7.2.0 | 2425 | `ed3340ffa916133bcae66609e624cecece60f674db8b8a4de766750ee813f998` |
| Blunder 7.4.0 | 2532 | `160e0866a6811b9c769505a1e463017241682e1dc4449c9aecbae4f65be927d1` |
| Blunder 8.0.0 | 2674 | `7dbf4b3915b37318c688538b7368f54e4046c957d9ee0fcfdb4cbea45556beec` |
| Counter 3.8 | 2994 | `d260174182ea10c5b902a8d25e217166f5b5b54dc4fa64c220b498acce6dec11` |

## Decision rule

Fixed 200 games; no early rating verdict or stopping on a favorable estimate.
Report per-anchor W/D/L, exclusions/errors, pooled rating and 95% CI. The 2600
threshold is supported if the **lower CI exceeds 2600**, both engine families
remain represented, anchors bracket 50%, and no NGN error or flag-out occurred.
Otherwise report the target as unconfirmed and investigate; do not silently
extend or pool favorable extra games. Pre-existing >2% per-anchor error exclusion
remains in effect and must be disclosed. Any new rating is a local estimate,
not an official CCRL result. The changed harness precludes attributing a rating
delta versus July solely to engine improvements.

The gauntlet is against anchors, not self-play. Its shared game logic has unit
regressions plus the newly completed A/A preflight. It runs its own engine
handshake/warmup preflight at startup.

```text
gauntlet_20260904.exe -ngn .\ngn_20260904_fast.exe -anchors ratings.json -tc 120+1 -games 40 -concurrency 8 -lowpower=false -openings sprt_openings.txt -only v6.1.0,v7.2.0,v7.4.0,v8.0.0,counter-3.8 -tally-out r0904pin_tally.json -pgn r0904pin.pgn -no-record
```

Run name **`r0904pin`**, in `C:\Users\ehrli\ngn\repin`.
Poll with `scripts/boxrepin.sh tail r0904pin`.
Results and raw logs to `output/recovery-2026-09-04/` after completion.

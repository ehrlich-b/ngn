# 2026-07-19 T7 — history pruning modernization (depth<=6, depth-scaled threshold)

Pre-registered BEFORE launch (CLAUDE.md work-loop step 5). Isolated single-change per-change gate: candidate
= T5-keep HEAD engine (`5b93966`, == working-tree HEAD `c940fe8` — engine byte-identical, T10b reverted) +
`output/t7.patch`, base = T5-keep HEAD engine (`5b93966`) unchanged, reusing the on-box `ngn_t5.exe` (the exact
T5-keep binary), real-clock 10+0.1 c8 on the LAN 9800X3D box. Highest-ranked unblocked queue item after the
T10b shelve (biggest remaining measured prior, +23.7 [E] CMH-pruning class). Engine diff stays UNCOMMITTED per
policy (shelved as `output/t7.patch`, tree restored); commit only on a keep/provisional verdict.

## Mechanism (one paragraph)

NGN's quiet-move history pruning (search.go ~1730) was the flat form the 2026-07-01 audit flagged ("history
pruning depth<=3 flat"): a flat depth cap `depth <= 3`, a flat constant threshold `histScore < -1000`, the
`legalTried > 3` late-move requirement, guards `!isPV && !inCheck && !IsCapture && PromoType()==NoType && move
!= ttMove`, and NO killer/counter exemption. T7 modernizes that single block to the Ethereal-standard shape:
(1) the depth cap extends `depth <= 3` -> `depth <= 6`; (2) the flat threshold becomes depth-scaled
`histScore < -512*depth` (shallow nodes prune moderately-bad history, deep nodes demand a larger deficit — the
standard `margin*depth` form); (3) killers and the counter-move are exempted
(`!IsKillerMove(move, ply) && move != GetCounterMove(GetLastMovePlayed())`), mirroring the adjacent
futility-prune W1 guard, because now that pruning reaches depth 6, dropping a good ordering move here would fail
the node low and trigger a re-search (the closed-position tree inflation the futility W1 comment warns about).
The combined `GetHistoryScore(move, GetLastMovePlayed(), prev2)` main+continuation+followup score path, the
`legalTried > 3` guard, the `info.HistPrunes` counter, and the `SearchToggles.HistPrune` default-on are all
unchanged; the main move loop only runs at `depth >= 1` (depth 0 dispatches to quiescence at search.go:1269), so
`-512*depth >= -512` — no zero/negative-depth edge. This is a pruning (search-behavior) change: fixed-depth node
counts MOVE (nodecheck below), so it is gated by real-clock games, not proxies. The -512 coefficient and the
depth-6 cap are noted as future SPSA dims (T8); the revamp doc Part 3 T7 entry prescribes the shape ("depth<=6,
depth-scaled threshold") but no exact constants, so the Ethereal-standard linear form is used.

```yaml
id: 2026-07-19-t7-history-pruning
date: 2026-07-19
change_class: search/eval heuristic (search behavior — quiet-move pruning; real-clock games gate)
hypothesis: >
  modernizing NGN's flat history pruning (depth<=3, flat -1000 threshold, no killer/counter exemption) to the
  Ethereal-standard shape (depth<=6 cap, depth-scaled -512*depth threshold, killer/counter exempt) nets positive
  Elo at real clock by pruning more low-history quiets at shallow-to-mid depth without dropping good ordering
  moves. Measured prior: Ethereal 23b841e CMH prune+sort +23.7 / +34.5 LTC (the biggest single classical-Ethereal
  item); history pruning +4.1 [V]. H1: candidate >= +3 Elo over base; H0: candidate <= -3.
base_commit: 5b93966          # T5-keep HEAD engine (== working-tree HEAD c940fe8, engine byte-identical)
candidate_commit: 5b93966 + output/t7.patch   # engine == 5b93966 + t7; patch uncommitted per policy
base_binary_sha256: 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721      # on-box ngn_t5.exe (reused T5-keep binary, hash-verified on box)
candidate_binary_sha256: 34bbfc66a70e121e187d571a213fbf46f3cf91aa0b2d31c429aa04fd29e3ecbf  # output/ngn_t7.exe (reproducible build, hash-verified on box)
harness_commit: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-aware validated mill, box-staged, untouched); launcher scripts/boxsprt.sh untouched
command: sprt.exe -new .\ngn_t7.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
go_version: go1.26.2 (local cross-compile, darwin/arm64 host)
goarch_goamd64: windows/amd64 GOAMD64=v3
tc: 10+0.1 (seconds; bullet)
concurrency: 8
openings: canonical sprt_openings (5000 lines)
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: STANDARD-ON (-resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80; M5-validated non-distorting)
maxgames: 8000
mingames: 300
aa_preflight: standing 2026-07-03 M5-enable adjudication-ON A/A (experiments/2026-07-03-m5-adjudication-aa.md) — same TC (10+0.1) / concurrency (8) / adjudication-flag set / machine (LAN 9800X3D) / openings config, penta -1.1 [-13,+11], pLLR -0.18, 0/0 flag-outs, 77.6% adjudicated, 961 g/hr, DONE_EXIT_0; the SAME convention adopted by the KEPT T1b/T4c/T1e/T5 and SHELVED T13/T1c/T16/T10b runs, so no fresh A/A required.
decision_rule: >
  SPRT [-3,+3] on pentanomial pLLR, bounds +/-2.94, maxgames cap 8000; pLLR >= +2.94 -> KEEP (Batch-2 keep per
  CLAUDE.md batch-certification path); pLLR <= -2.94 -> SHELVE and revert; capped at 8000g with point est >= +1,
  pLLR > 0, and 0 excess flag-outs -> Batch-2 PROVISIONAL keep; capped-nonpositive -> shelve; any candidate
  flag-out above the A/A baseline (0) -> HALT and investigate.
games_or_pairs: (pending)
result: (pending)
flags_errors: (pending)
verdict: (pending)
next_action: (pending)
```

## Nodecheck (node counts EXPECTED TO MOVE — pruning change, NOT re-locked)

Baselines (5b93966): 346662 / 149587 / 765656. Pruning change => counts MOVE (expected, not node-identical).
Deterministic (two identical runs produced identical counts):

| position | depth | baseline (5b93966) | T7     | delta |
|----------|-------|--------------------|--------|-------|
| kiwipete | d12   | 346662             | 346665 | +3    |
| mid      | d12   | 149587             | 149587 | 0     |
| end      | d16   | 765656             | 765654 | -2    |

Counts moved (confirming behavioral, not node-identical) but only slightly on these three positions: history
pruning is a rare event (needs `hist < -512*depth` AND `legalTried > 3`), and NGN's gravity-form history entries
cluster near 0, so the deeper depth-4..6 thresholds (-2048/-2560/-3072) seldom fire while the killer/counter
exemption removes a few prunes — the two roughly offset here. Single-position node shape is not a strength
signal — the real-clock SPRT owns strength. Baselines are NOT re-locked; that happens only on a keep verdict.

## Tests (patched tree, all GREEN — verbatim in output/t7-notes.md)

- `go test -short ./engine -count=1` -> ok
- `go test -short -race ./engine -count=1` -> ok
- `go test -short ./... -count=1` -> ok (engine, cmd/sprt, internal/uci)

No new regression test: the change reshapes an existing pruning threshold with no new seam a deterministic unit
test would pin better than nodecheck already does (the `HistPrunes` counter is exercised by the fixed-depth
search); a crafted "prune fires" position would be brittle and is not forced, per the work-loop guidance.

## Binaries

Cross-compiled with the record-convention command (Makefile `build` Windows line):
`GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -o output/ngn_t7.exe main.go`, go1.26.2, no ldflags/trimpath,
from clean T5-keep HEAD `5b93966` + `output/t7.patch`. Tree restored after build; patch stays uncommitted.

- **Candidate** `output/ngn_t7.exe` sha256 `34bbfc66a70e121e187d571a213fbf46f3cf91aa0b2d31c429aa04fd29e3ecbf`
  (authority = output/t7-notes.md; reproducible — a second identical build reproduced the sha; local
  `shasum -a 256` and on-box `Get-FileHash` both match).
- **Base** reuses the on-box `ngn_t5.exe` sha256 `0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721`
  — the exact `5b93966` (T5 keep) engine binary already staged and hash-verified on the box; no rebuild.

Patch: `output/t7.patch` (git diff of engine/search.go only; single-block change).

## Bounds — [-3,+3] per-change gate (NOT the [0,+6] cert shape)

Isolated per-change SPRT: `-elo0 -3 -elo1 3` (H0: elo<=-3, H1: elo>=3), same shape as the KEPT T1b/T4c/T1e/T5
and SHELVED T13/T1c/T16/T10b runs. `alpha=beta=0.05` yields pLLR bounds +/-2.94. Expected header:
`H0: elo<=-3.0 H1: elo>=3.0 (alpha=0.05 beta=0.05 -> LLR bounds [-2.94, 2.94])`.

## Planned box run (staging + launch)

Base already on box; candidate staged and both on-box hashes re-verified before launch:

```bash
scripts/boxsprt.sh push output/ngn_t7.exe ngn_t7.exe   # candidate — new to box (DONE)
scripts/boxsprt.sh hash ngn_t7.exe          # 34BBFC66…  (verified)
scripts/boxsprt.sh hash ngn_t5.exe          # 0A8F65B7…  (verified, reused base)
```

Launch (writes `run_t7.bat`, out-file `t7_out.txt`):

```bash
scripts/boxsprt.sh launch t7 -- -new '.\ngn_t7.exe' -base '.\ngn_t5.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
```

## Launch

(pending — appended after confirmed-live)

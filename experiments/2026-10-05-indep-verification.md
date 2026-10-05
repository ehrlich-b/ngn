# NGN independence verification — 2026-10-05

Baseline `indep/main` = `74d63eec8f1ee3d18f4c7a3014e486ae0e1f964f`; old main measurement reference = `e4f138ca425ac2585ec2cfd54ae782dfb2177887`. Work stayed in the Mac `ngn-clean` tree and isolated WSL worktrees. All Go compilation, tests, engines and benchmarks ran on WSL (Go 1.25.5, GOAMD64=v3).

## Similarity

Build-derived scope: main → engine initially (37 non-test Go files); final main → engine → internal/sliderray (36 files). No third-party Go module; the final standard-library closure contains neither `testing` nor `os/exec`. The Stockfish client is reachable only from tooling.

`scripts/provenance_similarity.py` reuses the October 5 audit’s lexical approach: strip comments/whitespace; retain identifiers for lines; normalize identifiers for tokens while preserving keywords/operators/literals. Integers are canonicalized across hex/decimal. T is the longest exact token run and directional NGN token coverage by runs ≥20. Coverage conservatively unions **all** such runs across the reference corpus (a suffix automaton), rather than only SequenceMatcher-selected blocks. L uses SequenceMatcher with autojunk=False on nonblank normalized code lines. Identifier normalization makes unrelated getters, enums, argument lists and field declarations appear similar; these percentages are **not copied-code percentages**. The automaton was checked against brute-force longest-run/coverage results on 60 deterministic cases.

Pinned measurement-only sources: Zahak `72ff8066e2d8348d8d3847fab62c4a6b8344240d`; Counter 3.8 `b172b99b3128d82c44ef1d8d8a278aa09f2d721b`; Counter 5.5 `63c487ca724c620f71c129d62129c6fb9109c872`; Stash `13e0a81a4af7fb6af43405228535c5058ee71657`; Stockfish 18 `cb3d4ee9b47d0c5aae855b12379378ea1439675c`; Disservin chess-library `53e6a841dcda7059a2af363d85f785ef1817304a`. PeSTO comparison uses Counter 3.8’s `evalpesto/evaluation.go`. Exact reference-file SHA-256s, matched paths and line spans are in the [JSON receipt](2026-10-05-indep-verification.json). External snapshots are excluded from the committed code. C/Go syntax differences limit negative cross-language findings.

| Linked source | indep T / % | final T / % | final L / % | Longest-run classification |
|---|---:|---:|---:|---|
| `engine/algebraic.go` | 27 / 6.51 | 27 / 6.51 | 11 / 11.35 | generic syntax/control flow |
| `engine/bitboard.go` | 46 / 18.22 | 46 / 18.22 | 8 / 21.29 | occupancy getters |
| `engine/book_embedded.go` | 15 / 0.00 | 15 / 0.00 | 2 / 13.04 | generic syntax/control flow |
| `engine/cache.go` | 44 / 28.92 | 44 / 28.92 | 6 / 20.00 | record getters/struct shape |
| `engine/crash_handler.go` | 17 / 0.00 | 17 / 0.00 | 3 / 18.75 | generic syntax/control flow |
| `engine/enpassant_geometry.go` | — | 16 / 0.00 | 3 / 50.00 | generic syntax/control flow |
| `engine/eval.go` | 20 / 4.63 | 20 / 4.63 | 2 / 9.33 | generic syntax/control flow |
| `engine/evaluation_core.go` | 22 / 1.28 | 22 / 1.28 | 4 / 15.36 | analytic HCE; no PST seed |
| `engine/evaluator_selection.go` | 18 / 0.00 | 18 / 0.00 | 3 / 24.47 | generic syntax/control flow |
| `engine/fen.go` | 18 / 0.00 | 18 / 0.00 | 5 / 13.02 | generic syntax/control flow |
| `engine/global.go` | 21 / 5.16 | 21 / 5.16 | 3 / 26.76 | generic syntax/control flow |
| `engine/has_legal_moves.go` | 23 / 2.48 | 23 / 2.48 | 4 / 16.13 | generic syntax/control flow |
| `engine/hash.go` | 34 / 10.10 | 34 / 10.10 | 8 / 21.29 | random-array initialization |
| `engine/hce_lifecycle.go` | 15 / 0.00 | 15 / 0.00 | 4 / 26.98 | generic syntax/control flow |
| `engine/history_policy_11.go` | 14 / 0.00 | 14 / 0.00 | 2 / 40.00 | generic syntax/control flow |
| `engine/history_policy_experiment.go` | 18 / 0.00 | 18 / 0.00 | 3 / 28.75 | generic syntax/control flow |
| `engine/move.go` | 61 / 24.57 | 61 / 24.57 | 3 / 25.00 | packed move accessors |
| `engine/movegen.go` | 37 / 6.21 | 37 / 6.21 | 4 / 8.64 | generic syntax/control flow |
| `engine/moveorder.go` | 43 / 13.26 | 43 / 13.26 | 4 / 19.26 | killer reset loop |
| `engine/pawn_geometry.go` | 26 / 7.32 | 26 / 7.32 | 5 / 21.28 | generic syntax/control flow |
| `engine/piece.go` | 74 / 46.78 | 74 / 46.78 | 27 / 58.59 | piece-character switch |
| `engine/polyglot.go` | 27 / 6.11 | — | — | external books removed |
| `engine/polyglot_random.go` | 1562 / 99.30 | — | — | FOREIGN DATA, removed |
| `engine/position.go` | 96 / 37.02 | 96 / 37.02 | 9 / 11.02 | bitflag/turn API helpers |
| `engine/search.go` | 52 / 5.94 | 52 / 5.94 | 4 / 5.53 | uniform struct fields |
| `engine/search_control.go` | 25 / 13.31 | 25 / 13.31 | 4 / 9.15 | generic syntax/control flow |
| `engine/search_smp.go` | 23 / 2.87 | 23 / 2.87 | 4 / 11.06 | generic syntax/control flow |
| `engine/search_stats.go` | 18 / 0.00 | 18 / 0.00 | 3 / 16.35 | generic syntax/control flow |
| `engine/slider_attacks.go` | 27 / 9.57 | 27 / 12.19 | 3 / 27.54 | generic syntax/control flow |
| `engine/slider_magics_generated.go` | — | 9 / 0.00 | 1 / 2.26 | NGN-generated numbers |
| `engine/square.go` | 83 / 50.54 | 83 / 50.54 | 64 / 67.15 | standard rank/file/square enums |
| `engine/stockfish_client.go` | 21 / 2.24 | — | — | tooling only |
| `engine/syzygy.go` | 19 / 0.00 | 19 / 0.00 | 4 / 3.88 | generic syntax/control flow |
| `engine/testutil.go` | 40 / 10.22 | — | — | test-only |
| `engine/time.go` | 26 / 5.18 | 26 / 5.18 | 4 / 6.63 | generic syntax/control flow |
| `engine/uci.go` | 55 / 6.87 | 55 / 7.04 | 5 / 3.56 | format argument list |
| `engine/uci_evaluator.go` | 18 / 0.00 | 18 / 0.00 | 3 / 27.03 | generic syntax/control flow |
| `engine/worker_evaluator.go` | 23 / 8.06 | 23 / 8.06 | 3 / 18.42 | generic syntax/control flow |
| `internal/sliderray/ray.go` | — | 18 / 0.00 | 4 / 30.43 | coordinate walking |
| `main.go` | 13 / 0.00 | 13 / 0.00 | 2 / 25.68 | generic syntax/control flow |

Flag above generic-idiom level: **Polyglot random data**, 781/781 integers identical in order to Disservin (100% data equality; 1,562-token run; 99.30% whole-file token coverage), removed. Highest remaining file is `position.go`: 96 tokens across six elementary tag/turn helpers, lines 63–90; retained interface operations, not a larger board/search algorithm. Other long runs are the standard rank/file enums (83), required piece-character mapping (74), packed-move accessors (61), a formatting argument list (55), and repeated struct fields (52). These API/enum overlaps are explicitly disclosed; the bounded comparison cannot certify every line’s origin.

Exact matching Zahak-file T / coverage: bitboard 46/11.46%, cache 37/22.42%, hash 33/7.06%, move 24/5.13%, piece 74/41.39%, position 96/32.76%, square 83/46.87%. The broader table also tests each file against the other selected references. No non-generic search/draw implementation clone was identified in these snapshots. The HCE’s MG material bases `[100,315,335,510,975]` and EG `[100,330,350,540,985]` match none of the ten corresponding PeSTO bases; square values are generated analytically from zeroed arrays, not descendants of a foreign PST seed.

## Removals and retained attributed numbers

External Polyglot reader, 781 constants, external-book UCI state/scan, active book oracles and fixtures were deleted. Embedded book remains NGN-authored FEN/move choices only, with OwnBook=false in both root and UCI constructor. Raw EP geometry moved into an independently written adjacency helper; legal-EP repetition tests remain. Historical replay/probe code no longer calls Polyglot. Stockfish subprocess support moved to internal/oracle; testutil is test-only, with its benchmark helper renamed to avoid Go’s test-discovery signature rule. README and current-state notes were corrected while all historical LICENSES and attribution remain.

Aspiration divisor: 81 → **80**, a round NGN placeholder. Pawn-correction scale: 6245/131072 → **3/64**, an NGN placeholder (at most 1 cp change over every clamped entry -1024…1024). Neither is presented as NGN-tuned.

Other numeric search/order/time comments that cite foreign engines, retained as disclosed published-idea parameters: futility coefficient **103** and max depth **8** (Counter); LMP `LMP_BASE=4`, depth² and improving divisor **2** (Counter-shaped curve; historical prose mentions base 5/depth 8); reverse-futility current coefficient **88** and depth cap **7** (comment cites SF11’s historical 217 and Ethereal/Weiss, not current value provenance); null reduction `NULL_MOVE_R=3`, `+1`, `depth/6` (Counter/Ethereal/Weiss); singular `SINGULAR_MARGIN=48`, `/32` (comment’s historical “64 => 2*depth” is stale); log LMR divisor **2.0**, non-PV **+1**, and **64**-entry depth/move table bounds (Stockfish-style formula); correction limit **1024**, EMA depth divisor **8**, update cap **limit/4** (Stockfish StatsEntry analogy); capture-order bands **80000/-40000**, offset **1000**, and capture-history divisor **8** (ordering class acknowledgment to Weiss/Ethereal/Counter/SF, not evidence these numbers were copied); time-effort bounds **0.85/1.25** and neutral **1.0** (Stockfish-style idea). Original aspiration prose also acknowledged Stash’s window shape with `ASP_INIT=13`; only its borrowed score divisor was replaced. The comments describing later-root null-window PVS and TT fail-low guards attribute ideas without extra calibrated constants.

## Magic construction and correctness

`cmd/magicgen` is NGN-authored; seed `0x4e474e4d41474943` (NGNMAGIC). Its own xorshift/multiply/rotate stream proposes sparse AND-of-three candidates and checks every blocker subset against internal/sliderray. Candidate budget is bounded per square. It found 128 multipliers in 7,052,437 candidates (~0.199 s); a second run reproduced the generated Go file byte-for-byte. None equals a hex literal in either pinned Counter bitboard file. Constants and generator are committed; startup constructs and verifies 102,400 rook + 5,248 bishop rows (841 KiB attack arrays). GODEBUG inittrace measured the engine package at **3.4 ms**.

`TestMagicEveryRelevantOccupancy` derives relevance independently by single-blocker changes to a separate coordinate oracle, then checks all **107,648** relevant subsets with every irrelevant bit both absent and present. Existing full-occupancy/random/invalid-origin/no-allocation checks pass. Full ray/magic slider replay files (including random full occupancies) are byte-identical. Fixed-depth-4 HCE dumps are byte-identical, including scores, moves, PVs, hashes, perft statistics and nodes; node counts: start **1184**, Kiwipete **4620**, tactical check **274**, rook ending **553**, promotions **387**, EP **332**. The intentionally changed search scalars were applied before saving the ray comparison.

Six [standard perft positions](https://www.chessprogramming.org/Perft_Results) pass on rays and magics with identical counts: start d5 **4,865,609**; Kiwipete d4 **4,085,603**; rook ending d4 **43,238**; promotions/castling d4 **422,333**; tactical promotions d4 **2,103,487**; middlegame d4 **3,894,594**. Root FEN/hash restoration is checked.

## Speed and validation

Three middlegame FENs, fresh process/TT per sample, HCE, OwnBook=false, Hash=64 MiB, Threads=1/GOMAXPROCS=1, GOAMD64=v3, `go movetime 10000`. NPS is the engine’s latest completed-iteration report; partial interrupted iterations are not included. FENs and raw nodes/time/depth appear in the receipt. The configured 10 s search returns around 9.91 s due to the engine time reserve. Shared-host load and tree shape limit single-sample comparisons.

Initial baseline NPS (positions 1/2/3): indep **1,010,895 / 988,375 / 1,001,671**; old main **1,451,125 / 1,504,549 / 1,561,563**. First post-cleanup comparison: rays **545,542 / 577,397 / 643,174**, magics **687,584 / 709,384 / 698,531** (per-position +26.0%, +22.9%, +8.6%). All three pairs finished the same depth with the same cumulative nodes. The initial and cleanup measurements shifted enough to warrant an interleaved repeat after the match.

Interleaved repeat after the match (AMD Ryzen 7 9800X3D, same settings; rotate binary order between positions):

| Position | old main | indep/main rays | cleaned rays | cleaned magics | magic/ray gain |
|---|---:|---:|---:|---:|---:|
| 1 | 1,405,580 | 904,480 | 859,329 | 985,714 | 14.7% |
| 2 | 1,479,285 | 917,234 | 971,847 | 1,144,716 | 17.8% |
| 3 | 1,534,892 | 1,089,287 | 1,085,156 | 1,224,483 | 12.8% |
| Mean | 1,473,252 | 970,334 | 972,111 | 1,118,304 | 15.0% |

The repeat supports a **15.0%** mean gain from magics over the cleaned ray build, and **15.2%** versus the original independent baseline; the original independent baseline remains slower than old main. Engine launch-to-version over 20 runs: median **2.90 ms**, maximum **3.77 ms**. These are directional three-position samples, not a universal speed guarantee.

WSL checks: `go test -p 2 -short ./...` PASS (engine 15.777 s); `go test -short -race ./engine` PASS (90.387 s); `go vet -p 2 ./...` PASS (empty diagnostic output); exhaustive slider tests, standard perft, fixed-depth/slider replay identity and deterministic generator replay PASS. Cleanup short suite and ray perft also passed before its commit. Build concurrency and normal checks were capped at two Go workers; no unrelated process was killed.

## Strength orientation and commits

100 games at **10+0.1**, concurrency **4**, one search thread, first 50 six-ply NGN opening prefixes with reversed colors; 12m15s, under the 40-minute cap. Cleaned HCE **4 W / 9 D / 87 L**, score **8.5%**; estimated **−412.8 Elo** versus old main (paired 95% interval **[−572, −324]**; unpaired **[−533, −293]**). **Zero time forfeits**. Pair buckets LL/LD/LW-or-DD/WD/WW = **39/6/4/1/0**. The runner’s SPRT verdict text is deliberately ignored: minGames=101 forced all 100 games for orientation. This is a substantial relative strength regression, not an absolute rating and not evidence that the 3000-Elo goal is reached. The magic substitution itself preserved search nodes/PVs.

Openings: `experiments/2026-09-05-nnue-smp-b0-opening-prefixes-100.txt`, SHA-256 `1c3bad721bde116e6cd3fcdc2c04fb1ef96acdd410e7c441a84b4b444f5a3a9a`; preferred WSL output/sprt_openings.txt was absent. Old main ran its HCE startup with its default OwnBook setting; cleaned HCE used OwnBook=false. Six-ply prefixes can still hit old main’s small embedded book, a further limit on interpreting this orientation. An earlier launch passed an unsupported old-main -own-book CLI flag and completed zero games; it was corrected before this recorded 100-game run. No forfeits or games from that failed launch are included.

Cleanup commit: `e56987ddddcdcfd8f6bd551314956036594d5882`; magic commit: `6cc43615154685d9a684f3c8d3f35683fc3520fe`. Both were tested and pushed to private origin/own/clean from the isolated WSL checkout. Mac source files match the tested code; the Mac tool sandbox denied writes to the common Git metadata outside this worktree. Final documentation/evidence preserves the raw receipts (27 MiB of WSL outputs, well under the 2 GiB limit) and fixes the perft source mode to 0644; Bryan can reconcile local metadata with `git fetch origin own/clean && git reset --mixed FETCH_HEAD` from his normal terminal.

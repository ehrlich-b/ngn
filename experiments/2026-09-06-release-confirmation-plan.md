# Release confirmation after the pretrained control

The current Counter-versus-HCE match isolates the evaluator with identical accepted source, one thread per side, and a fixed 30+0.3 paired SPRT. Its opening set is the already audited 200-line, six-ply development corpus. Neither the result nor the accepted SMP gain is a fresh absolute rating. The running manifest and stopping rule remain unchanged.

## Decisions after the current test

- H1 accepted with clean audits: keep the exact network and adapter as the candidate. Confirm NNUE with eight workers at a longer clock against HCE with eight workers, keeping code, total hash, CPU mask, and time controls matched. This tests the interaction between the two features.
- H0 accepted: preserve the negative result and profile the exact candidate before changing arithmetic, score policy, or search. A new architecture or more training data is not the default response. The known upstream network separates learned-network quality from NGN integration and runtime cost.
- Cap without boundary crossing: report inconclusive. Preserve the fixed test; do not extend it opportunistically or turn a favorable interim score into acceptance.
- Operational failure: preserve the failure, identify its cause, and distinguish recovered game evidence from a clean completed prospective run.

The reviewed HCE2v1 and HCE4v1 tests follow sequentially after the Counter compute window. Their accepted manifests use the original SMP binary, so evaluator work does not confound width scaling.

## Independent opening confirmation

The later longer-clock test should use a second, fixed opening source selected before its results are observed. Prefer the official Stockfish books repository's historical `8moves_v3.pgn` for the first independent confirmation: it provides 34,700 lines of 16 plies, and its PGN format preserves the complete opening move history. Pin the repository commit, archive digest, extraction, deterministic selection seed and ordered identities. Deduplicate selected positions after legal replay, not only identical move strings. [Official book inventory](https://github.com/official-stockfish/books/blob/master/README.md)

This is a recommendation for testing generalization, not a claim that a balanced book necessarily gives more statistical power. Fishtest documents opening-selection bias and evaluates book efficiency with fixed-game time-odds controls; its guidance favors unbalanced books for efficient engine testing. A second book must not be selected because the candidate wins on it. [Fishtest FAQ](https://github.com/official-stockfish/fishtest/wiki/Fishtest-faq)

Stockfish's current progression comparisons use 60+0.6 at both one and eight threads and the UHO_Lichess_4852_v1 EPD book. That is useful precedent for separating thread counts and checking a longer clock, not an assurance that its ratings transfer to NGN. NGN's current auditor deliberately requires start-position PGNs; adding arbitrary-FEN openings would need history/repetition and terminal-audit support. Using an established PGN corpus first keeps that change bounded. [Stockfish progression-test conditions](https://official-stockfish.github.io/docs/stockfish-wiki/Regression-Tests.html)

Before any confirmation game: extend the runner's six-ply-only contract deliberately to the selected fixed opening depth, retain exact Round/color/EPD association, audit every selected opening legally, and test negative prefix/depth cases. Freeze the complete prospective contract. Proposed confirmation clock is 60+0.6, one game at a time, equal eight-worker physical-core masks and 128 MiB total hash per engine, no adjudication, the same normalized 0/20 hypotheses and 400-game cap. This is a plan, not an approved or launched match.

## Opponents and deployment

The first external comparison remains the pinned Counter 5.5 release, with a fresh handshake under the actual CPU mask and matched threads/hash/clock. Counter supports multiple workers; its earlier one-CPU handshake was not an engine-wide limit. Zahak 10, Blunder 8.5.5 and Chess-3 v4.0 remain provisioned opponents, with no completed external games yet. A latest-release refresh and provenance check precedes any world-ranking claim.

Publish per-opponent completed results and uncertainty under the actual WSL conditions. Do not add internal gain estimates to the historical native-Windows 2884.14 estimate. Once the required isolated, combined and external checks pass, build the final integrated source, verify its protocol and stop behavior on WSL, archive deployment and rollback artifacts, then replace the preserved 53e4d1b release under the user's existing authorization.

Research checked 2026-09-06. This plan changes no active run, source arithmetic, or deployment.

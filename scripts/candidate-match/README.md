# Candidate match runner v1

This runner is scoped to a same-binary, one-worker NGN HCE versus NGN-v1 diagnostic. The Python validator is authoritative; manifest.schema.json documents its external JSON shape.

The manifest first exists in HELD state with the exact null approval block. manifest.review_subject_sha256 canonicalizes the complete JSON with sorted keys and compact separators after replacing only status and approval with their HELD values. Root reviews that digest, then changes only those two authorization fields. An approved run recomputes the subject digest before copying or launching anything. The CLI approval token must also equal the approved manifest token.

Both roles send options in this exact order: Threads, OwnBook=false, EvalFile, EvalBackend, Hash, Move Overhead. EvalFile therefore stages a validated NGN-v1 file before selecting that backend. HCE uses <empty>. The role preflight uses sequential isready barriers, checks the selected evaluator, preserves OwnBook=false through ucinewgame, and requires a real start-position depth-2 search with more than one node and no fixed book signature.

The wrapper copies every reviewed source, binary, network, opening, and audit tool into a new run directory. It uses the accepted bounded process supervisor for preflights, the match stage, operational trace audit, and independent chess audit. Match children are checked through /proc for exact executable, working directory, CPU mask, and GOMAXPROCS. The trace audit rejects engine errors, warnings, illegal move/PV reports, nonzero exits, unexpected stderr, option-order drift, and probable embedded-book signatures. The independent frozen auditor validates complete opening histories, every played move, natural terminal state, paired orientation, final EPD, telemetry, and reports zero probable embedded-book signature plies.

No timed match is authorized by this source. A concrete manifest remains HELD until root supplies its bound review digest, token, and exclusive match-window approval.

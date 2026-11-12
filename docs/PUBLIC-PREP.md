# Public repository preparation

Bryan: this prepares the development repository above `e59a5817`; releases and
binary downloads remain in [ngn-releases](https://github.com/ehrlich-b/ngn-releases).
Publishing the branch and changing repository visibility are still your steps.

## Current tree

Nine historical notes now live in `docs/notes/`, thirteen root data files in
`testdata/`, and four Stockfish analysis scripts in `scripts/`. Active consumers
use the new paths. Generated profiles, coverage output and the old tactical-test
executable are removed and ignored. The root retains the build sources,
`default.pgo`, Makefile, module file and all license and attribution files.

The README describes development, links downloads, and distinguishes the official
0.3.0 rating from the development estimate copied from the release repository.
Engine runtime sources, embedded networks and the PGO input are unchanged.
Compare clean source archives with the same Go toolchain and environment: Git
checkouts normally stamp their different revisions into their executables.

## Operational context that remains

The public guidance in README.md, CLAUDE.md, TODO.md and `docs/` outside
`docs/notes/` has no matches for the checked personal home paths, workspace and
repository-store paths, private IPv4 addresses, host names or author email.
Neutral user/host placeholders replace private locations in that guidance.

The inventory below counts occurrences, with distinct matching files per folder.
It scans tracked bytes as UTF-8 with replacement, expands gzip payloads once,
and matches host words case-insensitively. Author email in LICENSE and NOTICE
is excluded. Binary strings and the compressed PGO input are included.

| Folder | Matching files | Occurrences |
| --- | ---: | ---: |
| (root) | 1 | 1 |
| cmd | 3 | 5 |
| docs/notes | 1 | 3 |
| engine | 1 | 1 |
| experiments | 2079 | 26144 |
| internal | 3 | 9 |
| scripts | 17 | 29 |
| trainer | 2 | 18 |

The one root hit is an executable path inside the compressed `default.pgo`.
That file stays byte-for-byte intact because it is a build input. Historical
notes keep their recorded paths. Experiment records and hash-bound receipts
remain intact, including their machine paths, addresses, host names and emails.
Scripts and training tools still contain old machine defaults; claim tooling
still contains fixed artifact locations. Those tools need local configuration
before use. Some host-word matches are generic prose in comments and diagnostics.
These counts describe retained context, not credentials or a new secrets audit.

Old experiment references keep their original spelling to preserve evidence.
For current use, the corpus formerly at the root is
`testdata/acpl_corpus.labels`; root PGN, EPD, FEN, label and result dumps are now
under `testdata/` with the same filenames. Historical note filenames now resolve
under `docs/notes/`. Output-local files with the same basenames are separate
artifacts and keep their existing paths.

## What Git history contains

September development included Rodent-derived compatibility evaluators and a
Counter-derived scaffold. Later work added named attribution and license texts,
then removed those runtime paths and the retired training scaffold. The record
is preserved in [NOTICE](../NOTICE) and [THIRD_PARTY.md](THIRD_PARTY.md), including
the October 1 removal at `6e2e4f57`, training cleanup at `78ec5eda`, and retained
historical licenses at `f5bd82a1`. A tidy current tree does not erase that origin.

History also retains NGN's own trained networks, small owned network fixtures,
and old NGN Linux and Windows binaries and build output. The earlier history
inventory found no third-party trained networks committed to the repository;
references to external opponents, weights and measurements remain in records.
Removing generated files from this head leaves their old Git objects available.

## Validation boundary

This preparation changes documentation, fixture locations and helper paths.
It makes no new playing-strength claim. Required validation is byte identity
for Linux and Windows AMD64 v3 builds, Go vet, the full short Go suite, and
whitespace checks against both specified bases. Kernel tests must run on a
suitable compute host when the preparation host prohibits them; such a blocked
check remains unfinished and must not be described as passing.

#!/usr/bin/env bash
# Build the CCRL-rated Blunder opponent ladder for the rating gauntlet.
#
# Blunder (github.com/algerbrex/blunder) is an open-source Go engine with
# published CCRL ratings, so it anchors NGN's absolute rating far better than
# Stockfish's UCI_Elo slider (which we measured ~+400 inflated vs CCRL). This
# clones Blunder once, checks out tagged releases, and builds each natively into
# opponents/. The built binaries are gitignored; opponents/ratings.json (the
# version->CCRL map) is the checked-in, reproducible spec.
#
# Usage: scripts/build-anchors.sh   (idempotent; re-run to rebuild from the tags)
set -euo pipefail
cd "$(dirname "$0")/.."

OUT=opponents
SRC="$OUT/.src"
mkdir -p "$OUT"

if [ ! -d "$SRC/.git" ]; then
	git clone https://github.com/algerbrex/blunder.git "$SRC"
fi
git -C "$SRC" fetch --tags --quiet

build() { # <tag> <outname>
	git -C "$SRC" -c advice.detachedHead=false checkout --quiet "$1"
	( cd "$SRC" && go build -o "../../$OUT/$2" blunder/main.go )
	echo "built $OUT/$2 from $1"
}

# version -> CCRL Blitz rating (kept in sync with opponents/ratings.json)
build v6.1.0 blunder_610   # 2155 — lower bracket
build v7.2.0 blunder_720   # 2425
build v7.4.0 blunder_740   # 2532 — the anchor that exposed the SF inflation
build v5.0.0 blunder_500   # 2080 — fallback bracket (NGN favored)
build v8.0.0 blunder_800   # 2674 — fallback bracket (NGN underdog)

git -C "$SRC" checkout --quiet - 2>/dev/null || true
echo "done: $(ls "$OUT" | grep '^blunder_' | tr '\n' ' ')"

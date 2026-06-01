#!/usr/bin/env bash
# nodecheck.sh — the canonical node-identity gate for [NEUTRAL] Lane-B changes.
#
# Prints the cumulative search node count for a binary on three phase-diverse
# canonical positions. A node-identical change MUST reproduce the baselines
# exactly; any difference means the change is secretly [BEHAVIOUR] (→ SPRT lane).
#
# WHY THIS SCRIPT (lessons from 2026-06-01, when ad-hoc /tmp checks kept flaking):
#   - `debug on` is REQUIRED or the `info string stats` line (with the cumulative
#     info.Nodes) is never printed — easy to forget.
#   - node counts are DETERMINISTIC (core/contention-independent), so a long
#     fixed window is contention-proof: under E-core load a search just takes
#     longer to REACH the target depth; give it enough time and the count is exact.
#   - perft (`go test -short -run Perft ./engine/`) is the stronger correctness
#     gate for movegen/board changes; this is the search-tree identity gate.
#
# Usage:
#   scripts/nodecheck.sh [binary]      # default build/ngn; prints the 3 counts
# Compare candidate vs the baselines below (which are main-relative — RECOMPUTE
# and update them here + docs/10 whenever main's SEARCH behavior changes, e.g.
# after a kept [BEHAVIOUR] change like de-pad/A2/A4).
set -uo pipefail
cd "$(dirname "$0")/.."
BIN="${1:-build/ngn}"
[ -x "$BIN" ] || { echo "no binary at $BIN"; exit 1; }

# name|fen|depth|baseline   (baselines = main @ T6+de-pad commit d1fb499, 2026-06-01;
# de-pad shifted them from the pre-de-pad values kiwipete 1669503 / mid 1165258 / end 862348)
POSITIONS=(
  "kiwipete|r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1|12|2016803"
  "mid|r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10|12|615508"
  "end|8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1|16|864252"
)

nodes() { # bin fen depth
  local w=$(( $3 + 8 ))  # generous window: deterministic count, contention-proof
  { printf 'uci\n'; sleep 1; printf 'isready\n'; sleep 0.4; printf 'debug on\n'; sleep 0.3;
    printf 'position fen %s\n' "$2"; sleep 0.4; printf 'go depth %d\n' "$3"; sleep "$w"; printf 'quit\n'; } \
  | taskpolicy -b "$1" 2>/dev/null | grep 'info string stats' | tail -1 | sed -E 's/.* nodes ([0-9]+).*/\1/'
}

echo "node-identity check: $BIN"
for row in "${POSITIONS[@]}"; do
  IFS='|' read -r nm fen d base <<<"$row"
  n=$(nodes "$BIN" "$fen" "$d")
  printf "  %-9s d%-2s = %-10s baseline %s\n" "$nm" "$d" "${n:-FAIL}" "$base"
done

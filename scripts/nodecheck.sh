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

# name|fen|depth|baseline   (baselines RE-LOCKED 2026-06-02 night with the wait-for-bestmove
# nodes() below. The OLD baselines (1372558/1003482/772119) were FIXED-SLEEP-WINDOW ARTIFACTS:
# the old nodes() slept depth+8s then grepped tail -1, so under E-core load it caught a MID-search
# stats line, not the depth-N-complete one — the "unstable ruler" (end d16 read 441388/772119/
# 1268761 depending on timing). go-depth searches ARE deterministic; waiting for the bestmove line
# makes the count EXACT and reproducible (verified 3x identical). EXACT-MATCH node-IDENTITY gate.
# Baselines RE-LOCKED 2026-06-03 after the night's kept [BEHAVIOUR] fixes (F1 legalTried-guard,
# C4 phase-6 king, C1 insufficient-material draw-in-search). These shift node counts legitimately;
# C1 especially (kiwipete -37% as deep lines trade into recognized draws, end +45% from corrected
# draw scores reordering the search). RE-LOCKED AGAIN 2026-06-03 LATE after the gradient tune
# (5915619) + #19 passed-pawn rank-scaling (0cf5c5b) shifted eval→cutoffs→tree.
# RE-LOCKED 2026-06-05 night to HEAD = ...+tune+#19+C1(king-dist 1695e20)+#20(passer-ext cda2033)
# +C2(rook-behind 964755e). #20 trims the 6th-rank extension (mid -37%); C2 (eval) COMPOUNDS the
# quiet rook-EG tree WITH DEPTH: end 571500(@C1+#20)->1023904 (+79% @d16, only +9% @d14 — a
# depth-compounding cost, see [[project_endgame_tree_diagnostic]] / TODO C2 caveat). Deterministic
# (wait-for-bestmove). NOTE for the speed lane: gate node-IDENTITY against THESE numbers.
# RE-LOCKED 2026-06-05 to HEAD = ...+threats (f60f9dd): the static threats eval term
# reorders the search a lot (kiwipete -46%, end -50%, mid +35% — eval butterfly, NOT a
# bug; tactical 30/30, SPRT +20.5/204g, faithfulness/symmetry green). Deterministic
# (verified reproduced exactly). Gate node-IDENTITY against THESE for the speed lane.
# RE-LOCKED 2026-06-07 to HEAD = ...+king-safety attacker-material scaling (7587eb6; SPRT
# +43.7 elo [+9,+78] over 400g, KEEP): the eval change reorders cutoffs (mid -20%, kiwipete
# +9%, end ~flat). Deterministic (wait-for-bestmove). Gate node-IDENTITY against THESE.
# RE-LOCKED 2026-06-10 to HEAD = ...+qmvv qsearch MVV/LVA selection-sort (3d86936; SPRT
# +22.6 elo over 400g at 10+0.1, KEEP): qsearch ordering shifts the tree both ways by
# position (najdorf -21%, kiwipete +55%, end -6%) — SPRT is the strength authority, these
# counts are the identity gate. The same commit's micro-opt stack (eval cache, lazy SEE,
# piece tables, stop checks) is byte-identical by construction (verified on this trio).
# RE-LOCKED 2026-06-10 AGAIN to HEAD = ...+A2a improving-gated RFP depth 3->7 (95379a8;
# SPRT +6.1 elo over 400g at 10+0.1, positive at cap, KEEP): deep RFP prunes hard at
# matched depth (kiwipete -60%, mid -44%, end -14%) — the intended width reduction.
# RE-LOCKED 2026-06-10 (3rd) to HEAD = ...+CH-grav gravity-form history (b659c4e; SPRT
# +13.0 over 400g / pooled +9.4 over 556g at 10+0.1, KEEP): history values now evolve
# asymptotically with no halving cliffs, reordering the tree both ways by position
# (kiwipete +7%, mid +55%, end +45% here; the 4 EBF-probe middlegames went -2..-10%) —
# matched-depth counts are the identity gate, the real-clock SPRT is the strength ruler.
POSITIONS=(
  "kiwipete|r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1|12|1087714"
  "mid|r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10|12|1076797"
  "end|8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1|16|611810"
)

nodes() { # bin fen depth — waits for the bestmove line so the count is depth-N-complete (exact)
  local out fifo pid i
  out=$(mktemp); fifo=$(mktemp -u); mkfifo "$fifo"
  taskpolicy -b "$1" < "$fifo" > "$out" 2>/dev/null &
  pid=$!
  exec 3>"$fifo"
  printf 'uci\n' >&3; sleep 0.4; printf 'isready\n' >&3; sleep 0.2; printf 'debug on\n' >&3; sleep 0.2
  printf 'position fen %s\n' "$2" >&3
  printf 'go depth %d\n' "$3" >&3
  for i in $(seq 1 240); do grep -q '^bestmove' "$out" && break; sleep 0.5; done
  printf 'quit\n' >&3; exec 3>&-
  wait "$pid" 2>/dev/null
  grep 'info string stats' "$out" | tail -1 | sed -E 's/.* nodes ([0-9]+).*/\1/'
  rm -f "$out" "$fifo"
}

echo "node-identity check: $BIN"
for row in "${POSITIONS[@]}"; do
  IFS='|' read -r nm fen d base <<<"$row"
  n=$(nodes "$BIN" "$fen" "$d")
  printf "  %-9s d%-2s = %-10s baseline %s\n" "$nm" "$d" "${n:-FAIL}" "$base"
done

#!/usr/bin/env python3
"""Synthetic campaign transcript fixture. Does not launch engines or play games."""
import sys
from pathlib import Path

args = sys.argv[1:]
count = int(args[args.index("-maxgames") + 1])
mode = next((a.split("=", 1)[1] for a in args if a.startswith("FixtureMode=")), "keep")
penta = [0, 0, count // 2, 0, 0] if count == 200 else [0, 0, count // 4, count // 4, 0]
if mode == "negative":
    penta = [0, count // 4, count // 4, 0, 0]
if mode == "incomplete":
    penta[2] -= 1
pgn, wins, draws, losses = [], 0, 0, 0
pair = 0
for bucket, n in enumerate(penta):
    for _ in range(n):
        pair += 1
        outcomes = {0: [0, 0], 1: [0, .5], 2: [.5, .5], 3: [1, .5], 4: [1, 1]}[bucket]
        for game, score in enumerate(outcomes, 1):
            white = "candidate" if game == 1 or mode == "same-color" else "base"
            black = "base" if white == "candidate" else "candidate"
            result = "1/2-1/2" if score == .5 else "1-0" if (score == 1) == (white == "candidate") else "0-1"
            wins += score == 1
            draws += score == .5
            losses += score == 0
            tags = {"Event": "SYNTHETIC FIXTURE", "Round": f"{pair}.{game}", "White": white, "Black": black,
                    "Result": result, "Termination": "draw-rule", "TimeControl": "10+0.1"}
            pgn.append("\n".join(f'[{k} "{v}"]' for k, v in tags.items()) + "\n\n" + result + "\n\n")
Path(args[args.index("-pgn") + 1]).write_text("".join(pgn))
low, high = (-1, 1) if count == 200 else (-100, -10) if mode == "negative" else (0, 100) if mode == "rounded-zero" else (10, 100)
print(f"Games: {count}\nW-D-L: {wins}-{draws}-{losses}")
print(f"Pentanomial [LL {penta[0]}  LD {penta[1]}  {{LW,DD}} {penta[2]}  WD {penta[3]}  WW {penta[4]}]")
print(f"Penta Elo: 20  95% CI [{low}, {high}]  pLLR 1.0")
print(f"Flag-outs (lost on time): new {int(mode == 'fault')}, base 0")
print("Verdict: INCONCLUSIVE (synthetic fixed sample)")

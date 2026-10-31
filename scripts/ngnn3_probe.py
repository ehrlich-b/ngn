"""Owned NGNN3 validation: equal-score nets, legacy identity, pinned paired NPS.

Run on WSL only. No games or opponents. Search topology is checked separately
before timing; tied king rows isolate the format's inference overhead.
"""

import argparse
import json
import math
import os
from pathlib import Path
import statistics
import subprocess


IDENTITY_KEYS = ("position", "kind", "depth", "nodes", "score", "best", "pv")


def prepare(root):
    import numpy as np
    from trainer.export import export_state
    from trainer.king_buckets import DEFAULT_MAP
    rng = np.random.default_rng(20261006)
    w1 = rng.normal(0, 0.07, (768, 256))
    w1 = (w1 + w1[np.arange(768) ^ 7]) / 2
    state = {"w1": w1, "b1": rng.uniform(-0.1, 0.6, 256),
             "o": rng.uniform(-0.3, 0.3, (8, 512)), "ob": rng.uniform(-0.1, 0.1, 8)}
    export_state(state, root/"probe2.nnue", 2)
    export_state(dict(state, w1=np.tile(w1, (8, 1)), king_map=np.asarray(DEFAULT_MAP)), root/"probe3.nnue", 3)
    export_state(dict(state, o=state["o"][0], ob=state["ob"][0]), root/"probe1.nnue", 1)


def invoke(binary, network, *options, cpu=None):
    command = [str(binary), "-net", str(network), *map(str, options)]
    if cpu is not None:
        command = ["taskset", "-c", str(cpu), *command]
    env = dict(os.environ, GOAMD64="v3", GOMAXPROCS="1")
    completed = subprocess.run(command, env=env, capture_output=True, text=True, timeout=180)
    if completed.returncode:
        raise RuntimeError(completed.stderr)
    return [json.loads(line) for line in completed.stdout.splitlines() if line.strip()]


def identity(root, binary, depth):
    reports = []
    for version in (1, 2, 3):
        rows = invoke(binary, root/f"probe{version}.nnue", "-depth", depth)
        (root/f"after{version}.jsonl").write_text("".join(json.dumps(row)+"\n" for row in rows))
        reference = root/f"before{version}.jsonl" if version < 3 else root/"after2.jsonl"
        before = [json.loads(line) for line in reference.read_text().splitlines()]
        if len(before) != len(rows):
            raise RuntimeError("Search result count changed")
        for a, b in zip(before, rows):
            if any(a[key] != b[key] for key in IDENTITY_KEYS):
                raise RuntimeError(f"Search identity changed for NGNN{version}: {a} -> {b}")
        reports.append({"format": version, "cases": len(rows), "exact": True})
    (root/"identity.json").write_text(json.dumps(reports, indent=2)+"\n")
    print(json.dumps(reports), flush=True)


def paired(root, binary, cpu, repeats, duration):
    # Two warmups, then reverse pair order on alternating repeats. Time each
    # position independently; all search state and TT start cold each time.
    for version in (2, 3):
        invoke(binary, root/f"probe{version}.nnue", "-mode", "timed", "-position", 0,
               "-duration", duration, cpu=cpu)
    pairs = []
    for repeat in range(repeats):
        for position in range(6):
            order = (2, 3) if repeat % 2 == 0 else (3, 2)
            rows = {}
            for version in order:
                result = invoke(binary, root/f"probe{version}.nnue", "-mode", "timed",
                                "-position", position, "-duration", duration, cpu=cpu)
                if len(result) != 1 or result[0]["nps"] <= 0:
                    raise RuntimeError("Invalid timed probe")
                rows[version] = result[0]
            pairs.append({"repeat": repeat, "position": position, "order": list(order),
                          "ngnn2": rows[2], "ngnn3": rows[3], "ratio": rows[3]["nps"]/rows[2]["nps"]})
    ratios = [pair["ratio"] for pair in pairs]
    summary = {"cpu": cpu, "repeats": repeats, "duration": duration, "pairs": len(pairs),
               "h": 256, "kb": 8, "nb": 8, "geometric_ratio": math.exp(statistics.mean(map(math.log, ratios))),
               "median_ratio": statistics.median(ratios),
               "ngnn2_mean_nps": statistics.mean(pair["ngnn2"]["nps"] for pair in pairs),
               "ngnn3_mean_nps": statistics.mean(pair["ngnn3"]["nps"] for pair in pairs),
               "position_ratios": [math.exp(statistics.mean(math.log(pair["ratio"]) for pair in pairs
                                                             if pair["position"] == position)) for position in range(6)]}
    (root/"nps.json").write_text(json.dumps({"summary": summary, "pairs": pairs}, indent=2)+"\n")
    print(json.dumps(summary), flush=True)


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument("action", choices=("prepare", "identity", "paired"))
    cli.add_argument("--root", type=Path, required=True)
    cli.add_argument("--binary", type=Path)
    cli.add_argument("--depth", type=int, default=7)
    cli.add_argument("--cpu", type=int, default=0)
    cli.add_argument("--repeats", type=int, default=4)
    cli.add_argument("--duration", default="3s")
    args = cli.parse_args()
    args.root.mkdir(parents=True, exist_ok=True)
    if args.action == "prepare":
        prepare(args.root)
    else:
        if args.binary is None or args.repeats < 1:
            cli.error("binary and positive repeats are required")
        if args.action == "identity":
            identity(args.root, args.binary, args.depth)
        else:
            paired(args.root, args.binary, args.cpu, args.repeats, args.duration)


if __name__ == "__main__":
    main()

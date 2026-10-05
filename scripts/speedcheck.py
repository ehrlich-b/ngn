#!/usr/bin/env python3
"""NGN-only paired search measurement and exact search identity checks."""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def probe(binary, cpu, mode, net, depth, duration, position=-1):
    args = ["taskset", "-c", str(cpu), binary, "-mode", mode,
            "-depth", str(depth), "-duration", f"{duration}s", "-position", str(position)]
    if net:
        args += ["-net", net]
    env = dict(os.environ, GOMAXPROCS="1")
    output = subprocess.run(args, env=env, text=True, capture_output=True,
                            timeout=600 if mode == "identity" else duration + 60, check=True)
    rows = [json.loads(line) for line in output.stdout.splitlines()]
    expected = 12 if mode == "identity" and position == -1 else 1
    if len(rows) != expected or any(row["nodes"] <= 0 for row in rows):
        raise RuntimeError(f"incomplete probe: {output.stdout!r}")
    if mode == "timed" and any(not duration * .98 <= row["seconds"] <= duration + 2 for row in rows):
        raise RuntimeError(f"unexpected timed search duration: {rows!r}")
    return rows


def identity(rows):
    return [{key: value for key, value in row.items() if key not in ("seconds", "nps")} for row in rows]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline")
    parser.add_argument("candidate")
    parser.add_argument("--net", required=True)
    parser.add_argument("--cpu", type=int, default=2)
    parser.add_argument("--depth", type=int, default=9)
    parser.add_argument("--duration", type=float, default=10)
    parser.add_argument("--repeats", type=int, default=3)
    parser.add_argument("--out", required=True)
    parser.add_argument("--identity-only", action="store_true")
    args = parser.parse_args()
    if args.repeats < 1 or args.duration <= 0 or args.cpu not in os.sched_getaffinity(0):
        parser.error("invalid trial bounds or CPU")
    result = {"baseline_sha256": digest(args.baseline), "candidate_sha256": digest(args.candidate),
              "net_sha256": digest(args.net), "cpu": args.cpu, "depth": args.depth,
              "duration": args.duration, "repeats": args.repeats, "identity": {}, "trials": []}
    output = Path(args.out)
    for backend, net in [("hce", ""), ("nnue", args.net)]:
        a = probe(args.baseline, args.cpu, "identity", net, args.depth, args.duration)
        b = probe(args.candidate, args.cpu, "identity", net, args.depth, args.duration)
        if identity(a) != identity(b):
            result["identity"][backend] = {"equal": False, "baseline": a, "candidate": b}
            output.write_text(json.dumps(result, indent=2) + "\n")
            raise RuntimeError(f"{backend} identity mismatch; see {output}")
        result["identity"][backend] = {"equal": True, "searches": identity(a)}
    if not args.identity_only:
        with output.with_suffix(".jsonl").open("w") as journal:
            for repeat in range(args.repeats):
                for position in range(6):
                    for backend_index, (backend, net) in enumerate([("hce", ""), ("nnue", args.net)]):
                        pair = {}
                        order = ["baseline", "candidate"]
                        if (repeat + position + backend_index) % 2:
                            order.reverse()
                        for side in order:
                            binary = args.baseline if side == "baseline" else args.candidate
                            row = probe(binary, args.cpu, "timed", net, args.depth, args.duration, position)[0]
                            pair[side] = row
                        trial = dict(repeat=repeat, position=position, backend=backend, order=order,
                                     ratio=pair["candidate"]["nps"] / pair["baseline"]["nps"], **pair)
                        result["trials"].append(trial)
                        journal.write(json.dumps(trial) + "\n")
                        journal.flush()
        result["summary"] = {}
        for backend in ("hce", "nnue"):
            trials = [trial for trial in result["trials"] if trial["backend"] == backend]
            result["summary"][backend] = {
                "geomean_gain_pct": 100 * (math.exp(statistics.mean(math.log(t["ratio"]) for t in trials)) - 1),
                "pooled_gain_pct": 100 * ((sum(t["candidate"]["nodes"] for t in trials) / sum(t["candidate"]["seconds"] for t in trials)) /
                                          (sum(t["baseline"]["nodes"] for t in trials) / sum(t["baseline"]["seconds"] for t in trials)) - 1),
                "repeat_gain_pct": [100 * (math.exp(statistics.mean(math.log(t["ratio"]) for t in trials if t["repeat"] == r)) - 1)
                                    for r in range(args.repeats)],
                "position_gain_pct": [100 * (math.exp(statistics.mean(math.log(t["ratio"]) for t in trials if t["position"] == p)) - 1)
                                      for p in range(6)],
                "baseline_nps": sum(t["baseline"]["nodes"] for t in trials) / sum(t["baseline"]["seconds"] for t in trials),
                "candidate_nps": sum(t["candidate"]["nodes"] for t in trials) / sum(t["candidate"]["seconds"] for t in trials),
            }
    output.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result.get("summary", result["identity"])))


if __name__ == "__main__":
    main()

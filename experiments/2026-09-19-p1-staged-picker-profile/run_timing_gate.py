#!/usr/bin/env python3
"""Run the frozen ten-block interleaved P1 fixed-node timing gate."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import statistics
import subprocess
import time
from pathlib import Path


FIXTURES = (
    "start_white",
    "start_black",
    "kiwipete_white",
    "kiwipete_black",
    "asymmetric_white",
    "asymmetric_black",
)
BENCHMARK = "BenchmarkRodentV12OutputGateColdFixedNodes"
MODEL_SHA256 = "c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053"
MODEL_BYTES = 4_744_768
BLOCKS = 10
MAX_MEDIAN_RATIO = 0.97
ROW = re.compile(
    rf"^{BENCHMARK}/([^\s]+)\s+1\s+(\d+)\s+ns/op\s+"
    r"400000\s+minimum_nodes/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op$"
)


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def run_once(name: str, binary: Path, model: Path, cpu: int, raw_path: Path) -> dict:
    command = [
        "taskset",
        "-c",
        str(cpu),
        str(binary),
        "-test.run=^$",
        f"-test.bench=^{BENCHMARK}$",
        "-test.benchtime=1x",
        "-test.count=1",
        "-test.benchmem",
        "-test.timeout=10m",
    ]
    env = os.environ.copy()
    env.update(
        {
            "GOMAXPROCS": "1",
            "GOAMD64": "v3",
            "RODENT_V12_DEFAULT_MODEL": str(model),
        }
    )
    started = time.monotonic()
    completed = subprocess.run(
        command,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=300,
        check=False,
    )
    elapsed = time.monotonic() - started
    raw_path.write_text(
        completed.stdout + ("\n--- STDERR ---\n" + completed.stderr if completed.stderr else ""),
        encoding="utf-8",
    )
    if completed.returncode != 0 or "PASS\n" not in completed.stdout:
        raise RuntimeError(f"{name} benchmark failed with exit {completed.returncode}: {raw_path}")

    fixtures = {}
    for line in completed.stdout.splitlines():
        match = ROW.match(line.strip())
        if not match:
            continue
        fixture, ns, bytes_per_op, allocs = match.groups()
        if fixture in fixtures:
            raise RuntimeError(f"duplicate fixture {fixture} in {raw_path}")
        fixtures[fixture] = {
            "ns": int(ns),
            "bytes_per_op": int(bytes_per_op),
            "allocs_per_op": int(allocs),
        }
    if tuple(fixtures) != FIXTURES:
        raise RuntimeError(f"fixture order/content mismatch in {raw_path}: {tuple(fixtures)}")
    return {
        "engine": name,
        "elapsed_seconds": elapsed,
        "aggregate_ns": sum(row["ns"] for row in fixtures.values()),
        "fixtures": fixtures,
        "raw_path": str(raw_path),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", type=Path, required=True)
    parser.add_argument("--candidate", type=Path, required=True)
    parser.add_argument("--model", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--cpu", type=int, default=2)
    parser.add_argument("--base-source", required=True)
    parser.add_argument("--candidate-source", required=True)
    args = parser.parse_args()

    for path in (args.base, args.candidate, args.model):
        if not path.is_file():
            raise SystemExit(f"missing input: {path}")
    if args.model.stat().st_size != MODEL_BYTES or digest(args.model) != MODEL_SHA256:
        raise SystemExit("model identity mismatch")
    args.out.mkdir(parents=True, exist_ok=False)

    blocks = []
    ratios = []
    for block in range(1, BLOCKS + 1):
        order = ("base", "candidate") if block % 2 else ("candidate", "base")
        results = {}
        for engine_name in order:
            binary = args.base if engine_name == "base" else args.candidate
            raw = args.out / f"block-{block:02d}-{engine_name}.txt"
            results[engine_name] = run_once(engine_name, binary, args.model, args.cpu, raw)
        ratio = results["candidate"]["aggregate_ns"] / results["base"]["aggregate_ns"]
        ratios.append(ratio)
        blocks.append({"block": block, "order": list(order), "ratio": ratio, **results})

    median_ratio = statistics.median(ratios)
    result = {
        "schema": "ngn-p1-staged-picker-timing-v1",
        "base_source": args.base_source,
        "candidate_source": args.candidate_source,
        "base_binary_sha256": digest(args.base),
        "candidate_binary_sha256": digest(args.candidate),
        "model_sha256": MODEL_SHA256,
        "cpu": args.cpu,
        "blocks": blocks,
        "ratios": ratios,
        "median_candidate_base_ratio": median_ratio,
        "maximum_median_ratio": MAX_MEDIAN_RATIO,
        "passes": median_ratio <= MAX_MEDIAN_RATIO,
    }
    result_path = args.out / "result.json"
    result_path.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["passes"] else 1


if __name__ == "__main__":
    raise SystemExit(main())

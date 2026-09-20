#!/usr/bin/env python3
"""Summarize the frozen P1 base/candidate searchsnapshot node falsifier."""

from __future__ import annotations

import argparse
import json
import math
from pathlib import Path


def load(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("baseline", type=Path)
    parser.add_argument("candidate", type=Path)
    args = parser.parse_args()

    baseline = load(args.baseline)
    candidate = load(args.candidate)
    if len(baseline["fixtures"]) != len(candidate["fixtures"]):
        raise SystemExit("fixture count differs")

    rows = []
    log_ratios = []
    max_row = None
    stage_totals = {
        "PickerTTFirstMoves": 0,
        "PickerTacticalStages": 0,
        "PickerQuietStages": 0,
        "PickerTacticalMoves": 0,
        "PickerQuietMoves": 0,
        "PickerBadCaptureMoves": 0,
    }
    for base_fixture, candidate_fixture in zip(
        baseline["fixtures"], candidate["fixtures"], strict=True
    ):
        if base_fixture["name"] != candidate_fixture["name"]:
            raise SystemExit("fixture order differs")
        for base_pass, candidate_pass in zip(
            base_fixture["passes"], candidate_fixture["passes"], strict=True
        ):
            if base_pass["state"] != candidate_pass["state"]:
                raise SystemExit("pass order differs")
            base_nodes = int(base_pass["final"]["nodes"])
            candidate_nodes = int(candidate_pass["final"]["nodes"])
            ratio = None
            if base_nodes > 0:
                ratio = candidate_nodes / base_nodes
                log_ratios.append(math.log(ratio))
            row = {
                "fixture": base_fixture["name"],
                "state": base_pass["state"],
                "base_nodes": base_nodes,
                "candidate_nodes": candidate_nodes,
                "ratio": ratio,
                "base_best_move": base_pass["final"]["best_move"],
                "candidate_best_move": candidate_pass["final"]["best_move"],
            }
            rows.append(row)
            if ratio is not None and (max_row is None or ratio > max_row["ratio"]):
                max_row = row

            diagnostics = candidate_pass["final"]["diagnostics"]
            for name in stage_totals:
                stage_totals[name] += int(diagnostics.get(name, 0))

    geometric_mean = math.exp(sum(log_ratios) / len(log_ratios))
    result = {
        "schema": "ngn-p1-snapshot-summary-v1",
        "rows": rows,
        "ratio_count": len(log_ratios),
        "geometric_mean_node_ratio": geometric_mean,
        "maximum_node_ratio": max_row,
        "aggregate_limit": 1.05,
        "per_pass_limit": 1.15,
        "aggregate_pass": geometric_mean <= 1.05,
        "per_pass_pass": max_row is not None and max_row["ratio"] <= 1.15,
        "stage_totals": stage_totals,
    }
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["aggregate_pass"] and result["per_pass_pass"] else 1


if __name__ == "__main__":
    raise SystemExit(main())

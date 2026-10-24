#!/usr/bin/env python3
"""Profile the accepted SMP search without changing engine source in the checkout."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import statistics
import subprocess
import sys
import tempfile

from time_v import environment, run, telemetry
from resources import capture_resource_evidence, summarize_resources

BASE = "8ba878cb03a7eb81ca465b3d562953d0f45e81c7"
MODEL_SHA = "c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053"
HERE = Path(__file__).resolve().parent
REPO = HERE.parent.parent


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def validate_model(path):
    if path is None or not Path(path).is_file():
        raise ValueError("pinned V1.2 model unavailable; supply a local --model; no downloads permitted")
    path = Path(path).resolve()
    digest = sha(path)
    if path.stat().st_size != 4744768 or digest != MODEL_SHA:
        raise ValueError("pinned model identity mismatch")
    return path, digest


def wire_report_resources(report, evidence):
    report["resources"] = summarize_resources(evidence, requested_width=None)
    report["resources"]["evidence_reference"] = "initial_telemetry and raw_evidence references"
    return report


def wire_cell_resources(row, receipt, width, cell):
    evidence = dict(receipt.get("resource_evidence") or {})
    evidence["gomaxprocs"] = row.get("gomaxprocs")
    evidence["effective_worker_width"] = row.get("effective_threads")
    row["resources"] = summarize_resources(evidence, requested_width=width)
    row["resources"]["evidence_reference"] = str(Path(cell) / "time.json")
    row["non_owned_busy_cpu_seconds_estimate"] = receipt.get("non_owned_busy_cpu_seconds_estimate")
    row["measurement_interval_seconds"] = receipt.get("measurement_interval_seconds")
    row["shared_host_telemetry_reference"] = str(Path(cell) / "telemetry.json")
    row["shared_host_load_caveat"] = (
        "Non-owned busy CPU is a coarse host-counter-minus-owned-CPU estimate; it does not attribute "
        "work to individual jobs, and shared-host scheduling and staggered measurements limit causal inference.")
    if width in (4, 8) and row.get("gomaxprocs") == 2:
        row["measurement_classification"] = "oversubscribed worker measurement"
        row["scaling_interpretation"] = "not core scaling or playing efficiency"
    return row


def git(*args):
    return subprocess.check_output(["git", *args], cwd=REPO, env=environment(), text=True).strip()


def duplication(workers):
    """Hash overlap, not a claim that repeated positions had identical search windows."""
    union = set()
    memberships = 0
    visits = 0
    for worker in workers:
        samples = worker.get("samples", {})
        union.update(samples)
        memberships += len(samples)
        visits += sum(samples.values())
    return {"sampled_visits": visits, "worker_unique_sum": memberships,
            "union_unique": len(union), "cross_worker_excess": memberships-len(union),
            "cross_worker_excess_fraction": (memberships-len(union))/memberships if memberships else 0,
            "within_worker_revisits": visits-memberships}


def overlays(folder):
    # Both builds add only the package probe. Only diagnostic adds hash sampling.
    probe = (HERE / "probe_test.go.txt").read_text()
    output = {}
    for kind in ["native", "diagnostic"]:
        target = folder / kind
        target.mkdir()
        source = probe
        replacements = {}
        if kind == "diagnostic":
            assert source.count("return nil }") == 1
            source = source.replace("return nil }", "return info.smpObserved }")
            search = (REPO / "engine/search.go").read_text()
            assert search.count("type SearchInfo struct {\n") == 1
            prefix, search = search.split("func alphaBetaPV(", 1)
            assert search.count("\thash := pos.Hash()\n") == 2
            prefix = prefix.replace("type SearchInfo struct {\n",
                                    "type SearchInfo struct {\n\tsmpObserved map[uint64]uint64\n")
            search = search.replace("\thash := pos.Hash()\n", """\thash := pos.Hash()
\tif hash & 63 == 0 {
\t\tif info.smpObserved == nil { info.smpObserved = make(map[uint64]uint64) }
\t\tinfo.smpObserved[hash]++
\t}
""")
            changed = target / "search.go"
            changed.write_text(prefix + "func alphaBetaPV(" + search)
            replacements[str(REPO / "engine/search.go")] = str(changed)
        fixture = target / "probe_test.go"
        fixture.write_text(source)
        replacements[str(REPO / "engine/smp_profile_probe_test.go")] = str(fixture)
        overlay = target / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replacements}, indent=2)+"\n")
        binary = target / "engine.test"
        run(["go", "test", "-p", "2", "-overlay", str(overlay), "-c", "-o", str(binary),
             "./engine"], target / "build", cwd=REPO)
        output[kind] = binary
    return output


def tt_annotation(binary, profile, folder):
    """An unsynchronized one-worker search may have no TT lock samples."""
    pattern = "Cache.*(Get|Set)"
    try:
        run(["go", "tool", "pprof", "-list="+pattern, str(binary), str(profile)],
            folder, cwd=REPO)
    except RuntimeError:
        receipt = json.loads((folder / "time.json").read_text())
        error = (folder / "stderr.txt").read_text().strip()
        if receipt["exit_code"] != 2 or error != "no matches found for regexp: "+pattern:
            raise
        return "NO_TT_SAMPLES"
    return "ANNOTATED"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", type=Path, default=os.environ.get("RODENT_V12_DEFAULT_MODEL"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--check-harness", action="store_true")
    args = parser.parse_args()
    args.output = args.output.resolve()
    if args.output == REPO or REPO in args.output.parents:
        parser.error("evidence must be outside the checkout")
    args.output.mkdir(parents=True, exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix="run-", dir=args.output))
    report = {"status": "RUNNING", "base": BASE, "head": git("rev-parse", "HEAD"),
              "model_sha256_required": MODEL_SHA, "gomaxprocs": 2,
              "model_requested": str(args.model) if args.model else None,
              "host": platform.uname()._asdict(), "affinity": sorted(os.sched_getaffinity(0)),
              "initial_telemetry": telemetry(), "evidence": str(folder),
              "mechanisms_admitted": [], "matches_run": 0,
              "protocol": {"milliseconds": 2000, "repetitions": 3, "widths": [1,2,4,8],
                           "hash_mb": 128, "diagnostic_hash_mask": 63,
                           "control_nodes": 400000, "seed": 20261023}}
    initial_resources = capture_resource_evidence()
    initial_resources["gomaxprocs"] = None
    wire_report_resources(report, initial_resources)
    try:
        for tree in ["engine", "rodentv12eval", "internal", "go.mod", "go.sum"]:
            if git("diff", BASE, "--", tree):
                raise ValueError("accepted source changed: "+tree)
        report["source_tree"] = git("rev-parse", "HEAD:engine")
        historical = json.loads((REPO / "experiments/2026-09-19-e0-v12-anand-strength/manifest-release-v1.json").read_text())
        # Preserve the recorded model location check as an explicit prerequisite witness.
        def model_record(value):
            if isinstance(value, dict):
                if "v12_model" in value:
                    return value["v12_model"]
                for child in value.values():
                    found = model_record(child)
                    if found:
                        return found
            return None
        pinned = model_record(historical)
        assert pinned and pinned["sha256"] == MODEL_SHA
        report["historical_model"] = {**pinned, "exists":Path(pinned["path"]).is_file()}
        if args.check_harness:
            binaries = overlays(folder)
            assert duplication([{"samples":{"1":2,"2":1}}, {"samples":{"2":3,"3":1}}]) == {
                "sampled_visits":7,"worker_unique_sum":4,"union_unique":3,
                "cross_worker_excess":1,"cross_worker_excess_fraction":0.25,"within_worker_revisits":3}
            report.update(status="HARNESS_COMPILED_ONLY", binaries={k:sha(v) for k,v in binaries.items()})
            print(json.dumps({"status":report["status"],"evidence":str(folder)}))
            return 0
        args.model, model_digest = validate_model(args.model)
        report["model_path"] = str(args.model)
        report["model_sha256"] = model_digest
        binaries = overlays(folder)
        report["binaries"] = {k:sha(v) for k,v in binaries.items()}
        positions = json.loads((HERE / "positions.json").read_text())
        rng = random.Random(20261023)
        rows = []
        # Fixed-node controls are strictly one-worker; every scaling cell uses time.
        schedule = [("control",0,1,p) for p in positions]
        for repetition in range(3):
            block = [("throughput",repetition,w,p) for w in [1,2,4,8] for p in positions]
            rng.shuffle(block)
            schedule += block
        for mode in ["profile","duplication"]:
            block = [(mode,0,w,p) for w in [1,2,4,8] for p in positions]
            rng.shuffle(block)
            schedule += block
        for index, (mode, repetition, width, position) in enumerate(schedule):
            cell = folder / f"{index:03}-{mode}-{repetition}-{width}-{position['name']}"
            kind = "diagnostic" if mode == "duplication" else "native"
            env = environment()
            env.update(RODENT_V12_DEFAULT_MODEL=str(args.model), SMP_WIDTH=str(width),
                       SMP_MS="2000", SMP_NODES="400000" if mode == "control" else "0",
                       SMP_FEN=position["fen"], SMP_OUT=str(cell),
                       SMP_PROFILES="1" if mode == "profile" else "0")
            command = [str(binaries[kind]), "-test.run=^TestSMPProfile$", "-test.count=1",
                       "-test.timeout=20s", "-test.v"]
            receipt = run(command, cell, env, REPO)
            row = json.loads((cell / "result.json").read_text())
            row.update(mode=mode, repetition=repetition, position=position["name"], evidence=str(cell))
            wire_cell_resources(row, receipt, width, cell)
            if mode == "duplication":
                row["duplication"] = duplication(row["workers"])
                if row["duplication"]["sampled_visits"] == 0:
                    raise ValueError("empty diagnostic sampling")
            if mode == "profile":
                for profile in ["cpu", "mutex", "block"]:
                    command = ["go", "tool", "pprof", "-top", "-nodecount=40"]
                    if profile == "cpu":
                        command += ["-tagfocus=scope=search"]
                    run(command+[str(binaries[kind]),str(cell / (profile+".pprof"))],
                        cell / (profile+"-top"), cwd=REPO)
                    if profile != "cpu":
                        row[profile+"_tt_annotation"] = tt_annotation(
                            binaries[kind], cell / (profile+".pprof"), cell / (profile+"-tt"))
            rows.append(row)
            print(f"{index+1}/{len(schedule)} {mode} {width} {position['name']}", flush=True)
        summary = {}
        for width in [1,2,4,8]:
            selected = [r for r in rows if r["mode"] == "throughput" and r["width"] == width]
            summary[str(width)] = {"nodes":sum(r["nodes"] for r in selected),
                                   "seconds":sum(r["seconds"] for r in selected),
                                   "median_nps":statistics.median(r["nps"] for r in selected),
                                   "depths":[r["depth"] for r in selected]}
            summary[str(width)]["aggregate_nps"] = summary[str(width)]["nodes"]/summary[str(width)]["seconds"]
        for width in [1,2,4,8]:
            speedup = summary[str(width)]["aggregate_nps"]/summary["1"]["aggregate_nps"]
            summary[str(width)].update(speedup=speedup)
            summary[str(width)]["logical_worker_ratio"] = speedup / width
            summary[str(width)]["interpretation"] = ("oversubscribed worker measurement; not core scaling or playing efficiency"
                                                        if width in (4, 8) else "logical worker measurement")
        report.update(status="MEASURED_CLOSE_NO_POLICY", rows=rows, summary=summary,
                      reopen_condition="On a host authorized for >=8 Go execution slots, repeat the pinned "
                      "matrix; require a repeatable measured dominant cost and a mechanism distinct from "
                      "the three shelved variants before admitting one candidate; require an independent "
                      "playing-efficiency gate on an authorized match host before strength claims.")
        return 0
    except (ValueError, RuntimeError, OSError, subprocess.CalledProcessError) as exc:
        report.update(status="BLOCKED", error=str(exc),
                      reopen_condition="Provide the hash-pinned local model, then execute this frozen protocol; "
                      "no SMP policy admitted without measurements.")
        print(str(exc), file=sys.stderr)
        return 1
    finally:
        (folder / "report.json").write_text(json.dumps(report, indent=2)+"\n")
        print("report: "+str(folder / "report.json"), flush=True)


if __name__ == "__main__":
    sys.exit(main())

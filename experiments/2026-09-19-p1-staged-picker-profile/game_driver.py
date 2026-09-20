#!/usr/bin/env python3
"""Frozen P1 reference-picker pilot and independent confirmation. WSL only."""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import sys


SCHEMA = "ngn-p1-reference-picker-game-gate-v1"
HELD = "HELD_PENDING_STATIC_REVIEW"
RELEASED = "RELEASED_AFTER_STATIC_REVIEW"
PROTOCOL = {
    "pilot_pairs": 64,
    "confirmation_pairs": 200,
    "tc": "10+0.1",
    "concurrency": 4,
    "physical_cpu_mask": ["0", "2", "4", "6"],
    "seed": 20260919,
    "pilot_bootstrap_seed": 2026091904,
    "confirmation_bootstrap_seed": 2026091905,
    "bootstrap_replicates": 100000,
}
RUN_ROOT = "/home/ehrli/p1-staged-picker-game-20260919/run-001"
DECISION_RULE = (
    "Run one fixed 64-pair candidate/base pilot on canonical openings 1-64. "
    "SHELVE if the paired-bootstrap upper95 Elo is below zero, or if point Elo "
    "is non-positive without an independent mechanism advantage. Otherwise run "
    "one fixed 200-pair confirmation on independent canonical source lines "
    "201-400. ACCEPT only if its paired-bootstrap lower95 Elo is strictly "
    "positive and all legal, process, telemetry and supervisor gates pass; "
    "otherwise SHELVE; never extend or rerun after observing score"
)
CLAIM_BOUNDARY = (
    "Relative Threads=1 Rodent-V1.2 staged-picker evidence at 10+0.1 on the "
    "shared WSL host; not an absolute rating, 3300 proof, wider-thread claim, "
    "installation or release approval"
)
SHARED_HOST_CONTRACT = (
    "the Lean hopper remains running; ten-second prelaunch and five-second "
    "continuous non-owned CPU samples are retained as telemetry, but background "
    "CPU alone does not reject the sample; scheduler and time-management "
    "interference remain a disclosed limitation"
)
MECHANISM_GATE = {
    "snapshot_summary_sha256": "1d0508beba00463bb3c1a79339a315ee9c14df0e9e35e919667e3fabaa63f0cd",
    "snapshot_geometric_mean_node_ratio": 1.0064150954406066,
    "snapshot_maximum_node_ratio": 1.147810218978102,
    "snapshot_passes": True,
    "timing_result_sha256": "5ea2b3ecd0a7f8fb2adc22c1200bbe5f8b7536d609e2256a88a6fd7d687746ae",
    "timing_median_candidate_base_ratio": 0.950269675008514,
    "timing_maximum_ratio": 0.97,
    "timing_passes": True,
    "full_short_tests": "PASS",
    "engine_race_tests": "PASS",
}
EXPECTED_SOURCES = {
    "e0_driver.py", "role_exec.py", "run_match_stage.py", "process_supervisor.py",
    "trace_audit.py", "audit_fastchess_match.py", "uci_preflight.py", "common.py",
    "fastchess", "stockfish", "pilot_openings", "pilot_prefixes",
    "confirmation_openings", "confirmation_prefixes", "v12_model",
}
ENGINE_KEYS = {
    "path", "sha256", "source_commit", "source_tree", "go_version",
    "goamd64", "cgo_enabled", "build_flags",
}


def die(message: str) -> None:
    raise SystemExit(f"p1-reference-picker-game: {message}")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def atomic_json(path: Path, value: object) -> None:
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.replace(temporary, path)


def checked(spec: dict, label: str, expected_keys: set[str] | None = None) -> Path:
    if expected_keys is not None and set(spec) != expected_keys:
        die(f"{label}: unexpected identity keys")
    if "path" not in spec or "sha256" not in spec:
        die(f"{label}: incomplete identity")
    path = Path(spec["path"]).resolve()
    if not path.is_file() or sha256(path) != spec["sha256"]:
        die(f"{label}: identity mismatch at {path}")
    return path


def load_module(path: Path):
    spec = importlib.util.spec_from_file_location("p1_frozen_e0_driver", path)
    if spec is None or spec.loader is None:
        die("cannot load frozen candidate-match helper")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def validate_manifest(manifest: dict, driver_hash: str) -> None:
    expected = {
        "schema", "driver_sha256", "release_state", "run_root", "protocol",
        "decision_rule", "claim_boundary", "shared_host_contract",
        "mechanism_gate", "engines", "frozen_sources",
    }
    if set(manifest) != expected or manifest.get("schema") != SCHEMA:
        die("manifest keys/schema differ from contract")
    if manifest["driver_sha256"] != driver_hash:
        die("driver hash differs from manifest")
    if manifest["release_state"] not in {HELD, RELEASED}:
        die("invalid release state")
    if manifest["run_root"] != RUN_ROOT or Path(RUN_ROOT).resolve().exists():
        die("run root differs or already exists")
    if manifest["protocol"] != PROTOCOL:
        die("protocol differs from contract")
    contracts = {
        "decision_rule": DECISION_RULE,
        "claim_boundary": CLAIM_BOUNDARY,
        "shared_host_contract": SHARED_HOST_CONTRACT,
        "mechanism_gate": MECHANISM_GATE,
    }
    for key, value in contracts.items():
        if manifest[key] != value:
            die(f"{key} differs from contract")
    if not (
        manifest["mechanism_gate"]["snapshot_passes"]
        and manifest["mechanism_gate"]["timing_passes"]
        and manifest["mechanism_gate"]["timing_median_candidate_base_ratio"]
        <= manifest["mechanism_gate"]["timing_maximum_ratio"]
    ):
        die("mechanism gate does not admit games")
    if set(manifest["frozen_sources"]) != EXPECTED_SOURCES:
        die("frozen source set differs")
    if set(manifest["engines"]) != {"base", "candidate"}:
        die("engine role set differs")
    hashes = set()
    for role, engine in manifest["engines"].items():
        if set(engine) != ENGINE_KEYS:
            die(f"{role} engine provenance differs")
        if engine["go_version"] != "go1.25.5 linux/amd64":
            die(f"{role} Go version differs")
        if engine["goamd64"] != "v3" or engine["cgo_enabled"] != "0" or engine["build_flags"] != ["-trimpath"]:
            die(f"{role} build contract differs")
        for key in ("source_commit", "source_tree"):
            value = engine[key]
            if len(value) != 40 or any(character not in "0123456789abcdef" for character in value):
                die(f"{role} {key} is not a full object ID")
        hashes.add(engine["sha256"])
    if len(hashes) != 2:
        die("base and candidate binaries must differ")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--check-held", action="store_true")
    args = parser.parse_args()
    if sys.platform != "linux" or "microsoft" not in Path("/proc/version").read_text(encoding="utf-8").lower():
        die("WSL execution required")
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    driver_hash = sha256(Path(__file__).resolve())
    validate_manifest(manifest, driver_hash)
    sources = {
        name: checked(spec, name, {"path", "sha256"})
        for name, spec in manifest["frozen_sources"].items()
    }
    engines = {
        role: checked(spec, f"engine {role}", ENGINE_KEYS)
        for role, spec in manifest["engines"].items()
    }
    if args.check_held:
        print(json.dumps({
            "state": "HELD_STATIC_CHECK_COMPLETE",
            "driver_sha256": driver_hash,
            "engines": {role: sha256(path) for role, path in engines.items()},
            "frozen_sources": len(sources),
            "run_root": "absent",
        }, sort_keys=True))
        return
    if manifest["release_state"] != RELEASED:
        die("manifest is held")

    helper = load_module(sources["e0_driver.py"])
    run_root = Path(RUN_ROOT).resolve()
    run_root.mkdir(parents=True)
    helper.baseline_cpu(10, run_root / "prelaunch-non-owned-cpu.json")
    atomic_json(run_root / "frozen-manifest.json", manifest)
    source_dir = run_root / "source"
    tools_dir = run_root / "tools"
    inputs_dir = run_root / "inputs"
    for directory in (source_dir, tools_dir, inputs_dir):
        directory.mkdir()
    shutil.copy2(Path(__file__).resolve(), source_dir / "game_driver.py")
    for name in (
        "e0_driver.py", "role_exec.py", "run_match_stage.py", "process_supervisor.py",
        "trace_audit.py", "audit_fastchess_match.py", "uci_preflight.py", "common.py",
    ):
        helper.copy_frozen(sources[name], source_dir / name, manifest["frozen_sources"][name]["sha256"], executable=True)
    for name in ("fastchess", "stockfish"):
        helper.copy_frozen(sources[name], tools_dir / name, manifest["frozen_sources"][name]["sha256"], executable=True)
    copied = {}
    for name in ("pilot_openings", "pilot_prefixes", "confirmation_openings", "confirmation_prefixes", "v12_model"):
        destination = inputs_dir / Path(sources[name]).name
        helper.copy_frozen(sources[name], destination, manifest["frozen_sources"][name]["sha256"])
        copied[name] = destination
    frozen_engines = {}
    for role, source in engines.items():
        destination = inputs_dir / f"ngn-p1-{role}"
        helper.copy_frozen(source, destination, manifest["engines"][role]["sha256"], executable=True)
        frozen_engines[role] = destination

    phase_results = {}

    def run_phase(name: str, pairs: int, book: str) -> None:
        phase = run_root / name
        phase.mkdir()
        roles = []
        for side, role in (("a", "candidate"), ("b", "base")):
            role_dir = phase / f"role-{side}"
            role_dir.mkdir()
            engine = role_dir / "ngn"
            helper.copy_frozen(frozen_engines[role], engine, manifest["engines"][role]["sha256"], executable=True)
            launcher = role_dir / "launch"
            helper.copy_frozen(source_dir / "role_exec.py", launcher, manifest["frozen_sources"]["role_exec.py"]["sha256"], executable=True)
            helper.role_config(role_dir / "role-config.json", engine, manifest["engines"][role]["sha256"], role_dir)
            roles.append({
                "id": f"{name}-{side}",
                "name": f"P1-{role}-{side.upper()}",
                "semantic": role,
                "backend": "rodent-v1.2-default",
                "model": copied["v12_model"],
                "cwd": role_dir,
                "engine": engine,
                "sha256": manifest["engines"][role]["sha256"],
                "launcher": launcher,
            })
        for role in roles:
            helper.preflight(source_dir, role, phase)
        openings = copied[f"{book}_openings"]
        prefixes = copied[f"{book}_prefixes"]
        command = helper.phase_command(tools_dir, phase, roles, openings, pairs, manifest["protocol"])
        atomic_json(phase / "command.json", {"argv": command})
        helper.supervise(
            source_dir, command, phase, roles,
            manifest["protocol"]["physical_cpu_mask"],
            3600 if book == "pilot" else 7200,
        )
        (phase / "match.exit").write_text("0\n", encoding="utf-8")
        audited = helper.audit(source_dir, tools_dir, phase, roles, openings, prefixes, pairs * 2, pairs)
        bootstrap_seed = manifest["protocol"][f"{book}_bootstrap_seed"]
        phase_results[name] = {
            "engine_a": "candidate",
            "engine_b": "base",
            "games": pairs * 2,
            "pairs": pairs,
            "legal_plies": audited["legal_plies"],
            "engine_a_points": audited["engine_a_points"],
            "engine_a_score_rate": audited["engine_a_score_rate"],
            "penta_0_to_4": audited["penta_0_to_4"],
            "paired_normal_ci": helper.paired_ci(audited["penta_0_to_4"]),
            "paired_bootstrap_ci": helper.paired_bootstrap(
                audited["pair_audit"], bootstrap_seed,
                manifest["protocol"]["bootstrap_replicates"],
            ),
            "evidence_sha256": {
                artifact: sha256(phase / artifact) for artifact in (
                    "audit-match.json", "audit-operational.json", "match-witness.json",
                    "supervisor.json", "games.pgn", "final.epd", "match.stdout",
                    "match.stderr", "fastchess.log", "non-owned-cpu.json",
                )
            },
        }

    run_phase("pilot", manifest["protocol"]["pilot_pairs"], "pilot")
    pilot_ci = phase_results["pilot"]["paired_bootstrap_ci"]
    mechanism_advantage = (
        manifest["mechanism_gate"]["timing_passes"]
        and manifest["mechanism_gate"]["timing_median_candidate_base_ratio"]
        <= manifest["mechanism_gate"]["timing_maximum_ratio"]
    )
    if pilot_ci["upper95_elo"] < 0:
        decision = "SHELVE_CLEARLY_BAD_PILOT"
    elif pilot_ci["elo"] <= 0 and not mechanism_advantage:
        decision = "SHELVE_NONPOSITIVE_WITHOUT_MECHANISM_ADVANTAGE"
    else:
        run_phase("confirmation", manifest["protocol"]["confirmation_pairs"], "confirmation")
        decision = (
            "ACCEPT" if phase_results["confirmation"]["paired_bootstrap_ci"]["lower95_elo"] > 0
            else "SHELVE"
        )

    decision_path = run_root / "decision.json"
    atomic_json(decision_path, {
        "schema": SCHEMA + "-decision",
        "decision": decision,
        "rule": DECISION_RULE,
        "independent_mechanism_advantage": mechanism_advantage,
        "results": phase_results,
        "frozen_manifest_sha256": sha256(run_root / "frozen-manifest.json"),
    })
    inventory = {}
    for path in sorted(run_root.rglob("*")):
        if path.is_file() and path.name not in {"inventory.sha256.json", "terminal.json"}:
            inventory[str(path.relative_to(run_root))] = sha256(path)
    inventory_path = run_root / "inventory.sha256.json"
    atomic_json(inventory_path, {"schema": SCHEMA + "-inventory", "files": inventory})
    atomic_json(run_root / "terminal.json", {
        "schema": SCHEMA + "-terminal",
        "state": "COMPLETE",
        "decision": decision,
        "decision_sha256": sha256(decision_path),
        "inventory_sha256": sha256(inventory_path),
        "no_extension_after_observation": True,
        "shared_host_observational_evidence": True,
    })


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Recover P1's zero-game WSL infrastructure-null attempt without score drift."""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile


SCHEMA = "ngn-p1-reference-picker-game-recovery-v1"
HELD = "HELD_PENDING_STATIC_REVIEW"
RELEASED = "RELEASED_AFTER_STATIC_REVIEW"
FAILED_ROOT = Path("/home/ehrli/p1-staged-picker-game-20260919/run-001")
RECOVERY_ROOT = "/home/ehrli/p1-staged-picker-game-20260919/run-002"
MIN_WINDOWS_FREE_BYTES = 20 * 1024 * 1024 * 1024
RECOVERY_RULE = (
    "Treat run-001 as infrastructure-null only after exact hashes prove its "
    "supervisor remained RUNNING, witness remained STARTING, zero games finished, "
    "PGN and final EPD are empty, no decision or terminal receipt exists, and no "
    "match process survives. Do not inspect partial engine trace for selection. "
    "Permit one fresh run-002 of the byte-frozen original game protocol only when "
    "the Windows host volume has at least 20 GiB free; never reuse run-001"
)
EXPECTED_ARTIFACTS = {
    "frozen_manifest", "prelaunch_cpu", "supervisor", "witness", "games",
    "final_epd", "match_stdout", "match_stderr", "fastchess_log", "process_samples",
}


def die(message: str) -> None:
    raise SystemExit(f"p1-reference-picker-recovery: {message}")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def checked(spec: dict, label: str) -> Path:
    if set(spec) != {"path", "sha256"}:
        die(f"{label}: unexpected identity keys")
    path = Path(spec["path"]).resolve()
    if not path.is_file() or sha256(path) != spec["sha256"]:
        die(f"{label}: identity mismatch at {path}")
    return path


def load_module(path: Path):
    spec = importlib.util.spec_from_file_location("p1_original_game_driver", path)
    if spec is None or spec.loader is None:
        die("cannot load original game driver")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def validate_manifest(manifest: dict, driver_hash: str) -> tuple[Path, Path, dict[str, Path]]:
    expected = {
        "schema", "driver_sha256", "release_state", "failed_root", "recovery_root",
        "minimum_windows_free_bytes", "recovery_rule", "original_driver",
        "recovery_template", "failed_artifacts",
    }
    if set(manifest) != expected or manifest.get("schema") != SCHEMA:
        die("manifest keys/schema differ")
    if manifest["driver_sha256"] != driver_hash:
        die("recovery driver hash differs")
    if manifest["release_state"] not in {HELD, RELEASED}:
        die("invalid release state")
    if manifest["failed_root"] != str(FAILED_ROOT) or manifest["recovery_root"] != RECOVERY_ROOT:
        die("run roots differ from recovery contract")
    if manifest["minimum_windows_free_bytes"] != MIN_WINDOWS_FREE_BYTES:
        die("host free-space floor differs")
    if manifest["recovery_rule"] != RECOVERY_RULE:
        die("recovery rule differs")
    if Path(RECOVERY_ROOT).exists():
        die("immutable recovery root already exists")
    if set(manifest["failed_artifacts"]) != EXPECTED_ARTIFACTS:
        die("failed artifact set differs")
    original_driver = checked(manifest["original_driver"], "original driver")
    recovery_template = checked(manifest["recovery_template"], "recovery template")
    artifacts = {
        name: checked(record, f"failed artifact {name}")
        for name, record in manifest["failed_artifacts"].items()
    }
    return original_driver, recovery_template, artifacts


def live_processes_containing(needles: tuple[str, ...]) -> list[dict[str, object]]:
    matches = []
    for entry in Path("/proc").iterdir():
        if not entry.name.isdigit() or int(entry.name) == os.getpid():
            continue
        try:
            command = (entry / "cmdline").read_bytes().replace(b"\0", b" ").decode("utf-8", "replace")
        except (FileNotFoundError, ProcessLookupError, PermissionError):
            continue
        if any(needle in command for needle in needles):
            matches.append({"pid": int(entry.name), "command": command})
    return matches


def validate_null_attempt(artifacts: dict[str, Path]) -> dict[str, object]:
    supervisor = json.loads(artifacts["supervisor"].read_text(encoding="utf-8"))
    witness = json.loads(artifacts["witness"].read_text(encoding="utf-8"))
    if supervisor.get("state") != "RUNNING" or witness.get("state") != "STARTING":
        die("failed attempt advanced beyond its frozen null state")
    for name in ("games", "final_epd", "match_stderr"):
        if artifacts[name].stat().st_size != 0:
            die(f"failed attempt {name} is not empty")
    stdout = artifacts["match_stdout"].read_text(encoding="utf-8")
    trace = artifacts["fastchess_log"].read_text(encoding="utf-8", errors="replace")
    if stdout.count("Started game ") != 4 or "Finished game" in stdout or "Finished game" in trace:
        die("failed attempt contains an unexpected completed-game marker")
    for forbidden in (FAILED_ROOT / "decision.json", FAILED_ROOT / "terminal.json"):
        if forbidden.exists():
            die(f"failed attempt unexpectedly has {forbidden.name}")
    survivors = live_processes_containing((str(FAILED_ROOT), RECOVERY_ROOT))
    if survivors:
        die(f"match process survives: {survivors}")
    return {
        "supervisor_state": "RUNNING",
        "witness_state": "STARTING",
        "games_started": 4,
        "games_finished": 0,
        "empty_pgn_epd_stderr": True,
        "decision_absent": True,
        "terminal_absent": True,
        "surviving_match_processes": 0,
    }


def validate_template(path: Path, original_driver_hash: str) -> dict:
    template = json.loads(path.read_text(encoding="utf-8"))
    if template.get("schema") != "ngn-p1-reference-picker-game-gate-v1":
        die("execution template schema differs")
    if template.get("driver_sha256") != original_driver_hash:
        die("execution template does not bind the original driver")
    if template.get("release_state") != "HELD_PENDING_STATIC_REVIEW":
        die("execution template is not held")
    if template.get("run_root") != str(FAILED_ROOT):
        die("held execution template is not the original run-001 contract")
    return template


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--check-held", action="store_true")
    args = parser.parse_args()
    if sys.platform != "linux" or "microsoft" not in Path("/proc/version").read_text(encoding="utf-8").lower():
        die("WSL execution required")
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    driver_hash = sha256(Path(__file__).resolve())
    original_driver, recovery_template, artifacts = validate_manifest(manifest, driver_hash)
    null_state = validate_null_attempt(artifacts)
    original_driver_hash = manifest["original_driver"]["sha256"]
    template = validate_template(recovery_template, original_driver_hash)
    windows_free = shutil.disk_usage("/mnt/c").free
    ready = windows_free >= MIN_WINDOWS_FREE_BYTES
    if args.check_held:
        print(json.dumps({
            "state": "HELD_RECOVERY_CHECK_COMPLETE",
            "driver_sha256": driver_hash,
            "null_attempt": null_state,
            "windows_free_bytes": windows_free,
            "minimum_windows_free_bytes": MIN_WINDOWS_FREE_BYTES,
            "disk_ready": ready,
            "recovery_root": "absent",
        }, sort_keys=True))
        return
    if manifest["release_state"] != RELEASED:
        die("recovery manifest is held")
    if not ready:
        die(f"Windows free space {windows_free} is below {MIN_WINDOWS_FREE_BYTES}")

    released_template = dict(template)
    released_template["release_state"] = "RELEASED_AFTER_STATIC_REVIEW"
    released_template["run_root"] = RECOVERY_ROOT
    temporary_path: Path | None = None
    previous_argv = sys.argv
    try:
        with tempfile.NamedTemporaryFile(
            mode="w", encoding="utf-8", prefix="p1-recovery-release-", suffix=".json",
            dir="/tmp", delete=False,
        ) as handle:
            json.dump(released_template, handle, indent=2, sort_keys=True)
            handle.write("\n")
            temporary_path = Path(handle.name)
        original = load_module(original_driver)
        original.RUN_ROOT = RECOVERY_ROOT
        sys.argv = [str(original_driver), str(temporary_path)]
        original.main()
    finally:
        sys.argv = previous_argv
        if temporary_path is not None:
            temporary_path.unlink(missing_ok=True)


if __name__ == "__main__":
    main()

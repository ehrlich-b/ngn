#!/usr/bin/env python3
"""Build an owned H16 consumer, test private embedding, compare four depth4 FENs.

No training corpus or games. All generated source, bundle and logs go to --output.
The committed result is produced by --record and compared on every normal run.
"""
import argparse
import json
import os
from pathlib import Path
import sys
import tarfile
import tempfile

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO))
from scripts.wave2 import net_job as nj  # noqa: E402

FIX = Path(__file__).resolve().parent


def source_fixture(destination):
    # Copy a bounded, explicit owned source fixture. Original models and data
    # are not required: embed() adds the owned H16 file before the first build.
    paths = [REPO / name for name in ("go.mod", "main.go", "main_test.go", "build_info.go", "build_info_test.go")]
    for directory in ("engine", "internal/sliderray"):
        paths += [p for p in (REPO / directory).iterdir() if p.is_file() and p.suffix in (".go", ".s")]
    paths += [p for p in (REPO / "testdata/ngnn1").iterdir() if p.is_file()]
    with tarfile.open(destination, "w:gz") as archive:
        for path in sorted(paths):
            archive.add(path, arcname=str(path.relative_to(REPO)))
    return nj.artifact(destination)


def run(output, record=False):
    output.mkdir(parents=True, exist_ok=True)
    directory = Path(tempfile.mkdtemp(prefix="consumer-", dir=output))
    source = source_fixture(directory / "owned-source.tar.gz")
    selected = {"net": nj.artifact(FIX / "tiny-owned.nnue"), "source": source}
    result = {"verdict": "KEEP", "synthetic_fixture": True, "games_played": 0, "elo_claim": False}
    nj.write(directory / "result.json", result)
    cpus = sorted(os.sched_getaffinity(0))[:2]
    nj.adopt({"retention_bytes": 5242880}, directory, None, selected,
             {"synthetic_training_fixture": True, "selected": selected}, result, cpus)
    manifest = nj.read(directory / "adoption/manifest.json")
    identity = nj.read(directory / "adoption/identity.json")
    embedded = directory / "adoption/source/engine/nets" / ("candidate-" + selected["net"]["sha256"] + ".nnue")
    nj.check_artifact(manifest["bundle"])
    assert manifest["status"] == "READY" and manifest["private"] and not manifest["main_updated"]
    assert nj.sha(embedded) == selected["net"]["sha256"]
    assert nj.net_format(embedded) == (1, 16)
    result = {"version": 1, "fixture_only": True, "games_played": 0, "elo_claim": False,
              "net_sha256": selected["net"]["sha256"], "embedded_sha256": nj.sha(embedded),
              "format": "NGNN1", "hidden": 16, "identity": identity,
              "parity_passed": True, "startup_identity_passed": True, "candidate_status": manifest["status"]}
    nj.write(directory / "fixture-result.json", result)
    expected = FIX / "result.json"
    if record:
        nj.write(expected, result)
    elif result != nj.read(expected):
        raise nj.Invalid("real consumer fixture differs from committed result: " + str(directory))
    print(json.dumps({"status": "pass", "evidence": str(directory), "net_sha256": selected["net"]["sha256"],
                      "depth": 4, "fens": 4, "games": 0, "retained_bytes": nj.retained_bytes(directory)}))
    return directory


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--record", action="store_true")
    args = parser.parse_args()
    run(args.output.resolve(), args.record)

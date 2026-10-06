#!/usr/bin/env python3
"""NGN generation stages; orchestration uses only Python's standard library.

Run one foreground stage at a time on WSL. See genloop.md for commands and
resume semantics. Training subprocesses use the existing NumPy/PyTorch trainer.
"""

import argparse
from contextlib import contextmanager
import hashlib
import json
import math
import os
from pathlib import Path
import random
import re
import shlex
import shutil
import signal
import struct
import subprocess
import sys
import time

REPO = Path(__file__).resolve().parents[1]
GIB = 1024 ** 3


def read_json(path):
    return json.loads(Path(path).read_text())


def atomic_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    with temporary.open("w") as stream:
        json.dump(value, stream, indent=2, allow_nan=False)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)
    fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def sha256(path):
    digest = hashlib.sha256()
    with Path(path).open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def identity(path):
    path = Path(path).resolve()
    return {"path": str(path), "bytes": path.stat().st_size, "sha256": sha256(path)}


def verify(items):
    for item in items:
        if identity(item["path"]) != item:
            raise ValueError("Changed artifact: " + item["path"])


def records(root):
    return sum(p.stat().st_size // 32 for p in Path(root).rglob("*.ngnp"))


def stop_reason(count, target, elapsed, seconds, disk, minimum):
    # Safety takes priority even if the target was reached on the same tick.
    if any(free < minimum for free in disk.values()):
        return "disk-low"
    if target and count >= target:
        return "target"
    if seconds and elapsed >= seconds:
        return "time-limit"
    return None


def disk_free(root, windows_mount):
    """Fail closed if neither drvfs nor Windows PowerShell can report C:."""
    wsl = shutil.disk_usage(root).free
    mount = Path(windows_mount)
    if mount.is_mount():
        windows = shutil.disk_usage(mount).free
    else:
        result = subprocess.run(
            ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
             "[Console]::Write((Get-PSDrive -Name C).Free)"],
            capture_output=True, text=True, timeout=10, check=True)
        windows = int(result.stdout.strip())
    return {"wsl": wsl, "windows_c": windows}


def closed_shards(root):
    items = []
    for shard in sorted(Path(root).rglob("*.ngnp")):
        meta = read_json(str(shard) + ".json")
        item = identity(shard)
        if (meta["format"] != "NGNP1" or not meta["closed"] or
                meta["status"] not in ("complete", "interrupted") or item["bytes"] % 32 or
                meta["positions"] != item["bytes"] // 32 or
                meta["bytes"] != item["bytes"] or meta["sha256"] != item["sha256"]):
            raise ValueError("Unclosed or corrupt shard: " + str(shard))
        items.append(item)
    return items


def split_shards(items, fraction, seed):
    nonempty = sorted((item for item in items if item["bytes"]), key=lambda x: x["path"])
    if len(nonempty) < 2:
        raise ValueError("Need at least two nonempty closed shards for shard holdout")
    random.Random(seed).shuffle(nonempty)
    count = min(len(nonempty) - 1, max(1, math.ceil(len(nonempty) * fraction)))
    return {"validation": nonempty[:count], "training": nonempty[count:]}


def record_fen(raw):
    occupancy = struct.unpack_from("<Q", raw)[0]
    board, piece = {}, 0
    alphabet = "PNBRQKpnbrqk"
    for square in range(64):
        if occupancy & (1 << square):
            code = (raw[8 + piece // 2] >> (4 * (piece % 2))) & 15
            board[square] = alphabet[code]
            piece += 1
    ranks = []
    for rank in range(7, -1, -1):
        row, empty = "", 0
        for file in range(8):
            value = board.get(rank * 8 + file)
            if value:
                row += (str(empty) if empty else "") + value
                empty = 0
            else:
                empty += 1
        ranks.append(row + (str(empty) if empty else ""))
    return "/".join(ranks) + (" b" if raw[27] else " w") + " - - 0 1"


def parity_fens(items):
    total = sum(item["bytes"] // 32 for item in items)
    if total < 20:
        raise ValueError("Need 20 held-out records for parity")
    indices = [i * (total - 1) // 19 for i in range(20)]
    fens, offset = [], 0
    for item in items:
        count = item["bytes"] // 32
        with Path(item["path"]).open("rb") as stream:
            for index in indices:
                if offset <= index < offset + count:
                    stream.seek((index - offset) * 32)
                    fens.append(record_fen(stream.read(32)))
        offset += count
    return fens


def gate_result(log, games):
    """Reconcile the complete-pair footer; incomplete/operational runs fail."""
    if re.search(r"WATCHDOG|illegal|no-move|panic:|fatal error|info string error", log, re.I):
        raise ValueError("Operational error in gate log")
    wdls = re.findall(r"Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)", log)
    pentas = re.findall(r"Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\] over (\d+) pairs", log)
    flags = re.findall(r"Flag-outs \(lost on time\): new (\d+), base (\d+)\s+of (\d+) games", log)
    if len(wdls) != 1 or len(pentas) != 1 or len(flags) != 1:
        raise ValueError("Missing or duplicate gate footer")
    n, w, d, l = map(int, wdls[0])
    counts = list(map(int, pentas[0][:5]))
    pairs = int(pentas[0][5])
    if (n != games or w + d + l != n or pairs * 2 != n or sum(counts) != pairs or
            sum(i * c for i, c in enumerate(counts)) != 2 * w + d or
            tuple(map(int, flags[0])) != (0, 0, games)):
        raise ValueError("Incomplete pairs, inconsistent tally, or clock losses")
    # Same pair estimator as cmd/sprt, preserving precision in the manifest.
    mean = sum(i / 4 * c for i, c in enumerate(counts)) / pairs
    variance = max(0.001, sum((i / 4) ** 2 * c for i, c in enumerate(counts)) / pairs - mean ** 2)
    se = math.sqrt(variance / pairs)

    def elo(p):
        return -800.0 if p <= 1e-4 else 800.0 if p >= 1 - 1e-4 else 400 * math.log10(p / (1 - p))

    return {"games": n, "w": w, "d": d, "l": l, "pairs": pairs,
            "pentanomial": counts, "score": mean, "paired_elo": elo(mean),
            "paired_ci95": [elo(mean - 1.96 * se), elo(mean + 1.96 * se)],
            "method": "cmd/sprt pair-score normal CI; variance floor 0.001; tails clipped at +/-800"}


@contextmanager
def generation_lock(root):
    import fcntl
    root.mkdir(parents=True, exist_ok=True)
    with (root / ".lock").open("a") as stream:
        try:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ValueError("Another stage is running in this generation") from None
        yield


class Manifest:
    def __init__(self, root):
        self.root = Path(root)
        self.path = self.root / "manifest.json"
        self.data = read_json(self.path) if self.path.exists() else {
            "version": 1, "generation": self.root.name, "created": time.time(), "stages": {}}
        if self.data["version"] != 1:
            raise ValueError("Unsupported manifest version")

    def save(self):
        atomic_json(self.path, self.data)

    def event(self, **value):
        with (self.root / "events.jsonl").open("a") as stream:
            stream.write(json.dumps({"time": time.time(), **value}) + "\n")

    def begin(self, stage, config, resume):
        old = self.data["stages"].get(stage)
        if old:
            if old["config"] != config:
                raise ValueError("Stage options or input identities changed; use a new generation")
            if old["status"] == "complete":
                verify(old.get("artifacts", []))
                verify(old.get("sidecars", []))
                if old.get("split"):
                    verify([old["split"]])
                    split = read_json(old["split"]["path"])
                    verify(split["training"] + split["validation"])
                for step in old.get("steps", {}).values():
                    verify(step.get("artifacts", []))
                return False
            if not resume:
                raise ValueError("Existing unfinished stage; pass --resume")
            for attempt in old.get("attempts", []):
                pid = attempt.get("pid")
                if pid and Path(f"/proc/{pid}").exists():
                    # Never signal/adopt a process from a previous invocation.
                    raise ValueError(f"Recorded process {pid} still exists; inspect it before resuming")
        else:
            old = {"config": config, "steps": {}, "attempts": []}
            self.data["stages"][stage] = old
        self.stage, self.state = stage, old
        old.update(status="running", started=time.time(), error=None)
        self.save()
        self.event(stage=stage, action="begin", resume=resume)
        return True

    def finish(self, status="complete", **values):
        self.state.update(status=status, finished=time.time(), **values)
        self.save()
        self.event(stage=self.stage, action=status, **values)


def stop_child(child, driver=False):
    if child.poll() is not None:
        return
    # The datagen driver forwards TERM and waits for its own workers.
    if driver:
        child.send_signal(signal.SIGTERM)
    else:
        os.killpg(child.pid, signal.SIGTERM)
    try:
        child.wait(timeout=30)
    except subprocess.TimeoutExpired:
        os.killpg(child.pid, signal.SIGKILL)
        child.wait()
        raise RuntimeError("Child exceeded orderly shutdown deadline; outputs need inspection")


def run_command(manifest, name, command, input_text=None, timeout=None):
    step = manifest.state["steps"].setdefault(name, {"status": "pending", "attempts": []})
    if step["status"] == "complete":
        verify(step["artifacts"])
        return None
    folder = manifest.root / manifest.stage
    folder.mkdir(exist_ok=True)
    log = folder / f"{name}-{len(step['attempts']) + 1}.log"
    attempt = {"command": command, "log": str(log), "started": time.time()}
    step["attempts"].append(attempt)
    manifest.state["attempts"].append(attempt)
    if input_text is not None:
        source = log.with_suffix(".stdin")
        source.write_text(input_text)
        attempt["stdin"] = identity(source)
    manifest.event(stage=manifest.stage, step=name, action="command", **attempt)
    print(f"{name}: {shlex.join(command)}", flush=True)
    child = None
    try:
        with log.open("w") as output:
            child = subprocess.Popen(command, cwd=REPO, stdout=output, stderr=subprocess.STDOUT,
                                     stdin=subprocess.PIPE if input_text is not None else subprocess.DEVNULL,
                                     start_new_session=True, env={**os.environ, "GOMAXPROCS": "1",
                                                                 "OMP_NUM_THREADS": "2", "MKL_NUM_THREADS": "2",
                                                                 "OPENBLAS_NUM_THREADS": "2"})
            attempt["pid"] = child.pid
            step["status"] = "running"
            manifest.save()
            child.communicate(input_text.encode() if input_text is not None else None, timeout=timeout)
            if child.returncode:
                raise RuntimeError(f"{name} exited {child.returncode}; see {log}")
    finally:
        if child is not None:
            stop_child(child)
        attempt.pop("pid", None)
        attempt.update(finished=time.time(), returncode=child.returncode if child is not None else None)
        manifest.save()
    return log


def complete_step(manifest, name, paths):
    step = manifest.state["steps"][name]
    step.update(status="complete", artifacts=[identity(p) for p in paths])
    manifest.save()
    manifest.event(stage=manifest.stage, step=name, action="complete")


def datagen(manifest, args):
    root = manifest.root / "data"
    root.mkdir(exist_ok=True)
    existing = closed_shards(root)  # Never count unclosed crash fragments toward a resume.
    verify(manifest.state.get("artifacts", []))
    verify(manifest.state.get("sidecars", []))
    if args.positions and records(root) >= args.positions:
        manifest.finish(artifacts=existing, positions=records(root), stop="target")
        return
    disk = disk_free(manifest.root, args.windows_mount)
    if min(disk.values()) < args.min_free_gib * GIB:
        raise ValueError("Disk guard refused start: " + json.dumps(disk))
    index = len(manifest.state["attempts"])
    if args.seed + (index + 1) * args.workers > 9223372036854775807:
        raise ValueError("Resume seed range exhausted")
    command = ["bash", str(REPO / "scripts/datagen_run.sh"), "-out", str(root),
               "-workers", str(args.workers), "-nodes", str(args.nodes),
               "-seed", str(args.seed + index * args.workers), "-bin", str(args.binary)]
    if args.net:
        command += ["-net", str(args.net)]
    folder = manifest.root / "datagen"
    folder.mkdir(exist_ok=True)
    log = folder / f"driver-{index + 1}.log"
    attempt = {"command": command, "log": str(log), "started": time.time()}
    manifest.state["attempts"].append(attempt)
    manifest.event(stage="datagen", action="command", **attempt)
    child, reason = None, None
    started, initial = time.monotonic(), records(root)
    try:
        with log.open("w") as output:
            child = subprocess.Popen(command, cwd=REPO, stdout=output, stderr=subprocess.STDOUT,
                                     stdin=subprocess.DEVNULL, start_new_session=True)
            attempt["pid"] = child.pid
            manifest.save()
            print("datagen: " + shlex.join(command), flush=True)
            while child.poll() is None:
                count = records(root)
                elapsed = time.monotonic() - started
                disk = disk_free(manifest.root, args.windows_mount)
                reason = stop_reason(count, args.positions, elapsed, args.seconds,
                                     disk, args.min_free_gib * GIB)
                manifest.state["progress"] = {"positions": count, "seconds": elapsed,
                                              "rate": (count - initial) / max(elapsed, 0.001), "disk": disk}
                manifest.save()
                if reason:
                    child.send_signal(signal.SIGTERM)
                    child.wait(timeout=30)
                    break
                try:
                    child.wait(timeout=args.poll_seconds)
                except subprocess.TimeoutExpired:
                    pass
    finally:
        if child is not None:
            stop_child(child, driver=True)
        attempt.pop("pid", None)
        attempt.update(finished=time.time(), returncode=child.returncode if child is not None else None)
        manifest.save()
    items = closed_shards(root)
    count = records(root)
    if child.returncode:
        raise RuntimeError(f"Datagen driver failed; see {log}")
    status = "complete" if reason == "target" or (reason == "time-limit" and not args.positions) else "interrupted"
    manifest.finish(status, artifacts=items, positions=count, stop=reason or "driver-exit",
                    sidecars=[identity(x["path"] + ".json") for x in items])
    if status != "complete":
        raise RuntimeError("Datagen stopped before target: " + str(reason))


def train(manifest, args):
    data = manifest.data["stages"].get("datagen", {})
    if data.get("status") != "complete":
        raise ValueError("Complete datagen before training")
    verify(data["artifacts"])
    verify(data.get("sidecars", []))
    folder = manifest.root / "train"
    folder.mkdir(exist_ok=True)
    split_path = folder / "split.json"
    if split_path.exists():
        split = read_json(split_path)
        verify(split["training"] + split["validation"])
    else:
        split = split_shards(data["artifacts"], args.holdout, args.seed)
        atomic_json(split_path, split)
    manifest.state["split"] = identity(split_path)
    manifest.save()
    out = folder / "epochs"
    command = [str(args.python), "-u", "-m", "trainer.train",
               "--data", *[x["path"] for x in split["training"]],
               "--validation-data", *[x["path"] for x in split["validation"]],
               "--out", str(out), "--epochs", str(args.epochs), "--hidden", str(args.hidden),
               "--buckets", str(args.buckets), "--batch", str(args.batch), "--lr", str(args.lr),
               "--seed", str(args.seed), "--threads", str(args.threads), "--device", args.device]
    checkpoint = out / f"epoch-{args.epochs:04d}.pt"
    # Epoch checkpoints are atomic; a failed epoch resumes from the preceding one.
    saved = sorted(out.glob("epoch-*.pt"))
    if args.resume and saved:
        command += ["--resume", str(saved[-1])]
    log = run_command(manifest, "fit", command)
    if log:
        complete_step(manifest, "fit", [checkpoint, out / "history.json", split_path])
    fens = parity_fens(split["validation"])
    fen_file = folder / "parity.fens"
    fen_file.write_text("\n".join(fens) + "\n")
    exports = []
    for version in ([1, 2] if args.buckets == 1 else [2]):
        net = folder / f"network-ngnn{version}.nnue"
        name = f"export{version}"
        log = run_command(manifest, name, [str(args.python), "-m", "trainer.export",
                                         str(checkpoint), str(net), "--format", str(version)])
        if log:
            complete_step(manifest, name, [net])
        ref_name, engine_name = f"refeval{version}", f"engine-eval{version}"
        ref_log = run_command(manifest, ref_name, [str(args.python), "-m", "trainer.refeval",
                                                  str(net), "--fen-file", str(fen_file)], timeout=60)
        if ref_log:
            complete_step(manifest, ref_name, [ref_log, fen_file])
        else:
            ref_log = Path(manifest.state["steps"][ref_name]["attempts"][-1]["log"])
        script = ("uci\nsetoption name Hash value 64\nsetoption name Threads value 1\n"
                  "setoption name OwnBook value false\n" + f"setoption name EvalFile value {net}\n"
                  "setoption name UseNNUE value true\nisready\n")
        script += "".join(f"position fen {fen}\neval\n" for fen in fens) + "quit\n"
        engine_log = run_command(manifest, engine_name, [str(args.engine)], script, timeout=60)
        if engine_log is None:
            engine_log = Path(manifest.state["steps"][engine_name]["attempts"][-1]["log"])
        text = engine_log.read_text()
        actual = [int(x) for x in re.findall(rf"eval backend ngnn{version} score_cp (-?\d+)", text)]
        expected = [int(line.split("\t")[0]) for line in ref_log.read_text().splitlines() if "\t" in line]
        if len(expected) != 20 or actual != expected or "info string error" in text:
            raise ValueError(f"NGNN{version} reference/engine parity failed; see {engine_log}")
        complete_step(manifest, engine_name, [engine_log])
        exports.append(identity(net))
    manifest.finish(artifacts=exports, exports=exports, parity_positions=20,
                    history=read_json(out / "history.json"))


def gate(manifest, args):
    command = [str(args.sprt), "-new", str(args.engine), "-base", str(args.base_engine),
               "-tc", "10+0.1", "-concurrency", str(args.concurrency), "-lowpower=false",
               "-maxgames", str(args.games), "-mingames", str(args.games + 1),
               "-maxmoves", str(args.maxplies), "-openings", str(args.openings)]
    for side, net, options in [("new", args.candidate_net, args.newoption),
                               ("base", args.base_net, args.baseoption)]:
        settings = ["Hash=64", "Threads=1", "OwnBook=false"] + options
        # Evaluator selection comes last, so extra options cannot silently override it.
        settings += ([f"EvalFile={net}", "UseNNUE=true"] if net else ["UseNNUE=false"])
        for setting in settings:
            command += [f"-{side}option", setting]
    log = run_command(manifest, "match", command)
    if log is None:
        log = Path(manifest.state["steps"]["match"]["attempts"][-1]["log"])
    result = gate_result(log.read_text(), args.games)
    complete_step(manifest, "match", [log])
    manifest.finish(artifacts=[identity(log)], result=result)
    print(json.dumps(result), flush=True)


def status(root):
    manifest = Manifest(root)
    output = {"generation": manifest.data["generation"], "stages": {}}
    output["positions"] = records(root / "data")
    for name, state in manifest.data["stages"].items():
        item = {"status": state["status"]}
        if name == "datagen":
            progress = state.get("progress", {})
            rate = progress.get("rate", 0)
            target = state["config"]["positions"]
            item.update(rate_positions_per_second=rate, target=target,
                        eta_seconds=max(0, target - output["positions"]) / rate if target and rate else None,
                        progress_updated=progress.get("seconds"), rate_is_last_saved_sample=True)
        if name == "train":
            history = root / "train/epochs/history.json"
            item["epochs"] = read_json(history) if history.exists() else []
        if name == "gate":
            logs = list((root / "gate").glob("match-*.log"))
            if logs:
                log = max(logs, key=lambda p: p.stat().st_mtime)
                with log.open("rb") as stream:
                    stream.seek(max(0, log.stat().st_size - 8192))
                    rows = re.findall(r"^G\d+.*$", stream.read().decode(errors="replace"), re.M)
                item["last_game"] = rows[-1] if rows else None
        if state.get("error"):
            item["error"] = state["error"]
        output["stages"][name] = item
    if root.exists():
        try:
            output["disk_free_bytes"] = disk_free(root, manifest.data.get("windows_mount", "/mnt/c"))
        except (OSError, ValueError, subprocess.SubprocessError) as error:
            output["disk_error"] = str(error)
    print(json.dumps(output, indent=2))


def parser():
    cli = argparse.ArgumentParser(description=__doc__)
    stages = cli.add_subparsers(dest="stage", required=True)
    for name in ("datagen", "train", "gate", "status"):
        p = stages.add_parser(name)
        p.add_argument("--generation", type=Path, required=True)
        if name == "status":
            continue
        p.add_argument("--resume", action="store_true")
        p.add_argument("--dry-run", action="store_true", help="Print a plan; no writes or subprocesses")
        if name == "datagen":
            p.add_argument("--binary", type=Path, default=REPO / "build/datagen")
            p.add_argument("--workers", type=int, default=2)
            p.add_argument("--nodes", type=int, default=5000)
            p.add_argument("--net", "-net", type=Path)
            p.add_argument("--positions", type=int, default=0, help="Target summed records; 0 = timed generation")
            p.add_argument("--seconds", type=float, default=0, help="Foreground time limit per attempt")
            p.add_argument("--seed", type=int, default=2026100600)
            p.add_argument("--min-free-gib", type=float, default=6)
            p.add_argument("--windows-mount", default="/mnt/c")
            p.add_argument("--poll-seconds", type=float, default=2, help="Disk/target reaction interval")
        elif name == "train":
            p.add_argument("--python", type=Path, default=Path(sys.executable))
            p.add_argument("--engine", type=Path, default=REPO / "build/ngn")
            p.add_argument("--holdout", type=float, default=0.1)
            p.add_argument("--seed", type=int, default=20261006)
            p.add_argument("--epochs", type=int, default=10)
            p.add_argument("--hidden", type=int, default=256)
            p.add_argument("--buckets", type=int, choices=(1, 2, 4, 8), default=1)
            p.add_argument("--batch", type=int, default=16384)
            p.add_argument("--lr", type=float, default=0.001)
            p.add_argument("--device", default="cpu", help="Use cuda only with separate job authorization")
            p.add_argument("--threads", type=int, default=2)
        else:
            p.add_argument("--engine", type=Path, default=REPO / "build/ngn")
            p.add_argument("--base-engine", type=Path)
            p.add_argument("--sprt", type=Path, default=REPO / "build/sprt")
            p.add_argument("--candidate-net", type=Path)
            p.add_argument("--base-net", type=Path, help="Omit for HCE")
            p.add_argument("--openings", type=Path, required=True)
            p.add_argument("--games", type=int, default=160)
            p.add_argument("--concurrency", type=int, default=2)
            p.add_argument("--maxplies", type=int, default=400)
            p.add_argument("--newoption", action="append", default=[])
            p.add_argument("--baseoption", action="append", default=[])
    return cli


def main(argv=None):
    args = parser().parse_args(argv)
    for key, value in vars(args).items():
        if isinstance(value, Path):
            # A venv interpreter is commonly a symlink to the system Python.
            # Its lexical path selects pyvenv.cfg and must survive normalization.
            setattr(args, key, value.absolute() if key == "python" else value.resolve())
    if args.stage == "status":
        status(args.generation)
        return 0
    config = {key: str(value) if isinstance(value, Path) else value for key, value in vars(args).items()
              if key not in ("stage", "generation", "resume", "dry_run")}
    if args.stage == "datagen":
        if (args.workers < 1 or args.workers > 256 or args.nodes < 1 or args.positions < 0 or
                args.seconds < 0 or not (args.positions or args.seconds) or args.seed < 0 or
                args.seed > 9223372036854775807 - 256 or args.poll_seconds <= 0 or args.min_free_gib <= 0):
            raise ValueError("Invalid datagen bounds")
    if args.stage == "train" and (not 0 < args.holdout < 1 or args.epochs < 1 or
                                 args.threads < 1 or args.batch < 1 or args.lr <= 0):
        raise ValueError("Invalid training bounds")
    if args.stage == "gate":
        args.base_engine = args.base_engine or args.engine
        if args.candidate_net is None:
            state = Manifest(args.generation).data["stages"].get("train", {})
            if state.get("status") == "complete":
                args.candidate_net = Path(state["exports"][-1]["path"])
            elif not args.dry_run:
                raise ValueError("Pass --candidate-net or complete train first")
        config.update(base_engine=str(args.base_engine), candidate_net=str(args.candidate_net))
        if args.games < 2 or args.games % 2 or args.concurrency < 1 or args.maxplies < 1:
            raise ValueError("Gate needs positive even games and positive concurrency/ply cap")
        for option in args.newoption + args.baseoption:
            if "=" not in option or any(c in option for c in "\r\n"):
                raise ValueError("UCI options must be single-line NAME=VALUE")
    if args.dry_run:
        print(json.dumps({"stage": args.stage, "generation": str(args.generation), "options": config,
                          "steps": {"datagen": ["disk guard", "driver", "SIGTERM on target/time/disk", "closed-shard hashes"],
                                    "train": ["shard holdout", "fit/resume checkpoint", "export NGNN1/NGNN2", "20-position reference/engine parity"],
                                    "gate": ["fixed paired 10+0.1 match", "verify all games/pairs and zero operational errors", "W/D/L and paired CI"]}[args.stage]}, indent=2))
        return 0
    if sys.platform != "linux":
        raise ValueError("Execute stages only on the authorized WSL host")
    # Hash inputs on every invocation, so a completed stage cannot mask changes.
    inputs = [REPO / "scripts/genloop.py"]
    if args.stage == "datagen":
        inputs += [REPO / "scripts/datagen_run.sh", args.binary] + ([args.net] if args.net else [])
    elif args.stage == "train":
        inputs += [args.engine, *sorted((REPO / "trainer").glob("*.py"))]
    else:
        inputs += [args.engine, args.base_engine, args.sprt, args.openings, args.candidate_net]
        inputs += [args.base_net] if args.base_net else []
    config["inputs"] = [identity(path) for path in inputs]
    with generation_lock(args.generation):
        manifest = Manifest(args.generation)
        if not manifest.begin(args.stage, config, args.resume):
            print(args.stage + ": already complete; hashes verified")
            return 0
        manifest.state.setdefault("source_commit", subprocess.run(
            ["git", "rev-parse", "HEAD"], cwd=REPO, capture_output=True,
            text=True, check=True).stdout.strip())
        if args.stage == "datagen":
            manifest.data["windows_mount"] = args.windows_mount
        try:
            {"datagen": datagen, "train": train, "gate": gate}[args.stage](manifest, args)
        except BaseException as error:
            manifest.finish("interrupted" if isinstance(error, KeyboardInterrupt) else "failed", error=str(error))
            raise
    return 0


if __name__ == "__main__":
    def interrupt(_signum, _frame):
        raise KeyboardInterrupt("Received termination signal")
    signal.signal(signal.SIGTERM, interrupt)
    try:
        sys.exit(main())
    except (ValueError, RuntimeError, OSError, subprocess.SubprocessError, KeyboardInterrupt) as error:
        print("genloop: " + str(error), file=sys.stderr)
        sys.exit(1)

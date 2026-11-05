#!/usr/bin/env python3
"""Durable, receipt-driven training and fixed net gates. See net_job.schema.json.

The state directory belongs to one experiment. A detached supervisor waits only
on reservations and receipts; it never reads live match scores or retries work.
"""
from __future__ import annotations

import argparse
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import selectors
import shutil
import socket
import struct
import subprocess
import sys
import tarfile
import time
import zlib

from scripts import campaign_queue as campaign

BASE_NET = "2dcb6d936c0275927cb9e881f42e22558e618ad7e6fd998df6c4e5046d42c443"
TERMINAL = {"SUCCEEDED", "FAILED", "BLOCKED", "READY"}
FENS = (
    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    "r3k2r/ppp2ppp/2n1bn2/3qp3/3P4/2N1PN2/PPP2PPP/R2QKB1R w KQkq - 4 8",
    "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
    "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1",
)


class Invalid(ValueError):
    pass


def sha(path):
    h = hashlib.sha256()
    with Path(path).open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def identity(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"),
                                     allow_nan=False).encode()).hexdigest()


def read(path):
    def unique(pairs):
        out = {}
        for key, value in pairs:
            if key in out:
                raise Invalid("duplicate JSON key: " + key)
            out[key] = value
        return out
    return json.loads(Path(path).read_text(), object_pairs_hook=unique)


def write(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    with temporary.open("w") as stream:
        stream.write(json.dumps(value, sort_keys=True, indent=2, allow_nan=False) + "\n")
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)
    fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def artifact(path):
    return {"path": str(Path(path).resolve()), "sha256": sha(path)}


def check_artifact(value):
    if set(value) != {"path", "sha256"} or not Path(value["path"]).is_absolute():
        raise Invalid("artifact requires absolute path and sha256")
    if not re.fullmatch("[0-9a-f]{64}", value["sha256"]):
        raise Invalid("invalid artifact digest")
    path = Path(value["path"])
    if not path.is_file() or sha(path) != value["sha256"]:
        raise Invalid("input hash mismatch: " + str(path))
    return path


@contextmanager
def lock(path):
    with Path(path).open("a+") as stream:
        try:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            yield False
        else:
            try:
                yield True
            finally:
                fcntl.flock(stream, fcntl.LOCK_UN)


def environment(extra=None):
    env = {k: v for k, v in os.environ.items()
           if not re.search("credential|token|secret|password|api.?key|access.?key|authorization|cookie|private.?key", k, re.I)}
    env.update(GOMAXPROCS="2", GOFLAGS="-p=1", OMP_NUM_THREADS="2", MKL_NUM_THREADS="2",
               OPENBLAS_NUM_THREADS="2", NUMEXPR_NUM_THREADS="2", CUDA_VISIBLE_DEVICES="",
               GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod",
               GOPATH="/work/cache/go")
    if extra:
        env.update(extra)
    return env


def execute(argv, cwd, log, cpus, extra=None):
    """Bound CPU affinity, clear credentials and retain combined output."""
    with Path(log).open("w") as output:
        return subprocess.run(["nice", "-n", "10", "taskset", "-c", ",".join(map(str, cpus)),
                               *map(str, argv)], cwd=cwd, env=environment(extra),
                              stdout=output, stderr=subprocess.STDOUT, check=False).returncode


def unpack(source, target):
    """Only owned regular source files, never links or archive path escapes."""
    target = Path(target)
    target.mkdir()
    with tarfile.open(check_artifact(source)) as archive:
        members = archive.getmembers()
        for member in members:
            p = Path(member.name)
            if p.is_absolute() or ".." in p.parts or not (member.isfile() or member.isdir()):
                raise Invalid("unsafe source member: " + member.name)
        archive.extractall(target, members=members, filter="data")
    return target


def files(root):
    return {p.relative_to(root).as_posix(): sha(p) for p in Path(root).rglob("*") if p.is_file()}


def matching_sources(base, new, exception):
    a, b = files(base), files(new)
    changes = {key: [a.get(key), b.get(key)] for key in a.keys() | b.keys() if a.get(key) != b.get(key)}
    if not changes:
        if exception:
            raise Invalid("evaluator exception declares no change")
        return changes
    if not exception or sorted(changes) != sorted(exception["files"]) or identity(changes) != exception["sha256"]:
        raise Invalid("source mismatch outside selected evaluator")
    if any(not re.fullmatch(r"engine/(?:ngnn[^/]*|worker_evaluator)\.go", name)
           or name.endswith("_test.go") for name in changes):
        raise Invalid("exception is not an evaluator implementation")
    return changes


def frozen_book(book, claims, destination):
    def entries(path):
        lines = [" ".join(line.split()) for line in path.read_text().splitlines() if line.strip()]
        if not lines or any(len(line.split()) != 6 or "/" not in line for line in lines):
            raise Invalid("books require one full FEN per line; export claim positions as FENs")
        # FEN positions remove move-order ambiguity and ignore move counters.
        return {" ".join(line.split()[:4]) for line in lines}
    development = entries(check_artifact(book))
    if not claims:
        raise Invalid("claim book inventory is required")
    for claim in claims:
        for a in entries(check_artifact(claim)):
            if a in development:
                raise Invalid("development book overlaps claim data")
    shutil.copyfile(book["path"], destination)
    return artifact(destination)


def net_format(path):
    data = Path(path).read_bytes()
    if len(data) < 28 or zlib.crc32(data[:-4]) != struct.unpack("<I", data[-4:])[0]:
        raise Invalid("network checksum")
    magic, version, hidden = struct.unpack("<4sII", data[:12])
    if {1: b"NGNN", 2: b"NGN2", 3: b"NGN3"}.get(version) != magic:
        raise Invalid("network format")
    if not 16 <= hidden <= 2048 or hidden % 16:
        raise Invalid("network hidden dimension")
    kb, nb, header, offset = 1, 1, 24, 12
    if version == 2:
        if len(data) < 32:
            raise Invalid("network header")
        nb, header, offset = struct.unpack_from("<I", data, 12)[0], 28, 16
    elif version == 3:
        if len(data) < 100:
            raise Invalid("network header")
        kb, nb = struct.unpack_from("<II", data, 12)
        header, offset = 96, 20
        if not 1 <= kb <= 8 or any(bucket >= kb for bucket in data[32:96]):
            raise Invalid("network king buckets")
    if nb not in (1, 2, 4, 8) or struct.unpack_from("<III", data, offset) != (255, 64, 400):
        raise Invalid("network quantization/buckets")
    if len(data) != header + 2 * (kb * 768 * hidden + hidden + nb * 2 * hidden) + nb * 4 + 4:
        raise Invalid("network payload length")
    return version, hidden


def validate(job):
    common = {"version", "id", "stage", "inputs"}
    stage = job.get("stage")
    required = {"train": {"argv"}, "await-result": {"training"},
                "gate": {"training", "book", "claim_books", "retention_bytes"}}
    optional = {"prerequisite"} | ({"evaluator_exception"} if stage == "gate" else set())
    if stage not in required or not common | required[stage] <= job.keys():
        raise Invalid("missing job fields")
    if job.keys() - (common | required[stage] | optional):
        raise Invalid("unknown job fields")
    if job["version"] != 1 or not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,79}", job["id"]):
        raise Invalid("invalid job version/id")
    if stage == "train" and (not job["argv"] or not all(isinstance(x, str) and x for x in job["argv"])):
        raise Invalid("training requires an argv array")
    if stage == "train" and "source" not in job["inputs"]:
        raise Invalid("training source required")
    if stage == "gate" and not {"base_source", "base_net"} <= job["inputs"].keys():
        raise Invalid("baseline source and net required")
    if stage != "train":
        training = job["training"]
        if set(training) != {"id", "input_sha256", "export", "net_sha256", "source_sha256"}:
            raise Invalid("training selection fields")
        if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,79}", training["id"]):
            raise Invalid("invalid training id")
        for key in ("input_sha256", "net_sha256", "source_sha256"):
            if not re.fullmatch("[0-9a-f]{64}", training[key]):
                raise Invalid("invalid training digest")
    if stage == "gate" and (type(job["retention_bytes"]) is not int or job["retention_bytes"] < 0):
        raise Invalid("invalid retention budget")


def resolve_training(root, selection):
    path = root / selection["id"] / "receipt.json"
    if not path.exists():
        raise Invalid("missing training receipt")
    receipt = read(path)
    if receipt["input_sha256"] != selection["input_sha256"] or receipt["stage"] != "train":
        raise Invalid("training receipt identity mismatch")
    if receipt["status"] in ("WAITING", "RUNNING"):
        return None
    if receipt["status"] != "SUCCEEDED":
        raise Invalid("failed training predecessor")
    selected = receipt["exports"].get(selection["export"])
    if selected is None:
        raise Invalid("selected export missing")
    for kind in ("net", "source"):
        check_artifact(selected[kind])
        if selected[kind]["sha256"] != selection[kind + "_sha256"]:
            raise Invalid("selected " + kind + " hash mismatch")
    return {"receipt": artifact(path), "selected": selected}


def admission(root, job, key):
    path = root / "reservations.json"
    if not path.exists():
        return None
    value = read(path)
    entry = value.get("jobs", {}).get(job["id"])
    if not entry or entry.get("status") != "ADMITTED":
        return None
    if value.get("version") != 1 or entry.get("input_sha256") != key or entry.get("stage") != job["stage"]:
        raise Invalid("reservation identity mismatch")
    cpus = entry.get("cpus", [])
    if not 1 <= len(cpus) <= 2 or any(type(c) is not int for c in cpus) or len(set(cpus)) != len(cpus):
        raise Invalid("reservation requires one or two distinct CPU cores")
    if not set(cpus) <= os.sched_getaffinity(0):
        raise Invalid("reserved CPUs unavailable")
    if not entry.get("ticket"):
        raise Invalid("reservation ticket required")
    return {"receipt": artifact(path), "entry": entry}


def train(job, key, directory, cpus):
    source = unpack(job["inputs"]["source"], directory / "source")
    output = directory / "training-result.json"
    rc = execute(job["argv"], source, directory / "train.log", cpus,
                 {"NET_JOB_RESULT": str(output), "NET_JOB_INPUT_SHA256": key})
    if rc or not output.exists():
        raise Invalid(f"training failed or omitted terminal receipt (exit {rc})")
    result = read(output)
    if result.get("version") != 1 or result.get("status") != "SUCCEEDED" or result.get("input_sha256") != key:
        raise Invalid("training did not explicitly certify successful completion")
    exports = result.get("exports", {})
    if not exports:
        raise Invalid("training exported no nets")
    for selected in exports.values():
        for kind in ("net", "source"):
            check_artifact(selected[kind])
        if selected["source"] != job["inputs"]["source"]:
            raise Invalid("training export source mismatch")
        net_format(selected["net"]["path"])
    return {"exports": exports, "training_receipt": artifact(output)}


def control_identity(job, selected, book, binaries, cpus):
    return identity({"candidate": selected, "baseline": job["inputs"], "book": book["sha256"],
                     "binaries": {k: v["sha256"] for k, v in binaries.items()}, "cpus": cpus,
                     "machine": [socket.gethostname(), os.uname().machine], "tc": "10+0.1",
                     "options": ["Threads=1", "Hash=64", "OwnBook=false", "UseNNUE=true"],
                     "concurrency": 6, "no_retries": True})


def campaign_job(job, directory, binaries, selected, book, key, cpus, aa=False):
    base = str(check_artifact(job["inputs"]["base_net"]))
    net = str(check_artifact(selected["net"]))
    options = ["Threads=1", "Hash=64", "OwnBook=false", "UseNNUE=true"]
    artifacts = {v["path"]: v["sha256"] for v in
                 [*binaries.values(), selected["net"], job["inputs"]["base_net"], book]}
    return {"name": job["id"] + ("-aa" if aa else "-net"), "class": "aa" if aa else "net",
            "output": str(directory / ("aa" if aa else "net")), "control_key": key,
            "artifacts": artifacts, "cpus": ",".join(map(str, cpus)),
            "harness": binaries["harness"]["path"],
            "candidate": binaries["base" if aa else "candidate"]["path"], "base": binaries["base"]["path"],
            "tc": "10+0.1", "concurrency": 6, "min_games": 200 if aa else 1000,
            "max_games": 200 if aa else 1000, "book": book["path"], "cwd": str(directory),
            "candidate_options": options + ["EvalFile=" + (base if aa else net)],
            "base_options": options + ["EvalFile=" + base], "no_retries": True, "strict_pairs": True}


def build(source, output, target, log, cpus):
    if execute(["go", "build", "-p", "1", "-o", str(output), target], source, log, cpus):
        raise Invalid("source build failed: " + str(log))
    return artifact(output)


def reusable_control(root, key):
    path = root / "controls" / (key + ".json")
    if not path.exists():
        return None
    control = read(path)
    if control["key"] != key:
        raise Invalid("control identity mismatch")
    result = read(check_artifact(control["result"]))
    if (result.get("verdict") != "AA_PASS" or result.get("faults") or
            result.get("games") != 200 or result.get("pairs") != 100):
        raise Invalid("invalid cached A/A control")
    # Bind all the original game evidence, not just an editable summary.
    for value in control["evidence"]:
        check_artifact(value)
    return control


def strict_result(result, games, verdict):
    return (result.get("verdict") == verdict and not result.get("faults") and
            result.get("games") == games and result.get("pairs") == games // 2 and
            result.get("orphan_games") == 0 and
            (verdict != "KEEP" or (result["printed_elo_ci95"][1] > 0 and result["exact_elo_ci95"][1] > 0)))


def gate(job, directory, root, resolved, cpus):
    selected = resolved["selected"]
    base_net = check_artifact(job["inputs"]["base_net"])
    if sha(base_net) != BASE_NET or net_format(base_net) != (3, 768):
        raise Invalid("baseline must be exact current-main H768")
    book = frozen_book(job["book"], job["claim_books"], directory / "development-book.txt")
    base = unpack(job["inputs"]["base_source"], directory / "base-source")
    source = unpack(selected["source"], directory / "gated-source")
    changes = matching_sources(base, source, job.get("evaluator_exception"))
    binaries = {"base": build(base, directory / "base", ".", directory / "base-build.log", cpus),
                "candidate": build(source, directory / "candidate-evalfile", ".", directory / "candidate-build.log", cpus),
                "harness": build(base, directory / "sprt", "./cmd/sprt", directory / "harness-build.log", cpus)}
    key = control_identity(job, selected, book, binaries, cpus)
    write(directory / "gate-inputs.json", {"training": resolved, "book": book, "source_changes": changes,
                                           "binaries": binaries, "control_key": key})
    registry = directory / "campaign-controls.json"
    status = directory / "campaign-status.jsonl"
    control = reusable_control(root, key)
    if control is None:
        aa = campaign_job(job, directory, binaries, selected, book, key, cpus, aa=True)
        result = campaign.run(aa, registry, status)
        if not strict_result(result, 200, "AA_PASS"):
            raise Invalid("fixed 200-game A/A did not pass")
        control = {"key": key, "result": artifact(directory / "aa/result.json"),
                   "evidence": [artifact(directory / "aa" / name)
                                for name in ("manifest.json", "match.log", "games.pgn")]}
        write(root / "controls" / (key + ".json"), control)
    write(registry, {key: {"result": control["result"]["path"], "harness_sha256": binaries["harness"]["sha256"]}})
    net_job = campaign_job(job, directory, binaries, selected, book, key, cpus)
    result = campaign.run(net_job, registry, status)
    write(directory / "result.json", result)
    if strict_result(result, 1000, "KEEP"):
        adopt(job, directory, source, selected, resolved, result, cpus)
    elif result.get("verdict") == "KEEP":
        raise Invalid("invalid KEEP receipt")
    elif result.get("verdict") != "SHELVED":
        raise Invalid("net gate has operational faults or an incomplete sample")
    return {"result": artifact(directory / "result.json"), "verdict": result["verdict"], "control": control}


class UCI:
    """One owned engine process with bounded reads and no handshake retries."""
    def __init__(self, binary, cpus):
        self.child = subprocess.Popen(["nice", "-n", "10", "taskset", "-c", ",".join(map(str, cpus)), str(binary)],
                                      stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                      env=environment(), bufsize=0)
        self.pending = b""
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.child.stdout, selectors.EVENT_READ)

    def send(self, line):
        self.child.stdin.write((line + "\n").encode())

    def wait(self, token):
        lines, deadline = [], time.monotonic() + 20
        while True:
            while b"\n" in self.pending:
                line, self.pending = self.pending.split(b"\n", 1)
                text = line.decode().strip()
                lines.append(text)
                if re.search(r"\berror\b|failed|panic:|fatal", text, re.I):
                    raise Invalid("UCI fault: " + text)
                if text.split()[:1] == [token]:
                    return lines
            remaining = deadline - time.monotonic()
            if remaining <= 0 or not self.selector.select(remaining):
                raise Invalid("UCI timeout: " + token)
            data = os.read(self.child.stdout.fileno(), 65536)
            if not data:
                raise Invalid("UCI exited before " + token)
            self.pending += data

    def close(self):
        try:
            self.send("quit")
            self.child.wait(timeout=3)
        except (BrokenPipeError, subprocess.TimeoutExpired):
            self.child.kill()
            self.child.wait()
        finally:
            self.child.stdin.close()
            self.child.stdout.close()
            self.selector.close()


def consumer_identity(binary, net, cpus, fens=FENS):
    reports = []
    for external in (False, True):
        rows = []
        for fen in fens:
            engine = UCI(binary, cpus)
            try:
                engine.send("uci")
                engine.wait("uciok")
                for name, value in (("Threads", "1"), ("Hash", "64"), ("OwnBook", "false"), ("UseNNUE", "true")):
                    engine.send(f"setoption name {name} value {value}")
                if external:
                    engine.send("setoption name EvalFile value " + str(net))
                engine.send("isready")
                engine.wait("readyok")
                engine.send("position fen " + fen)
                engine.send("eval")
                engine.send("isready")
                evaluation = [line for line in engine.wait("readyok") if line.startswith("info string eval backend ")]
                if len(evaluation) != 1:
                    raise Invalid("missing explicit evaluator identity")
                engine.send("go depth 4")
                lines = engine.wait("bestmove")
                final = [line.split() for line in lines if line.startswith("info depth 4 ")]
                if not final:
                    raise Invalid("missing completed depth4 search")
                fields = final[-1]
                pick = lambda key: fields[fields.index(key) + 1]
                rows.append({"fen": fen, "eval": evaluation[0], "depth": pick("depth"), "nodes": pick("nodes"),
                             "score": fields[fields.index("score") + 1:fields.index("score") + 3],
                             "pv": fields[fields.index("pv") + 1:], "bestmove": lines[-1]})
            finally:
                engine.close()
        reports.append(rows)
    if reports[0] != reports[1]:
        raise Invalid("embedded/default versus EvalFile depth4 mismatch")
    return {"depth": 4, "cases": len(fens), "identical": True, "rows": reports[0]}


def embed(source, net):
    """Update only a private source snapshot; the working checkout is untouched."""
    net = check_artifact(net)
    version, hidden = net_format(net)
    digest = sha(net)
    name = "candidate-" + digest + ".nnue"
    target = source / "engine/nets" / name
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(net, target)
    default = source / "engine/default_net.go"
    text, count = re.subn(r'const DefaultNetSHA256 = "[0-9a-f]{64}"',
                          'const DefaultNetSHA256 = "' + digest + '"', default.read_text())
    text, embeds = re.subn(r"//go:embed nets/[^\n]+", "//go:embed nets/" + name, text)
    if count != 1 or embeds != 1:
        raise Invalid("unsupported embedded net source")
    default.write_text(text)
    # Existing startup identity is format-specific; keep the expected backend
    # in sync in the private candidate, including browser digest assertions.
    startup = source / "main_test.go"
    if startup.exists():
        startup.write_text(startup.read_text().replace("eval backend ngnn3 ", f"eval backend ngnn{version} "))
    browser = source / "cmd/browser/core_test.go"
    if browser.exists():
        browser.write_text(browser.read_text().replace(BASE_NET, digest))
    for name in ("build_info.go", "build_info_test.go"):
        info = source / name
        if info.exists():
            info.write_text(info.read_text().replace('"ngnn3"', f'"ngnn{version}"'))
    test = '''package engine
import ("crypto/sha256"; "fmt"; "testing")
func TestCandidateEmbeddedFormat(t *testing.T) {
 raw := DefaultNetBytes()
 if fmt.Sprintf("%x", sha256.Sum256(raw)) != DefaultNetSHA256 { t.Fatal("digest") }
 n, err := LoadDefaultNet()
 if err != nil { t.Fatal(err) }
 if n.version != VERSION || n.hidden != HIDDEN { t.Fatal("format identity") }
}
'''.replace("VERSION", str(version)).replace("HIDDEN", str(hidden))
    (source / "engine/candidate_embedded_test.go").write_text(test)
    oracle = ("_, want := ngnn3Oracle(n, pos)" if version == 3 else
              f"want := ngnn{version}Oracle(n, pos)")
    parity = '''package engine
import "testing"
func TestCandidateEmbeddedParity(t *testing.T) {
 n, err := LoadDefaultNet()
 if err != nil { t.Fatal(err) }
 for _, fen := range []string{FENS} {
  pos, err := ParseFEN(fen)
  if err != nil { t.Fatal(err) }
  e := ngnn1NewWorker(t, n, pos)
  ORACLE
  if got := e.SearchSTM(pos); got != want { t.Fatalf("%s: %d != %d", fen, got, want) }
 }
}
'''.replace("FENS", ",".join(json.dumps(fen) for fen in FENS)).replace("ORACLE", oracle)
    (source / "engine/candidate_parity_test.go").write_text(parity)
    return version


def retained_bytes(directory):
    return sum(p.stat().st_size for p in Path(directory).rglob("*") if p.is_file())


def adopt(job, directory, gated_source, selected, resolved, result, cpus):
    candidate = directory / "adoption"
    candidate.mkdir()
    manifest_path = candidate / "manifest.json"
    manifest = {"version": 1, "status": "BUILDING", "private": True, "main_updated": False,
                "gated_net": selected["net"], "source": selected["source"], "training": resolved,
                "result": artifact(directory / "result.json"), "verdict": result["verdict"]}
    write(manifest_path, manifest)
    try:
        source = unpack(selected["source"], candidate / "source")
        version = embed(source, selected["net"])
        # Bundle exists even if a subsequent build or identity check fails.
        bundle = candidate / "candidate.tar.gz"
        with tarfile.open(bundle, "w:gz") as archive:
            archive.add(source, arcname="source")
        manifest["bundle"] = artifact(bundle)
        write(manifest_path, manifest)
        binary = candidate / "ngn"
        manifest["binary"] = build(source, binary, ".", candidate / "build.log", cpus)
        identity_report = consumer_identity(binary, selected["net"]["path"], cpus)
        write(candidate / "identity.json", identity_report)
        fragment = f"^(TestCandidateEmbeddedFormat|TestCandidateEmbeddedParity|TestNGNN{version}TrainerParity)$"
        argv = ["go", "test", "-p", "1", "-count=1", "./engine", "-run", fragment]
        if execute(argv, source, candidate / "parity.log", cpus):
            raise Invalid("candidate format parity failed")
        if (source / "main_test.go").exists():
            startup = ["go", "test", "-p", "1", "-count=1", ".", "-run",
                       "^(TestRunDefaultEmbedsNNUEAndKeepsItForEchoedDefaults|TestEmbeddedNetDigest|TestBuildInfoCommand)$"]
            if execute(startup, source, candidate / "startup.log", cpus):
                raise Invalid("candidate startup identity failed")
        manifest.update(status="READY", identity=artifact(candidate / "identity.json"),
                        parity={"argv": argv, "log": artifact(candidate / "parity.log")})
    except Exception as error:
        manifest.update(status="FAILED", error=str(error))
        raise
    finally:
        write(manifest_path, manifest)
        size = retained_bytes(directory)
        manifest.update(retained_bytes=size, retention_budget=job["retention_bytes"],
                        keep_reason="Required source, gated net, builds and game evidence are preserved."
                        if size > job["retention_bytes"] else "Within budget; all required evidence preserved.")
        write(manifest_path, manifest)
        # Include the manifest itself in the measured size, to a fixed point.
        while manifest["retained_bytes"] != retained_bytes(directory):
            manifest["retained_bytes"] = retained_bytes(directory)
            write(manifest_path, manifest)


def step(root, job):
    validate(job)
    root = Path(root).resolve()
    root.mkdir(parents=True, exist_ok=True)
    directory = root / job["id"]
    directory.mkdir(exist_ok=True)
    key = identity(job)
    with lock(directory / "job.lock") as acquired:
        receipt_path = directory / "receipt.json"
        if not acquired:
            return read(receipt_path) if receipt_path.exists() else {"status": "WAITING", "reason": "job lock"}
        spec = directory / "job.json"
        if spec.exists() and read(spec) != job:
            raise Invalid("job id already bound to different inputs")
        if not spec.exists():
            write(spec, job)
        previous = read(receipt_path) if receipt_path.exists() else None
        if previous and previous["status"] in TERMINAL:
            return previous
        receipt = {"version": 1, "id": job["id"], "stage": job["stage"], "input_sha256": key,
                   "status": "WAITING", "reason": "admission"}

        def save(status, **fields):
            receipt.update(status=status, **fields)
            adoption = directory / "adoption/manifest.json"
            if status in TERMINAL and adoption.exists():
                receipt["adoption"] = artifact(adoption)
                if (directory / "result.json").exists():
                    receipt["result"] = artifact(directory / "result.json")
            write(receipt_path, receipt)
            if status in TERMINAL and adoption.exists():
                manifest = read(adoption)
                while manifest["retained_bytes"] != retained_bytes(directory):
                    manifest["retained_bytes"] = retained_bytes(directory)
                    write(adoption, manifest)
                receipt["adoption"] = artifact(adoption)
                write(receipt_path, receipt)
            return receipt

        try:
            if previous and previous["status"] == "RUNNING":
                raise Invalid("unfinished work requires inspection; never restart")
            for value in job["inputs"].values():
                check_artifact(value)
            if job.get("prerequisite"):
                prerequisite = job["prerequisite"]
                value = read(check_artifact(prerequisite))
                if value.get("status") not in ("SUCCEEDED", "READY"):
                    raise Invalid("failed prerequisite receipt")
                receipt["prerequisite"] = prerequisite
            resolved = None
            if job["stage"] != "train":
                resolved = resolve_training(root, job["training"])
                if resolved is None:
                    return save("WAITING", reason="training receipt")
                receipt["training"] = resolved
                if job["stage"] == "await-result":
                    return save("READY", reason="terminal successful training receipt")
            reservation = admission(root, job, key)
            if reservation is None:
                return save("WAITING", reason="reservation")
            receipt["admission"] = reservation
            stage_lock = "training.lock" if job["stage"] == "train" else "gate.lock"
            with lock(root / stage_lock) as acquired:
                if not acquired:
                    return save("WAITING", reason=stage_lock)
                save("RUNNING", reason="admitted", started_ns=time.time_ns())
                cpus = reservation["entry"]["cpus"]
                result = (train(job, key, directory, cpus) if job["stage"] == "train" else
                          gate(job, directory, root, resolved, cpus))
                return save("SUCCEEDED", reason="completed", finished_ns=time.time_ns(), **result)
        except Exception as error:
            return save("FAILED" if receipt["status"] == "RUNNING" else "BLOCKED",
                        reason=str(error), finished_ns=time.time_ns())


def supervise(root, job, watch=False, interval=2):
    while True:
        receipt = step(root, job)
        if not watch or receipt["status"] in TERMINAL:
            return receipt
        time.sleep(interval)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("job", type=Path)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--watch", action="store_true")
    parser.add_argument("--detach", action="store_true")
    args = parser.parse_args()
    job = read(args.job)
    validate(job)
    args.root = args.root.resolve()
    if args.detach:
        args.root.mkdir(parents=True, exist_ok=True)
        with (args.root / (job["id"] + "-supervisor.log")).open("ab") as output:
            child = subprocess.Popen([sys.executable, "-m", "scripts.wave2.net_job", str(args.job.resolve()),
                                      "--root", str(args.root), "--watch"], stdin=subprocess.DEVNULL,
                                     stdout=output, stderr=subprocess.STDOUT, start_new_session=True,
                                     env=environment())
        print(json.dumps({"supervisor_pid": child.pid, "receipt": str(args.root / job["id"] / "receipt.json")}))
        return 0
    receipt = supervise(args.root, job, args.watch)
    print(json.dumps(receipt, sort_keys=True))
    return int(receipt["status"] in {"FAILED", "BLOCKED"})


if __name__ == "__main__":
    raise SystemExit(main())

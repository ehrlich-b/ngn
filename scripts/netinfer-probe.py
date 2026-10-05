#!/usr/bin/env python3
"""NGN-owned bounded UCI node identity / throughput / smoke probe."""
import argparse
import json
import os
import queue
import re
import subprocess
import threading
import time

POSITIONS = (
    ("start", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"),
    ("kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"),
    ("endgame", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"),
)


def run(binary, fen, go, network, threads, deadline):
    env = dict(os.environ, GOAMD64="v3", GOMAXPROCS=str(threads))
    cpu_start = os.times()
    process = subprocess.Popen([binary], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, text=True, env=env)
    lines = queue.Queue()

    def read():
        for line in process.stdout:
            lines.put(line.rstrip())
        lines.put(None)

    reader = threading.Thread(target=read, daemon=True)
    reader.start()

    def send(command):
        process.stdin.write(command + "\n")
        process.stdin.flush()

    def receive(terminal, timeout):
        end = time.monotonic() + timeout
        received = []
        while True:
            line = lines.get(timeout=max(0.01, end - time.monotonic()))
            if line is None:
                raise RuntimeError("engine exited: " + process.stderr.read())
            received.append(line)
            if line.startswith("info string error"):
                raise RuntimeError(line)
            if line.startswith(terminal):
                return received

    try:
        send("uci")
        receive("uciok", 20)
        for command in ("setoption name OwnBook value false", "setoption name Hash value 16",
                        "setoption name Move Overhead value 0", f"setoption name Threads value {threads}"):
            send(command)
        if network:
            send("setoption name EvalFile value " + network)
            send("setoption name UseNNUE value true")
        send("debug on")  # End-of-search stats include the unfinished iteration.
        send("ucinewgame")
        send("position fen " + fen)
        send("isready")
        receive("readyok", 20)
        start = time.monotonic()
        send(go)
        output = receive("bestmove", deadline)
        elapsed = time.monotonic() - start
        info = [line for line in output if line.startswith("info depth ")]
        if not info:
            raise RuntimeError("search produced no depth info")
        values = {key: int(value) for key, value in re.findall(r"\b(depth|nodes|nps|time) (\d+)", info[-1])}
        stats = [line for line in output if line.startswith("info string stats ")]
        if not stats:
            raise RuntimeError("missing final search statistics")
        values["reported_nps"] = values["nps"]
        values["nodes"] = int(re.search(r"\bnodes (\d+)", stats[-1]).group(1))
        values["nps"] = round(values["nodes"]/elapsed)
        values.update(elapsed_s=round(elapsed, 4), bestmove=output[-1].split()[1])
    finally:
        if process.poll() is None:
            send("quit")
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()  # Only this probe's own child.
                process.wait()
        process.stdin.close()
        process.stdout.close()
        process.stderr.close()
        reader.join(timeout=1)
    # Reaped child CPU time exposes scheduling pressure on the shared WSL box.
    cpu_end = os.times()
    cpu_seconds = (cpu_end.children_user + cpu_end.children_system
                   - cpu_start.children_user - cpu_start.children_system)
    values.update(child_cpu_s=round(cpu_seconds, 4),
                  cpu_nps=round(values["nodes"]/cpu_seconds) if cpu_seconds else 0)
    return values


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary")
    parser.add_argument("--network", default="")
    parser.add_argument("--threads", type=int, default=1)
    parser.add_argument("--depth", type=int, default=8)
    parser.add_argument("--seconds", type=float, default=0)
    parser.add_argument("--deadline", type=float, default=300)
    args = parser.parse_args()
    go = f"go movetime {round(args.seconds*1000)}" if args.seconds else f"go depth {args.depth}"
    for name, fen in POSITIONS:
        result = run(args.binary, fen, go, args.network, args.threads, args.deadline)
        print(json.dumps(dict(position=name, go=go, threads=args.threads, network=bool(args.network), **result)), flush=True)


if __name__ == "__main__":
    main()

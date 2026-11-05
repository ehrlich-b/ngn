#!/usr/bin/env python3
"""Owned UCI process fixture; deliberately no chess or strength simulation."""
import os
import sys

error_mode = ""
playing = False
for line in sys.stdin:
    line = line.strip()
    if line == "uci":
        print("id name owned-net-job-fixture\nuciok", flush=True)
    elif line == "isready":
        print("readyok", flush=True)
    elif line == "eval":
        print("info string eval backend ngnn1 score_cp 0", flush=True)
    elif line.startswith("setoption name FixtureError value "):
        error_mode = line.split()[-1]
    elif line == "ucinewgame":
        playing = True
    elif line.startswith("go "):
        if os.environ.get("NET_JOB_FAKE_UCI_FAIL"):
            raise SystemExit(1)
        if error_mode == "warmup" or (error_mode == "search" and playing):
            print("info string error search recovered panic: owned fixture failure", flush=True)
        print("info depth 4 score cp 0 nodes 8 pv e2e4\nbestmove e2e4", flush=True)
    elif line == "quit":
        break

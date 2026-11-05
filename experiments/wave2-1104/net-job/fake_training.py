#!/usr/bin/env python3
"""Owned, bounded process fixture: emit an explicit training export receipt."""
import hashlib
import json
import os
from pathlib import Path
import sys
import time

net, source, mode = sys.argv[1:]
if mode == "fail":
    raise SystemExit(2)
if mode == "no-receipt":
    raise SystemExit(0)
if mode == "slow":
    time.sleep(.25)


def artifact(path):
    p = Path(path).resolve()
    return {"path": str(p), "sha256": hashlib.sha256(p.read_bytes()).hexdigest()}


result = {"version": 1, "status": "SUCCEEDED", "input_sha256": os.environ["NET_JOB_INPUT_SHA256"],
          "exports": {"selected": {"net": artifact(net), "source": artifact(source)}}}
if mode == "failed-receipt":
    result["status"] = "FAILED"
if mode == "wrong-source":
    result["exports"]["selected"]["source"]["sha256"] = "0" * 64
Path(os.environ["NET_JOB_RESULT"]).write_text(json.dumps(result) + "\n")

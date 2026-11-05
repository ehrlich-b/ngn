#!/usr/bin/env python3
"""Check the frozen experiment inputs and author/default policy."""
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent

def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()

mode = sys.argv[1]
m = json.loads((HERE / "manifest.json").read_text())
if mode == "host":
    host = socket.gethostname()
    print(json.dumps(dict(host=host, kernel_checks="none", device_access="none", gpu="none", cpu_limit=2)))
    assert m["fixture"]["threads"] == 1
    # Exactly forge is restricted; there is no substring host match.
    if host == "forge":
        print("Only ordinary Go fixture and registered Go gates are declared; no kernel work.")
elif mode == "artifacts":
    cases = json.loads((HERE / "off-depth4.json").read_text())
    assert len(cases) == 4
    assert all(c["move"] and c["nodes"] > 0 for c in cases)
    assert hashlib.sha256((ROOT / m["fixture"]["network"]).read_bytes()).hexdigest() == m["fixture"]["network_sha256"]
    assert os.access(HERE / "run.py", os.X_OK)
    result = (HERE / m["result"]).read_text()
    stats = json.loads(result.strip().splitlines()[-1])
    assert stats["exit_code"] == 0 and stats["cpu_limit"] == 2 and stats["tensor_threads"] == 2
    assert stats["argv"][0:4] == ["nice", "-n", "10", "go"]
    assert "-p" in stats["argv"] and stats["argv"][stats["argv"].index("-p")+1] == "1"
    assert "PASS" in result and "FAIL" not in result
    assert len([line for line in result.splitlines() if "BASELINE " in line]) == 4
    print(json.dumps(dict(fixture_cases=len(cases), net_sha256=m["fixture"]["network_sha256"], recorded_result=stats)))
elif mode == "policy":
    assert m["strength_status"] == "DEFERRED" and m["strength_claim"] is None
    assert m["option"] == dict(name="W2CorrMargin", default=0, min=0, max=1)
    assert m["formula"] == "min(128, abs(corrected_static_before_TT - raw_static) / 2)"
    assert "var W2CorrMargin = 0" in (ROOT / "engine/wave2_corr_margin.go").read_text()
    authors = git("log", m["base_head"]+"..HEAD", "--format=%an <%ae>|%cn <%ce>").splitlines()
    expected = "Bryan Ehrlich <ehrlich.bryan@gmail.com>"
    assert authors and all(a == expected+"|"+expected for a in authors)
    print(json.dumps(dict(authors=authors, strength=m["strength_status"], default=m["option"]["default"])))
else:
    raise ValueError(mode)

#!/usr/bin/env python3
# ebfprobe.py — the EBF/depth-at-fixed-nodes probe (the conthist-lane filter + the
# 3000-path falsification instrument). Committed 2026-06-10 after a host reboot
# wiped the /tmp original (instruments live in scripts/, not /tmp).
#
# Usage: python3 scripts/ebfprobe.py BASE_BIN CAND_BIN   (run x2 — output is
# deterministic, byte-identical reruns; engines run as-launched, wrap the whole
# call in `taskpolicy -b` for E-cores.)
#
# Reads: depth reached at 400K fixed nodes on 6 out-of-book positions (4 quiet/
# closed/late middlegames + 2 endgames). Deltas are what matter; matched-depth
# node counts disambiguate a coarse 0-ply delta (reduction-where-it-counts = real,
# inflation-in-sharp = reject — see TODO Lane A exec log, qmvv vs A2 precedents).
#
# STANDING BASELINE (2026-06-11, post-conthist HEAD fa0926d = grav+fmh kept):
#   mg depths QGD 11 / closed 11 / lateMg-ph6-11 11 / najdorf 10
#   rook-eg 12, pawn-eg 24
# Prior (pre-conthist 95379a8): mg 12/9/11/10, rook-eg 15, pawn-eg 25.
# CONTHIST SYSTEM CHECKPOINT (2026-06-11): FAIL — cumulative mg net +1 (closed
# +2, QGD -1) vs the pre-registered >=+2 bar; statlmr stage SPRT-rejected. The
# depth-17-19 class gap (EBF ~2.0-2.2 at this same 400K) is INTACT; attribution
# now owned by the docs/11 §3 diagnostic (memory: project_ebf_lane).
import subprocess, select, sys

# Out-of-book positions: quiet middlegames + endgames (deltas are what matter).
POS = [
    ("quiet-mg-QGD",   "r1bq1rk1/pp2bppp/2n1pn2/3p4/3P4/2NBPN2/PP3PPP/R1BQ1RK1 w - - 0 1"),
    ("quiet-mg-closed","r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2NBPN2/PP3PPP/R1BQ1RK1 w - - 0 1"),
    ("lateMg-ph6-11",  "2rq1rk1/pp1bppbp/3p1np1/8/3NP3/1BN1BP2/PPPQ2PP/2KR3R w - - 0 1"),
    ("najdorf-mg",     "r2q1rk1/1b1nbppp/p2ppn2/1p6/3NPP2/2N1B3/PPPQB1PP/R4RK1 w - - 0 1"),
    ("rook-eg-R4P",    "8/5pk1/4p1p1/4P3/5PP1/r6P/5K2/1R6 w - - 0 1"),
    ("pawn-eg",        "8/3k4/2p1p3/2P1P3/3K4/8/8/8 w - - 0 1"),
]
NODES = 400000

def run(engine, fen, nodes):
    p = subprocess.Popen([engine], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                         stderr=subprocess.DEVNULL, text=True, bufsize=1)
    def send(s):
        p.stdin.write(s + "\n"); p.stdin.flush()
    send("uci")
    def wait(tok, tmo=20.0):
        buf = []
        while True:
            r,_,_ = select.select([p.stdout], [], [], tmo)
            if not r:
                return buf
            line = p.stdout.readline()
            if not line:
                return buf
            buf.append(line.rstrip("\n"))
            if line.startswith(tok):
                return buf
    wait("uciok")
    send("isready"); wait("readyok")
    send("position fen " + fen)
    send(f"go nodes {nodes}")
    depth = seldepth = lastnodes = 0
    while True:
        r,_,_ = select.select([p.stdout], [], [], 40.0)
        if not r:
            break
        line = p.stdout.readline()
        if not line:
            break
        if line.startswith("info") and " depth " in line:
            t = line.split()
            try:
                if "depth" in t:    depth = int(t[t.index("depth")+1])
                if "seldepth" in t: seldepth = int(t[t.index("seldepth")+1])
                if "nodes" in t:    lastnodes = int(t[t.index("nodes")+1])
            except (ValueError, IndexError):
                pass
        if line.startswith("bestmove"):
            break
    send("quit")
    try: p.wait(timeout=5)
    except subprocess.TimeoutExpired: p.kill()
    return depth, seldepth, lastnodes

base, cand = sys.argv[1], sys.argv[2]
print(f"{'position':<18}{'base d':>7}{'cand d':>7}{'Δd':>4}{'base sd':>9}{'cand sd':>9}{'base N':>10}{'cand N':>10}")
for name, fen in POS:
    bd, bsd, bn = run(base, fen, NODES)
    cd, csd, cn = run(cand, fen, NODES)
    print(f"{name:<18}{bd:>7}{cd:>7}{cd-bd:>4}{bsd:>9}{csd:>9}{bn:>10}{cn:>10}")

#!/usr/bin/env bash
# Prepare a match-only Berserk 12 anchor outside the NGN checkout, after the screen.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
A=/home/ehrli/ngn-data/claim-v3-20261008/anchors/berserk12
EXPECTED=94d48b044bbad8144f7fce7a2848ac888812a1fd
mkdir -p "$A"
# Screen owns the exclusive rig. train.pid is written only after screen termination.
for ((i=0; i<24; i++)); do
  [[ -s $R/session1/train.pid ]] && break
  kill -0 "$(cat "$R/session1/controller.pid")" 2>/dev/null || { echo 'Campaign controller ended before training.' >&2; exit 1; }
  sleep 300
done
[[ -s $R/session1/train.pid ]] || { echo 'Screen completion wait exhausted.' >&2; exit 1; }
exec > "$A/build.log" 2>&1
date -Is
taskset -c 13,15 nice -n 19 git clone --depth 1 --branch 12 https://github.com/jhonnold/berserk.git "$A/source"
[[ $(git -C "$A/source" rev-parse HEAD) == "$EXPECTED" ]]
taskset -c 13,15 nice -n 19 make -C "$A/source/src" basic
cp "$A/source/src/berserk" "$A/berserk-12-linux-native"
git -C "$A/source" submodule status > "$A/submodules.txt"
git -C "$A/source" status --porcelain > "$A/source-status.txt"
taskset -c 13,15 nice -n 19 python3 - "$A" "$EXPECTED" <<'PY'
import hashlib, json, os, subprocess, sys
from pathlib import Path
a=Path(sys.argv[1]); binary=a/'berserk-12-linux-native'
probe=subprocess.run([str(binary)],cwd='/tmp',input='uci\nsetoption name Threads value 1\nsetoption name Hash value 64\nisready\nquit\n',
                     capture_output=True,text=True,timeout=20,env=dict(os.environ,GOMAXPROCS='1'))
(a/'uci.stdout').write_text(probe.stdout); (a/'uci.stderr').write_text(probe.stderr)
assert probe.returncode==0 and 'uciok' in probe.stdout and 'readyok' in probe.stdout
assert 'Berserk 12' in probe.stdout
assert 'option name Threads ' in probe.stdout and 'option name Hash ' in probe.stdout
digest=lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
net=a/'source/src/networks/berserk-fb675dad41b4.nn'
assert net.is_file()
assert digest(net).startswith('fb675dad41b4')
receipt=dict(source_commit=sys.argv[2],build_argv=['taskset','-c','13,15','nice','-n','19','make','-C',str(a/'source/src'),'basic'],
             compiler=subprocess.check_output(['gcc','--version'],text=True).splitlines()[0],
             binary=dict(path=str(binary),sha256=digest(binary)),network=dict(path=str(net),sha256=digest(net)),
             uci_stdout_sha256=digest(a/'uci.stdout'),uci_stderr_sha256=digest(a/'uci.stderr'),probe_exit=probe.returncode,
             ccrl_label=3713,ccrl_version='Berserk 12 64-bit (1CPU)',ccrl_date='2026-10-03',
             admission='Build and UCI only; stability games and orientation are still required.')
(a/'build-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(receipt))
PY
date -Is

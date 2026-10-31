#!/usr/bin/env bash
# Fresh TT-only isolation on forge; rig samples are interrupted and unpooled.
set -euo pipefail
R=/work/lanes/ngn-s47
S=$R/tt
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH GOAMD64=v3 GOMAXPROCS=1
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session47-tt-forge',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; diagnostic only; resume paused SPSA with SIGCONT at terminal"; kill -CONT -- -88166 2>/dev/null || true; }
trap finish EXIT
date -Is > controller.started
st wait 'native compact completion and SPSA inotify-close-write checkpoint1825; no polling'
python3 - "$R" <<'PY'
import json,os,select,signal,sys
from pathlib import Path
p=Path(sys.argv[1])
# Pause the unchanged tuner at its next durable checkpoint, preserving live RNG.
ns={};exec((p/'checkpoint-wait.py').read_text().split("if __name__ == '__main__':")[0],ns)
r=ns['wait'](Path('/work/lanes/ngn-spsa-20261009/spsa.jsonl'),88178,1825,3*3600)
os.killpg(88166,signal.SIGSTOP)
(p/'spsa-pause.json').write_text(json.dumps(r,indent=2)+'\n')
# A completed speed failure also frees the lane; an invalid sample does not
# affect this independently pinned archived TT diagnostic.
if not (p/'compact/controller.done').exists():
 fd=os.pidfd_open(int((p/'compact/controller.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN);assert poll.poll(3600*1000),'compact waiter expired';os.close(fd)
assert (p/'compact/controller.done').exists()
PY
sha256sum -c inputs.sha256 > input-check-before.txt
COMMON=(-tc 60+0.6 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'forge4-11/c2/host_pinning:on; fresh A/A40 at60+0.6'
timeout --signal=TERM --kill-after=15s 4h taskset -c 4-11 nice -n 10 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
sha256sum -c inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly400 independent TT-only paired60+0.6; no source adoption or extension'
timeout --signal=TERM --kill-after=15s 48h taskset -c 4-11 nice -n 10 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
python3 - <<'PY'
import json,re
from pathlib import Path
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
j=json.loads(Path('gate-summary.json').read_text());ci=j['exact_elo_ci95']
r=dict(result='TT-regression-implicated' if ci[2]<0 else 'TT-positive' if ci[1]>0 else 'TT-unresolved',gate_summary=j,playing_adoption=False)
Path('diagnostic-summary.json').write_text(json.dumps(r,indent=2)+'\n')
PY
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat diagnostic-summary.json)"

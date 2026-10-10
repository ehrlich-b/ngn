#!/usr/bin/env bash
# New SEE-only forge diagnosis; no adoption or pooling with the rig sample.
set -euo pipefail
R=/work/lanes/ngn-s48
S=$R/see
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH GOAMD64=v3 GOMAXPROCS=1
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session48-see-isolation',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 if [[ -f spsa-pause.json ]]; then kill -CONT -- -88166 2>/dev/null || true; fi
 st end "rc $rc; diagnostic only; paused SPSA continued; session review required"
}
trap finish EXIT
date -Is > controller.started
st wait 'native SPSA close-write checkpoint1850; then fresh SEE-only A/A40 and fixed1000; data gate has priority'
python3 - <<'PY'
import json,os,signal
from pathlib import Path
ns={}
exec(Path('/work/lanes/ngn-s47/checkpoint-wait.py').read_text().split("if __name__ == '__main__':")[0],ns)
r=ns['wait'](Path('/work/lanes/ngn-spsa-20261009/spsa.jsonl'),88178,1850,3*3600)
os.killpg(88166,signal.SIGSTOP)
Path('spsa-pause.json').write_text(json.dumps(r,indent=2)+'\n')
PY
sha256sum -c inputs.sha256 > input-check-before.txt
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'forge4-11/c2/host_pinning:on; fresh40 at10+0.1; no other NGN games'
timeout --signal=TERM --kill-after=15s 1h taskset -c 4-11 nice -n 10 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 isolation-parser.py aa aa.log > aa-summary.json
sha256sum -c inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly1000 paired SEE-only10+0.1; diagnostic interval/no adoption; interrupt for reviewed data gate'
timeout --signal=TERM --kill-after=15s 8h taskset -c 4-11 nice -n 10 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 1000 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 isolation-parser.py gate gate.log > diagnostic-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat diagnostic-summary.json); review before redesign; no playing change"

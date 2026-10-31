#!/usr/bin/env bash
# Launch only after session31 review selects the retained architecture baseline.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session32-vnni512
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='ngnn4-vnni512-speed',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > speed-controller.exit; date -Is > speed-controller.done; st end "rc $rc; review exact fixed sample before topic retention"; }
trap finish EXIT
date -Is > speed-controller.started
[[ $(cat controller.exit) == 0 ]]
sha256sum -c speed-inputs.sha256 > speed-input-check-before.txt
st timing-start 'reviewed immediate architecture baseline;48 identities/exactly20 balanced blocks on14; optional12 cost blocks only on speed keep'
if python3 speed.py > speed.log 2>&1; then
 st speed-keep 'fixed rule passes; source/raw review required before topic commit; cost is budget only'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete fixed sample failed rule; no extension'; else exit "$rc"; fi
fi
sha256sum -c speed-inputs.sha256 > speed-input-check-after.txt

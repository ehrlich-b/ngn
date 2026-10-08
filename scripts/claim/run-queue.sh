#!/usr/bin/env bash
# Authorized v3 fixed samples. A failed screen or operational failure ends the queue.
set -Eeuo pipefail
if [[ $# != 6 ]]; then
  printf 'Usage: run-queue.sh ID.json SOURCE TOOL BRIDGE TOOL_BUILD.json NEW_QUEUE_DIR\n' >&2
  exit 2
fi
candidate=$1 source=$2 tool=$3 bridge=$4 tool_build=$5 queue=$6
mkdir "$queue"
stage=screen
verdict='NO CLAIM'
failure=
started=$(date -u +%FT%TZ)
finish() {
  rc=$?
  trap - EXIT
  set +e
  # Even a rejected/interrupted sample gets an explicit immutable disposition.
  python3 - "$queue" "$stage" "$verdict" "$failure" "$rc" "$started" <<'PY'
import datetime, json, pathlib, sys
root, stage, verdict, failure, rc, started = sys.argv[1:]
status = dict(stage=stage, verdict=verdict, failure=failure, exit_code=int(rc),
              started_utc=started, finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
              status='complete' if int(rc)==0 else 'operationally invalid; no replay, extension or restart')
with (pathlib.Path(root)/'queue-status.json').open('x') as f:
    json.dump(status, f, indent=2); f.write('\n')
PY
  chmod 444 "$queue/queue-status.json"
  exit "$rc"
}
trap finish EXIT
trap 'failure="line $LINENO: $BASH_COMMAND"' ERR
trap 'failure="interrupted"; exit 130' INT TERM
common=(--candidate "$candidate" --source "$source" --tool "$tool"
        --windows-bridge "$bridge" --tool-build-argv "$tool_build" --execute)
bash "$source/scripts/claim/screen.sh" "${common[@]}" --out "$queue/screen"
if ! python3 - "$queue/screen/analysis.json" "$queue/screen/post-run-hashes.json" <<'PY'
import json, sys
a, h = [json.load(open(p)) for p in sys.argv[1:]]
sys.exit(0 if a['Schema']=='NGN-CLAIM-3' and a['ClaimThreshold']==3600 and a['ScreenEligible'] and a['Bracketed'] and a['Primary']['CI95'][0]>3600 and h['Schema']=='NGN-CLAIM-3' and h['Unchanged'] else 1)
PY
then
  stage=screen-rejected
  exit 0
fi
stage=confirmation
bash "$source/scripts/claim/confirmation.sh" "${common[@]}" --out "$queue/confirmation" \
  --screen-receipt "$queue/screen/receipt.json" --screen-analysis "$queue/screen/analysis.json"
if python3 - "$queue/confirmation/analysis.json" "$queue/confirmation/post-run-hashes.json" <<'PY'
import json, sys
a, h = [json.load(open(p)) for p in sys.argv[1:]]
sys.exit(0 if a['Schema']=='NGN-CLAIM-3' and a['ClaimThreshold']==3600 and a['ClaimEligible'] and a['Bracketed'] and a['Primary']['CI95'][0]>3600 and h['Schema']=='NGN-CLAIM-3' and h['Unchanged'] else 1)
PY
then
  verdict=CLAIM
fi
stage=finished

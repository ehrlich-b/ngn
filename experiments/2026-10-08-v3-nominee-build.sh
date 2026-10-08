#!/usr/bin/env bash
# Prepare a clean candidate engine while retaining the admitted v3 tool.
set -euo pipefail
[[ $# == 2 ]] || { echo 'Usage: controller.sh BUNDLE EXPECTED_COMMIT' >&2; exit 2; }
BUNDLE=$1 EXPECTED=$2
R=/home/ehrli/ngn-data/night-20261007
S=/home/ehrli/ngn-data/claim-v3-20261008/nominee-session10b
V=/home/ehrli/ngn-data/claim-v3-20261008/tooling
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3
export GOMAXPROCS=2 GOCACHE=$S/go-cache PYTHONDONTWRITEBYTECODE=1
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='v3-nominee-preparation',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve nominee-session10b; no retry or nomination"; fi
}
trap finish EXIT
cd "$S"
[[ ! -e controller.started ]]
sha256sum -c preparation-inputs.sha256 > input-check-before.log
python3 - <<'PY'
import shutil
assert shutil.disk_usage('/mnt/c').free >= 25*1024**3
PY
date -Is > controller.started
st start "pid $$; clean engine $EXPECTED; CPU 13,15; training/tuner unchanged; no games"
git clone --shared --no-checkout /home/ehrli/repos/ngn source > clone.log 2>&1
git -C source fetch "$BUNDLE" HEAD >> clone.log 2>&1
git -C source checkout --detach FETCH_HEAD >> clone.log 2>&1
[[ $(git -C source rev-parse HEAD) == "$EXPECTED" ]]
[[ -z $(git -C source status --porcelain --untracked-files=all) ]]
mkdir frozen checks
taskset -c 13,15 nice -n 19 python3 check.py inputs "$S" "$EXPECTED" > source-check.log 2>&1
cd source
[[ $(go version) == 'go version go1.25.5 linux/amd64' ]]
taskset -c 13,15 nice -n 19 go build -buildvcs=true -p 2 -o "$S/frozen/ngn" . > "$S/ngn-build.log" 2>&1
taskset -c 13,15 nice -n 19 go build -buildvcs=true -p 2 -o "$S/frozen/speedprobe" ./cmd/speedprobe > "$S/probe-build.log" 2>&1
go version -m "$S/frozen/ngn" > "$S/frozen/ngn-buildinfo.txt"
go version -m "$S/frozen/speedprobe" > "$S/frozen/speedprobe-buildinfo.txt"
cd "$S"
taskset -c 13,15 nice -n 19 python3 check.py artifacts "$S" "$EXPECTED" > artifact-check.log 2>&1
taskset -c 13,15 nice -n 19 python3 check.py identities "$S" "$EXPECTED" > identity-check.log 2>&1
st identities-pass '36 accepted-delta records and 18 actual UCI/common-engine cases exact; no timing or tuner pause'
taskset -c 13,15 nice -n 19 "$V/frozen/claimtool" identity \
  --engine "$S/frozen/ngn" --engine-sha "$(sha256sum "$S/frozen/ngn" | cut -c1-64)" \
  --net "$S/frozen/development-be34532c.nnue" --net-sha be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a \
  --source "$S/source" --build-argv "$S/frozen/ngn-build.json" \
  --provenance "$S/frozen/provenance.json" --out "$S/frozen/development-identity.json" > identity.log 2>&1
taskset -c 13,15 nice -n 19 python3 check.py plans "$S" "$EXPECTED" > plan-check.log 2>&1
sha256sum -c preparation-inputs.sha256 > input-check-after.log
taskset -c 13,15 nice -n 19 python3 check.py finish "$S" "$EXPECTED" > final-check.log 2>&1
st done 'clean candidate build, inherited tests, 54 exact identities and dry controls pass; final nomination/protocol still required'

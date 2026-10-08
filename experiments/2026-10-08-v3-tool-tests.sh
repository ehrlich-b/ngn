#!/usr/bin/env bash
# Test the prospective v3 instrument after the exclusive screen and PGO probe.
# No games, engine behavior change, claim launch, or automatic promotion.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session2
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='v3-tool-tests',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
trap 'rc=$?; [[ $rc == 0 ]] || st fail "rc $rc; inspect session2/tests.log; no retry"' EXIT
st wait "pid $$; final source pins and the existing PGO controller must finish"
ready=0
for ((i=0; i<24; i++)); do
  pstate=$(ps -o stat= -p 534228 2>/dev/null || true)
  if [[ -f $S/inputs-ready && ( -z $pstate || $pstate == Z* ) && -s $R/session1/train.pid ]]; then
    ready=1; break
  fi
  sleep 300
done
[[ $ready == 1 ]] || { st fail 'dependency wait exhausted'; exit 1; }
cd "$S"
sha256sum -c inputs.sha256 > input-check.log
mkdir baseline source
tar -xf source.tar -C baseline
tar -xf source.tar -C source
git -C source apply "$S/candidate.patch"
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOFLAGS=-p=2
st start 'old-fail threshold witness; narrow v3 tests; required short/race suites on CPU 13,15'
cat > baseline/internal/claim/campaign_v3_regression_test.go <<'GO'
package claim
import "testing"
func TestCampaignThresholdRejectsSub3600(t *testing.T) {
    for _, mode := range []string{"screen", "confirmation"} {
        pairs := 80
        if mode == "screen" { pairs = 40 }
        p := PoolAnalysis{CI95: [2]float64{3500, 3550}}
        if s, c := eligible(mode, decisionAnchors(pairs), p, true, 0); s || c {
            t.Fatalf("%s still admits a lower bound below 3600", mode)
        }
    }
}
GO
cd "$S/baseline"
set +e
taskset -c 13,15 nice -n 19 go test -short ./internal/claim -run '^TestCampaignThresholdRejectsSub3600$' -count=1 > "$S/old-fail.log" 2>&1
rc=$?
set -e
[[ $rc == 1 ]] && rg -q 'still admits a lower bound below 3600' "$S/old-fail.log"
cd "$S/source"
taskset -c 13,15 nice -n 19 go test -short ./internal/claim ./cmd/claimtool -count=1 > "$S/claim-tests.log" 2>&1
taskset -c 13,15 nice -n 19 go test -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -short ./... -count=1 > "$S/all-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -short -race ./... -count=1 > "$S/all-race.log" 2>&1
sha256sum -c "$S/inputs.sha256" > "$S/post-test-input-check.log"
python3 - "$S" <<'PY'
import datetime,hashlib,json,sys
from pathlib import Path
s=Path(sys.argv[1])
files=['old-fail.log','claim-tests.log','engine-short.log','engine-race.log','all-short.log','all-race.log']
with (s/'test-receipt.json').open('x') as f:
    json.dump(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   passed=True,old_threshold_reproduced=True,cpu='13,15',nice=19,
                   environment=dict(GOMAXPROCS='2',GOAMD64='v3',GOTOOLCHAIN='local',GOPROXY='off',GOFLAGS='-p=2'),
                   logs={p:hashlib.sha256((s/p).read_bytes()).hexdigest() for p in files}),f,indent=2)
    f.write('\n')
PY
st done 'short/race tests passed; session2/test-receipt.json; v3 games still require final protocol freeze and stability admission'

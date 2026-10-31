#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session88/tt-pressure
cd "$D/source"
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2
export NGN_PRESSURE_NET=$R/session82/net-final115/candidate.nnue
cp "$D/pressure.go.txt" engine/campaign_pressure_test.go
cat > engine/campaign_pressure_counts.go <<'GO'
package engine
// Disposable single-thread instrumentation, never part of a playing binary.
type campaignPressureCounts struct {
    Stores, Matching, MatchingAccepted, MatchingRejected uint64
    OldMatchingRejected, RejectedDifferentScore, RejectedDifferentMove uint64
    Evictions, EmptyInserts, CollisionRejected, ProbeSlots, OccupiedMismatches uint64
}
var campaignPressure campaignPressureCounts
GO
gofmt -w engine/campaign_pressure_test.go engine/campaign_pressure_counts.go
go test -c -o "$D/uninstrumented.test" ./engine
GOMAXPROCS=1 NGN_PRESSURE_IDENTITY=1 NGN_PRESSURE_OUT="$D/identity-base.jsonl" \
  "$D/uninstrumented.test" -test.run '^TestCampaignTTPressure$' -test.timeout 2m
python3 "$D/instrument.py" "$D/source"
gofmt -w engine/cache.go
go test -c -o "$D/instrumented.test" ./engine
GOMAXPROCS=1 NGN_PRESSURE_IDENTITY=1 NGN_PRESSURE_OUT="$D/identity-instrumented.jsonl" \
  "$D/instrumented.test" -test.run '^TestCampaignTTPressure$' -test.timeout 2m
python3 - <<'PY'
import json
from pathlib import Path
D=Path('/home/ehrli/ngn-data/night-20261007/session88/tt-pressure')
base=[json.loads(x) for x in (D/'identity-base.jsonl').read_text().splitlines()]
instrumented=[json.loads(x) for x in (D/'identity-instrumented.jsonl').read_text().splitlines()]
for a,b in zip(base,instrumented):
    a.pop('pressure'); b.pop('pressure')
assert len(base)==len(instrumented)==12 and base==instrumented, 'instrumentation changed search'
(D/'identity.json').write_text(json.dumps(dict(passed=True,identities=12,node_cap=100000))+'\n')
PY
GOMAXPROCS=1 NGN_PRESSURE_OUT="$D/pressure.jsonl" \
  "$D/instrumented.test" -test.run '^TestCampaignTTPressure$' -test.timeout 15m
python3 - <<'PY'
import datetime,json
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007'); D=R/'session88/tt-pressure'
rows=[json.loads(x) for x in (D/'pressure.jsonl').read_text().splitlines()]
assert len(rows)==24
result=dict(state='DIAGNOSED',cells=24,identity_cases=12,
    interpretation='Counters and Hash64/256 sensitivity only; no Elo or policy acceptance verdict',
    finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(D/'result.json').write_text(json.dumps(result,indent=2)+'\n')
with (R/'status.jsonl').open('a') as f:
    f.write(json.dumps(dict(utc=result['finished_utc'],stage='session88-tt-pressure',event='complete',detail=result))+'\n')
PY

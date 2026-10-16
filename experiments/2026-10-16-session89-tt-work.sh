#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session89/tt-history
cd "$D/source"
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2
export NGN_HISTORY_NET=$R/session82/net-final115/candidate.nnue
cp "$D/history.go.txt" engine/campaign_tt_history_test.go
gofmt -w engine/campaign_tt_history_test.go
go test -c -o "$D/base.test" ./engine
GOMAXPROCS=1 NGN_HISTORY_IDENTITY=1 NGN_HISTORY_OUT="$D/identity-base.jsonl" \
  "$D/base.test" -test.run '^TestCampaignTTHistory$' -test.timeout 5m
python3 "$D/instrument.py" "$D/source"
gofmt -w engine/cache.go
go test -c -o "$D/instrumented.test" ./engine
GOMAXPROCS=1 NGN_HISTORY_IDENTITY=1 NGN_HISTORY_OUT="$D/identity-instrumented.jsonl" \
  "$D/instrumented.test" -test.run '^TestCampaignTTHistory$' -test.timeout 5m
python3 - <<'PY'
import json
from pathlib import Path
D=Path('/home/ehrli/ngn-data/night-20261007/session89/tt-history')
a=[json.loads(x) for x in (D/'identity-base.jsonl').read_text().splitlines()]
b=[json.loads(x) for x in (D/'identity-instrumented.jsonl').read_text().splitlines()]
for row in a+b: row.pop('pressure')
assert len(a)==len(b)==96 and a==b, 'instrumentation changed persistent roots'
(D/'identity.json').write_text(json.dumps(dict(passed=True,roots=96,node_cap=100000))+'\n')
PY
GOMAXPROCS=1 NGN_HISTORY_OUT="$D/history.jsonl" \
  "$D/instrumented.test" -test.run '^TestCampaignTTHistory$' -test.timeout 25m
python3 "$D/review.py" "$D"

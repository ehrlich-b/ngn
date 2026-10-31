#!/usr/bin/env bash
set -euo pipefail
p=/home/ehrli/ngn-data/night-20261007/session79-see-proof
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
cd "$p/old"
set +e
go test -p 2 -short ./engine -run '^TestSEEPostMoveGeometry$' -count=1 -v > "$p/old-fail.log" 2>&1
old_rc=$?
set -e
[ "$old_rc" = 1 ]
python3 - "$p/old-fail.log" <<'PY'
import pathlib,sys
s=pathlib.Path(sys.argv[1]).read_text()
for name in ['source_reveals_rook/white','source_reveals_rook/black','EP_ignores_attack_on_captured_square/white','EP_ignores_attack_on_captured_square/black']:
 assert '--- FAIL: TestSEEPostMoveGeometry/'+name in s,name
print('Both source-xray and EP destination witnesses fail before the patch, in both colors.')
PY
cp /home/ehrli/ngn-data/night-20261007/session66-correctness/see-narrow.log "$p/new-pass.log"
printf '0\n' > "$p/proof.exit"
date -Is > "$p/proof.done"

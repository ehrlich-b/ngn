#!/usr/bin/env bash
# Source checks only, on forge after the key diagnostics; games remain on rig.
set -euo pipefail
R=/work/lanes/ngn-spsa-20261009
S=$R/key-plan
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
mkdir "$S"
cd "$S"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
date -Is > controller.started
mkdir source
tar -xf "$R/base.tar" -C source
(cd source && git apply "$R/key-plan.patch")
(cd source && timeout 20m go test -p 2 -short ./... -count=1) > full-short.log 2>&1
(cd source && timeout 20m go test -p 2 -short -race ./... -count=1) > full-race.log 2>&1
(cd source && timeout 10m go vet -p 2 ./...) > vet.log 2>&1
(cd source && timeout 10m go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',candidate_topic='night/key-plan-current',machine='forge / Ryzen6900HX Debian13 VM',go_version=subprocess.check_output(['go','version'],text=True).strip(),os=platform.platform(),goamd64='v3',cpus=[8,9,10,11],memory_max=2684354560,nice=10,checks=['full-short ./...','full-race ./...','go vet ./...','explicit accepted-PGO build'],files_sha256={p:sha(p) for p in ['source/engine/hash.go','source/engine/uci_process_test.go','source/default.pgo','candidate-ngn']},patch_sha256=sha('../key-plan.patch'),strength_games=0)
Path('receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY

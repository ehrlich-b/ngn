#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s46-compact
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session46-compact',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; compact source checks only; rig speed verdict required"; }
trap finish EXIT
date -Is > controller.started
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
(cd candidate && gofmt -w engine/ngnn*.go engine/worker_evaluator.go)
st checks-start 'loader-proved compact accumulators on nominated base; forge8-11/2GiB'
for tags in normal purego; do
 flags=(); [[ $tags != purego ]] || flags=(-tags=purego)
 (cd candidate && timeout 15m taskset -c 8-11 nice -n 10 go test -p 2 "${flags[@]}" -run '^TestNGNN(Compact|3|NullFrame)' -count=1 -v ./engine) > "$tags-controls.log" 2>&1
done
for mode in short race; do
 flags=(); [[ $mode != race ]] || flags=(-race)
 (cd candidate && timeout 30m taskset -c 8-11 nice -n 10 go test -p 2 -short "${flags[@]}" ./... -count=1) > "full-$mode.log" 2>&1
done
(cd candidate && timeout 15m taskset -c 8-11 nice -n 10 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 for spec in probe:./cmd/speedprobe ngn:.; do
  name=${spec%%:*}; package=${spec#*:}
  (cd "$kind" && timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-$name" "$package") > "$kind-$name-build.log" 2>&1
 done
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
changed=[line[6:] for line in (r/'candidate.patch').read_text().splitlines() if line.startswith('+++ b/')]
j=dict(base_source=json.loads((r/'nomination.json').read_text())['base_source'],utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source_archive_sha256=sha(r/'base.tar'),patch_sha256=sha(r/'candidate.patch'),source_sha256={f:sha(r/'candidate'/f) for f in changed},binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},uci_sha256={k:sha(r/f'{k}-ngn') for k in ['baseline','candidate']},profile_sha256=sha(r/'baseline/default.pgo'),old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='forge Debian13',build_test_cpus=[8,9,10,11],nice=10,tests_pass=True,memory_max=2147483648,checks=['compact bound/kernel/oracle/cache/null normal controls','compact purego controls','full short ./...','full race ./...','vet','explicit PGO builds'],strength_games=0)
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st checks-pass 'compact source checks/builds complete; rig36-identity/fixed40-block speed later'

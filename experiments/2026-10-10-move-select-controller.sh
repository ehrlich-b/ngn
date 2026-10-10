#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session40-move-select
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session40-move-select',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; review fixed speed verdict before adoption; no extension"
}
trap finish EXIT
date -Is > controller.started
st tests-start 'current6c0fc26 independent selection lanes; rig12,14; existing gates untouched'
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 12,14 nice -n 19 gofmt -w candidate/engine/moveorder.go candidate/engine/move_select_test.go
(cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestMoveSelectScalarIdentity$' -count=1 -v ./engine) > selection.log 2>&1
for mode in engine-short engine-race full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 package=./engine; [[ $mode != full-* ]] || package=./...
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" "$package" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for spec in probe:./cmd/speedprobe ngn:.; do
 name=${spec%%:*}; package=${spec#*:}
 (cd candidate && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/candidate-$name" "$package") > "candidate-$name-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
j=dict(base_source='6c0fc26',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
 source_archive_sha256=sha(r/'base.tar'),patch_sha256=sha(r/'candidate.patch'),source_sha256={f:sha(r/'candidate'/f) for f in ['engine/moveorder.go','engine/move_select_test.go']},
 binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},profile_sha256=sha(r/'baseline/default.pgo'),old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),
 candidate_uci_sha256=sha(r/'candidate-ngn'),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',build_test_cpus=[12,14],nice=19,tests_pass=True,
 baseline_probe_reused_from='forge current-key source check106279',concurrent_workloads=['provenance0-11/3GiB','architecture13,15/2GiB','hopper unchanged'],memory_max=2684354560,
 build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','PACKAGE'])
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st tests-pass 'scalar ordering/full checks pass;36 exact identities and fixed20 timing blocks next'
export GOMAXPROCS=1
if python3 check.py > check.log 2>&1; then
 st speed-keep 'complete fixed rule passes; review before isolated topic commit; architecture base held'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete fixed sample fails rule; no extension'; else exit "$rc"; fi
fi

#!/usr/bin/env bash
# Three isolated time candidates on the new accepted key base; no games.
set -euo pipefail
S=/work/lanes/ngn-s39-time-key
I=/work/lanes/ngn-s39-key-integration
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session39-time-key-checks',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; separate candidates only; long-clock games remain required"; }
trap finish EXIT
date -Is > controller.started
st wait 'native completion of the separate forge integration scope; no polling'
python3 - "$I" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(90*60*1000),'integration dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists() and int((p/'controller.exit').read_text())==0
PY
sha256sum -c inputs.sha256 > input-check-before.txt
for spec in admission:admission.patch:TestIterationAdmission horizon:horizon.patch:TestTimeHorizon partial:partial.patch:TestPartialRoot; do
 IFS=: read -r kind patch pattern <<< "$spec"
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
 (cd "$kind" && git apply "../$patch")
 st checks-start "isolated $kind on6c0fc26; current-key source changed since session38"
 (cd "$kind" && timeout 15m taskset -c 8-11 nice -n 10 go test -p 2 -run "^$pattern" -count=1 -v ./engine) > "$kind-narrow.log" 2>&1
 test "$(rg -c '^--- PASS:' "$kind-narrow.log")" -ge 1
 for mode in full-short full-race; do
  flags=(); [[ $mode != *race ]] || flags=(-race)
  (cd "$kind" && timeout 30m taskset -c 8-11 nice -n 10 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$kind-$mode.log" 2>&1
 done
 (cd "$kind" && timeout 15m taskset -c 8-11 nice -n 10 go vet -p 2 ./...) > "$kind-vet.log" 2>&1
 (cd "$kind" && timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/$kind/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
 python3 - "$kind" "$patch" <<'PY'
import datetime,hashlib,json,platform,subprocess,sys
from pathlib import Path
kind,patch=sys.argv[1:];p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
changed=[line.split(' b/',1)[1] for line in Path(patch).read_text().splitlines() if line.startswith('diff --git ')]
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='6c0fc26a17e359442098802f9f33ee673c118c90',change=kind,checks=['existing narrow controls','full-short ./...','full-race ./...','go vet ./...','explicit PGO build'],machine='forge / Ryzen6900HX Debian13 VM',go_version=subprocess.check_output(['go','version'],text=True).strip(),os=platform.platform(),cpus=[8,9,10,11],memory_max=2684354560,nice=10,files_sha256={n:sha(p/n) for n in [patch,'base.tar',kind+'-ngn',kind+'/default.pgo',kind+'/engine/hash.go']},changed_source_sha256={n:sha(p/kind/n) for n in changed},strength_games=0,automatic_promotion=False)
(p/(kind+'-receipt.json')).write_text(json.dumps(j,indent=2)+'\n')
PY
 st checks-pass "$kind: current-source code ready; no behavior or strength acceptance"
done
sha256sum -c inputs.sha256 > input-check-after.txt

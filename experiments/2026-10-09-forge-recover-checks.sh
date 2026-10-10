#!/usr/bin/env bash
# The independent tune and this scope each have2.5G, totaling at most5G.
set -euo pipefail
R=/work/lanes/ngn-spsa-20261009
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$R"
finish() { local rc=$?; echo "$rc" > recovery/controller.exit; date -Is > recovery/controller.done; }
trap finish EXIT
date -Is > recovery/controller.started

# The cancellation notification precedes unlocking. Check eventual lock access
# with the output still blocked, and prove the test rejects a join holding it.
(cd baseline && timeout 5m go test -p 2 -race -run '^TestUCICompletionFollowsFinalOutputWithoutHoldingLifecycleLock$' -count=20 ./engine) > recovery/test-fix.log 2>&1
python3 - <<'PY'
import shutil
from pathlib import Path
shutil.copytree('baseline', 'lock-negative-control')
p=Path('lock-negative-control/engine/uci.go')
s=p.read_text()
old='\tuci.stopSearchSession(session, suppress)\n\t<-session.done'
assert s.count(old)==1
p.write_text(s.replace(old,'\tuci.stopSearchSession(session, suppress)\n\tuci.lifecycleMu.Lock()\n\tdefer uci.lifecycleMu.Unlock()\n\t<-session.done'))
PY
set +e
(cd lock-negative-control && timeout 30s go test -p 2 -race -timeout 20s -run '^TestUCICompletionFollowsFinalOutputWithoutHoldingLifecycleLock$' -count=1 ./engine) > recovery/negative-control.log 2>&1
control_rc=$?
set -e
test "$control_rc" = 1
rg -q 'final output or join held lifecycleMu while blocking' recovery/negative-control.log

(cd baseline && timeout 20m go test -p 2 -short ./... -count=1) > recovery/baseline-full-short.log 2>&1
(cd baseline && timeout 20m go test -p 2 -short -race ./... -count=1) > recovery/baseline-full-race.log 2>&1

# Preserve the completed admission narrow/full-short stages. Its failure was
# the independent test's transient TryLock, so resume from that failed race.
(cd admission && timeout 20m go test -p 2 -short -race ./... -count=1) > recovery/admission-full-race.log 2>&1
(cd admission && timeout 10m go vet -p 2 ./...) > admission-vet.log 2>&1
(cd admission && timeout 10m go build -p 2 -pgo="$R/baseline/default.pgo" -o "$R/admission-ngn" .) > admission-build.log 2>&1
date -Is > admission.done

for spec in horizon:2026-10-09-time-horizon.patch:TestTimeHorizon partial:2026-10-09-partial-root.patch:TestPartialRoot; do
 IFS=: read -r kind patch pattern <<< "$spec"
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
 (cd "$kind" && git apply ../test-fix.patch && git apply "../$patch")
 (cd "$kind" && timeout 10m go test -p 2 -run "^$pattern" -count=1 -v ./engine) > "$kind-narrow.log" 2>&1
 (cd "$kind" && timeout 20m go test -p 2 -short ./... -count=1) > "$kind-full-short.log" 2>&1
 (cd "$kind" && timeout 20m go test -p 2 -short -race ./... -count=1) > "$kind-full-race.log" 2>&1
 (cd "$kind" && timeout 10m go vet -p 2 ./...) > "$kind-vet.log" 2>&1
 (cd "$kind" && timeout 10m go build -p 2 -pgo="$R/baseline/default.pgo" -o "$R/$kind-ngn" .) > "$kind-build.log" 2>&1
 date -Is > "$kind.done"
done
python3 key-prepare.py
export GOMAXPROCS=1 NGN_KEY_NET=$R/kb8bin/src/engine/nets/ngn-kb8h512-be34532c.nnue
for kind in oldkey newkey; do
 (cd "$kind" && NGN_KEY_OUT="$R/$kind.json" timeout 10m go test -p 2 -run '^TestCampaignKeyDiagnostics$' -count=1 -v ./engine) > "$kind.log" 2>&1
done
printf 'Recovered actual-base checks and diagnostic fixtures complete. No strength games.\n'

#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-next-smp
finish(){ rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
export NGN_CAMPAIGN_NET=/home/ehrli/ngn-data/night-20261007/session77-see-rig/base.nnue
for TOPIC in root-ties helper-depths; do
 C=$D/$TOPIC
 cd "$C/source"
 gofmt -w engine/search.go engine/search_smp.go $(find engine -maxdepth 1 -name "search_smp_*ties*.go" -o -name "search_smp_depths*.go")
 go test -p 1 -short ./engine -run '^(TestSMPHelperRootTies|TestSMPHelperDepth|TestLazySMP)' -count=1 -v > "$C/focused.log" 2>&1
 go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
 for width in 2 4; do GOMAXPROCS=$width go test -p 1 -short -race ./... -count=1 > "$C/full-race-$width.log" 2>&1; done
 go vet -p 1 ./... > "$C/vet.log" 2>&1
 go build -p 1 -pgo=default.pgo -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
 sha256sum "$C/candidate-ngn" > "$C/candidate.sha256"
 # Same fixed12 full histories/36 pairs, each against the unchanged immediate base.
 sed -e "s@session77-consensus'@session77-next-smp/$TOPIC'@" -e "s@p.parent/'session77-see-rig@p.parent.parent/'session77-see-rig@g" -e "s@p.parent/'session77-smp-panel@p.parent.parent/'session77-smp-panel@g" /home/ehrli/ngn-data/night-20261007/session77-consensus/identity.py > "$C/identity.py"
 python3 "$C/identity.py" > "$C/identity.log" 2>&1
 echo 0 > "$C/checks.exit"; date -Is > "$C/checks.done"
done

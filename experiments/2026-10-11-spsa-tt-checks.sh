#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s56/spsa-final
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$S"
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; }
trap finish EXIT
date -Is > checks.started
cp -a /work/lanes/ngn-s53/tt-refresh/candidate source
cp /work/lanes/ngn-s54/spsa-final/final-live.go.txt source/engine/spsa_final_live_test.go
(cd source && NGN_SPSA_FINAL_VECTOR="$S/vector.json" timeout 10m go test -p 2 -count=1 -run '^TestCampaignSPSAFinalLiveVector$' -v ./engine) > handler.log 2>&1
for kind in accepted factorizer; do
 mkdir "$kind"
 cp vector.json endpoint.json handler.log "$kind/"
 ln -s ../source "$kind/source"
 ln -s /work/lanes/ngn-s53/tt-refresh/candidate-ngn "$kind/ngn"
 if [[ $kind == accepted ]]; then
  ln -s /work/lanes/ngn-s53/tt-refresh/base.nnue "$kind/base.nnue"
 else
  ln -s /work/lanes/ngn-s51/factorizer/trained/best.nnue "$kind/base.nnue"
 fi
 timeout 10m python3 final-live.py "$S/$kind" > "$kind/probe.log" 2>&1
 done

#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session94
trap 'rc=$?; echo "$rc" > "$D/playing-build.exit"; touch "$D/playing-build.done"' EXIT
export GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
python3 "$D/build-playing.py"

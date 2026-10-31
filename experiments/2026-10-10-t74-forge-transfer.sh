#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session41-remote-archive
D=$R/session19-data
A=/home/ehrli/nnue-owned-k4-20260920/archive-a1
cd "$S"
finish() { local rc=$?; echo "$rc" > transfer.exit; date -Is > transfer.done; }
trap finish EXIT
date -Is > transfer.started
token=$(cat token)
url=http://192.168.4.65:8766/$token
upload() { taskset -c 12,14 nice -n 19 curl --fail --silent --show-error --max-time 7200 -T "$1" "$url/$2"; }
upload "$D/tool/target/release/ngnk4archive" converter
upload /home/ehrli/nnue-owned-k4-20260920/runs/k4-production-pilot-20260921/selection/conflicts.bin conflicts.bin
upload "$A/calibration-probe.keys" calibration-probe.keys
upload "$A/score-map.json" score-map.json
upload "$D/scatter-t80stats/scatter.json" t80-statistics.json
upload "$D/scatter-t76/scatter.json" t76-statistics.json
upload "$D/fit-source-scale.py" fit-source-scale.py
upload "$D/raw/farseerT74.binpack" farseerT74.binpack
date -Is > upload-complete
upload upload-complete transfer.done

#!/usr/bin/env bash
set -euo pipefail
s=$1
taskset -pc 4-11 $$ > "$s/controller-affinity.txt"
cd "$s"
finish() {
  local rc=$?
  echo "$rc" > controller.exit
  date -Is > controller.done
}
trap finish EXIT
date -Is > controller.started
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
python3 games.py

#!/usr/bin/env bash
# One NGN-owned search worker per process. Each invocation reserves a new run dir.
set -euo pipefail

repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
out=""
binary="$repo/build/datagen"
workers=14
seed=$(date +%s%N)
args=()
usage() {
  printf '%s\n' 'Usage: datagen_run.sh -out DIR [-workers 14] [-seed BASE] [-bin PATH] [-net PATH] [-games N] [-nodes 5000] [-random-plies N] [-maxply 400] [-hash 16]'
}
while (($#)); do
  if (($# < 2)); then usage >&2; exit 2; fi
  case "$1" in
    -out) out=$2 ;;
    -workers|-n) workers=$2 ;;
    -seed) seed=$2 ;;
    -bin) binary=$2 ;;
    -net|-games|-nodes|-random-plies|-maxply|-hash) args+=("$1" "$2") ;;
    *) usage >&2; exit 2 ;;
  esac
  shift 2
done
if [[ $(uname -s) != Linux ]]; then
  printf '%s\n' 'Run datagen on the authorized Linux/WSL host.' >&2; exit 2
fi
if [[ -z $out || ! $workers =~ ^[0-9]+$ || ! $seed =~ ^[0-9]+$ || ${#workers} -gt 3 || ${#seed} -gt 19 ]]; then
  usage >&2; exit 2
fi
# Normalize decimal strings before Bash arithmetic; reserve room for all seeds.
workers=$((10#$workers)); seed=$((10#$seed))
if ((workers < 1 || workers > 256 || seed < 0 || seed > 9223372036854775807-workers)); then
  printf '%s\n' 'Invalid worker count or seed range.' >&2; exit 2
fi
if [[ ! -x $binary ]]; then printf 'Missing executable: %s\n' "$binary" >&2; exit 2; fi
mkdir -p -- "$out"
run=$(mktemp -d "$out/run-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX")
printf 'run_dir=%s\n' "$run"
pids=()
stopping=0
stop_workers() {
  stopping=1
  if ((${#pids[@]})); then kill -TERM "${pids[@]}" 2>/dev/null || true; fi
}
trap stop_workers INT TERM
for ((i=0; i<workers; i++)); do
  if ((stopping)); then break; fi
  worker_seed=$((seed+i))
  GOMAXPROCS=1 "$binary" -out "$run/worker-$i-seed-$worker_seed.ngnp" -seed "$worker_seed" "${args[@]}" >"$run/worker-$i.log" 2>&1 &
  pids+=("$!")
done
printf '%s\n' "${pids[@]}" >"$run/pids.txt"
failed=0
for pid in "${pids[@]}"; do
  while true; do
    status=0
    wait "$pid" || status=$?
    if ((status > 128 && stopping)); then
      # A trap may interrupt wait just as the child finishes; collect its real
      # exit status even if it has already disappeared from the process table.
      status=0
      wait "$pid" || status=$?
    fi
    # wait can be interrupted by the driver's signal trap before a child exits.
    if kill -0 "$pid" 2>/dev/null; then continue; fi
    if ((status != 0)); then failed=1; stop_workers; fi
    break
  done
done
printf 'exit_status=%d\n' "$failed"
exit "$failed"

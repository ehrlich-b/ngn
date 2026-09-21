#!/usr/bin/env bash
set -euo pipefail

# Run inside WSL. The Windows volume backs the sparse WSL virtual disk, so its
# physical free space is the authoritative launch/stop gate.
readonly LAUNCH_KIB=$((60 * 1024 * 1024))
readonly STOP_KIB=$((25 * 1024 * 1024))
readonly CHECK_SECONDS=60

usage() {
  echo "usage: run_wsl_stage.sh COMMAND [ARG ...]" >&2
  exit 2
}

available_kib() {
  df -Pk /mnt/c | awk 'NR == 2 { print $4 }'
}

[[ $# -gt 0 ]] || usage
available="$(available_kib)"
[[ "$available" =~ ^[0-9]+$ ]] || {
  echo "could not read physical free space for /mnt/c" >&2
  exit 1
}
if (( available < LAUNCH_KIB )); then
  printf 'disk launch gate failed: %.2f GiB free, require at least 60 GiB\n' \
    "$(awk -v kib="$available" 'BEGIN { print kib / 1024 / 1024 }')" >&2
  exit 1
fi

setsid taskset -c 10-11 nice -n 10 "$@" &
job_pid=$!

stop_job() {
  kill -TERM -- "-$job_pid" 2>/dev/null || true
}
trap stop_job INT TERM HUP

(
  while kill -0 "$job_pid" 2>/dev/null; do
    sleep "$CHECK_SECONDS"
    current="$(available_kib)"
    if [[ ! "$current" =~ ^[0-9]+$ ]] || (( current < STOP_KIB )); then
      echo "disk stop gate reached; terminating process group $job_pid" >&2
      kill -TERM -- "-$job_pid" 2>/dev/null || true
      sleep 10
      kill -KILL -- "-$job_pid" 2>/dev/null || true
      exit 70
    fi
  done
) &
monitor_pid=$!

set +e
wait "$job_pid"
job_status=$?
kill "$monitor_pid" 2>/dev/null
wait "$monitor_pid" 2>/dev/null
set -e

trap - INT TERM HUP
exit "$job_status"

#!/usr/bin/env bash
# NGN-owned fixed screen/confirmation launch. No engines in --dry-run.
set -euo pipefail
mode=$1
shift
candidate= out= source= tool= bridge= tool_build= screen_receipt= screen_analysis=
dry=false
execute=false
while (($#)); do
  case "$1" in
    --candidate) candidate=$2; shift 2;;
    --out) out=$2; shift 2;;
    --source) source=$2; shift 2;;
    --tool) tool=$2; shift 2;;
    --windows-bridge) bridge=$2; shift 2;;
    --tool-build-argv) tool_build=$2; shift 2;;
    --screen-receipt) screen_receipt=$2; shift 2;;
    --screen-analysis) screen_analysis=$2; shift 2;;
    --dry-run) dry=true; shift;;
    --execute) execute=true; shift;;
    *) printf 'Unknown argument: %s\n' "$1" >&2; exit 2;;
  esac
done
if [[ -z $candidate || -z $out || -z $source || -z $tool || -z $bridge || -z $tool_build ]]; then
  printf 'Required: --candidate ID.json --out NEW_DIR --source ROOT --tool claimtool --windows-bridge claimbridge.exe --tool-build-argv BUILD.json [--dry-run|--execute]\n' >&2
  exit 2
fi
if $dry && $execute; then printf 'Choose --dry-run or --execute.\n' >&2; exit 2; fi
if ! $dry && ! $execute; then
  printf 'Use --dry-run to check inputs; --execute is for a separately authorized %s workload.\n' "$mode" >&2
  exit 2
fi
args=(prepare --mode "$mode" --candidate "$candidate" --out "$out" --source "$source" --windows-bridge "$bridge" --tool-build-argv "$tool_build")
if [[ $mode == confirmation ]]; then
  if ! $dry && [[ -z $screen_receipt || -z $screen_analysis ]]; then
    printf 'Confirmation requires --screen-receipt and --screen-analysis.\n' >&2; exit 2
  fi
  args+=(--screen-receipt "$screen_receipt" --screen-analysis "$screen_analysis")
fi
if $dry; then args+=(--dry-run); fi
if $dry; then pinned=(taskset -c 13,15 nice -n 19)
else pinned=(taskset -c 0,2,4,6,8,10,12,14 nice -n 19)
fi
"${pinned[@]}" "$tool" "${args[@]}"
if $dry; then exit 0; fi
# Receipt and external digest seal now exist; the game directory cannot exist.
"${pinned[@]}" "$tool" run --receipt "$out/receipt.json" --games "$out/games" --out "$out/run-status.json"
"${pinned[@]}" "$tool" audit --receipt "$out/receipt.json" --games "$out/games" --out "$out/audit.json"
"${pinned[@]}" "$tool" analyze --receipt "$out/receipt.json" --games "$out/games" --audit "$out/audit.json" --out "$out/analysis.json"
"${pinned[@]}" "$tool" recheck --receipt "$out/receipt.json" --out "$out/post-run-hashes.json"

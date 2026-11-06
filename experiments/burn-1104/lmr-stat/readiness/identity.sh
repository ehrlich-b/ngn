#!/usr/bin/env bash
set -euo pipefail

repo=$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
candidate=d065c6c19b45c30933d37e63928d1149834d545c
baseline=$candidate^
out=
cpu=$(python3 -c 'import os; print(min(os.sched_getaffinity(0)))')
depth=10
while (($#)); do
    case "$1" in
        --baseline|--out|--cpu|--depth)
            if (($# < 2)); then
                echo "missing value for $1" >&2
                exit 2
            fi
            case "$1" in
                --baseline) baseline=$2 ;;
                --out) out=$2 ;;
                --cpu) cpu=$2 ;;
                --depth) depth=$2 ;;
            esac
            shift 2
            ;;
        *)
            echo "usage: $0 [--baseline REV] [--out FILE] [--cpu CPU] [--depth DEPTH]" >&2
            exit 2
            ;;
    esac
done
baseline=$(git -C "$repo" rev-parse --verify "$baseline^{commit}")
scratch=$(mktemp -d "${TMPDIR:?}/lmr-identity.XXXXXXXX")
mkdir "$scratch/baseline" "$scratch/candidate"
# Retain temporary sources and binaries for inspection; never build the caller's HEAD.
git -C "$repo" archive "$baseline" | tar -x -C "$scratch/baseline"
git -C "$repo" archive "$candidate" | tar -x -C "$scratch/candidate"
(cd "$scratch/baseline" && nice -n 10 go build -p 2 -o "$scratch/sp-base" ./cmd/speedprobe)
(cd "$scratch/candidate" && nice -n 10 go build -p 2 -o "$scratch/sp-new" ./cmd/speedprobe)
out=${out:-$scratch/identity.json}
printf 'baseline=%s\ncandidate=%s\nT1104LMRFractional=0 (compiled default; speedprobe sets no tunables)\nscratch=%s\noutput=%s\n' "$baseline" "$candidate" "$scratch" "$out"
nice -n 10 python3 "$scratch/candidate/scripts/speedcheck.py" \
    "$scratch/sp-base" "$scratch/sp-new" \
    --net "$scratch/candidate/engine/nets/ngn-h512-81215229.nnue" \
    --cpu "$cpu" --depth "$depth" --identity-only --out "$out"

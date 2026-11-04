#!/usr/bin/env bash
set -euo pipefail
if [ "${TTFIRST_CLEAN_ENV:-}" != 1 ]; then
  exec env -i PATH="$PATH" TTFIRST_CLEAN_ENV=1 bash "$0" "$@"
fi
repo=$(git rev-parse --show-toplevel)
lane=$(dirname "$repo")
folder="$repo/experiments/burn-1104/ttfirst"
mode=${1:-all}
case "$mode" in all|focused|corpus|gates) ;; *) exit 2;; esac
export GOCACHE=/work/cache/go-build GOMODCACHE=/work/cache/go-mod GOPATH=/work/cache/go GOMAXPROCS=2 TMPDIR="$lane/tmp" GOAMD64=v3
mkdir -p "$TMPDIR" "$lane/evidence"
scratch=$(mktemp -d "$TMPDIR/ttfirst.XXXXXX")
evidence=$(mktemp -d "$lane/evidence/check.XXXXXX")
echo "Evidence: $evidence"
exec > >(tee "$evidence/run.log") 2>&1
run() { echo "+ $*"; timeout 120s nice -n 10 "$@"; }
run git clone --quiet --shared --no-checkout "$repo" "$scratch/base"
run git -C "$scratch/base" checkout --quiet --detach faaf3741eaf0e2add133ffe6574a3c24db415907
run git clone --quiet --shared --no-checkout "$repo" "$scratch/candidate"
run git -C "$scratch/candidate" checkout --quiet --detach faaf3741eaf0e2add133ffe6574a3c24db415907
run git -C "$scratch/candidate" apply --check --whitespace=error-all "$folder/candidate.patch"
run git -C "$scratch/candidate" apply --whitespace=error-all "$folder/candidate.patch"
run python3 "$folder/verify.py" reference "$scratch/base" "$scratch/candidate"
if [[ "$mode" == all || "$mode" == corpus ]]; then
for tree in base candidate; do
  mkdir -p "$scratch/$tree/cmd/ttfirstprobe"
  cp "$folder/probe.go.txt" "$scratch/$tree/cmd/ttfirstprobe/main.go"
  (cd "$scratch/$tree" && run go build -trimpath -o "$scratch/probe-$tree" ./cmd/ttfirstprobe)
done
fi
if [[ "$mode" == all || "$mode" == focused ]]; then
(cd "$scratch/candidate" && run go test -p 2 -short -count=1 ./engine -run '^TestTTFirst' -v) > "$evidence/focused.log" 2>&1
cat "$evidence/focused.log"
run python3 "$folder/mutations.py" "$scratch/candidate" "$evidence"
fi
if [[ "$mode" == all || "$mode" == corpus ]]; then
for tree in base candidate; do
  run "$scratch/probe-$tree" "$folder/corpus.json" > "$evidence/$tree.stdout"
  # Strip only the runner's command announcement; probe records are JSON lines.
  python3 - "$evidence/$tree.stdout" "$evidence/$tree.jsonl" <<'PY'
import pathlib,sys,json
lines=pathlib.Path(sys.argv[1]).read_text().splitlines()
records=[json.loads(line) for line in lines if line.startswith('{')]
pathlib.Path(sys.argv[2]).write_text(''.join(json.dumps(r,sort_keys=True)+'\n' for r in records))
PY
done
run python3 "$folder/compare.py" "$evidence/base.jsonl" "$evidence/candidate.jsonl"
fi
if [[ "$mode" == all || "$mode" == gates ]]; then
(cd "$scratch/candidate" && run go vet -p 2 ./...) > "$evidence/vet.log" 2>&1
cat "$evidence/vet.log"
(cd "$scratch/candidate" && run go test -p 2 -short -count=1 ./...) > "$evidence/short.log" 2>&1
cat "$evidence/short.log"
fi
run python3 "$folder/verify.py" default
printf '%s\n' 'PASS: TT-first bounded correctness; full timing and adoption remain pending.'

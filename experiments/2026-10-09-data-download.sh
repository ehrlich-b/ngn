#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session19-data
REV=1e095a758c630bc58d0b6dac4da44fcd38ac89c2
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='data-volume',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$D/download.exit"; date -Is > "$D/download.done"; st download-end "rc $rc; see download.log"; }
trap finish EXIT
mkdir -p "$D/raw"
st download-start "pid $$; pinned T76 then T74; no training or games"
for name in farseerT76 farseerT74; do
  # The converter verifies each completed source before decoding.
  expected=6141321549; [[ $name != farseerT74 ]] || expected=20144023865
  if [[ -f "$D/raw/$name.binpack" && $(stat -c %s "$D/raw/$name.binpack") == "$expected" ]]; then continue; fi
  free=$(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ')
  # Both compressed inputs occupy 24.5 GiB. No older artifacts are deleted.
  [[ "$free" -ge $((55*1024*1024*1024)) ]]
  timeout --signal=TERM --kill-after=10s 3h taskset -c 13,15 nice -n 19 curl --fail --location --http1.1 --retry 3 --retry-delay 20 --speed-limit 100000 --speed-time 300 --output "$D/raw/$name.binpack" "https://huggingface.co/datasets/official-stockfish/master-binpacks/resolve/$REV/$name.binpack"
  st downloaded "$name; converter will verify its pinned hash before use"
done

#!/usr/bin/env bash
# Build and dry-check the v3 instrument from tested, clean committed source.
# Preparation only: no engines or games are launched by claimtool --dry-run.
set -euo pipefail
[[ $# == 2 ]] || { echo 'Usage: v3-freeze-build.sh BUNDLE EXPECTED_COMMIT' >&2; exit 2; }
BUNDLE=$1 EXPECTED=$2
R=/home/ehrli/ngn-data/night-20261007
S=$R/session2
V=/home/ehrli/ngn-data/claim-v3-20261008/tooling
SRC=$V/source
F=$V/frozen
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='v3-tool-build',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
trap 'rc=$?; [[ $rc == 0 ]] || st fail "rc $rc; inspect session2/build.log; no retry"' EXIT
python3 - "$S/test-receipt.json" <<'PY'
import json,sys
assert json.load(open(sys.argv[1]))['passed']
PY
st start "pid $$; source $EXPECTED; build and dry-run only"
mkdir "$V"
git clone --shared --no-checkout /home/ehrli/repos/ngn "$SRC"
git -C "$SRC" fetch "$BUNDLE" HEAD
git -C "$SRC" checkout --detach FETCH_HEAD
[[ $(git -C "$SRC" rev-parse HEAD) == "$EXPECTED" ]]
[[ -z $(git -C "$SRC" status --porcelain --untracked-files=all) ]]
mkdir "$F"
python3 - "$SRC" "$S/source" "$F" <<'PY'
import hashlib,json,subprocess,sys
from pathlib import Path
src,tested,out=map(Path,sys.argv[1:])
digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
inputs={}
for rel in subprocess.check_output(['git','-C',str(src),'ls-files'],text=True).splitlines():
    if rel.endswith(('.go','.s','.h','.c','.nnue')) or rel in ('go.mod','go.sum'):
        assert digest(src/rel)==digest(tested/rel),f'Untested compilation input: {rel}'
        inputs[rel]=digest(src/rel)
assert digest(src/'default.pgo')=='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0'
with (out/'tested-source-inputs.json').open('x') as f:json.dump(inputs,f,indent=2);f.write('\n')
PY
cd "$SRC"
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2
[[ $(go version) == 'go version go1.25.5 linux/amd64' ]]
taskset -c 13,15 nice -n 19 go build -buildvcs=true -p 2 -o "$F/claimtool" ./cmd/claimtool
taskset -c 13,15 nice -n 19 go build -buildvcs=true -p 2 -o "$F/ngn" .
go version -m "$F/claimtool" > "$F/claimtool-buildinfo.txt"
go version -m "$F/ngn" > "$F/ngn-buildinfo.txt"
cp "$R/kb8h512/best.nnue" "$F/candidate-be34532c.nnue"
chmod 555 "$F/claimtool" "$F/ngn"
chmod 444 "$F/candidate-be34532c.nnue"
python3 - "$F" "$R/screen-kb8h512-session1/identity.json" <<'PY'
import hashlib,json,sys
from pathlib import Path
f=Path(sys.argv[1]); prior=json.load(open(sys.argv[2]))
for name,target in [('claimtool','./cmd/claimtool'),('ngn','.')]:
    with (f/f'{name}-build.json').open('x') as out:
        json.dump(['/usr/local/go/bin/go','build','-buildvcs=true','-p','2','-o',str(f/name),target],out);out.write('\n')
with (f/'provenance.json').open('x') as out:json.dump(prior['Provenance'],out,indent=2);out.write('\n')
assert hashlib.sha256((f/'candidate-be34532c.nnue').read_bytes()).hexdigest()=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
PY
taskset -c 13,15 nice -n 19 "$F/claimtool" identity \
  --engine "$F/ngn" --engine-sha "$(sha256sum "$F/ngn" | cut -c1-64)" \
  --net "$F/candidate-be34532c.nnue" --net-sha be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a \
  --source "$SRC" --build-argv "$F/ngn-build.json" --provenance "$F/provenance.json" --out "$F/development-identity.json"
for mode in screen stability-screen stability-confirmation tooling-check; do
  taskset -c 13,15 nice -n 19 "$F/claimtool" prepare --mode "$mode" --candidate "$F/development-identity.json" \
    --source "$SRC" --tool-build-argv "$F/claimtool-build.json" --dry-run --out "$V/dry-$mode"
done
python3 - "$V" "$EXPECTED" <<'PY'
import hashlib,json,sys
from pathlib import Path
v=Path(sys.argv[1]); f=v/'frozen'
for mode in ('screen','stability-screen','stability-confirmation','tooling-check'):
    p=json.load(open(v/f'dry-{mode}/dry-run.json'))['Plan']
    assert p['Schema']=='NGN-CLAIM-3' and p['ClaimThreshold']==3600
    if mode=='screen':assert [a['Label'] for a in p['Anchors']]==[3317,3359,3557,3713]
assert not any(v.rglob('receipt.json')) and not any(v.rglob('games'))
artifacts={p.name:dict(path=str(p),sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in f.iterdir() if p.is_file()}
with (v/'build-receipt.json').open('x') as out:
    json.dump(dict(source_commit=sys.argv[2],source_clean=True,artifacts=artifacts,go_version='go1.25.5',
                   environment=dict(GOAMD64='v3',GOTOOLCHAIN='local',GOPROXY='off',GOMAXPROCS='2'),
                   build_cpu='13,15',nice=19,dry_modes=['screen','stability-screen','stability-confirmation','tooling-check'],
                   admission='Build and dry-run only; no operational games or final claim nomination.'),out,indent=2);out.write('\n')
PY
st done "clean source $EXPECTED; tooling/build-receipt.json; no v3 game launched"

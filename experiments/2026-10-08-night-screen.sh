#!/bin/bash
# v2 10+0.1 320-game 4-anchor screen for a night net: night-screen.sh NAME [TRAINER_PID_FILE] [--dry-run]
# Same frozen claim engine/tool/anchors/openings as the 0.3.0 screen; only the net changes.
# A running trainer is paused (SIGSTOP) for the screen so no game shares a CPU with it.
set -u
NAME=$1; PAUSE=${2:-}; MODE=${3:---execute}
C=/home/ehrli/ngn-data/claim-run-20261007
N=/home/ehrli/ngn-data/night-20261007
SRC=/home/ehrli/wt/claim-run-20261007
W=$N/screen-$NAME
[ "$MODE" = --dry-run ] && W=$N/screen-$NAME-dry
mkdir "$W" "$W/prov" || exit 1
export WSL_INTEROP=/run/WSL/2_interop  # the persistent interop socket (Counter 3.8 runs as a Windows exe)
cp $N/$NAME/best.nnue $W/candidate.nnue
cp $C/frozen/provenance/corpus-digest.json $W/prov/corpus-digest.json
cp $N/$NAME/history.json $W/prov/training-history.json
cp $N/$NAME/config.json $W/prov/training-config.json
cp $N/$NAME/parity.json $W/prov/parity.json
cp $N/$NAME.cmd $W/prov/manifest-training-command.txt
python3 - "$W" <<'P'
import hashlib, json, sys
w = sys.argv[1]
roles = dict(manifest='manifest-training-command.txt', data='corpus-digest.json', training='training-history.json',
             export='training-config.json', parity='parity.json')
out = {r: dict(Path=f'{w}/prov/{f}', SHA256=hashlib.sha256(open(f'{w}/prov/{f}', 'rb').read()).hexdigest()) for r, f in roles.items()}
json.dump(out, open(f'{w}/prov.json', 'w'), indent=1)
P
taskset -c 13,15 nice -n 19 $C/frozen/claimtool identity --engine $C/frozen/ngn --engine-sha 8fb36cfb68b78e4df65a1fc6b9f05232cb234418db1ab609b3a4468e12e22f4e \
  --net $W/candidate.nnue --net-sha $(sha256sum $W/candidate.nnue | cut -c1-64) --source $SRC \
  --build-argv $C/frozen/ngn-build.json --provenance $W/prov.json --out $W/identity.json || exit 1
[ -n "$PAUSE" ] && [ "$MODE" = --execute ] && kill -STOP $(cat $PAUSE)
bash $SRC/scripts/claim/screen.sh --candidate $W/identity.json --source $SRC --tool $C/frozen/claimtool \
  --windows-bridge $C/frozen/claimbridge.exe --tool-build-argv $C/frozen/claimtool-build.json $MODE --out $W/screen
rc=$?
[ -n "$PAUSE" ] && [ "$MODE" = --execute ] && kill -CONT $(cat $PAUSE)
[ -f $W/screen/analysis.json ] && python3 -c "
import json; a = json.load(open('$W/screen/analysis.json'))
print(json.dumps(dict(ScreenEligible=a.get('ScreenEligible'), Bracketed=a.get('Bracketed'), Primary=a.get('Primary'))))"
exit $rc

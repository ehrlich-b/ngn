#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s51
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=1 GOPROXY=off GOTOOLCHAIN=local
export NGN_PRESSURE_NET=/work/lanes/ngn-spsa-20261009/kb8bin/src/engine/nets/ngn-kb8h512-be34532c.nnue
cd "$S"
st() { python3 - status.jsonl "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session51-tt-pressure',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > pressure.exit; date -Is > pressure.done; st end "rc $rc; diagnostic only, no strength verdict"; }
trap finish EXIT
date -Is > pressure.started
mkdir legacy own
tar -xf base.tar -C legacy
tar -xf base.tar -C own
(cd own && git apply ../candidate.patch)
python3 pressure-stage.py "$S"
for design in legacy own; do
 (cd "$design" && gofmt -w engine/campaign_pressure_test.go engine/campaign_pressure_support_test.go)
 (cd "$design" && timeout 10m go test -c -p 2 -pgo="$S/$design/default.pgo" -o "$S/$design-test" ./engine) > "$design-build.log" 2>&1
 NGN_PRESSURE_TT="$design" timeout 10m "./$design-test" -test.run '^TestCampaignPressureControls$' -test.v -test.timeout 9m > "$design-controls.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
names=['base.tar','candidate.patch','pressure-stage.py','pressure-test.go.txt','pressure.md','pressure.sh','legacy-test','own-test','legacy-controls.log','own-controls.log']
for design in ['legacy','own']:
 names += [design+'/engine/'+n for n in ['cache.go','hash.go','campaign_pressure_test.go','campaign_pressure_support_test.go']]
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source_snapshot='a49ebfc',purpose='archived TT/key fixed-position diagnosis only',machine='forge Debian13 VM/Ryzen6900HX',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',pgo_sha256=sha(p/'own/default.pgo'),cpus=[5,6,7],nice=10,memory_max_bytes=2*1024**3,host_pinning='on',hash_mb=64,net_sha256=sha(Path('/work/lanes/ngn-spsa-20261009/kb8bin/src/engine/nets/ngn-kb8h512-be34532c.nnue')),concurrent_workloads=['SPSA88166/88178 forge8-11/2.5GiB/two games','reader117631 unchanged','rig TT games and factorizer separate machine'],cells=[{'tt':design,'keys':keys,'depth':20,'reports_at_depths':[16,20],'fens':8,'argv':['./'+design+'-test','-test.run','^TestCampaignPressureSearch$','-test.v','-test.timeout','29m'],'environment':{'GOMAXPROCS':'1','NGN_PRESSURE_TT':design,'NGN_PRESSURE_KEYS':keys,'NGN_PRESSURE_OUT':str(p/(design+'-'+keys+'.jsonl'))}} for design in ['legacy','own'] for keys in ['old','new']],raw_counted_depth10_identity=True,files_sha256={n:sha(p/n) for n in names},decision='diagnostic only; no acceptance threshold, adoption, sample extension or retry')
(p/'pressure-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st controls-pass 'legacy/own raw-counted depth10 identities and counter reconciliation; four frozen cells next'
for design in legacy own; do
 for keys in old new; do
  st cell-start "$design/$keys cold depth20, reports16/20, eight FENs"
  NGN_PRESSURE_TT="$design" NGN_PRESSURE_KEYS="$keys" NGN_PRESSURE_OUT="$S/$design-$keys.jsonl" timeout --signal=TERM --kill-after=10s 30m "./$design-test" -test.run '^TestCampaignPressureSearch$' -test.v -test.timeout 29m > "$design-$keys.log" 2>&1
  st cell-done "$design/$keys complete"
 done
done

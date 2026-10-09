#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session29
mkdir "$S"
cd "$R"
python3 - "$R" <<'PY'
import datetime,json,os,subprocess,sys
from pathlib import Path
r=Path(sys.argv[1]);free=os.statvfs('/mnt/c');free=free.f_bavail*free.f_frsize
assert free>=32*1024**3,free
ps=subprocess.check_output(['ps','-eo','pid,ppid,args'],text=True)
assert not any('/spsa/spsa -engine' in x or 'bash '+str(r)+'/session' in x for x in ps.splitlines()),ps
checks=['session21/board/receipt.json','session21/key/game-manifest.json','session21/see/receipt.json','session21/pawn/receipt.json','session22-row/build-receipt.json','session23-time/trace-manifest.json','session23-qstatic/receipt.json','session24-continuation/build-receipt.json','session25-tt-scan/build-receipt.json']
assert not any((r/f).exists() for f in checks),'completed work exists: resume its stage instead'
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),host_c_free_bytes=free,spsa_checkpoint=json.loads((r/'spsa/kb8/spsa.jsonl').read_text().splitlines()[-1]),interruption='Bryan-approved WSL shutdown/compaction; all controllers were waiting before first heavy stage',processes=ps.splitlines())
(r/'session29/pre-recovery.json').write_text(json.dumps(j,indent=2)+'\n')
# Preserve misleading zero exit markers written during shutdown separately.
for directory,prefixes in [('session21',['controller','checks','data']),('session21/data',['controller']),('session22-row',['controller']),('session23-time',['controller']),('session23-qstatic',['controller']),('session24-continuation',['controller']),('session25-tt-scan',['controller'])]:
 p=r/directory;stash=p/'wsl-interrupted-session29';stash.mkdir()
 for stem in prefixes:
  for suffix in ['started','pid','exit','done','log']:
   f=p/(stem+'.'+suffix)
   if f.is_file():f.rename(stash/f.name)
PY
SP=$R/spsa/kb8
cp "$SP/spsa.pid" "$S/pre-spsa.pid"
setsid nohup bash -c "$(cat "$SP/spsa.cmd") -resume" < /dev/null >> "$SP/spsa.log" 2>&1 &
echo $! > "$SP/spsa.pid"
launch() {
 local dir=$1 script=$2 stem=$3
 (cd "$dir"; setsid nohup bash "$script" < /dev/null > "$stem.log" 2>&1 & echo $! > "$stem.pid")
}
launch "$R/session21" controller.sh controller
launch "$R/session21" checks.sh checks
launch "$R/session21" data.sh data
for d in session22-row session23-time session23-qstatic session24-continuation session25-tt-scan; do
 launch "$R/$d" controller.sh controller
done
python3 - "$R" <<'PY'
import datetime,json,sys
from pathlib import Path
r=Path(sys.argv[1]);j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session29-recovery',event='relaunched',detail='WSL compaction interrupted wait-only controllers; same manifests/scripts/inputs, no completed work repeated; SPSA legacy resume from iter1225; data waits for70GiB',pids={p:int((r/p).read_text()) for p in ['spsa/kb8/spsa.pid','session21/controller.pid','session21/checks.pid','session21/data.pid',*[d+'/controller.pid' for d in ['session22-row','session23-time','session23-qstatic','session24-continuation','session25-tt-scan']]]})
(r/'session29/launch.json').write_text(json.dumps(j,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(j)+'\n')
print(json.dumps(j,indent=2))
PY

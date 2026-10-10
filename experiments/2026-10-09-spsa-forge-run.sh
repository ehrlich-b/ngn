#!/usr/bin/env bash
# Detached inside the authorized four-CPU/5G forge scope.
set -euo pipefail
R=/work/lanes/ngn-spsa-20261009
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=1 GOPROXY=off GOTOOLCHAIN=local
cd "$R"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
from pathlib import Path
p=Path('status.jsonl')
with p.open('a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='spsa-forge',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; final-vector games still required"; }
trap finish EXIT
date -Is > controller.started
st wait 'native completion of the separate forge code-check scope; then frozen SPSA continuation'
python3 - <<'PY'
import os,select
from pathlib import Path
if not Path('checks.done').exists():
 fd=os.pidfd_open(int(Path('prepare.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(60*60*1000), 'code-check scope still running; no tune overlap'
 os.close(fd)
# These independent time/key checks cannot invalidate the already verified
# frozen legacy engine or bounded tuner. Preserve their outcome for review.
assert Path('checks.done').exists()
PY
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
saved=json.loads(Path('spsa.jsonl').read_text().splitlines()[-1])
assert sha('spsa.jsonl')==Path('checkpoint.sha256').read_text().split()[0]
assert sha('openings.txt')=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
args=['./spsa-forge','-engine',str(Path('ngn-kb8').resolve()),'-tc','10+0.1','-iters','2000','-pairs','6','-concurrency','2','-openings',str(Path('openings.txt').resolve()),'-out',str(Path('spsa.jsonl').resolve()),'-lowpower=false','-seed','20261008','-resume']
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),machine='forge / Ryzen6900HX Debian13 VM',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',cpus=[8,9,10,11],cpuquota='400%',memory_max='5G',swap_max=0,nice=10,environment={'GOMAXPROCS':'1'},argv=args,checkpoint=saved,durable_games=saved['iter']*12,endpoint_games=24000,simultaneous_games=2,engine_processes=4,hash_mb=128,legacy_rng_reset_on_resume=True,independent_checks_exit=int(Path('checks.exit').read_text()),decision='no optimizer acceptance; converged-vector games on rig after endpoint',files_sha256={n:sha(n) for n in ['spsa-forge','ngn-kb8','spsa.jsonl','openings.txt','worker.patch','controller.sh']})
Path('launch-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st start 'frozen old-base legacy tune; pairs6/update, two concurrent games; machine change recorded; no strength verdict'
./spsa-forge -engine "$R/ngn-kb8" -tc 10+0.1 -iters 2000 -pairs 6 -concurrency 2 -openings "$R/openings.txt" -out "$R/spsa.jsonl" -lowpower=false -seed 20261008 -resume > spsa.log 2>&1

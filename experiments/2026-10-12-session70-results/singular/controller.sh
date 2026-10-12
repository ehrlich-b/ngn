#!/usr/bin/env bash
set -euo pipefail
s=$1
taskset -pc 4-11 $$ > "$s/controller-affinity.txt"
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/controller.exit"; date -Is > "$s/controller.done"; }
trap finish EXIT
date -Is > controller.started
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
python3 - <<'PY'
import ctypes,datetime,json,os,select
from pathlib import Path
p=Path.cwd(); dependency=Path('/work/lanes/ngn-s69-admission-final14')
def st(event,detail):
    with (p/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session69-singular-correction',event=event,detail=detail))+'\n')
st('waiting-for-cpu-release','admission checks only; no game starts while NGN checks own forge')
lib=ctypes.CDLL(None);fd=lib.inotify_init1(os.O_NONBLOCK)
assert fd>=0 and lib.inotify_add_watch(fd,os.fsencode(dependency),0x8|0x80)>=0
while not (dependency/'checks.done').exists():
    select.select([fd],[],[])
    os.read(fd,65536)
os.close(fd)
st('cpu-released',dict(admission_exit=(dependency/'checks.exit').read_text().strip(),automatic_admission_promotion=False))
PY
python3 games.py

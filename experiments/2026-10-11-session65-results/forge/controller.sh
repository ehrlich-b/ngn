#!/usr/bin/env bash
set -euo pipefail
s=/work/lanes/ngn-s65-data-aa
cd "$s"
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
python3 - <<'PY'
import ctypes,os,select,pathlib,time
p=pathlib.Path('/work/lanes/ngn-s65-pairmatch');lib=ctypes.CDLL(None);fd=lib.inotify_init1(os.O_NONBLOCK);lib.inotify_add_watch(fd,os.fsencode(p),0x8|0x80)
end=time.monotonic()+1800
while not (p/'final.done').exists():
 t=end-time.monotonic();assert t>0
 assert select.select([fd],[],[],t)[0]
 os.read(fd,65536)
assert (p/'final.exit').read_text().strip()=='0','source checks failed; control not admitted'
PY
date -Is > controller.started
python3 aa.py

#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-consensus-gate
cd "$D"
finish(){ rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; }
trap finish EXIT
export GOMAXPROCS=1 GOAMD64=v3 PYTHONDONTWRITEBYTECODE=1
# Tests use SMT siblings of two S-game cores; await their natural terminal.
python3 - <<'PYWAIT'
import ctypes,os,pathlib,select
p=pathlib.Path('/home/ehrli/ngn-data/night-20261007/session77-next-smp')
if not (p/'checks.done').exists():
 libc=ctypes.CDLL(None);fd=libc.inotify_init1(os.O_NONBLOCK|os.O_CLOEXEC)
 assert fd>=0 and libc.inotify_add_watch(fd,os.fsencode(p),0x8|0x80)>=0
 try:
  while not (p/'checks.done').exists():
   assert select.select([fd],[],[],24*3600)[0];os.read(fd,65536)
 finally:os.close(fd)
PYWAIT
python3 games.py

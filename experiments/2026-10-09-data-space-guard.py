"""Keep C: above its campaign floor while this one corpus controller writes."""
import datetime,json,os,select,signal,sys,time
from pathlib import Path
parent=int(sys.argv[1]);journal=Path(sys.argv[2])
fd=os.pidfd_open(parent);poll=select.poll();poll.register(fd,select.POLLIN)
while not poll.poll(60000):
    disk=os.statvfs('/mnt/c');free=disk.f_bavail*disk.f_frsize
    if free>=32*1024**3:continue
    # Only descendants of this exact campaign controller are stopped.
    links={}
    for p in Path('/proc').iterdir():
        if not p.name.isdigit():continue
        try:links[int(p.name)]=int((p/'stat').read_text().rsplit(')',1)[1].split()[1])
        except (FileNotFoundError,ProcessLookupError,PermissionError):continue
    owned={parent}
    while True:
        children={p for p,pp in links.items() if pp in owned}
        if children<=owned:break
        owned|=children
    with journal.open('a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='data-volume',event='space-stop',detail=f'C: {free} bytes; stop own corpus group before 25GiB floor'))+'\n')
    for p in owned-{parent}:
        try:os.kill(p,signal.SIGTERM)
        except ProcessLookupError:pass
    time.sleep(10)
    for p in owned-{parent}:
        try:os.kill(p,signal.SIGKILL)
        except ProcessLookupError:pass
    break
os.close(fd)

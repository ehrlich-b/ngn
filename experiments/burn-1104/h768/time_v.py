#!/usr/bin/env python3
"""Run proof commands with a clean environment and native wait4 accounting."""
import json, os, re, shlex, subprocess, sys, time
from pathlib import Path
scratch_root=Path(os.environ.get('TMPDIR',str(Path.cwd().parent/'tmp')))

def environment():
    env={k:v for k,v in os.environ.items() if not re.search(r'credential|token|secret|password',k,re.I)}
    env.update(GOCACHE='/work/cache/go-build',GOMODCACHE='/work/cache/go-mod',GOPATH='/work/cache/go',GOMAXPROCS='2',npm_config_cache='/work/cache/npm',UV_CACHE_DIR='/work/cache/uv',UV_PYTHON_INSTALL_DIR='/work/cache/uv-python',TMPDIR=str(scratch_root),PLAYWRIGHT_BROWSERS_PATH='/work/cache/ms-playwright',PYTHONDONTWRITEBYTECODE='1')
    env['PATH']='/usr/local/go/bin:'+env.get('PATH',os.defpath)
    return env

def run(args, log):
    log=Path(log); log.parent.mkdir(parents=True,exist_ok=True)
    started=time.monotonic()
    with log.open('w') as out:
        out.write('$ '+shlex.join(args)+'\n'); out.flush()
        process=subprocess.Popen(args,env=environment(),stdout=out,stderr=subprocess.STDOUT)
        _,status,usage=os.wait4(process.pid,0)
        rc=os.waitstatus_to_exitcode(status); process.returncode=rc
        stats=dict(exit_code=rc,elapsed_seconds=time.monotonic()-started,user_seconds=usage.ru_utime,system_seconds=usage.ru_stime,peak_rss_kib=usage.ru_maxrss)
        out.write('\n'+json.dumps(stats,sort_keys=True)+'\n')
    print(log, json.dumps(stats),flush=True)
    return rc

if __name__=='__main__':
    sys.exit(run(sys.argv[2:],sys.argv[1]))

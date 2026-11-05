#!/usr/bin/env python3
"""Execute a proof on two CPU cores; emit native wait4 resource measurements."""
import json
import os
from pathlib import Path
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from scripts.wave2.net_job import environment  # noqa: E402


def main():
    command = sys.argv[1:]
    if not command:
        raise SystemExit("expected an executable command")
    cpus = sorted(os.sched_getaffinity(0))[:2]
    start = time.monotonic()
    sys.stdout.flush()
    sys.stderr.flush()
    pid = os.fork()
    if pid == 0:
        os.sched_setaffinity(0, cpus)
        try:
            os.execvpe(command[0], command, environment())
        except OSError as error:
            print(str(error), file=sys.stderr, flush=True)
            os._exit(127)
    _, status, usage = os.wait4(pid, 0)
    code = os.waitstatus_to_exitcode(status)
    print(json.dumps({"command": command, "exit_code": code, "cpus": cpus,
                      "elapsed_seconds": round(time.monotonic() - start, 6),
                      "user_seconds": usage.ru_utime, "system_seconds": usage.ru_stime,
                      "peak_rss_kib": usage.ru_maxrss}), flush=True)
    return code if code >= 0 else 128 - code


if __name__ == "__main__":
    raise SystemExit(main())

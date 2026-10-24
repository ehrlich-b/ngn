#!/usr/bin/env python3
"""Native wait4 timing and shared-host telemetry; never invokes /usr/bin/time."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

from resources import capture_resource_evidence
from resources import external_busy_cpu_estimate


def environment():
    env = {k: v for k, v in os.environ.items()
           if not re.search(r"credential|token|secret|password", k, re.I)}
    env.update(GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod",
               GOPATH="/work/cache/go", GOMAXPROCS="2", GOAMD64="v3",
               PYTHONDONTWRITEBYTECODE="1",
               npm_config_cache="/work/cache/npm", UV_CACHE_DIR="/work/cache/uv",
               UV_PYTHON_INSTALL_DIR="/work/cache/uv-python")
    return env


def telemetry(pid=None):
    paths = ["/proc/stat", "/proc/loadavg", "/proc/pressure/cpu", "/proc/meminfo",
             "/proc/self/cgroup"]
    try:
        for line in Path("/proc/self/cgroup").read_text().splitlines():
            if line.startswith("0::"):
                base = Path("/sys/fs/cgroup") / line[3:].lstrip("/")
                paths += [str(base / name) for name in
                          ["cpu.stat", "cpu.max", "cpu.pressure", "cpuset.cpus.effective"]]
    except OSError:
        pass
    if pid:
        paths += [f"/proc/{pid}/{name}" for name in ["stat", "schedstat", "status"]]
        try:
            paths += [str(p / "schedstat") for p in Path(f"/proc/{pid}/task").iterdir()]
        except OSError:
            pass
    result = {"monotonic": time.monotonic(), "unix": time.time()}
    for path in paths:
        try:
            result[path] = Path(path).read_text()
        except OSError as exc:
            result[path] = {"unavailable": str(exc)}
    return result


def run(command, folder, env=None, cwd=None):
    folder = Path(folder)
    folder.mkdir(parents=True, exist_ok=False)
    env = dict(environment() if env is None else env)
    env["TMPDIR"] = str(folder.resolve() / "tmp")
    Path(env["TMPDIR"]).mkdir()
    samples = []
    start = time.monotonic()
    with (folder / "stdout.txt").open("wb") as out, (folder / "stderr.txt").open("wb") as err:
        proc = subprocess.Popen(["nice", "-n", "10", *command], cwd=cwd, env=env,
                                stdout=out, stderr=err)
        resource_evidence = capture_resource_evidence(proc.pid)
        resource_evidence["gomaxprocs"] = env.get("GOMAXPROCS")
        try:
            while True:
                samples.append(telemetry(proc.pid))
                pid, status, usage = os.wait4(proc.pid, os.WNOHANG)
                if pid:
                    break
                time.sleep(0.25)
        except BaseException:
            proc.kill()
            os.wait4(proc.pid, 0)
            raise
        proc.returncode = os.waitstatus_to_exitcode(status)
    receipt = {"command": command, "exit_code": proc.returncode,
               "elapsed_seconds": time.monotonic()-start,
               "user_seconds": usage.ru_utime, "system_seconds": usage.ru_stime,
               "peak_rss_kib": usage.ru_maxrss,
               "voluntary_context_switches": usage.ru_nvcsw,
               "involuntary_context_switches": usage.ru_nivcsw}
    receipt["measurement_interval_seconds"] = receipt["elapsed_seconds"]
    receipt["resource_evidence"] = resource_evidence
    def host_busy(sample):
        ticks = list(map(int, sample["/proc/stat"].splitlines()[0].split()[1:]))
        # Guest counters are already included in user/nice; idle and iowait excluded.
        return sum(ticks[i] for i in [0, 1, 2, 5, 6, 7]) / os.sysconf("SC_CLK_TCK")
    busy = host_busy(samples[-1]) - host_busy(samples[0])
    receipt["host_busy_cpu_seconds"] = busy
    receipt["non_owned_busy_cpu_seconds_estimate"] = external_busy_cpu_estimate(
        busy, usage.ru_utime + usage.ru_stime)
    (folder / "time.json").write_text(json.dumps(receipt, indent=2)+"\n")
    (folder / "telemetry.json").write_text(json.dumps(samples, indent=2)+"\n")
    if proc.returncode:
        raise RuntimeError(f"exit {proc.returncode}: {command}; see {folder}")
    return receipt


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command:
        parser.error("provide a command after --")
    args.output.mkdir(parents=True, exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix="run-", dir=args.output.resolve()))
    receipt = run(command, folder / "execution")
    print(json.dumps({"evidence":str(folder / "execution"), **receipt}))

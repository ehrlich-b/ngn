#!/usr/bin/env python3
"""Native wait4 timing and shared-host telemetry; never invokes /usr/bin/time."""
from collections.abc import Mapping, Sequence
import json
import math
from numbers import Real
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import sys
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


def stop_owned_group(proc):
    """Stop our private session, including helpers whose parent already exited.

    Helpers that deliberately create another session escape this ownership
    contract. Never discover groups by name or use the caller's process group.
    """
    try:
        try:
            os.killpg(proc.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        else:
            # Give cooperative helpers a bounded grace period.
            time.sleep(0.1)
    finally:
        # Escalate even if the grace period is interrupted. The original leader
        # need not still be alive, and a helper may ignore TERM.
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        finally:
            # Native wait4 sets returncode on success; wait tolerates ECHILD
            # when a failure happened just after a native reap.
            proc.wait(timeout=2)



class CommandTimeoutError(TimeoutError):
    """A profiling command exceeded its wall-clock limit."""


def validate_command(command):
    if (isinstance(command, (str, bytes)) or not isinstance(command, Sequence)
            or not command):
        raise ValueError("command must be a nonempty sequence of string arguments")
    if any(not isinstance(arg, str) for arg in command):
        raise ValueError("command arguments must all be strings")
    if not command[0]:
        raise ValueError("command executable must be a nonempty string")
    if any("\0" in arg for arg in command):
        raise ValueError("command arguments must not contain NUL")


def validate_cwd(cwd):
    if cwd is None:
        return None
    try:
        path = os.fspath(cwd)
        if not stat.S_ISDIR(os.stat(path).st_mode):
            raise NotADirectoryError(f"not a directory: {path!r}")
    except (TypeError, ValueError, OSError) as exc:
        raise ValueError(f"invalid cwd {cwd!r}: {exc}") from exc
    return path


def run(command, folder, env=None, cwd=None, timeout_seconds=None):
    if isinstance(command, (str, bytes)) or not isinstance(command, Sequence):
        raise ValueError("command must be a nonempty sequence of string arguments")
    return_tuple_command = isinstance(command, tuple)
    command = list(command)
    validate_command(command)
    if timeout_seconds is not None and (
        isinstance(timeout_seconds, bool)
        or not isinstance(timeout_seconds, (int, float))
        or timeout_seconds <= 0
        or (isinstance(timeout_seconds, float) and not math.isfinite(timeout_seconds))
    ):
        raise ValueError("timeout_seconds must be None or a positive finite int or float (excluding bool)")
    if env is None:
        env = environment()
    else:
        if not isinstance(env, Mapping):
            raise ValueError("env must be a mapping of strings to strings")
        env = dict(env)
        for key, value in env.items():
            if (not isinstance(key, str) or not key or "=" in key or "\0" in key
                    or not isinstance(value, str) or "\0" in value):
                raise ValueError("env contains an invalid key or value")
    cwd = validate_cwd(cwd)
    folder = Path(folder)
    folder.mkdir(parents=True, exist_ok=False)
    env["TMPDIR"] = str(folder.resolve() / "tmp")
    Path(env["TMPDIR"]).mkdir()
    measurement_failure = None

    def measure(stage, operation, *args):
        nonlocal measurement_failure
        try:
            return operation(*args)
        except BaseException as exc:
            measurement_failure = exc
            try:
                (folder / "failure.json").write_text(json.dumps({
                    "stage": stage, "exception_class": type(exc).__name__[:128],
                }, indent=2) + "\n")
            except BaseException:
                try:
                    exc.add_note("Could not write failure.json")
                except BaseException:
                    pass
            raise

    samples = []
    start = time.monotonic()
    with (folder / "stdout.txt").open("wb") as out, (folder / "stderr.txt").open("wb") as err:
        proc = None
        try:
            try:
                proc = subprocess.Popen(["nice", "-n", "10", *command], cwd=cwd, env=env,
                                        stdout=out, stderr=err, start_new_session=True)
            except OSError as exc:
                try:
                    (folder / "failure.json").write_text(json.dumps({
                        "stage": "spawn", "exception_class": type(exc).__name__,
                        "message": str(exc)[:1024],
                    }, indent=2) + "\n")
                except Exception as diagnostic_error:
                    exc.add_note(f"Could not write failure.json: {diagnostic_error!r}")
                raise
            resource_evidence = measure("capture_resource_evidence", capture_resource_evidence, proc.pid)
            resource_evidence["gomaxprocs"] = env.get("GOMAXPROCS")
            while True:
                if timeout_seconds is not None and time.monotonic() - start >= timeout_seconds:
                    raise CommandTimeoutError(
                        f"command exceeded wall timeout of {timeout_seconds} seconds: {command}; see {folder}")
                samples.append(measure("telemetry", telemetry, proc.pid))
                if timeout_seconds is not None and time.monotonic() - start >= timeout_seconds:
                    raise CommandTimeoutError(
                        f"command exceeded wall timeout of {timeout_seconds} seconds: {command}; see {folder}")
                next_sample = time.monotonic() + 0.25
                while True:
                    now = time.monotonic()
                    if timeout_seconds is not None and now - start >= timeout_seconds:
                        raise CommandTimeoutError(
                            f"command exceeded wall timeout of {timeout_seconds} seconds: {command}; see {folder}")
                    pid, status, usage = os.wait4(proc.pid, os.WNOHANG)
                    if pid:
                        proc.returncode = os.waitstatus_to_exitcode(status)
                        break
                    now = time.monotonic()
                    if timeout_seconds is not None and now - start >= timeout_seconds:
                        raise CommandTimeoutError(
                            f"command exceeded wall timeout of {timeout_seconds} seconds: {command}; see {folder}")
                    if now >= next_sample:
                        break
                    delay = next_sample - now
                    if timeout_seconds is not None:
                        delay = min(delay, 0.01)
                        if timeout_seconds <= now - start + delay:
                            delay = min(delay, timeout_seconds - (now - start))
                    time.sleep(max(0, delay))
                if pid:
                    break
        except CommandTimeoutError as exc:
            if exc is measurement_failure:
                raise
            try:
                (folder / "failure.json").write_text(json.dumps({
                    "stage": "timeout", "error": "wall timeout",
                    "timeout_seconds": timeout_seconds,
                    "elapsed_seconds": time.monotonic() - start,
                }, indent=2) + "\n")
            except Exception as diagnostic_error:
                exc.add_note(f"Could not write failure.json: {diagnostic_error!r}")
            raise
        finally:
            if proc is not None:
                original = sys.exc_info()[1]
                try:
                    stop_owned_group(proc)
                except BaseException:
                    # A cleanup interruption must not replace the failure that
                    # triggered cleanup. Without an earlier failure, report it.
                    if original is None:
                        raise
        # Include the completed command and owned-group cleanup in host accounting.
        samples.append(measure("telemetry", telemetry, proc.pid))
    receipt = {"command": command, "exit_code": proc.returncode,
               "elapsed_seconds": time.monotonic()-start,
               "user_seconds": usage.ru_utime, "system_seconds": usage.ru_stime,
               "peak_rss_kib": usage.ru_maxrss,
               "voluntary_context_switches": usage.ru_nvcsw,
               "involuntary_context_switches": usage.ru_nivcsw}
    receipt["measurement_interval_seconds"] = receipt["elapsed_seconds"]
    receipt["resource_evidence"] = resource_evidence
    def host_counters(sample, endpoint):
        stat = sample.get("/proc/stat")
        if not isinstance(stat, str):
            raise ValueError(f"{endpoint} /proc/stat unavailable: {stat!r}")
        lines = stat.splitlines()
        fields = lines[0].split() if lines else []
        if len(fields) < 9 or fields[0] != "cpu":
            raise ValueError(f"{endpoint} /proc/stat has a malformed aggregate CPU line")
        try:
            ticks = [int(value) for value in fields[1:9]]
        except ValueError:
            raise ValueError(f"{endpoint} /proc/stat has nonnumeric CPU counters") from None
        if any(tick < 0 for tick in ticks):
            raise ValueError(f"{endpoint} /proc/stat has negative CPU counters")
        return ticks

    try:
        first = host_counters(samples[0], "start")
        last = host_counters(samples[-1], "end")
        # Guest counters are already included in user/nice; idle and iowait excluded.
        busy_fields = [0, 1, 2, 5, 6, 7]
        if any(last[i] < first[i] for i in busy_fields):
            raise ValueError("host CPU counters decreased between start and end")
        try:
            ticks_per_second = os.sysconf("SC_CLK_TCK")
        except (OSError, ValueError):
            raise ValueError("SC_CLK_TCK unavailable") from None
        try:
            valid_ticks = (not isinstance(ticks_per_second, bool)
                           and isinstance(ticks_per_second, Real)
                           and ticks_per_second > 0
                           and math.isfinite(ticks_per_second))
        except (OverflowError, ValueError):
            valid_ticks = False
        if not valid_ticks:
            raise ValueError("SC_CLK_TCK must be a finite positive real number")
        busy = sum(last[i] - first[i] for i in busy_fields) / ticks_per_second
    except ValueError as exc:
        receipt["host_busy_cpu_seconds"] = None
        receipt["non_owned_busy_cpu_seconds_estimate"] = None
        receipt["host_busy_cpu_seconds_unavailable_reason"] = str(exc)
    else:
        receipt["host_busy_cpu_seconds"] = busy
        receipt["non_owned_busy_cpu_seconds_estimate"] = external_busy_cpu_estimate(
            busy, usage.ru_utime + usage.ru_stime)
    (folder / "time.json").write_text(json.dumps(receipt, indent=2)+"\n")
    (folder / "telemetry.json").write_text(json.dumps(samples, indent=2)+"\n")
    if proc.returncode:
        raise RuntimeError(f"exit {proc.returncode}: {command}; see {folder}")
    if return_tuple_command:
        receipt["command"] = tuple(command)
    return receipt


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--timeout-seconds", type=float, default=None)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    try:
        validate_command(command)
    except ValueError as exc:
        parser.error(str(exc))
    if args.timeout_seconds is not None and (not math.isfinite(args.timeout_seconds) or args.timeout_seconds <= 0):
        parser.error("--timeout-seconds must be finite and positive")
    args.output.mkdir(parents=True, exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix="run-", dir=args.output.resolve()))
    receipt = run(command, folder / "execution", timeout_seconds=args.timeout_seconds)
    print(json.dumps({"evidence":str(folder / "execution"), **receipt}))

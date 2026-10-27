"""Offline parsing and reporting for SMP execution capacity evidence."""
import math
import os
from pathlib import Path


def _read(path):
    try:
        return Path(path).read_text()
    except (OSError, ValueError) as exc:
        return {"unavailable": f"{type(exc).__name__}: {exc}"}


def _unescape_mount(value):
    return value.replace("\\040", " ").replace("\\011", "\t").replace("\\134", "\\")


def capture_resource_evidence(pid=None):
    """Collect raw references; safe to call without a model or a search."""
    pid = pid or os.getpid()
    prefix = f"/proc/{pid}"
    evidence = {"affinity": None, "affinity_reference": f"sched_getaffinity({pid})",
                "cgroup_reference": f"{prefix}/cgroup", "mountinfo_reference": f"{prefix}/mountinfo",
                "files": {}}
    try:
        evidence["affinity"] = sorted(os.sched_getaffinity(pid))
    except (AttributeError, OSError) as exc:
        evidence["affinity_error"] = f"{type(exc).__name__}: {exc}"
    for key, path in (("cgroup", f"{prefix}/cgroup"), ("mountinfo", f"{prefix}/mountinfo")):
        try:
            evidence[key] = Path(path).read_text()
        except OSError as exc:
            evidence[key] = {"unavailable": f"{type(exc).__name__}: {exc}"}
    if isinstance(evidence["cgroup"], str) and isinstance(evidence["mountinfo"], str):
        for mount, root, rel, version, controllers in _mounts(evidence["mountinfo"], evidence["cgroup"]):
            current = os.path.normpath(os.path.join(mount, os.path.relpath(rel, root)))
            while current == mount or current.startswith(mount.rstrip("/") + "/"):
                names = (["cpu.max", "cpuset.cpus.effective", "cpuset.cpus"] if version == 2 else
                         (["cpu.cfs_quota_us", "cpu.cfs_period_us"] if "cpu" in controllers else []) +
                         (["cpuset.cpus", "cpuset.effective_cpus"] if "cpuset" in controllers else []))
                for name in names:
                    path = os.path.join(current, name)
                    evidence["files"][path] = _read(path)
                if current == mount:
                    break
                current = os.path.dirname(current)
    return evidence


def _mounts(mountinfo, cgroup):
    groups = []
    for line in cgroup.splitlines():
        parts = line.split(":", 2)
        if len(parts) == 3:
            groups.append((parts[1].split(",") if parts[1] else [], os.path.normpath("/" + parts[2].lstrip("/"))))
    for line in mountinfo.splitlines():
        halves = line.split(" - ", 1)
        if len(halves) != 2:
            continue
        left, right = halves[0].split(), halves[1].split()
        if len(left) < 5 or len(right) < 3 or right[0] not in ("cgroup", "cgroup2"):
            continue
        mount, root = map(_unescape_mount, (left[4], left[3]))
        version = 2 if right[0] == "cgroup2" else 1
        controllers = set((right[2] if version == 1 else "").split(",")) - {""}
        for available, path in groups:
            if (version == 2 and not available) or (version == 1 and controllers.intersection(available)):
                if path == root or path.startswith(root.rstrip("/") + "/"):
                    yield mount, root, path, version, controllers


def _read_value(value):
    if isinstance(value, dict):
        return None
    return value.strip() if isinstance(value, str) else None


def _count_cpus(value):
    value = _read_value(value)
    if value is None or value == "":
        return None
    try:
        cpus = set()
        for chunk in value.split(","):
            bounds = chunk.split("-")
            if not all(token and token.isascii() and token.isdecimal() for token in bounds):
                return None
            if len(bounds) == 1:
                cpus.add(int(bounds[0]))
            elif len(bounds) == 2:
                lo, hi = map(int, bounds)
                if lo < 0 or hi < lo or hi - lo > 1_000_000:
                    return None
                cpus.update(range(lo, hi + 1))
            else:
                return None
        return len(cpus)
    except (ValueError, TypeError):
        return None


def _quota_summary(evidence):
    """Return observed bounds and completeness of visible quota evidence only."""
    files = evidence.get("files") or {}
    paths = {p for p in files if os.path.basename(p) in
             ("cpu.max", "cpu.cfs_quota_us", "cpu.cfs_period_us")}
    levels = {os.path.join(os.path.dirname(p), "cpu.cfs_quota_us")
              if os.path.basename(p).startswith("cpu.cfs_") else p for p in paths}
    cgroup, mountinfo = evidence.get("cgroup"), evidence.get("mountinfo")
    if isinstance(cgroup, str) and isinstance(mountinfo, str):
        mounts = list(_mounts(mountinfo, cgroup))
        v1_cpu = any(version == 1 and "cpu" in controllers
                     for _, _, _, version, controllers in mounts)
        if v1_cpu:
            # A controller belongs to only one hierarchy on hybrid hosts.
            levels = {p for p in levels if os.path.basename(p) != "cpu.max"}
        for mount, root, rel, version, controllers in mounts:
            if version == 1 and "cpu" not in controllers:
                continue
            if version == 2 and v1_cpu:
                continue
            current = os.path.normpath(os.path.join(mount, os.path.relpath(rel, root)))
            while current == mount or current.startswith(mount.rstrip("/") + "/"):
                path = os.path.join(current, "cpu.max" if version == 2 else "cpu.cfs_quota_us")
                raw = files.get(path)
                missing = path not in files or (isinstance(raw, dict) and
                          str(raw.get("unavailable", "")).startswith("FileNotFoundError:"))
                # cpu.max is not created at the v2 hierarchy root. A subtree
                # mount still has a quota, and permission failures stay unknown.
                if version == 2 and root == "/" and current == mount and missing:
                    levels.discard(path)
                else:
                    levels.add(path)
                if current == mount:
                    break
                current = os.path.dirname(current)
    candidates, unknown = [], []
    for path in sorted(levels):
        value = _read_value(files.get(path))
        try:
            if os.path.basename(path) == "cpu.max":
                fields = value.split() if value is not None else []
                if len(fields) != 2:
                    raise ValueError("invalid cpu.max")
                token, period_token = fields
                unlimited = token == "max"
            else:
                token = value
                period_token = _read_value(files.get(os.path.join(os.path.dirname(path), "cpu.cfs_period_us")))
                unlimited = token == "-1"
            # Kernel counters use decimal integers; reject signs, floats and extra tokens.
            if not period_token or not period_token.isascii() or not period_token.isdecimal() or int(period_token) <= 0:
                raise ValueError("invalid period")
            if not unlimited:
                if not token or not token.isascii() or not token.isdecimal() or int(token) <= 0:
                    raise ValueError("invalid quota")
                bound = int(token) / int(period_token)
                if not math.isfinite(bound) or bound <= 0:
                    raise ValueError("unrepresentable quota bound")
                candidates.append((bound, path))
        except (TypeError, ValueError, OverflowError):
            unknown.append(path)
    complete = bool(levels) and not unknown
    state = "limited" if candidates else "unlimited" if complete else "unknown"
    return candidates, state, complete, unknown, sorted(paths | levels)


def summarize_resources(evidence, requested_width):
    """Pure summary of captured raw evidence. Unknown inputs stay unknown."""
    files = evidence.get("files") or {}
    quota_candidates, quota_state, quota_complete, quota_unknown, quota_paths = _quota_summary(evidence)
    cpuset_counts = []
    for path, raw in files.items():
        name = os.path.basename(path)
        if name in ("cpuset.cpus.effective", "cpuset.effective_cpus", "cpuset.cpus"):
            count = _count_cpus(raw)
            if count is not None:
                cpuset_counts.append((count, path))
    affinity = evidence.get("affinity")
    affinity_count = len(affinity) if isinstance(affinity, (list, tuple, set)) else None
    affinity_count = affinity_count if affinity_count and all(isinstance(x, int) for x in affinity) else None
    quota = min(quota_candidates, default=None, key=lambda item: item[0])
    cpuset = min(cpuset_counts, default=None, key=lambda item: item[0])
    gomax = evidence.get("gomaxprocs")
    try:
        gomax = int(gomax) if gomax is not None else None
        if gomax <= 0:
            gomax = None
    except (TypeError, ValueError):
        gomax = None
    unknowns = []
    if gomax is None:
        unknowns.append("subprocess_gomaxprocs")
    if affinity_count is None:
        unknowns.append("affinity_cpu_count")
    if cpuset is None:
        unknowns.append("cpuset_cpu_count")
    if not quota_complete:
        unknowns.append("cgroup_cpu_quota_cores")
    return {
        "subprocess_gomaxprocs": gomax,
        "requested_worker_width": requested_width,
        "effective_worker_width": evidence.get("effective_worker_width", requested_width),
        "affinity_cpu_count": affinity_count,
        "affinity_evidence": evidence.get("affinity_reference", "unknown"),
        "cpuset_cpu_count": cpuset[0] if cpuset else None,
        "cpuset_evidence": [p for _, p in cpuset_counts] or
                           [p for p in files if os.path.basename(p).startswith("cpuset.")],
        "cgroup_cpu_quota_cores": quota[0] if quota else None,
        "cgroup_cpu_quota_state": quota_state,
        "cgroup_cpu_quota_evidence_complete": quota_complete,
        "cgroup_cpu_quota_unknown_levels": quota_unknown,
        "cgroup_quota_evidence": quota_paths,
        "capacity_is_exclusive_physical_cores": False,
        "core_scaling_qualified": False,
        "unknown_capacity_fields": unknowns,
        "raw_evidence": {"cgroup": evidence.get("cgroup_reference", "unknown"),
                          "mountinfo": evidence.get("mountinfo_reference", "unknown"),
                          "files": files},
    }


def external_busy_cpu_estimate(host_busy_cpu_seconds, owned_cpu_seconds):
    """Coarse shared-host estimate; does not attribute work to a process."""
    if host_busy_cpu_seconds is None or owned_cpu_seconds is None:
        return None
    try:
        host = float(host_busy_cpu_seconds)
        owned = float(owned_cpu_seconds)
    except (TypeError, ValueError, OverflowError):
        return None
    if not math.isfinite(host) or not math.isfinite(owned) or host < 0 or owned < 0:
        return None
    return max(0.0, host - owned)

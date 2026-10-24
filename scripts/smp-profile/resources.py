"""Offline parsing and reporting for SMP execution capacity evidence."""
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


def summarize_resources(evidence, requested_width):
    """Pure summary of captured raw evidence. Unknown inputs stay unknown."""
    files = evidence.get("files") or {}
    quota_candidates = []
    cpuset_counts = []
    for path, raw in files.items():
        name = os.path.basename(path)
        value = _read_value(raw)
        if name == "cpu.max" and value:
            fields = value.split()
            if len(fields) == 2 and fields[0] != "max":
                try:
                    quota, period = int(fields[0]), int(fields[1])
                    if quota > 0 and period > 0:
                        quota_candidates.append((quota / period, path))
                except ValueError:
                    pass
        elif name == "cpu.cfs_quota_us":
            period_path = os.path.join(os.path.dirname(path), "cpu.cfs_period_us")
            period_raw = _read_value(files.get(period_path))
            try:
                quota, period = int(value), int(period_raw)
                if quota > 0 and period > 0:
                    quota_candidates.append((quota / period, path))
            except (TypeError, ValueError):
                pass
        elif name in ("cpuset.cpus.effective", "cpuset.effective_cpus", "cpuset.cpus"):
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
    effective = min(x for x in (affinity_count, cpuset[0] if cpuset else None,
                                quota[0] if quota else None) if x is not None) if any(
                                    x is not None for x in (affinity_count, cpuset[0] if cpuset else None,
                                                            quota[0] if quota else None)) else None
    unknowns = []
    if gomax is None:
        unknowns.append("subprocess_gomaxprocs")
    if affinity_count is None:
        unknowns.append("affinity_cpu_count")
    if cpuset is None:
        unknowns.append("cpuset_cpu_count")
    if quota is None:
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
        "cgroup_quota_evidence": [p for _, p in quota_candidates] or
                                 [p for p in files if os.path.basename(p) in ("cpu.max", "cpu.cfs_quota_us", "cpu.cfs_period_us")],
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
        return max(0.0, float(host_busy_cpu_seconds) - float(owned_cpu_seconds))
    except (TypeError, ValueError):
        return None

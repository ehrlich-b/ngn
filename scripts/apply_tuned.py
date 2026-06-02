#!/usr/bin/env python3
"""Splice a `texel -mode gradient/tune` output blob into the engine source.

Usage: scripts/apply_tuned.py <tuned_blob.go> [--dry-run]

The blob contains `var pestoMGMaterial = [7]int{...}`, `var pestoEGMaterial = ...`,
the 12 `var <name>Table = [64]int{...}` PST blocks, and (after a
`// eval.go tunable weights:` marker) tab-indented `name = value` weight lines.
Each `var` block is matched by name and replaces the corresponding declaration in
engine/eval_pesto.go (order-independent); each weight replaces its definition in
engine/eval.go (the `\tname = N` line, never a usage site). Git is the undo.
"""
import re
import sys
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PESTO = os.path.join(ROOT, "engine", "eval_pesto.go")
EVAL = os.path.join(ROOT, "engine", "eval.go")


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    dry = "--dry-run" in sys.argv
    if len(args) != 1:
        sys.exit("usage: apply_tuned.py <tuned_blob.go> [--dry-run]")
    blob = open(args[0]).read()

    # Parse `var NAME = [N]int{ ... }` declarations. The body is `[^}]*` (flat int
    # arrays have no nested braces, and [^}] spans newlines), so this matches single-
    # line material arrays and multi-line tables uniformly without over-running.
    var_decls = re.findall(r"var (\w+) = (\[\d+\]int\{[^}]*\})", blob)
    # Parse weight lines after the marker.
    wmarker = blob.find("// eval.go tunable weights:")
    weights = []
    if wmarker >= 0:
        weights = re.findall(r"^\s*(\w+)\s*=\s*(-?\d+)\s*$", blob[wmarker:], re.MULTILINE)

    if not var_decls and not weights:
        sys.exit("no var declarations or weights found in blob")

    pesto = open(PESTO).read()
    changed = 0
    for name, decl in var_decls:
        pat = re.compile(r"var %s = \[\d+\]int\{[^}]*\}" % re.escape(name))
        if not pat.search(pesto):
            sys.exit("ERROR: %s not found in %s" % (name, PESTO))
        new = "var %s = %s" % (name, decl)
        pesto, n = pat.subn(new, pesto, count=1)
        if n:
            changed += 1
            print("  eval_pesto.go: %s" % name)

    evalsrc = open(EVAL).read()
    for name, val in weights:
        pat = re.compile(r"^(\t+%s\s*=\s*)-?\d+" % re.escape(name), re.MULTILINE)
        if not pat.search(evalsrc):
            sys.exit("ERROR: weight %s definition not found in %s" % (name, EVAL))
        evalsrc, n = pat.subn(r"\g<1>%s" % val, evalsrc, count=1)
        if n:
            changed += 1
            print("  eval.go: %s = %s" % (name, val))

    print("%d declarations spliced%s" % (changed, " (dry-run, not written)" if dry else ""))
    if not dry:
        open(PESTO, "w").write(pesto)
        open(EVAL, "w").write(evalsrc)


if __name__ == "__main__":
    main()

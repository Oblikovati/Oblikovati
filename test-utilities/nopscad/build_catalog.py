#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
#
# build_catalog.py — static-analysis catalog of every NopSCADlib part module.
#
# GOAL of the whole effort: re-model each OpenSCAD NopSCADlib part as a NATIVE
# Oblikovati feature-based design (a .obk with a sketch→feature tree), using the
# OpenSCAD source only as the reference to reproduce. This catalog is the ordered
# work queue for that re-modelling (low->high complexity). See PORTING.md.
#
# Scans vitamins/, printed/ and utils/ .scad files, scores each module's
# geometric complexity with a deterministic rubric, classifies its part-class
# (solid / assembly / bom-stub) and the kernel-gap flags its operations hit,
# then emits catalog.json (machine queue for the re-modelling) and CATALOG.md
# (human-readable, sorted low->high complexity).
#
# This is step 1 of the NopSCADlib-driven re-modelling + kernel-hardening program.
# It does NOT render geometry — golden STLs are rendered lazily per tier by
# render_goldens.py. Run:  python3 build_catalog.py /path/to/NopSCADlib

import json
import os
import re
import sys

# Complexity rubric: weight per operation token. Mirrors the plan's scoring.
OP_WEIGHTS = {
    "cube": 1, "cylinder": 1, "sphere": 1, "circle": 1, "square": 1,
    "polygon": 1,
    "difference": 1, "union": 1, "intersection": 1,
    "linear_extrude": 2, "offset": 2,
    "rotate_extrude": 3,
    "hull": 4,
    "polyhedron": 5, "minkowski": 5, "sweep": 5,
}
# Thread helpers are the heaviest single signal (helix + profile + taper).
THREAD_TOKENS = ("metric_thread", "metric_thread(", "thread(", "_thread")
# Tokens that force the advanced tier regardless of additive score.
ADVANCED_TOKENS = ("sweep", "minkowski", "polyhedron")

# Each op maps to the kernel area it exercises / the gap it may hit.
GAP_FLAGS = {
    "hull": "needs-convex-hull/boundary-blend op",
    "minkowski": "needs-3d-offset op",
    "offset": "needs-2d-sketch-offset",
    "sweep": "sweep-polyhedron-stability",
    "polyhedron": "direct-polyhedron-input",
    "rotate_extrude": "revolve",
}
THREAD_GAP = "needs-first-class-thread-feature"

MODULE_RE = re.compile(r"^\s*module\s+([A-Za-z0-9_]+)\s*\(", re.MULTILINE)
FUNC_RE = re.compile(r"\bfunction\s+([A-Za-z0-9_]+)\s*\(")
ACCESSOR_RE = re.compile(r"function\s+\w+\s*\(\s*type\s*\)\s*=\s*type\[")
PARAM_NAME_RE = re.compile(r"function\s+(\w+)\s*\(\s*type\s*\)\s*=\s*type\[\d+\]")
CALL_RE = re.compile(r"\b([a-z][A-Za-z0-9_]*)\s*\(")

# OpenSCAD builtins + NopSCADlib core helpers that never make a module an
# "assembly" — transforms, BOM/color wrappers, math, control flow.
BUILTINS = {
    "translate", "rotate", "scale", "mirror", "color", "resize", "offset",
    "linear_extrude", "rotate_extrude", "hull", "minkowski", "difference",
    "union", "intersection", "render", "children", "for", "if", "let",
    "echo", "assert", "str", "is_undef", "is_list", "len", "concat", "each",
    "cube", "cylinder", "sphere", "circle", "square", "polygon", "polyhedron",
    "text", "projection", "surface", "import", "min", "max", "abs", "sin",
    "cos", "tan", "atan", "atan2", "asin", "acos", "sqrt", "pow", "norm",
    "cross", "sign", "floor", "ceil", "round", "ln", "log", "exp",
    "vitamin", "translate_z", "vflip", "no_explode", "grey", "is_num",
}


def strip_comments(src: str) -> str:
    src = re.sub(r"/\*.*?\*/", "", src, flags=re.DOTALL)
    src = re.sub(r"//[^\n]*", "", src)
    return src


def count_ops(body: str) -> dict:
    counts = {}
    for op in OP_WEIGHTS:
        # word-boundary match so 'circle' doesn't catch 'semicircle' etc.
        n = len(re.findall(r"\b" + op + r"\s*\(", body))
        if n:
            counts[op] = n
    counts["for"] = len(re.findall(r"\bfor\s*\(", body))
    if any(t in body for t in THREAD_TOKENS):
        counts["thread"] = 1
    return counts


def score(counts: dict) -> int:
    s = 0
    for op, n in counts.items():
        if op == "for":
            s += n  # +1 per pattern loop
        elif op == "thread":
            s += 6
        else:
            s += OP_WEIGHTS.get(op, 0) * n
    return s


def tier(s: int, counts: dict) -> int:
    if counts.get("thread") or any(t in counts for t in ADVANCED_TOKENS):
        return 4
    if s <= 2:
        return 0
    if s <= 6:
        return 1
    if s <= 12:
        return 2
    if s <= 20:
        return 3
    return 4


def classify(src: str, body: str, local_funcs: set) -> str:
    if re.search(r"\bimport\s*\(", body):
        return "bom-stub"  # imported STL, no procedural geometry to re-model
    has_prim = any(re.search(r"\b" + op + r"\s*\(", body) for op in OP_WEIGHTS)
    # foreign call = a lowercase identifier( that is not a primitive/builtin
    # and not a local function/accessor (those just read the type vector).
    foreign = {c for c in CALL_RE.findall(body)
               if c not in BUILTINS and c not in local_funcs}
    if not has_prim:
        # no geometry of its own: composition (assembly) or pure BOM line.
        return "assembly" if foreign else "bom-stub"
    # has its own geometry; foreign part calls make it a composed assembly.
    # children() passthrough alone (washer/nut/screw) stays a solid.
    return "assembly" if foreign else "solid"


def gap_flags(counts: dict) -> list:
    flags = []
    for op in counts:
        if op in GAP_FLAGS:
            flags.append(GAP_FLAGS[op])
    if counts.get("thread"):
        flags.append(THREAD_GAP)
    return sorted(set(flags))


def params_for(src: str) -> list:
    # NopSCADlib exposes a part's dimensions as type[] accessor functions.
    return PARAM_NAME_RE.findall(src)


def analyze_file(path: str, rel: str) -> list:
    with open(path, encoding="utf-8", errors="replace") as f:
        raw = f.read()
    src = strip_comments(raw)
    modules = MODULE_RE.findall(src)
    if not modules:
        return []
    params = params_for(src)
    # local names (functions + sibling modules in this file) are never
    # "foreign" — only calls into OTHER part files mark a true assembly.
    local_names = set(FUNC_RE.findall(src)) | set(modules)
    entries = []
    for name in modules:
        # crude per-module body slice: from this module decl to the next one.
        start = src.find("module " + name)
        nxt = src.find("\nmodule ", start + 1)
        body = src[start: nxt if nxt != -1 else len(src)]
        counts = count_ops(body)
        if not counts or (len(counts) == 1 and "for" in counts):
            continue  # no geometry tokens -> skip helper/echo-only module
        s = score(counts)
        entries.append({
            "path": rel,
            "module": name,
            "params": params,
            "ops": counts,
            "score": s,
            "tier": tier(s, counts),
            "class": classify(src, body, local_names),
            "gap_flags": gap_flags(counts),
            "golden": None,  # filled by render_goldens.py when tested
        })
    return entries


def main(root: str):
    out_dir = os.path.dirname(os.path.abspath(__file__))
    entries = []
    for sub in ("vitamins", "printed", "utils", "utils/core"):
        d = os.path.join(root, sub)
        if not os.path.isdir(d):
            continue
        for fn in sorted(os.listdir(d)):
            if not fn.endswith(".scad"):
                continue
            rel = os.path.join(sub, fn)
            entries.extend(analyze_file(os.path.join(d, fn), rel))

    entries.sort(key=lambda e: (e["tier"], e["score"], e["path"], e["module"]))
    with open(os.path.join(out_dir, "catalog.json"), "w") as f:
        json.dump(entries, f, indent=2)

    write_markdown(os.path.join(out_dir, "CATALOG.md"), entries)
    by_tier = {}
    for e in entries:
        by_tier.setdefault(e["tier"], 0)
        by_tier[e["tier"]] += 1
    print("modules cataloged:", len(entries))
    for t in sorted(by_tier):
        print(f"  tier {t}: {by_tier[t]}")


def write_markdown(path: str, entries: list):
    tier_names = {0: "Trivial", 1: "Simple", 2: "Moderate",
                  3: "Complex", 4: "Advanced"}
    lines = ["# NopSCADlib Part Catalog (complexity-sorted)", "",
             "**Goal: re-model each OpenSCAD NopSCADlib part as a native Oblikovati "
             "feature-based design** (a `.obk` document with a real feature tree — "
             "sketch → extrude/revolve/cut/…), NOT a mesh import. The OpenSCAD "
             "source is the reference to reproduce; the deliverable is a parametric "
             "Oblikovati part. See PORTING.md.", "",
             "Generated by `build_catalog.py`. One row per geometric module, "
             "sorted low->high complexity (the order to re-model them in). `golden` is "
             "the rendered STL reference used to check a re-modelled part against the "
             "OpenSCAD original.", ""]
    cur = None
    for e in entries:
        if e["tier"] != cur:
            cur = e["tier"]
            lines += ["", f"## Tier {cur} — {tier_names[cur]}", "",
                      "| module | path | class | score | ops | gap flags |",
                      "|---|---|---|---|---|---|"]
        ops = ", ".join(f"{k}×{v}" for k, v in sorted(e["ops"].items()))
        gaps = ", ".join(e["gap_flags"]) or "—"
        lines.append(f"| `{e['module']}` | {e['path']} | {e['class']} | "
                     f"{e['score']} | {ops} | {gaps} |")
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


if __name__ == "__main__":
    root = sys.argv[1] if len(sys.argv) > 1 else \
        os.path.join(os.path.dirname(__file__),
                     "../../../NopSCADlib")
    main(os.path.abspath(root))

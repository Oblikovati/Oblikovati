#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
#
# render_goldens.py — render canonical single-instance STL goldens from
# NopSCADlib modules using the OpenSCAD CLI.
#
# Golden STLs are the *canonical expected output* a kernel-built body is asserted
# against (volume/area/bbox). We render one canonical instance per module (not the
# whole family layout the NopSCADlib tests/*.scad use) so the metric is a single
# solid. Faceting is pinned ($fa/$fs) for determinism.
#
# Invocations are declared per module in INSTANCES below, seeded lazily as each
# tier is brought under test (the program renders goldens on demand, not all 442
# modules up front). Each entry is the body of a wrapper that `include`s the
# NopSCADlib master lib (which pulls every data table + module) and calls the
# module once with canonical arguments.
#
# Usage:  python3 render_goldens.py [module ...]   (default: all declared)

import os
import shutil
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pathsafe import safe_name  # noqa: E402  (after sys.path bootstrap)

HERE = os.path.dirname(os.path.abspath(__file__))
NOPSCAD = os.path.abspath(os.path.join(HERE, "../../../NopSCADlib"))
GOLDENS = os.path.join(HERE, "goldens")

# Pinned faceting → deterministic tessellation across machines/runs.
HEADER = "$fa = 6; $fs = 0.25;\ninclude <{lib}>\n".format(
    lib=os.path.join(NOPSCAD, "lib.scad"))

# module -> canonical single-instance call. Grow this per tier.
INSTANCES = {
    # Tier 0/1 — simple solids
    "bearing_ball": "sphere(d = 5);",  # a 5mm ball; matches a steel ball BOM
    "washer": "washer(M3_washer);",
    "star_washer": "star_washer(M3_washer);",
    # rod(d,l): a smooth rod with 45-degree chamfered ends (chamfer = d/10).
    "rod": "rod(6, 20);",
    # O_ring(id, minor_d): a torus (revolved circle); centreline R = id/2 + minor_d/4.
    "o_ring": "O_ring(20, 3);",
    # Hex nut blank: a hexagonal prism (across-flats 10, so across-corners 10/cos30)
    # with a central through hole (Ø5), height 5 — a regular-polygon + hole part.
    "hex_nut": "difference() { cylinder(d = 10/cos(30), h = 5, $fn = 6); "
               "translate([0,0,-1]) cylinder(d = 5, h = 7); }",
}


def render(module: str, body: str) -> bool:
    os.makedirs(GOLDENS, exist_ok=True)
    # module is caller-supplied (argv); pin it to a bare name so the derived
    # .scad/.stl paths and the OpenSCAD invocation cannot escape GOLDENS.
    name = safe_name(module)
    scad = os.path.join(GOLDENS, name + ".scad")
    stl = os.path.join(GOLDENS, name + ".stl")
    with open(scad, "w") as f:
        f.write(HEADER + body + "\n")
    try:
        subprocess.run(openscad_cmd() + ["-o", stl, scad],
                       check=True, capture_output=True, text=True, timeout=300)
    except subprocess.CalledProcessError as e:
        print(f"  FAIL {module}: {e.stderr.strip().splitlines()[-1:]}", file=sys.stderr)
        return False
    finally:
        if os.path.exists(scad):
            os.remove(scad)  # keep only the .stl golden
    size = os.path.getsize(stl)
    print(f"  ok   {module}  ({size} bytes)")
    return True


def openscad_cmd() -> list[str]:
    env = os.environ.get("OPENSCAD")
    if env:
        return env.split()
    if shutil.which("openscad"):
        return ["openscad"]
    if shutil.which("flatpak-spawn"):
        return ["flatpak-spawn", "--host", "openscad"]
    return ["openscad"]


def main(argv):
    targets = argv or sorted(INSTANCES)
    missing = [m for m in targets if m not in INSTANCES]
    if missing:
        print("no INSTANCES entry for:", ", ".join(missing), file=sys.stderr)
        return 2
    ok = all(render(m, INSTANCES[m]) for m in targets)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

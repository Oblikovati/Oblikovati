#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Report volume and bounds for binary or ASCII STL files.

Usage:
  python3 stl_metrics.py part.stl [part2.stl ...]
"""

import json
import math
import struct
import sys
from pathlib import Path


def read_stl(path: Path):
    data = path.read_bytes()
    if len(data) >= 84:
        tri_count = struct.unpack_from("<I", data, 80)[0]
        if 84 + tri_count * 50 == len(data):
            return read_binary_stl(data, tri_count)
    return read_ascii_stl(data.decode("utf-8", errors="replace"))


def read_binary_stl(data: bytes, tri_count: int):
    triangles = []
    offset = 84
    for _ in range(tri_count):
        values = struct.unpack_from("<12fH", data, offset)
        triangles.append((values[3:6], values[6:9], values[9:12]))
        offset += 50
    return triangles


def read_ascii_stl(text: str):
    vertices = []
    triangles = []
    for line in text.splitlines():
        fields = line.strip().split()
        if len(fields) == 4 and fields[0] == "vertex":
            vertices.append(tuple(float(x) for x in fields[1:4]))
            if len(vertices) == 3:
                triangles.append(tuple(vertices))
                vertices = []
    return triangles


def metrics(triangles):
    volume = 0.0
    mins = [math.inf, math.inf, math.inf]
    maxs = [-math.inf, -math.inf, -math.inf]
    for a, b, c in triangles:
        volume += dot(a, cross(b, c)) / 6.0
        for p in (a, b, c):
            for axis in range(3):
                mins[axis] = min(mins[axis], p[axis])
                maxs[axis] = max(maxs[axis], p[axis])
    return {
        "triangles": len(triangles),
        "volume_mm3": abs(volume),
        "volume_cm3": abs(volume) / 1000.0,
        "bounds_mm": {"min": mins, "max": maxs},
    }


def cross(a, b):
    return (
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    )


def dot(a, b):
    return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]


def main(argv):
    if not argv:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    for name in argv:
        path = Path(name)
        print(json.dumps({"path": str(path), **metrics(read_stl(path))}, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
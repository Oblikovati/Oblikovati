#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Tests for pathsafe — run: python3 test_pathsafe.py (exits non-zero on failure)."""

import os
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pathsafe import resolved_within, safe_name  # noqa: E402


def _expect_value_error(fn, *args):
    try:
        fn(*args)
    except ValueError:
        return
    raise AssertionError(f"{fn.__name__}{args!r} should have raised ValueError")


def test_safe_name_accepts_bare_names():
    assert safe_name("hex_nut") == "hex_nut"
    assert safe_name("o_ring.v2") == "o_ring.v2"


def test_safe_name_rejects_traversal_and_separators():
    for bad in ("", ".", "..", "../etc/passwd", "a/b", "/abs", "x\0y".replace("\0", "/")):
        _expect_value_error(safe_name, bad)


def test_resolved_within_accepts_paths_inside_base():
    with tempfile.TemporaryDirectory() as base:
        full = resolved_within("sub/part.stl", base)
        assert full.startswith(os.path.realpath(base) + os.sep)
        assert resolved_within(base, base) == os.path.realpath(base)


def test_resolved_within_rejects_escapes():
    with tempfile.TemporaryDirectory() as base:
        _expect_value_error(resolved_within, "../../etc/passwd", base)
        _expect_value_error(resolved_within, "/etc/passwd", base)


def main():
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
    print(f"ok  {len(tests)} pathsafe tests passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())

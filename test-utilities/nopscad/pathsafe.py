#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Path-safety helpers for the NopSCADlib test utilities.

These scripts take file paths and module names off the command line. A caller
(a human, or an automated agent) could pass a value that escapes the directory
the script means to read or write — ``../../etc/passwd``, an absolute path, or a
name carrying a path separator. Validate such inputs here so a constructed path
can never resolve outside its intended base directory before any filesystem or
subprocess access (SonarCloud pythonsecurity:S8705/S8707).
"""

import os


def safe_name(name: str) -> str:
    """Return ``name`` when it is a single, separator-free filename stem; else raise.

    Use before joining a caller-supplied name onto a base directory:
    ``safe_name("hex_nut")`` -> ``"hex_nut"``; ``safe_name("../etc/x")`` raises.
    """
    if not name or name in (".", "..") or name != os.path.basename(name):
        raise ValueError(
            f"unsafe name {name!r}: want a bare filename with no path separators")
    return name


def resolved_within(path: str, base: str) -> str:
    """Return the absolute ``path`` once confirmed to sit inside ``base``; else raise.

    Resolves symlinks and ``..`` first, so a path that escapes the base is
    rejected before any access: ``resolved_within("a/b.stl", a_dir)`` returns the
    absolute path; ``resolved_within("../../etc/passwd", a_dir)`` raises.
    """
    base_real = os.path.realpath(base)
    target = path if os.path.isabs(path) else os.path.join(base_real, path)
    full = os.path.realpath(target)
    if full != base_real and not full.startswith(base_real + os.sep):
        raise ValueError(
            f"unsafe path {path!r}: resolves outside base directory {base!r}")
    return full

# SPDX-License-Identifier: GPL-2.0-only
"""Stage capture dependency caches against one observed lockfile."""

import os
import stat
from pathlib import Path
from typing import Mapping

from capture_build_environment import source_cargo_home
from capture_git_cache import stage_git_cache
from capture_registry_cache import stage_registry_cache


def read_capture_lock(source: Path) -> bytes:
    """Read a regular lockfile without following links; example: read_capture_lock(source)."""
    path = source / "Cargo.lock"
    if not hasattr(os, "O_NOFOLLOW"):
        raise ValueError("Cargo lock open: expected no-follow support")
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | getattr(os, "O_NONBLOCK", 0))
    try:
        if not stat.S_ISREG(os.fstat(descriptor).st_mode):
            raise ValueError(f"Cargo lock {path}: expected a regular file")
        with os.fdopen(descriptor, "rb", closefd=False) as file:
            return file.read()
    finally:
        os.close(descriptor)


def require_same_capture_lock(source: Path, expected: bytes) -> None:
    """Reject a changed lockfile; example: require_same_capture_lock(source, before)."""
    if read_capture_lock(source) != expected:
        raise ValueError(f"Cargo lock {source / 'Cargo.lock'}: expected unchanged capture lockfile")


def stage_capture_dependencies(source: Path, environment: Mapping[str, str], settings: Mapping[str, str]) -> bytes:
    """Stage dependencies using one lockfile read; example: stage_capture_dependencies(source, env, settings)."""
    lock = read_capture_lock(source)
    cache = source_cargo_home(environment)
    destination = Path(settings["CARGO_HOME"])
    stage_registry_cache(lock, cache, destination)
    stage_git_cache(lock, cache, destination)
    require_same_capture_lock(source, lock)
    return lock

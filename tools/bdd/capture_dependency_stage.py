# SPDX-License-Identifier: GPL-2.0-only
"""Stage capture dependency caches against one observed lockfile."""

from pathlib import Path
from typing import Mapping

from capture_build_environment import source_cargo_home
from capture_git_cache import stage_git_cache
from capture_registry_cache import stage_registry_cache


def stage_capture_dependencies(source: Path, environment: Mapping[str, str], settings: Mapping[str, str]) -> None:
    """Stage dependencies using one lockfile read; example: stage_capture_dependencies(source, env, settings)."""
    lock = (source / "Cargo.lock").read_bytes()
    cache = source_cargo_home(environment)
    destination = Path(settings["CARGO_HOME"])
    stage_registry_cache(lock, cache, destination)
    stage_git_cache(lock, cache, destination)

# SPDX-License-Identifier: GPL-2.0-only
"""Copy locked registry archives into a fresh capture Cargo home."""

import hashlib
import os
import stat
from pathlib import Path

from registry_vendor_check import REGISTRY, LockedPackage, locked_packages


def cache_directories(cargo_home: Path) -> list[Path]:
    """Find cached registry namespaces; example: cache_directories(cargo_home)."""
    cache = cargo_home / "registry" / "cache"
    if not cache.is_dir():
        return []
    return sorted(path for path in cache.iterdir() if path.is_dir())


def copy_archive(source: Path, target: Path, package: LockedPackage) -> None:
    """Copy bytes only if they match the lock checksum; example: copy_archive(src, dst, package)."""
    if not hasattr(os, "O_NOFOLLOW"):
        raise ValueError("archive open: expected operating-system no-follow support")
    descriptor = os.open(source, os.O_RDONLY | os.O_NOFOLLOW | getattr(os, "O_NONBLOCK", 0))
    try:
        if not stat.S_ISREG(os.fstat(descriptor).st_mode):
            raise ValueError(f"archive {source}: expected a regular file")
        with os.fdopen(descriptor, "rb", closefd=False) as content, target.open("xb") as output:
            digest = hashlib.sha256()
            while chunk := content.read(1024 * 1024):
                digest.update(chunk)
                output.write(chunk)
        if digest.hexdigest() != package.checksum:
            raise ValueError(f"archive {source}: expected locked SHA-256 {package.checksum}")
    finally:
        os.close(descriptor)


def copy_package(package: LockedPackage, caches: list[Path], destination: Path) -> None:
    """Copy each cached namespace for a locked package; example: copy_package(package, caches, dst)."""
    name = f"{package.name}-{package.version}.crate"
    present = [directory for directory in caches if (directory / name).exists() or (directory / name).is_symlink()]
    if not present:
        raise ValueError(f"archive {name}: expected a cached locked registry package")
    for directory in present:
        folder = destination / directory.name
        folder.mkdir(parents=True, exist_ok=True)
        copy_archive(directory / name, folder / name, package)


def stage_registry_cache(lock: bytes, cargo_home: Path, destination: Path) -> None:
    """Stage every locked registry archive; example: stage_registry_cache(lock, cache_home, isolated_home)."""
    caches = cache_directories(cargo_home)
    packages = locked_packages(lock)
    registry_cache = destination / "registry" / "cache"
    for package in packages.values():
        if package.source == REGISTRY:
            copy_package(package, caches, registry_cache)

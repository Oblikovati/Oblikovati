# SPDX-License-Identifier: GPL-2.0-only
"""Record a runtime tree without following symlinks or ignoring changes."""

import hashlib
import os
import stat
from pathlib import Path
from typing import Mapping

Metadata = dict[str, object]


def open_flags(directory: bool) -> int:
    """Require Linux no-follow and no-atime support; example: open_flags(True)."""
    required = ("O_NOFOLLOW", "O_NOATIME", "O_DIRECTORY")
    if os.name != "posix" or not all(hasattr(os, name) for name in required):
        raise ValueError(
            f"platform {os.name!r}: expected Linux no-follow and no-atime support"
        )
    return (
        os.O_RDONLY
        | os.O_NOFOLLOW
        | os.O_NOATIME
        | (os.O_DIRECTORY if directory else 0)
    )


def metadata(info: os.stat_result, kind: str) -> Metadata:
    """Record file identity and change metadata; example: metadata(os.stat(path), 'file')."""
    # readlink and exec can update access time on the CI filesystem (#3978).
    return {
        "kind": kind,
        "mode": info.st_mode,
        "uid": info.st_uid,
        "gid": info.st_gid,
        "device": info.st_dev,
        "inode": info.st_ino,
        "links": info.st_nlink,
        "size": info.st_size,
        "mtime_ns": info.st_mtime_ns,
        "ctime_ns": info.st_ctime_ns,
    }


def require_unchanged(before: os.stat_result, after: os.stat_result, name: str) -> None:
    """Refuse an entry that changed during inspection; example: require_unchanged(a, b, name)."""
    if metadata(before, "entry") != metadata(after, "entry"):
        raise ValueError(
            f"runtime input {name!r}: expected unchanged metadata during hashing"
        )


def file_digest(fd: int) -> str:
    """Hash all bytes from an open descriptor; example: file_digest(fd)."""
    digest = hashlib.sha256()
    while chunk := os.read(fd, 1024 * 1024):
        digest.update(chunk)
    return digest.hexdigest()


def snapshot_file(
    parent: int, name: str, path: str, before: os.stat_result
) -> Metadata:
    """Hash one file without following it; example: snapshot_file(parent, name, path, stat)."""
    fd = os.open(name, open_flags(False), dir_fd=parent)
    try:
        opened = os.fstat(fd)
        require_unchanged(before, opened, path)
        result = metadata(opened, "file")
        result["sha256"] = file_digest(fd)
        require_unchanged(opened, os.fstat(fd), path)
        return result
    finally:
        os.close(fd)


def snapshot_link(
    parent: int, name: str, path: str, before: os.stat_result
) -> Metadata:
    """Record one symlink without following it; example: snapshot_link(parent, name, path, stat)."""
    result = metadata(before, "symlink")
    result["target"] = os.readlink(name, dir_fd=parent)
    require_unchanged(before, os.stat(name, dir_fd=parent, follow_symlinks=False), path)
    return result


def child_names(fd: int) -> list[str]:
    """List all direct names in a stable order; example: child_names(directory_fd)."""
    with os.scandir(fd) as entries:
        return sorted(entry.name for entry in entries)


def snapshot_entry(
    parent: int, name: str, path: str, result: dict[str, Metadata]
) -> None:
    """Inventory one entry and any children; example: snapshot_entry(fd, name, path, result)."""
    info = os.stat(name, dir_fd=parent, follow_symlinks=False)
    if stat.S_ISREG(info.st_mode):
        result[path] = snapshot_file(parent, name, path, info)
    elif stat.S_ISDIR(info.st_mode):
        snapshot_directory(parent, name, path, info, result)
    elif stat.S_ISLNK(info.st_mode):
        result[path] = snapshot_link(parent, name, path, info)
    else:
        raise ValueError(
            f"runtime input {path!r}: expected a regular file, directory, or symlink"
        )


def snapshot_directory(
    parent: int,
    name: str,
    path: str,
    before: os.stat_result,
    result: dict[str, Metadata],
) -> None:
    """Inventory a directory while checking its metadata and membership; example: snapshot_directory(fd, name, path, stat, result)."""
    fd = os.open(name, open_flags(True), dir_fd=parent)
    try:
        opened = os.fstat(fd)
        require_unchanged(before, opened, path)
        record_directory(fd, path, opened, result)
    finally:
        os.close(fd)


def record_directory(
    fd: int, path: str, before: os.stat_result, result: dict[str, Metadata]
) -> None:
    """Record children and refuse membership changes; example: record_directory(fd, path, stat, result)."""
    names = child_names(fd)
    result[path] = metadata(before, "directory") | {"children": names}
    for name in names:
        child = name if path == "." else f"{path}/{name}"
        snapshot_entry(fd, name, child, result)
    if child_names(fd) != names:
        raise ValueError(
            f"runtime directory {path!r}: expected unchanged entries during hashing"
        )
    require_unchanged(before, os.fstat(fd), path)


def snapshot_runtime(root: Path) -> Mapping[str, Metadata]:
    """Inventory one bounded runtime tree; example: snapshot_runtime(Path('/tmp/runtime'))."""
    fd = os.open(root, open_flags(True))
    try:
        result: dict[str, Metadata] = {}
        record_directory(fd, ".", os.fstat(fd), result)
        return result
    finally:
        os.close(fd)

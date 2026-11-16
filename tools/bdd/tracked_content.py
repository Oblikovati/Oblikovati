# SPDX-License-Identifier: GPL-2.0-only
"""Compare checked-out tracked bytes with Git blobs without trusting the stat cache."""

import hashlib
import os
import stat
from pathlib import Path
from typing import Mapping


def git_blob_hash(content: bytes) -> str:
    """Hash raw Git blob bytes; example: git_blob_hash(b'hello')."""
    header = f"blob {len(content)}\0".encode("ascii")
    return hashlib.sha1(header + content, usedforsecurity=False).hexdigest()


def stable_identity(info: os.stat_result) -> tuple[int, ...]:
    """Identify a file without its access time; example: stable_identity(path.lstat())."""
    return (
        info.st_dev, info.st_ino, info.st_mode, info.st_size,
        info.st_mtime_ns, info.st_ctime_ns,
    )


def read_regular_file(path: Path, before: os.stat_result) -> bytes | None:
    """Read the opened inode without following its final symlink; example: read_regular_file(path, path.lstat())."""
    descriptor = os.open(path, os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
    with os.fdopen(descriptor, "rb") as stream:
        if stable_identity(os.fstat(stream.fileno())) != stable_identity(before):
            return None
        content = stream.read()
        if stable_identity(os.fstat(stream.fileno())) != stable_identity(before):
            return None
        return content


def read_tracked_file(path: Path, mode: str) -> bytes | None:
    """Read a regular file or link without accepting a changed entry; example: read_tracked_file(path, '100644')."""
    before = path.lstat()
    if mode == "120000" and stat.S_ISLNK(before.st_mode):
        content = os.fsencode(os.readlink(path))
    elif mode in ("100644", "100755") and stat.S_ISREG(before.st_mode):
        if os.name == "posix" and bool(before.st_mode & 0o111) != (mode == "100755"):
            return None
        content = read_regular_file(path, before)
    else:
        return None
    return content if stable_identity(path.lstat()) == stable_identity(before) else None


def real_parents(root: Path, name: str) -> bool:
    """Refuse tracked paths through links; example: real_parents(root, 'dir/file')."""
    current = root
    for part in Path(name).parts[:-1]:
        current = current / part
        if not stat.S_ISDIR(current.lstat().st_mode):
            return False
    return True


def tracked_content_matches(root: Path, tree: Mapping[str, str]) -> bool:
    """Reject tracked content different from its commit; example: tracked_content_matches(root, tree)."""
    try:
        for name, identity in tree.items():
            mode, kind, expected = identity.split(":")
            if kind != "blob" or not real_parents(root, name):
                return False
            content = read_tracked_file(root / name, mode)
            if content is None or git_blob_hash(content) != expected:
                return False
        return True
    except (OSError, ValueError):
        return False

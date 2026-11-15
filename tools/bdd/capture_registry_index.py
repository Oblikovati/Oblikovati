# SPDX-License-Identifier: GPL-2.0-only
"""Copy Cargo registry index data into a private capture home."""

import os
import shutil
import stat
from pathlib import Path


def copy_index_file(source: Path, target: Path) -> None:
    """Copy one regular index file without following links; example: copy_index_file(src, dst)."""
    if not hasattr(os, "O_NOFOLLOW"):
        raise ValueError("registry index open: expected no-follow support")
    descriptor = os.open(source, os.O_RDONLY | os.O_NOFOLLOW | getattr(os, "O_NONBLOCK", 0))
    try:
        if not stat.S_ISREG(os.fstat(descriptor).st_mode):
            raise ValueError(f"registry index file {source}: expected a regular file")
        with os.fdopen(descriptor, "rb", closefd=False) as input_file, target.open("xb") as output:
            shutil.copyfileobj(input_file, output)
    finally:
        os.close(descriptor)


def copy_index_directory(source: Path, target: Path) -> None:
    """Copy a directory without copying symlinks or special files; example: copy_index_directory(src, dst)."""
    if source.is_symlink() or not source.is_dir():
        raise ValueError(f"registry index {source}: expected a real directory")
    target.mkdir()
    for entry in sorted(source.iterdir()):
        mode = entry.lstat().st_mode
        if stat.S_ISDIR(mode):
            copy_index_directory(entry, target / entry.name)
        elif stat.S_ISREG(mode):
            copy_index_file(entry, target / entry.name)
        else:
            raise ValueError(f"registry index entry {entry}: expected a regular file or directory")


def stage_registry_index(source: Path, target: Path) -> None:
    """Copy the resolved registry index into a new directory; example: stage_registry_index(src, dst)."""
    copy_index_directory(source.resolve(strict=True), target)

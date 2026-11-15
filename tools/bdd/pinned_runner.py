# SPDX-License-Identifier: GPL-2.0-only
"""Pin one observed runner in a sealed Linux memory file for Bubblewrap."""

import hashlib
import os
import re
import stat
import subprocess
from pathlib import Path
from typing import Callable
from types import ModuleType

from runtime_snapshot import open_flags


def write_all(descriptor: int, data: bytes) -> None:
    """Write every byte to a descriptor; example: write_all(fd, b'data')."""
    pending = memoryview(data)
    while pending:
        count = os.write(descriptor, pending)
        if count <= 0:
            raise ValueError(f"runner write {count}: expected forward progress")
        pending = pending[count:]


def seal_runner(source: Path, expected: str) -> int:
    """Create an immutable runner matching the build digest; example: seal_runner(path, sha256)."""
    import fcntl
    if not re.fullmatch(r"[0-9a-f]{64}", expected):
        raise ValueError(f"runner digest {expected!r}: expected a SHA-256 hex digest")
    if not all(hasattr(os, name) for name in ("memfd_create", "MFD_ALLOW_SEALING", "MFD_CLOEXEC")):
        raise ValueError("runner memory file: expected Linux memfd sealing support")
    original = os.open(source, open_flags(False))
    try:
        return seal_open_runner(original, source, expected, fcntl)
    finally:
        os.close(original)


def seal_open_runner(original: int, source: Path, expected: str, seals: ModuleType) -> int:
    """Copy one regular executable into a sealed memory file; example: seal_open_runner(fd, path, sha, fcntl)."""
    info = os.fstat(original)
    if not stat.S_ISREG(info.st_mode) or not info.st_mode & 0o111:
        raise ValueError(f"runner {source}: expected an executable regular file")
    pinned = os.memfd_create("oblikovati-build-runner", os.MFD_ALLOW_SEALING | os.MFD_CLOEXEC)
    try:
        digest = hashlib.sha256()
        while chunk := os.read(original, 1024 * 1024):
            digest.update(chunk)
            write_all(pinned, chunk)
        if digest.hexdigest() != expected:
            raise ValueError(f"runner {source}: expected build SHA-256 {expected}")
        os.fchmod(pinned, 0o555)
        os.lseek(pinned, 0, os.SEEK_SET)
        seal_memory_file(pinned, seals)
        return pinned
    except BaseException:
        os.close(pinned)
        raise


def seal_memory_file(descriptor: int, seals: ModuleType) -> None:
    """Prevent writes and resizing of a memory file; example: seal_memory_file(fd, fcntl)."""
    flags = seals.F_SEAL_WRITE | seals.F_SEAL_GROW | seals.F_SEAL_SHRINK | seals.F_SEAL_SEAL
    seals.fcntl(descriptor, seals.F_ADD_SEALS, flags)


def bind_pinned_runner(command: list[str], source: Path, descriptor: int) -> list[str]:
    """Bind the pinned bytes where the runner belongs; example: bind_pinned_runner(command, path, fd)."""
    expected = ["--ro-bind", str(source), "/runner"]
    positions = [index for index in range(len(command) - 2) if command[index:index + 3] == expected]
    if len(positions) != 1:
        raise ValueError(f"runner mount {expected!r}: expected exactly one matching bind")
    index = positions[0]
    return command[:index] + ["--perms", "0555", "--ro-bind-data", str(descriptor), "/runner"] + command[index + 3:]


class PinnedRunnerProcess:
    """Run Bubblewrap with immutable bytes from one observed build artifact."""

    def __init__(self, source: Path, digest: str, launch: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run) -> None:
        self.source, self.digest, self.launch = source, digest, launch

    def __call__(self, command: list[str], log: Path) -> int:
        descriptor = seal_runner(self.source, self.digest)
        try:
            pinned = bind_pinned_runner(command, self.source, descriptor)
            with log.open("x", encoding="utf-8") as output:
                return self.launch(pinned, stdout=output, stderr=subprocess.STDOUT, check=False, env={}, pass_fds=(descriptor,)).returncode
        finally:
            os.close(descriptor)

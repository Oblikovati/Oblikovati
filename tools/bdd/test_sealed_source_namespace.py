# SPDX-License-Identifier: GPL-2.0-only
"""Exercise a sealed source archive in a private read-only mount namespace."""

import io
import os
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path


def source_archive(source: Path) -> bytes:
    """Archive one source file; example: source_archive(Path('source'))."""
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w") as archive:
        archive.add(source, arcname="source.txt")
    return buffer.getvalue()


def sealed_archive(contents: bytes) -> int:
    """Seal archive bytes in memory; example: fd = sealed_archive(b'bytes')."""
    import fcntl

    descriptor = os.memfd_create("test-source", os.MFD_ALLOW_SEALING)
    try:
        with os.fdopen(os.dup(descriptor), "wb") as output:
            output.write(contents)
        seals = (
            fcntl.F_SEAL_WRITE | fcntl.F_SEAL_GROW | fcntl.F_SEAL_SHRINK | fcntl.F_SEAL_SEAL
        )
        fcntl.fcntl(descriptor, fcntl.F_ADD_SEALS, seals)
        os.lseek(descriptor, 0, os.SEEK_SET)
        return descriptor
    except BaseException:
        os.close(descriptor)
        raise


def sandbox_prefix() -> list[str]:
    """Give the fixture its system tools; example: command = sandbox_prefix()."""
    return [
        "bwrap", "--unshare-all", "--unshare-user", "--ro-bind", "/usr", "/usr",
        "--symlink", "usr/bin", "/bin", "--symlink", "usr/lib", "/lib",
        "--symlink", "usr/lib64", "/lib64",
    ]


def source_sandbox(descriptor: int) -> list[str]:
    """Create a nested read-only source mount; example: source_sandbox(fd)."""
    inner = sandbox_prefix() + [
        "--disable-userns", "--ro-bind", "/tmp/tree", "/source", "--",
        "/bin/sh", "-c", 'cat /source/source.txt; ! printf altered > /source/source.txt',
    ]
    shell = 'mkdir /tmp/tree && tar -xf /input.tar -C /tmp/tree && exec "$@"'
    return sandbox_prefix() + [
        "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
        "--ro-bind-data", str(descriptor), "/input.tar", "--", "/bin/sh", "-c",
        shell, "source-stage", *inner,
    ]


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "memfd_create") and shutil.which("bwrap")
    and shutil.which("tar"), "requires Linux memfd, Bubblewrap and tar",
)
class SealedSourceNamespaceTests(unittest.TestCase):
    def test_sealed_bytes_survive_host_changes_and_mount_read_only(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            source = Path(folder) / "source.txt"
            source.write_text("committed\n")
            descriptor = sealed_archive(source_archive(source))
            self.addCleanup(os.close, descriptor)
            source.write_text("forged\n")
            with self.assertRaises(PermissionError):
                os.write(descriptor, b"forged")
            result = subprocess.run(source_sandbox(descriptor), pass_fds=(descriptor,),
                                    capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, "committed\n")
            self.assertEqual(source.read_text(), "forged\n")

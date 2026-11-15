# SPDX-License-Identifier: GPL-2.0-only
"""Check a fail-closed runtime filesystem inventory."""

import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import runtime_snapshot

from runtime_snapshot import snapshot_runtime


class RuntimeTreeFixture:
    """Make a small runtime tree; example: RuntimeTreeFixture(root)."""

    def __init__(self, root: Path) -> None:
        self.root = root
        (root / "lib").mkdir()
        (root / "lib" / "library").write_bytes(b"library")
        (root / "runner").write_bytes(b"executable")

    def change(self, name: str, contents: bytes) -> None:
        """Replace fixture bytes; example: fixture.change('runner', b'new')."""
        (self.root / name).write_bytes(contents)


class MutatingDigest:
    """Change a file while its descriptor is being hashed; example: MutatingDigest(path)."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.original = runtime_snapshot.file_digest
        self.called = False

    def __call__(self, fd: int) -> str:
        digest = self.original(fd)
        if not self.called:
            self.called = True
            self.path.write_bytes(b"changed while hashing")
        return digest


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"), "requires Linux file descriptors"
)
class RuntimeSnapshotTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.fixture = RuntimeTreeFixture(Path(self.temporary.name))

    def test_records_files_and_directories_and_detects_content_changes(self) -> None:
        before = snapshot_runtime(self.fixture.root)
        self.assertEqual(before["runner"]["kind"], "file")
        self.assertEqual(before["lib"]["kind"], "directory")
        self.fixture.change("runner", b"different")
        after = snapshot_runtime(self.fixture.root)
        self.assertNotEqual(before["runner"], after["runner"])

    def test_records_added_and_removed_directory_entries(self) -> None:
        before = snapshot_runtime(self.fixture.root)
        self.fixture.change("lib/new", b"new")
        after = snapshot_runtime(self.fixture.root)
        self.assertNotEqual(before["lib"], after["lib"])
        self.assertIn("lib/new", after)
        (self.fixture.root / "runner").unlink()
        self.assertNotIn("runner", snapshot_runtime(self.fixture.root))

    def test_symlink_target_and_mode_are_inputs(self) -> None:
        link = self.fixture.root / "link"
        link.symlink_to("runner")
        before = snapshot_runtime(self.fixture.root)
        link.unlink()
        link.symlink_to("lib/library")
        os.chmod(self.fixture.root / "runner", 0o700)
        after = snapshot_runtime(self.fixture.root)
        self.assertNotEqual(before["link"], after["link"])
        self.assertNotEqual(before["runner"], after["runner"])

    def test_mutation_during_hash_is_refused(self) -> None:
        fake = MutatingDigest(self.fixture.root / "lib" / "library")
        with patch("runtime_snapshot.file_digest", fake):
            with self.assertRaisesRegex(ValueError, "unchanged metadata"):
                snapshot_runtime(self.fixture.root)

    def test_external_symlink_is_not_followed(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            outside = Path(folder) / "outside"
            outside.write_bytes(b"first")
            (self.fixture.root / "link").symlink_to(outside)
            before = snapshot_runtime(self.fixture.root)
            outside.write_bytes(b"second")
            self.assertEqual(before, snapshot_runtime(self.fixture.root))

    def test_special_files_and_symlink_root_are_refused(self) -> None:
        pipe = self.fixture.root / "pipe"
        os.mkfifo(pipe)
        with self.assertRaisesRegex(ValueError, "regular file, directory, or symlink"):
            snapshot_runtime(self.fixture.root)
        pipe.unlink()
        alias = self.fixture.root.parent / (self.fixture.root.name + "-alias")
        alias.symlink_to(self.fixture.root)
        self.addCleanup(alias.unlink)
        with self.assertRaises(OSError):
            snapshot_runtime(alias)


if __name__ == "__main__":
    unittest.main()

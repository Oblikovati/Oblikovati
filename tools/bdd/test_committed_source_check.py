# SPDX-License-Identifier: GPL-2.0-only
"""Check that a mounted build source matches a selected committed tree."""

import os
import tempfile
import unittest
from pathlib import Path

from committed_build_source import materialize_source
from committed_source_check import committed_reference, verify_source_inventory
from runtime_snapshot import snapshot_runtime
from test_committed_build_source import SourceRepository


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"),
    "requires no-atime source snapshots",
)
class CommittedSourceCheckTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.repo = self.root / "repository"
        self.repo.mkdir()
        self.git = SourceRepository(self.repo)
        nested = self.repo / "nested/src"
        nested.mkdir(parents=True)
        (nested / "lib.rs").write_text("nested")
        self.git.git("add", "nested")
        self.git.git("commit", "-qm", "nested source")
        self.git.commit = self.git.git("rev-parse", "HEAD")
        self.source = self.root / "source"
        materialize_source(self.repo, self.git.commit, self.source)
        self.expected = committed_reference(self.repo, self.git.commit)

    def verify(self) -> None:
        """Check the current source; example: self.verify()."""
        verify_source_inventory(snapshot_runtime(self.source), self.expected)

    def test_committed_source_matches_even_when_worktree_is_dirty(self) -> None:
        (self.repo / "source.rs").write_text("dirty")
        (self.repo / "ignored.txt").write_text("ignored")
        self.verify()

    def test_changed_added_removed_and_empty_directory_are_rejected(self) -> None:
        changes = ("changed", "added", "removed", "empty")
        for change in changes:
            with self.subTest(change=change):
                item = self.source / "source.rs"
                original = item.read_bytes()
                if change == "changed":
                    item.write_bytes(b"altered")
                if change == "added":
                    (self.source / "ignored.txt").write_text("injected")
                if change == "removed":
                    item.unlink()
                if change == "empty":
                    (self.source / "empty").mkdir()
                with self.assertRaises(ValueError):
                    self.verify()
                if not item.exists():
                    item.write_bytes(original)
                elif change == "changed":
                    item.write_bytes(original)
                if change == "added":
                    (self.source / "ignored.txt").unlink()
                if change == "empty":
                    (self.source / "empty").rmdir()

    def test_mode_symlink_and_hardlink_are_rejected(self) -> None:
        item = self.source / "source.rs"
        item.chmod(0o755)
        with self.assertRaisesRegex(ValueError, "mode"):
            self.verify()
        item.chmod(0o644)
        item.unlink()
        item.symlink_to(self.repo / "source.rs")
        with self.assertRaisesRegex(ValueError, "regular file"):
            self.verify()
        item.unlink()
        os.link(self.repo / "source.rs", item)
        with self.assertRaisesRegex(ValueError, "single link"):
            self.verify()


if __name__ == "__main__":
    unittest.main()

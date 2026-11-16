# SPDX-License-Identifier: GPL-2.0-only
"""Reject tracked edits hidden by Git's stat cache."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

from scenario_dependencies import worktree_clean
from tracked_content import git_blob_hash, tracked_content_matches


class CachedStatRepository:
    """A Git checkout with a reproducible old file timestamp."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.file = root / "tracked.txt"
        self.git("init", "-q")
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "core.trustctime", "false")
        self.file.write_text("first", encoding="utf-8")
        os.utime(self.file, (1_700_000_000, 1_700_000_000))
        self.git("add", "tracked.txt")
        self.git("commit", "-qm", "baseline")

    def git(self, *args: str) -> str:
        """Read a Git result; example: repository.git('status', '--porcelain')."""
        return subprocess.check_output(
            ["git", "-C", str(self.root), *args], text=True
        ).strip()


class TrackedContentTests(unittest.TestCase):
    def test_git_blob_hash_includes_the_git_header(self) -> None:
        self.assertEqual(
            git_blob_hash(b"hello"), "b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0"
        )

    def test_same_stat_changed_content_is_not_clean(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repository = CachedStatRepository(Path(folder))
            self.assertTrue(worktree_clean(repository.root))
            repository.file.write_text("other", encoding="utf-8")
            os.utime(repository.file, (1_700_000_000, 1_700_000_000))
            self.assertEqual(repository.git("status", "--porcelain"), "")
            self.assertFalse(worktree_clean(repository.root))

    @unittest.skipUnless(os.name == "posix", "requires symbolic links")
    def test_matching_bytes_through_linked_parent_are_refused(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            actual = root / "actual"
            actual.mkdir()
            (actual / "file").write_bytes(b"hello")
            (root / "linked").symlink_to(actual, target_is_directory=True)
            tree = {"linked/file": f"100644:blob:{git_blob_hash(b'hello')}"}
            self.assertFalse(tracked_content_matches(root, tree))

    @unittest.skipUnless(os.name == "posix", "requires symbolic links")
    def test_link_target_and_executable_mode_are_checked(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            link = root / "link"
            link.symlink_to("target")
            tree = {"link": f"120000:blob:{git_blob_hash(b'target')}"}
            self.assertTrue(tracked_content_matches(root, tree))
            link.unlink()
            link.symlink_to("other")
            self.assertFalse(tracked_content_matches(root, tree))
            executable = root / "run"
            executable.write_bytes(b"hello")
            executable.chmod(0o644)
            tree = {"run": f"100755:blob:{git_blob_hash(b'hello')}"}
            self.assertFalse(tracked_content_matches(root, tree))
            executable.chmod(0o755)
            self.assertTrue(tracked_content_matches(root, tree))

# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for scenario snapshot verification."""

import hashlib
import os
import tempfile
import subprocess
import unittest
from pathlib import Path

from verify_snapshot import (
    checkout_files,
    checkout_namespace,
    git_revision,
    manifest_directories,
    read_snapshot_metadata,
    restore_snapshot,
    verify_artifact,
    verify_checkout,
)


class FakeRevision:
    def __init__(self, revision: str) -> None:
        self.revision = revision

    def __call__(self, checkout: Path) -> str:
        return self.revision


class SnapshotVerificationTests(unittest.TestCase):
    def test_restore_produces_a_clean_pinned_checkout(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            checkout = Path(folder) / "checkout"
            restore_snapshot(checkout)
            self.assertEqual(len(git_revision(checkout)), 40)
            self.assertTrue((checkout / "features/application").is_dir())
            (checkout / "features/application/untracked-empty").mkdir()
            metadata, manifest = read_snapshot_metadata()
            with self.assertRaisesRegex(ValueError, "untracked-empty"):
                verify_checkout(checkout, metadata, manifest, git_revision)

    def test_revision_rejects_a_dirty_checkout(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            item = root / "case.feature"
            item.write_text("Feature: example\n")
            subprocess.run(["git", "-C", str(root), "add", "case.feature"], check=True)
            subprocess.run(
                [
                    "git",
                    "-C",
                    str(root),
                    "-c",
                    "user.name=Test",
                    "-c",
                    "user.email=test@example.com",
                    "commit",
                    "-qm",
                    "fixture",
                ],
                check=True,
            )
            clean = git_revision(root)
            item.write_text("Feature: changed\n")
            with self.assertRaisesRegex(ValueError, "expected clean Git status"):
                git_revision(root)
            item.write_text("Feature: example\n")
            self.assertEqual(git_revision(root), clean)

    @unittest.skipUnless(
        os.name == "posix" and hasattr(os, "O_NOATIME"),
        "requires Linux no-atime descriptors",
    )
    def test_revision_inspection_does_not_modify_git_namespace(self) -> None:
        from runtime_snapshot import snapshot_runtime

        with tempfile.TemporaryDirectory() as folder:
            checkout = Path(folder) / "checkout"
            restore_snapshot(checkout)
            before = snapshot_runtime(checkout / ".git")
            git_revision(checkout)
            self.assertEqual(snapshot_runtime(checkout / ".git"), before)

    def test_checkout_detects_changed_and_added_files(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "scenario.feature").write_text("original")
            manifest = checkout_files(root)
            metadata = {"snapshot_commit": "abc"}
            verify_checkout(root, metadata, manifest, FakeRevision("abc"))
            (root / "scenario.feature").write_text("changed")
            with self.assertRaisesRegex(ValueError, "scenario.feature"):
                verify_checkout(root, metadata, manifest, FakeRevision("abc"))
            (root / "scenario.feature").unlink()
            with self.assertRaisesRegex(ValueError, "scenario.feature"):
                verify_checkout(root, metadata, manifest, FakeRevision("abc"))
            (root / "scenario.feature").write_text("original")
            (root / "extra.feature").write_text("extra")
            with self.assertRaisesRegex(ValueError, "extra.feature"):
                verify_checkout(root, metadata, manifest, FakeRevision("abc"))

    def test_namespace_reports_files_and_parent_directories(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "a" / "b").mkdir(parents=True)
            (root / "a" / "b" / "case.feature").write_text("Feature: example\n")
            files, directories = checkout_namespace(root)
            self.assertEqual(set(files), {"a/b/case.feature"})
            self.assertEqual(directories, {"a", "a/b"})
            self.assertEqual(manifest_directories(files), directories)

    def test_manifest_rejects_paths_outside_the_checkout(self) -> None:
        for name in ("/absolute/file", "../escape", "a/../file", ".", "a//b", "a\\b"):
            with self.subTest(name=name):
                with self.assertRaisesRegex(ValueError, "normalized relative"):
                    manifest_directories({name: "hash"})

    def test_checkout_rejects_extra_empty_directories(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            nested = root / "features" / "application"
            nested.mkdir(parents=True)
            (nested / "case.feature").write_text("Feature: example\n")
            manifest = checkout_files(root)
            metadata = {"snapshot_commit": "abc"}
            verify_checkout(root, metadata, manifest, FakeRevision("abc"))
            extra = nested / "untracked-empty"
            extra.mkdir()
            with self.assertRaisesRegex(ValueError, "untracked-empty"):
                verify_checkout(root, metadata, manifest, FakeRevision("abc"))

    @unittest.skipUnless(hasattr(os, "mkfifo"), "requires named pipes")
    def test_checkout_rejects_untracked_special_files(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "case.feature").write_text("Feature: example\n")
            manifest = checkout_files(root)
            pipe = root / "untracked-pipe"
            os.mkfifo(pipe)
            with self.assertRaisesRegex(ValueError, "untracked-pipe"):
                verify_checkout(
                    root, {"snapshot_commit": "abc"}, manifest, FakeRevision("abc")
                )

    def test_checkout_rejects_symlink_entries(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "case.feature").write_text("Feature: example\n")
            manifest = checkout_files(root)
            try:
                (root / "alias").symlink_to("case.feature")
            except OSError as cause:
                self.skipTest(f"symlink creation unavailable: {cause}")
            with self.assertRaisesRegex(ValueError, "alias"):
                verify_checkout(
                    root, {"snapshot_commit": "abc"}, manifest, FakeRevision("abc")
                )

    def test_wrong_revision_and_artifact_hash_are_refused(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            artifact = root / "bundle"
            artifact.write_bytes(b"actual")
            verify_artifact(artifact, hashlib.sha256(b"actual").hexdigest())
            with self.assertRaisesRegex(ValueError, "expected SHA-256"):
                verify_artifact(artifact, "wrong")
            with self.assertRaisesRegex(ValueError, "expected commit abc"):
                verify_checkout(
                    root, {"snapshot_commit": "abc"}, {}, FakeRevision("other")
                )


if __name__ == "__main__":
    unittest.main()

# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for scenario snapshot verification."""

import hashlib
import tempfile
import subprocess
import unittest
from pathlib import Path

from verify_snapshot import (
    checkout_files, git_revision, restore_snapshot, verify_artifact, verify_checkout,
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

    def test_revision_rejects_a_dirty_checkout(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            item = root / "case.feature"
            item.write_text("Feature: example\n")
            subprocess.run(["git", "-C", str(root), "add", "case.feature"], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Test", "-c",
                            "user.email=test@example.com", "commit", "-qm", "fixture"], check=True)
            clean = git_revision(root)
            item.write_text("Feature: changed\n")
            with self.assertRaisesRegex(ValueError, "expected clean Git status"):
                git_revision(root)
            item.write_text("Feature: example\n")
            self.assertEqual(git_revision(root), clean)

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

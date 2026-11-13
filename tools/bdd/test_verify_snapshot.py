# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for scenario snapshot verification."""

import hashlib
import tempfile
import unittest
from pathlib import Path

from verify_snapshot import checkout_files, verify_artifact, verify_checkout


class FakeRevision:
    def __init__(self, revision: str) -> None:
        self.revision = revision

    def __call__(self, checkout: Path) -> str:
        return self.revision


class SnapshotVerificationTests(unittest.TestCase):
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

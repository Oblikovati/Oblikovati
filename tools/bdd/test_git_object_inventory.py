# SPDX-License-Identifier: GPL-2.0-only
"""Check Git object identities without trusting a checkout."""

import hashlib
import io
import tempfile
import unittest
from pathlib import Path

from git_object_inventory import (
    GitBlobReader,
    git_blob_bytes,
    hash_blob_stream,
    locked_git_tree,
    sha1_git_blob,
)
from test_committed_build_source import SourceRepository


class GitObjectInventoryTests(unittest.TestCase):
    def test_committed_tree_and_blobs_ignore_dirty_checkout(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            fixture = SourceRepository(repo)
            tree = locked_git_tree(repo, fixture.commit)
            identity = tree["source.rs"].split(":")[-1]
            (repo / "source.rs").write_text("dirty")
            self.assertEqual(git_blob_bytes(repo, identity), b"committed")
            with GitBlobReader(repo) as reader:
                self.assertEqual(
                    reader.digest(identity), hashlib.sha256(b"committed").hexdigest()
                )
            self.assertEqual(sha1_git_blob(b"committed"), identity)

    def test_bad_revision_identity_and_blob_stream_are_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            SourceRepository(repo)
            with self.assertRaisesRegex(ValueError, "full SHA-1"):
                locked_git_tree(repo, "HEAD")
            with self.assertRaisesRegex(ValueError, "full SHA-1"):
                git_blob_bytes(repo, "short")
        identity = sha1_git_blob(b"abc")
        with self.assertRaisesRegex(ValueError, "more bytes"):
            hash_blob_stream(io.BytesIO(b"a"), identity, 3)
        with self.assertRaisesRegex(ValueError, "matching bytes"):
            hash_blob_stream(io.BytesIO(b"bad\n"), identity, 3)
        with self.assertRaisesRegex(ValueError, "terminator"):
            hash_blob_stream(io.BytesIO(b"abcx"), identity, 3)


if __name__ == "__main__":
    unittest.main()

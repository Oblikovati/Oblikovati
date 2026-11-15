# SPDX-License-Identifier: GPL-2.0-only
"""Test the capture registry index copy boundary."""

import os
import tempfile
import unittest
from pathlib import Path

from capture_registry_index import stage_registry_index


class CaptureRegistryIndexTests(unittest.TestCase):
    def test_copy_is_independent_after_staging(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            source, target = root / "index", root / "copy"
            (source / "cache").mkdir(parents=True)
            (source / "cache/item").write_bytes(b"first")
            stage_registry_index(source, target)
            (source / "cache/item").write_bytes(b"second")
            self.assertEqual((target / "cache/item").read_bytes(), b"first")

    def test_nested_symlink_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            source = root / "index"
            source.mkdir()
            (source / "link").symlink_to(root / "outside")
            with self.assertRaisesRegex(ValueError, "expected a regular file or directory"):
                stage_registry_index(source, root / "copy")

    @unittest.skipUnless(hasattr(os, "mkfifo"), "requires FIFO support")
    def test_fifo_is_rejected_without_blocking(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            source = root / "index"
            source.mkdir()
            os.mkfifo(source / "pipe")
            with self.assertRaisesRegex(ValueError, "expected a regular file or directory"):
                stage_registry_index(source, root / "copy")

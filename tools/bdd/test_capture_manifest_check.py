# SPDX-License-Identifier: GPL-2.0-only
"""Test capture manifest path and override checks."""

import tempfile
import unittest
from pathlib import Path

from capture_manifest_check import read_capture_manifests, require_same_manifests


class CaptureManifestTests(unittest.TestCase):
    def manifest(self, source: Path, content: str) -> Path:
        """Write a fixture manifest; example: self.manifest(source, text)."""
        path = source / "Cargo.toml"
        path.write_text(content)
        return path

    def test_internal_paths_and_current_workspace_are_allowed(self) -> None:
        source = Path(__file__).resolve().parents[2]
        self.assertGreater(len(read_capture_manifests(source)), 1)
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            self.manifest(root, '[dependencies]\ninternal={path="./crates/internal"}\n')
            self.assertEqual(len(read_capture_manifests(root)), 1)

    def test_external_paths_and_source_overrides_are_rejected(self) -> None:
        for content in (
            '[dependencies]\nbad={path="../outside"}\n',
            '[dependencies]\nbad={path="/outside"}\n',
            '[dependencies]\nbad={path="C:\\\\outside"}\n',
            '[patch.crates-io]\nfoo={path="foo"}\n',
            '[replace]\n',
        ):
            with self.subTest(content=content), tempfile.TemporaryDirectory() as folder:
                root = Path(folder)
                self.manifest(root, content)
                with self.assertRaisesRegex(ValueError, "expected"):
                    read_capture_manifests(root)

    def test_manifest_change_and_symlink_escape_are_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder) / "source"
            root.mkdir()
            manifest = self.manifest(root, '[dependencies]\nbad={path="inside"}\n')
            before = read_capture_manifests(root)
            manifest.write_text('[dependencies]\nbad={path="inside2"}\n')
            with self.assertRaisesRegex(ValueError, "expected unchanged"):
                require_same_manifests(root, before)
            manifest.write_text('[dependencies]\nbad={path="link"}\n')
            (root / "link").symlink_to(Path(folder), target_is_directory=True)
            with self.assertRaisesRegex(ValueError, "expected a path inside"):
                read_capture_manifests(root)

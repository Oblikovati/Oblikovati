# SPDX-License-Identifier: GPL-2.0-only
"""Check the native capture Cargo configuration policy."""

import os
import tempfile
import unittest
from pathlib import Path

from capture_source_config import read_capture_source_config, require_same_source_config


class CaptureSourceConfigTests(unittest.TestCase):
    def config(self, source: Path, content: str) -> Path:
        """Write a fixture configuration; example: self.config(source, '[build]')."""
        directory = source / ".cargo"
        directory.mkdir(exist_ok=True)
        result = directory / "config.toml"
        result.write_text(content)
        return result

    def test_current_project_config_is_allowed(self) -> None:
        source = Path(__file__).resolve().parents[2]
        self.assertIsInstance(read_capture_source_config(source), bytes)

    def test_empty_or_absent_config_is_allowed(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            source = Path(folder)
            self.assertIsNone(read_capture_source_config(source))
            self.config(source, "")
            self.assertEqual(read_capture_source_config(source), b"")

    def test_compiler_and_source_redirects_are_rejected(self) -> None:
        for content in (
            '[build]\nrustc-wrapper="/bad"\n',
            '[env]\nRUSTFLAGS="--cfg injected"\n',
            '[source.crates-io]\nreplace-with="external"\n',
            '[target.x86_64-unknown-linux-gnu]\nlinker="/bad"\n',
            '[alias]\nxtask="/bad"\n',
            'paths=["/outside"]\n',
        ):
            with self.subTest(content=content), tempfile.TemporaryDirectory() as folder:
                source = Path(folder)
                self.config(source, content)
                with self.assertRaisesRegex(ValueError, "expected an approved capture setting"):
                    read_capture_source_config(source)

    @unittest.skipUnless(hasattr(os, "mkfifo"), "requires FIFO support")
    def test_config_fifo_is_rejected_without_blocking(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            source = Path(folder)
            (source / ".cargo").mkdir()
            os.mkfifo(source / ".cargo/config.toml")
            with self.assertRaisesRegex(ValueError, "expected a regular file"):
                read_capture_source_config(source)

    def test_legacy_symlink_and_changed_config_are_refused(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            source = Path(folder)
            path = self.config(source, "")
            before = read_capture_source_config(source)
            path.write_text("# changed\n")
            with self.assertRaisesRegex(ValueError, "expected unchanged"):
                require_same_source_config(source, before)
            path.unlink()
            (source / ".cargo/config").symlink_to(source / "missing")
            with self.assertRaisesRegex(ValueError, "expected no legacy"):
                read_capture_source_config(source)

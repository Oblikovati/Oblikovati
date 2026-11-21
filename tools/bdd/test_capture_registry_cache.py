# SPDX-License-Identifier: GPL-2.0-only
"""Check lockfile-bound registry archive staging."""

import hashlib
import os
import tempfile
import unittest
from pathlib import Path

from capture_registry_cache import stage_registry_cache


class RegistryArchiveFixture:
    """Make one locked archive and a separate capture home."""

    def __init__(self, root: Path) -> None:
        self.home = root / "cargo"
        self.cache = self.home / "registry/cache/index.crates.io-test"
        self.cache.mkdir(parents=True)
        self.archive = self.cache / "example-1.0.0.crate"
        self.archive.write_bytes(b"locked archive")
        checksum = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        self.lock = (
            'version=3\n[[package]]\nname="example"\nversion="1.0.0"\n'
            'source="registry+https://github.com/rust-lang/crates.io-index"\n'
            f'checksum="{checksum}"\n'
        ).encode()
        self.destination = root / "isolated"
        self.destination.mkdir()


class CaptureRegistryCacheTests(unittest.TestCase):
    def test_stages_locked_bytes_without_sharing_extracted_sources(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = RegistryArchiveFixture(Path(folder))
            stage_registry_cache(fixture.lock, fixture.home, fixture.destination)
            staged = fixture.destination / "registry/cache/index.crates.io-test" / fixture.archive.name
            self.assertEqual(staged.read_bytes(), b"locked archive")
            self.assertFalse(staged.is_symlink())
            fixture.archive.write_bytes(b"changed afterward")
            self.assertEqual(staged.read_bytes(), b"locked archive")
            self.assertFalse((fixture.destination / "registry/src").exists())

    def test_tampered_missing_and_symlink_archives_are_rejected(self) -> None:
        for corruption in ("tampered", "missing", "symlink"):
            with self.subTest(corruption=corruption), tempfile.TemporaryDirectory() as folder:
                fixture = RegistryArchiveFixture(Path(folder))
                if corruption == "tampered":
                    fixture.archive.write_bytes(b"forged")
                else:
                    fixture.archive.unlink()
                    if corruption == "symlink":
                        fixture.archive.symlink_to(Path(folder) / "outside")
                with self.assertRaises((ValueError, OSError)):
                    stage_registry_cache(fixture.lock, fixture.home, fixture.destination)

    @unittest.skipUnless(hasattr(os, "mkfifo"), "requires FIFO support")
    def test_fifo_archive_is_refused_without_blocking(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = RegistryArchiveFixture(Path(folder))
            fixture.archive.unlink()
            os.mkfifo(fixture.archive)
            with self.assertRaisesRegex(ValueError, "expected a regular file"):
                stage_registry_cache(fixture.lock, fixture.home, fixture.destination)

    def test_bad_duplicate_namespace_is_not_ignored(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = RegistryArchiveFixture(Path(folder))
            duplicate = fixture.home / "registry/cache/other-index"
            duplicate.mkdir()
            (duplicate / fixture.archive.name).write_bytes(b"forged")
            with self.assertRaisesRegex(ValueError, "expected locked SHA-256"):
                stage_registry_cache(fixture.lock, fixture.home, fixture.destination)

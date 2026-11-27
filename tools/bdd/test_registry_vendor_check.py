# SPDX-License-Identifier: GPL-2.0-only
"""Check locked registry archives against an observed vendor tree."""

import hashlib
import io
import json
import os
import tarfile
import tempfile
import unittest
from pathlib import Path

from registry_vendor_check import (
    LockedPackage,
    archive_entries,
    locked_packages,
    verify_registry_vendor,
    verify_vendor_config,
)
from runtime_snapshot import snapshot_runtime

COMMENT = "This file only protects against accidental modifications. It is not a security mechanism and does not protect against malicious changes."


class RegistryVendorFixture:
    """A locked registry package and an explicitly unverified Git package."""

    def __init__(self, root: Path) -> None:
        self.vendor, self.archives = root / "vendor", root / "archives"
        self.vendor.mkdir()
        self.archives.mkdir()
        self.package = self.vendor / "witness"
        (self.package / "src").mkdir(parents=True)
        self.contents = {
            "Cargo.toml": b'[package]\nname="witness"\nversion="1.2.3"\n',
            "src/lib.rs": b"pub fn witness() {}\n",
        }
        self.write_archive()
        self.checksum = hashlib.sha256(
            (self.archives / "witness-1.2.3.crate").read_bytes()
        ).hexdigest()
        for name, content in self.contents.items():
            (self.package / name).write_bytes(content)
        self.write_checksum(self.contents)
        git = self.vendor / "git-dep"
        git.mkdir()
        (git / "Cargo.toml").write_text('[package]\nname="git-dep"\nversion="0.1.0"\n')
        self.lock = (
            f'[[package]]\nname="witness"\nversion="1.2.3"\n'
            f'source="registry+https://github.com/rust-lang/crates.io-index"\nchecksum="{self.checksum}"\n'
            '[[package]]\nname="git-dep"\nversion="0.1.0"\nsource="git+https://example.invalid/repo#0123456789"\n'
        ).encode()

    def write_archive(self) -> None:
        """Create a crate with one Cargo-excluded file; example: fixture.write_archive()."""
        with tarfile.open(self.archives / "witness-1.2.3.crate", "w:gz") as archive:
            for name, content in (self.contents | {".gitignore": b"ignored"}).items():
                member = tarfile.TarInfo(f"witness-1.2.3/{name}")
                member.size, member.mode = len(content), 0o644
                archive.addfile(member, io.BytesIO(content))

    def write_checksum(self, contents: dict[str, bytes]) -> None:
        """Generate fixture metadata independently of the verifier; example: fixture.write_checksum(files)."""
        files = {
            name: hashlib.sha256(content).hexdigest()
            for name, content in contents.items()
        }
        record = {"$comment": COMMENT, "files": files, "package": self.checksum}
        (self.package / ".cargo-checksum.json").write_text(
            json.dumps(record, sort_keys=True, separators=(",", ":"))
        )

    def verify(self) -> dict[str, object]:
        """Check the current fixture; example: fixture.verify()."""
        return verify_registry_vendor(
            self.lock, self.vendor, self.archives, snapshot_runtime(self.vendor)
        )


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"), "requires no-atime snapshots"
)
class RegistryVendorCheckTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.fixture = RegistryVendorFixture(Path(self.temp.name))

    def test_registry_matches_and_git_is_explicitly_unverified(self) -> None:
        result = self.fixture.verify()
        self.assertEqual(result["registry_packages_checked"], 1)
        self.assertEqual(result["git_packages_unverified"], ["git-dep 0.1.0"])
        self.assertFalse(result["complete"])

    def test_modified_file_and_forged_checksum_are_rejected(self) -> None:
        changed = dict(self.fixture.contents)
        changed["src/lib.rs"] = b"altered"
        (self.fixture.package / "src/lib.rs").write_bytes(changed["src/lib.rs"])
        self.fixture.write_checksum(changed)
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            self.fixture.verify()

    def test_altered_archive_and_metadata_are_rejected(self) -> None:
        archive = self.fixture.archives / "witness-1.2.3.crate"
        original = archive.read_bytes()
        archive.write_bytes(original + b"changed")
        with self.assertRaisesRegex(ValueError, "locked SHA-256"):
            self.fixture.verify()
        archive.write_bytes(original)
        (self.fixture.package / ".cargo-checksum.json").write_text("{}")
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            self.fixture.verify()

    def test_extra_missing_and_empty_paths_are_rejected(self) -> None:
        for name in ("extra", "missing", "empty"):
            with self.subTest(name=name):
                item = self.fixture.package / "src/lib.rs"
                if name == "extra":
                    (self.fixture.package / "extra").write_text("x")
                if name == "missing":
                    item.unlink()
                if name == "empty":
                    (self.fixture.package / "empty").mkdir()
                with self.assertRaisesRegex(ValueError, "expected package files"):
                    self.fixture.verify()
                if name == "extra":
                    (self.fixture.package / "extra").unlink()
                if name == "missing":
                    item.write_bytes(self.fixture.contents["src/lib.rs"])
                if name == "empty":
                    (self.fixture.package / "empty").rmdir()

    def test_changed_permissions_and_symlink_are_rejected(self) -> None:
        item = self.fixture.package / "src/lib.rs"
        item.chmod(0o755)
        with self.assertRaisesRegex(ValueError, "executable bits"):
            self.fixture.verify()
        item.unlink()
        item.symlink_to(self.fixture.archives / "witness-1.2.3.crate")
        with self.assertRaisesRegex(ValueError, "regular file"):
            self.fixture.verify()

    def test_stale_manifest_observation_is_rejected(self) -> None:
        observed = snapshot_runtime(self.fixture.vendor)
        (self.fixture.package / "Cargo.toml").write_text("changed")
        with self.assertRaisesRegex(ValueError, "recorded metadata"):
            verify_registry_vendor(
                self.fixture.lock, self.fixture.vendor, self.fixture.archives, observed
            )

    def test_archive_root_directory_is_allowed(self) -> None:
        package = LockedPackage("witness", "1.2.3", "registry", "a" * 64)
        with tempfile.TemporaryFile() as stream:
            with tarfile.open(fileobj=stream, mode="w") as archive:
                root = tarfile.TarInfo("witness-1.2.3/")
                root.type = tarfile.DIRTYPE
                archive.addfile(root)
                member = tarfile.TarInfo("witness-1.2.3/file")
                member.size = 1
                archive.addfile(member, io.BytesIO(b"x"))
            stream.seek(0)
            with tarfile.open(fileobj=stream, mode="r:") as archive:
                self.assertEqual(set(archive_entries(archive, package)), {"file"})

    def test_unsafe_archive_paths_and_links_are_rejected(self) -> None:
        package = LockedPackage("witness", "1.2.3", "registry", "a" * 64)
        for name, kind in (
            ("witness-1.2.3/../escape", tarfile.REGTYPE),
            ("witness-1.2.3/link", tarfile.SYMTYPE),
        ):
            with self.subTest(name=name), tempfile.TemporaryFile() as stream:
                with tarfile.open(fileobj=stream, mode="w") as archive:
                    member = tarfile.TarInfo(name)
                    member.type = kind
                    if kind == tarfile.SYMTYPE:
                        member.linkname = "/outside"
                    archive.addfile(member)
                stream.seek(0)
                with tarfile.open(fileobj=stream, mode="r:") as archive:
                    with self.assertRaises(ValueError):
                        archive_entries(archive, package)

    def test_git_only_replacement_configuration(self) -> None:
        lock = (
            b'[[package]]\nname="git-dep"\nversion="0.1.0"\n'
            b'source="git+https://example.invalid/repo?rev=abcdef0#0123456789"\n'
        )
        config = (
            b'[source."git+https://example.invalid/repo?rev=abcdef0"]\n'
            b'git="https://example.invalid/repo"\nrev="abcdef0"\nreplace-with="vendored-sources"\n'
            b'[source.vendored-sources]\ndirectory="/vendor"\n'
        )
        verify_vendor_config(lock, config)
        with self.assertRaisesRegex(ValueError, "expected only locked sources"):
            verify_vendor_config(lock, config.replace(b"/vendor", b"/usr"))

    def test_duplicate_and_unknown_sources_are_rejected(self) -> None:
        duplicate = (
            self.fixture.lock
            + b'[[package]]\nname="witness"\nversion="1.2.3"\nsource="git+https://example.invalid/repo#abc"\n'
        )
        with self.assertRaisesRegex(ValueError, "one external source"):
            locked_packages(duplicate)
        other = self.fixture.lock.replace(
            b"registry+https://github.com/rust-lang/crates.io-index",
            b"registry+https://other.invalid",
        )
        with self.assertRaisesRegex(ValueError, "expected crates.io or Git"):
            locked_packages(other)


if __name__ == "__main__":
    unittest.main()

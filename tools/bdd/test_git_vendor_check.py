# SPDX-License-Identifier: GPL-2.0-only
"""Test partial Git vendor verification with independent committed fixtures."""

import hashlib
import json
import os
import shutil
import tempfile
import unittest
from pathlib import Path

from git_vendor_check import package_entries, verify_git_vendor
from runtime_snapshot import snapshot_runtime
from test_committed_build_source import SourceRepository
from test_registry_vendor_check import COMMENT


class GitVendorFixture:
    """A committed crate and separately generated vendor manifest."""

    def __init__(self, root: Path) -> None:
        self.repo = root / "repo"
        self.repo.mkdir()
        self.git = SourceRepository(self.repo)
        crate = self.repo / "crates" / "witness"
        (crate / "src").mkdir(parents=True)
        (crate / "Cargo.toml").write_text(
            '[package]\nname="witness"\nversion="1.2.3"\n'
        )
        (crate / "src/lib.rs").write_text("committed\n")
        self.git.git("add", ".")
        self.git.git("commit", "-qm", "add witness")
        revision = self.git.git("rev-parse", "HEAD")
        self.source = f"git+https://example.invalid/repo?rev={revision}#{revision}"
        self.lock = f'[[package]]\nname="witness"\nversion="1.2.3"\nsource="{self.source}"\n'.encode()
        self.vendor = root / "vendor"
        self.package = self.vendor / "witness"
        (self.package / "src").mkdir(parents=True)
        (self.package / "Cargo.toml").write_text(
            '[package]\nname="witness"\nversion="1.2.3"\nedition="2015"\n'
            'readme=false\n'
            'autolib=false\nautobins=false\nautoexamples=false\nautotests=false\nautobenches=false\n'
            '[lib]\nname="witness"\npath="src/lib.rs"\n'
        )
        (self.package / "src/lib.rs").write_text("committed\n")
        self.write_checksum()

    def write_checksum(self) -> None:
        """Write Cargo-like metadata; example: fixture.write_checksum()."""
        names = ["Cargo.toml", "src/lib.rs"]
        files = {
            name: hashlib.sha256((self.package / name).read_bytes()).hexdigest()
            for name in names
        }
        record = {"$comment": COMMENT, "files": files, "package": None}
        (self.package / ".cargo-checksum.json").write_text(
            json.dumps(record, sort_keys=True, separators=(",", ":"))
        )

    def verify(self) -> dict[str, object]:
        """Check the fixture; example: fixture.verify()."""
        return verify_git_vendor(
            self.lock,
            self.vendor,
            snapshot_runtime(self.vendor),
            {self.source: self.repo},
        )


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"), "requires no-atime snapshots"
)
class GitVendorCheckTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.fixture = GitVendorFixture(Path(self.temp.name))

    def test_git_content_matches_but_manifest_remains_unverified(self) -> None:
        (self.fixture.repo / "crates/witness/src/lib.rs").write_text("dirty checkout")
        result = self.fixture.verify()
        self.assertEqual(result["git_files_checked"], 1)
        self.assertEqual(result["git_manifests_unverified"], ["witness 1.2.3"])
        self.assertFalse(result["complete"])
        self.assertFalse(result["reusable"])

    def test_partial_mode_reports_unchecked_sources_and_keeps_strict_default(self) -> None:
        other = self.fixture.vendor / "other"
        shutil.copytree(self.fixture.package, other)
        manifest = other / "Cargo.toml"
        manifest.write_text(manifest.read_text().replace('name="witness"', 'name="other"'))
        checksum = json.loads((other / ".cargo-checksum.json").read_text())
        checksum["files"]["Cargo.toml"] = hashlib.sha256(manifest.read_bytes()).hexdigest()
        (other / ".cargo-checksum.json").write_text(json.dumps(checksum, sort_keys=True, separators=(",", ":")))
        missing = "git+https://example.invalid/other?rev=" + "b" * 40 + "#" + "b" * 40
        lock = self.fixture.lock + f'[[package]]\nname="other"\nversion="1.2.3"\nsource="{missing}"\n'.encode()
        repositories = {self.fixture.source: self.fixture.repo}
        observed = snapshot_runtime(self.fixture.vendor)
        with self.assertRaisesRegex(ValueError, "expected exactly"):
            verify_git_vendor(lock, self.fixture.vendor, observed, repositories)
        result = verify_git_vendor(lock, self.fixture.vendor, observed, repositories, allow_partial=True)
        self.assertEqual(result["git_sources_unchecked"], [missing])
        self.assertEqual(result["git_packages_unchecked"], ["other 1.2.3"])
        self.assertEqual(result["git_manifests_unverified"], ["other 1.2.3", "witness 1.2.3"])
        self.assertFalse(result["complete"])
        self.assertFalse(result["reusable"])
        checked_file = self.fixture.package / "src/lib.rs"
        checked_file.write_text("tampered\n")
        with self.assertRaises(ValueError):
            verify_git_vendor(lock, self.fixture.vendor, snapshot_runtime(self.fixture.vendor), repositories, allow_partial=True)
        with self.assertRaisesRegex(ValueError, "nonempty subset"):
            verify_git_vendor(lock, self.fixture.vendor, observed, {}, allow_partial=True)
        with self.assertRaisesRegex(ValueError, "nonempty subset"):
            verify_git_vendor(lock, self.fixture.vendor, observed, {"unexpected": self.fixture.repo}, allow_partial=True)

    def test_forged_vendor_checksum_does_not_hide_modified_blob(self) -> None:
        (self.fixture.package / "src/lib.rs").write_text("changed")
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            self.fixture.verify()

    def test_forged_generated_build_script_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text().replace('[lib]', 'build="untrusted.rs"\n[lib]'))
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected no build script"):
            self.fixture.verify()

    def test_unchecked_manifest_metadata_remains_unverified(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text().replace('[lib]', 'unverified-field="value"\n[lib]'))
        self.fixture.write_checksum()
        result = self.fixture.verify()
        self.assertEqual(result["git_manifest_declarations_checked"], 1)
        self.assertFalse(result["complete"])
        self.assertFalse(result["reusable"])

    def test_forged_metadata_and_lints_are_refused_with_updated_checksum(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        for change in ('description="forged"\n[lib]', '[lib]'):
            with self.subTest(change=change):
                original = manifest.read_text()
                updated = original.replace('[lib]', change)
                if change == '[lib]':
                    updated += '[lints.rust]\nunsafe_code="allow"\n'
                manifest.write_text(updated)
                self.fixture.write_checksum()
                with self.assertRaisesRegex(ValueError, "expected locked source"):
                    self.fixture.verify()
                manifest.write_text(original)
                self.fixture.write_checksum()

    def test_forged_copied_metadata_with_updated_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text() + '[package.metadata.attack]\nflag=true\n')
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, 'metadata.*expected locked source'):
            self.fixture.verify()

    def test_forged_license_file_with_updated_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text().replace('[lib]', 'license-file="forged"\n[lib]'))
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, 'license-file.*expected locked'):
            self.fixture.verify()

    def test_forged_dependency_feature_and_target_are_refused(self) -> None:
        for section in (
            '[dependencies]\nforged="1"\n',
            '[features]\nforged=[]\n',
            '[target.\'cfg(unix)\'.dependencies]\nforged="1"\n',
        ):
            with self.subTest(section=section):
                path = self.fixture.package / "Cargo.toml"
                original = path.read_text()
                path.write_text(original + section)
                self.fixture.write_checksum()
                with self.assertRaisesRegex(ValueError, "expected locked source declaration"):
                    self.fixture.verify()
                path.write_text(original)
                self.fixture.write_checksum()

    def test_forged_proc_macro_manifest_with_updated_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text() + 'proc-macro=true\n')
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected .*src/lib.rs"):
            self.fixture.verify()

    def test_forged_normalized_target_with_updated_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text() + '[[test]]\nname="forged"\npath="src/lib.rs"\n')
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected locked targets"):
            self.fixture.verify()

    def test_missing_extra_or_executable_files_are_rejected(self) -> None:
        source = self.fixture.package / "src/lib.rs"
        for change in ("missing", "extra", "mode"):
            with self.subTest(change=change):
                if change == "missing":
                    source.unlink()
                elif change == "extra":
                    (self.fixture.package / "extra").write_text("extra")
                else:
                    source.chmod(0o755)
                with self.assertRaises(ValueError):
                    self.fixture.verify()
                source.write_text("committed\n")
                source.chmod(0o644)
                (self.fixture.package / "extra").unlink(missing_ok=True)

    def test_missing_repository_and_unsupported_tree_object_fail(self) -> None:
        with self.assertRaisesRegex(ValueError, "expected exactly"):
            verify_git_vendor(
                self.fixture.lock,
                self.fixture.vendor,
                snapshot_runtime(self.fixture.vendor),
                {},
            )
        with self.assertRaisesRegex(ValueError, "regular tracked blob"):
            package_entries({"crate/Cargo.toml": "120000:blob:bad"}, "crate")


if __name__ == "__main__":
    unittest.main()

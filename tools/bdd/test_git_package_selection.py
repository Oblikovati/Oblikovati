# SPDX-License-Identifier: GPL-2.0-only
"""Check that the Git package diagnostic rejects ambiguous or changed inputs."""

import hashlib
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from git_package_selection import (
    SandboxPaths,
    SelectionRequest,
    diagnose,
    execute_package_list,
    package_list_command,
    selected_entries,
)
from test_git_vendor_check import GitVendorFixture


class FakeCargoSelection:
    """Return a Cargo-like list and optionally change the staged input."""

    def __init__(self, listing: str, mutate: bool = False) -> None:
        self.listing = listing
        self.mutate = mutate
        self.command: list[str] = []
        self.vendor_to_mutate: Path | None = None

    def __call__(self, command: list[str]) -> str:
        self.command = command
        if self.mutate:
            source = Path(command[command.index("/source") - 1])
            (source / "crates/witness/src/lib.rs").write_text(
                "changed during selection"
            )
        if self.vendor_to_mutate is not None:
            self.vendor_to_mutate.write_text("changed during selection")
        return self.listing


class FakeProcessRun:
    """Capture the process options without running an external command."""

    def __init__(self) -> None:
        self.arguments: tuple[object, ...] = ()
        self.options: dict[str, object] = {}

    def __call__(
        self, *args: object, **kwargs: object
    ) -> subprocess.CompletedProcess[str]:
        self.arguments = args
        self.options = kwargs
        return subprocess.CompletedProcess(args[0], 0, stdout="Cargo.toml\n")


class SelectionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.entries = {
            "Cargo.toml": "manifest",
            "src/lib.rs": "source",
            "docs/a.md": "extra",
        }

    def test_selected_files_are_bounded_by_tracked_entries(self) -> None:
        listing = "Cargo.toml\nCargo.toml.orig\nsrc/lib.rs\n"
        self.assertEqual(
            selected_entries(listing, self.entries),
            {"Cargo.toml": "manifest", "src/lib.rs": "source"},
        )

    def test_duplicate_unsafe_missing_and_untracked_paths_are_refused(self) -> None:
        prefix = "Cargo.toml\nsrc/lib.rs\n"
        for listing in (
            "",
            prefix + "src/lib.rs\n",
            prefix + "../outside\n",
            prefix + "/outside\n",
            prefix + "src//bad\n",
            prefix + "untracked\n",
            prefix + "src\\bad\n",
            prefix + "src/bad\r\n",
            prefix.rstrip("\n"),
            "src/lib.rs\n",
            "Cargo.toml\n",
        ):
            with self.subTest(listing=listing), self.assertRaises(ValueError):
                selected_entries(listing, self.entries)

    def test_runner_captures_output_without_inheriting_environment(self) -> None:
        process = FakeProcessRun()
        with patch("git_package_selection.subprocess.run", process):
            self.assertEqual(
                execute_package_list(["/sandbox", "cargo"]), "Cargo.toml\n"
            )
        self.assertEqual(process.arguments, (["/sandbox", "cargo"],))
        self.assertEqual(
            process.options,
            {"capture_output": True, "text": True, "check": True, "env": {}},
        )

    def test_sandbox_command_is_offline_and_clears_inherited_environment(self) -> None:
        command = package_list_command(
            SandboxPaths(Path("/bwrap"), Path("/rust"), Path("/cache")),
            Path("/repo"),
            "witness",
        )
        for flag in (
            "--clearenv",
            "--unshare-all",
            "--disable-userns",
            "--offline",
            "CARGO_NET_OFFLINE",
        ):
            self.assertIn(flag, command)
        self.assertEqual(command[-2:], ["-p", "witness"])
        self.assertNotIn("RUSTFLAGS", command)


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"), "requires no-atime snapshots"
)
class DiagnosticTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.fixture = GitVendorFixture(Path(self.temp.name))
        old = self.fixture.package
        self.fixture.package = self.fixture.vendor / "witness-1.2.3"
        old.rename(self.fixture.package)
        self.revision = self.fixture.git.git("rev-parse", "HEAD")
        self.runner = FakeCargoSelection("Cargo.toml\nCargo.toml.orig\nsrc/lib.rs\n")

    def run_diagnostic(self) -> dict[str, object]:
        sandbox = SandboxPaths(Path("/bwrap"), Path("/rust"), Path("/cache"))
        request = SelectionRequest(
            self.fixture.repo,
            self.revision,
            "witness",
            "1.2.3",
            self.fixture.vendor,
            sandbox,
        )
        return diagnose(request, self.runner)

    def test_checks_locked_contents_and_never_claims_reusable_evidence(self) -> None:
        result = self.run_diagnostic()
        self.assertEqual(result["selected_files_checked"], 1)
        self.assertEqual(
            result["selection_sha256"],
            hashlib.sha256(self.runner.listing.encode()).hexdigest(),
        )
        self.assertFalse(result["manifest_complete"])
        self.assertFalse(result["complete"])
        self.assertFalse(result["reusable"])
        self.assertIn("--clearenv", self.runner.command)

    def test_forged_vendor_file_with_forged_checksum_is_refused(self) -> None:
        (self.fixture.package / "src/lib.rs").write_text("forged")
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            self.run_diagnostic()

    def test_invalid_package_name_is_rejected_before_running_cargo(self) -> None:
        sandbox = SandboxPaths(Path("/bwrap"), Path("/rust"), Path("/cache"))
        request = SelectionRequest(
            self.fixture.repo,
            self.revision,
            "../witness",
            "1.2.3",
            self.fixture.vendor,
            sandbox,
        )
        with self.assertRaisesRegex(ValueError, "expected Cargo package identifiers"):
            diagnose(request, self.runner)
        self.assertEqual(self.runner.command, [])

    def test_forged_manifest_with_forged_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(
            manifest.read_text().replace("[lib]", 'build="forged.rs"\n[lib]')
        )
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected no build script"):
            self.run_diagnostic()

    def test_forged_default_readme_with_forged_checksum_is_refused(self) -> None:
        manifest = self.fixture.package / "Cargo.toml"
        manifest.write_text(manifest.read_text().replace('readme=false', 'readme="forged"'))
        self.fixture.write_checksum()
        with self.assertRaisesRegex(ValueError, "readme.*expected locked"):
            self.run_diagnostic()

    def test_vendor_changed_during_selection_is_refused(self) -> None:
        self.runner.vendor_to_mutate = self.fixture.package / "src/lib.rs"
        with self.assertRaisesRegex(ValueError, "expected unchanged source and vendor"):
            self.run_diagnostic()

    def test_source_changed_during_selection_is_refused(self) -> None:
        self.runner.mutate = True
        with self.assertRaisesRegex(ValueError, "expected unchanged source"):
            self.run_diagnostic()


if __name__ == "__main__":
    unittest.main()

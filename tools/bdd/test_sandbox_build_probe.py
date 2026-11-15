# SPDX-License-Identifier: GPL-2.0-only
"""Test the deliberately non-reusable offline build probe."""

import json
import os
import tempfile
import unittest
from pathlib import Path
from dataclasses import replace
from unittest.mock import patch

from committed_build_source import materialize_source
from test_committed_build_source import SourceRepository
from test_registry_vendor_check import RegistryVendorFixture
from test_git_vendor_check import GitVendorFixture
from runtime_snapshot import snapshot_runtime

from sandbox_build_probe import (
    BuildProbe,
    build_environment,
    check_source_cargo_config,
    require_probe,
    run_build_probe,
    run_committed_build_probe,
    runner_path,
    sandbox_build_command,
)


class FakeBuildProcess:
    """Write a Cargo artifact response for a fixed test binary."""

    def __init__(self, target: Path, code: int = 0) -> None:
        self.target, self.code = target, code
        self.command: list[str] = []

    def __call__(self, command: list[str], messages: Path, log: Path) -> int:
        self.command = command
        runner = self.target / "debug/deps/head-test"
        runner.parent.mkdir(parents=True)
        runner.write_bytes(b"test runner")
        runner.chmod(0o755)
        messages.write_text(
            json.dumps(
                {
                    "reason": "compiler-artifact",
                    "target": {"name": "oblikovati_head"},
                    "profile": {"test": True},
                    "executable": "/build/debug/deps/head-test",
                }
            )
            + "\n"
        )
        log.write_text("build log")
        return self.code


class FakeInputSnapshot:
    """Return stable input identities or simulate a source change."""

    def __init__(self, changed: bool = False) -> None:
        self.count = 0
        self.changed = changed

    def __call__(self, path: Path) -> dict[str, object]:
        self.count += 1
        return {
            "path": str(path),
            "version": 2 if self.changed and self.count > 4 else 1,
        }


@unittest.skipUnless(
    os.name == "posix" and hasattr(os, "O_NOATIME"), "requires Linux no-atime snapshots"
)
class SandboxBuildProbeTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        root = Path(self.temporary.name)
        roots = [
            root / name
            for name in ("source", "vendor", "toolchain", "config", "target", "logs")
        ]
        for path in roots:
            path.mkdir()
        (roots[3] / "config.toml").write_text("[source.vendored-sources]\n")
        bwrap = root / "bwrap"
        bwrap.write_bytes(b"sandbox executable")
        self.probe = BuildProbe(*roots, bwrap)

    def test_build_observes_inputs_and_never_claims_reuse(self) -> None:
        process = FakeBuildProcess(self.probe.target)
        result = run_build_probe(self.probe, process, FakeInputSnapshot())
        self.assertFalse(result["reusable"])
        self.assertEqual(len(result["runner_sha256"]), 64)
        self.assertEqual(
            set(result["input_inventory_sha256"]),
            {"source", "vendor", "toolchain", "config"},
        )
        self.assertEqual(len(result["sandbox_sha256"]), 64)
        self.assertIn("--unshare-all", process.command)
        self.assertIn("--disable-userns", process.command)
        self.assertIn("--clearenv", process.command)
        self.assertIn("--offline", process.command)
        self.assertNotIn("RUSTFLAGS", build_environment())
        self.assertIn("/config/config.toml", sandbox_build_command(self.probe))

    def test_committed_source_is_checked_before_the_build(self) -> None:
        repo = self.probe.source.parent / "repository"
        repo.mkdir()
        fixture = SourceRepository(repo)
        self.probe.source.rmdir()
        materialize_source(repo, fixture.commit, self.probe.source)
        (self.probe.source / "source.rs").write_text("tampered")
        process = FakeBuildProcess(self.probe.target)
        with self.assertRaisesRegex(ValueError, "expected committed content"):
            run_committed_build_probe(self.probe, repo, fixture.commit, process)
        self.assertEqual(process.command, [])
        (self.probe.source / "source.rs").write_text("committed")
        result = run_committed_build_probe(self.probe, repo, fixture.commit, process)
        self.assertEqual(result["source_commit"], fixture.commit)
        self.assertFalse(result["reusable"])

    def test_registry_mismatch_stops_committed_build(self) -> None:
        root = self.probe.source.parent
        fixture_root = root / "registry-fixture"
        fixture_root.mkdir()
        vendor = RegistryVendorFixture(fixture_root)
        (self.probe.config / "config.toml").write_text(
            '[source.crates-io]\nreplace-with="vendored-sources"\n'
            '[source."git+https://example.invalid/repo"]\n'
            'git="https://example.invalid/repo"\nreplace-with="vendored-sources"\n'
            '[source.vendored-sources]\ndirectory="/vendor"\n'
        )
        repo = root / "repository"
        repo.mkdir()
        git = SourceRepository(repo)
        (repo / "Cargo.lock").write_bytes(vendor.lock)
        git.git("add", "Cargo.lock")
        git.git("commit", "-qm", "lock registry")
        revision = git.git("rev-parse", "HEAD")
        self.probe.source.rmdir()
        materialize_source(repo, revision, self.probe.source)
        probe = replace(self.probe, vendor=vendor.vendor)
        process = FakeBuildProcess(probe.target)
        (vendor.package / "src/lib.rs").write_text("forged")
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            run_committed_build_probe(
                probe, repo, revision, process, archives=vendor.archives
            )
        self.assertEqual(process.command, [])
        (vendor.package / "src/lib.rs").write_bytes(vendor.contents["src/lib.rs"])
        config_path = probe.config / "config.toml"
        valid_config = config_path.read_text()
        config_path.write_text(
            valid_config.replace('directory="/vendor"', 'directory="/usr"')
        )
        with self.assertRaisesRegex(ValueError, "expected only locked sources"):
            run_committed_build_probe(
                probe, repo, revision, process, archives=vendor.archives
            )
        self.assertEqual(process.command, [])
        config_path.write_text(valid_config)
        result = run_committed_build_probe(
            probe, repo, revision, process, archives=vendor.archives
        )
        self.assertEqual(result["registry_check"]["registry_packages_checked"], 1)
        self.assertEqual(
            result["registry_check"]["git_packages_unverified"], ["git-dep 0.1.0"]
        )
        self.assertFalse(result["reusable"])

    def test_forged_git_vendor_blob_stops_combined_build(self) -> None:
        root = self.probe.source.parent
        fixture_root = root / "git-fixture"
        fixture_root.mkdir()
        vendor = GitVendorFixture(fixture_root)
        archives = root / "archives"
        archives.mkdir()
        revision_selector = vendor.source.split("?rev=")[1].split("#")[0]
        (self.probe.config / "config.toml").write_text(
            f'[source."git+https://example.invalid/repo?rev={revision_selector}"]\n'
            f'git="https://example.invalid/repo"\nrev="{revision_selector}"\n'
            'replace-with="vendored-sources"\n'
            '[source.vendored-sources]\ndirectory="/vendor"\n'
        )
        repo = root / "repository"
        repo.mkdir()
        git = SourceRepository(repo)
        (repo / "Cargo.lock").write_bytes(vendor.lock)
        git.git("add", "Cargo.lock")
        git.git("commit", "-qm", "lock Git source")
        revision = git.git("rev-parse", "HEAD")
        self.probe.source.rmdir()
        materialize_source(repo, revision, self.probe.source)
        probe = replace(self.probe, vendor=vendor.vendor)
        process = FakeBuildProcess(probe.target)
        with self.assertRaisesRegex(ValueError, "expected registry archives"):
            run_committed_build_probe(
                probe,
                repo,
                revision,
                process,
                git_repositories={vendor.source: vendor.repo},
            )
        with self.assertRaisesRegex(ValueError, "expected exactly"):
            run_committed_build_probe(
                probe, repo, revision, process, archives=archives, git_repositories={}
            )
        self.assertEqual(process.command, [])
        source = vendor.package / "src/lib.rs"
        source.write_text("forged")
        vendor.write_checksum()
        with self.assertRaisesRegex(ValueError, "expected archive content"):
            run_committed_build_probe(
                probe,
                repo,
                revision,
                process,
                archives=archives,
                git_repositories={vendor.source: vendor.repo},
            )
        self.assertEqual(process.command, [])
        source.write_text("committed\n")
        vendor.write_checksum()
        result = run_committed_build_probe(
            probe,
            repo,
            revision,
            process,
            archives=archives,
            git_repositories={vendor.source: vendor.repo},
        )
        self.assertEqual(result["vendor_check"]["git"]["git_files_checked"], 1)
        self.assertEqual(
            result["vendor_check"]["git"]["git_manifests_unverified"], ["witness 1.2.3"]
        )
        self.assertFalse(result["reusable"])

    def test_changed_inputs_and_failed_build_are_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "inputs changed"):
            run_build_probe(
                self.probe, FakeBuildProcess(self.probe.target), FakeInputSnapshot(True)
            )
        for path in (self.probe.target, self.probe.logs):
            for child in path.iterdir():
                if child.is_dir():
                    import shutil

                    shutil.rmtree(child)
                else:
                    child.unlink()
        with self.assertRaisesRegex(ValueError, "build exit 3"):
            run_build_probe(
                self.probe, FakeBuildProcess(self.probe.target, 3), FakeInputSnapshot()
            )

    def test_source_cargo_overrides_are_rejected(self) -> None:
        folder = self.probe.source / ".cargo"
        folder.mkdir()
        config = folder / "config.toml"
        config.write_text("[build]\njobs=2\n")
        check_source_cargo_config(self.probe, snapshot_runtime(self.probe.source))
        config.write_text('[source.crates-io]\nreplace-with="host"\n')
        with self.assertRaisesRegex(ValueError, "expected no source overrides"):
            check_source_cargo_config(self.probe, snapshot_runtime(self.probe.source))
        config.unlink()
        (folder / "config").write_text("[build]\njobs=2\n")
        with self.assertRaisesRegex(ValueError, "no legacy Cargo configuration"):
            check_source_cargo_config(self.probe, snapshot_runtime(self.probe.source))

    def test_refuses_overlap_occupied_output_and_unsupported_host(self) -> None:
        overlap = BuildProbe(
            self.probe.source,
            self.probe.source,
            self.probe.toolchain,
            self.probe.config,
            self.probe.target,
            self.probe.logs,
            self.probe.bwrap,
        )
        with self.assertRaisesRegex(ValueError, "disjoint"):
            require_probe(overlap)
        (self.probe.logs / "existing").write_text("keep")
        with self.assertRaisesRegex(ValueError, "empty directories"):
            require_probe(self.probe)
        with patch("sandbox_build_probe.platform.machine", return_value="arm64"):
            with self.assertRaisesRegex(ValueError, "Linux x86_64"):
                require_probe(self.probe)

    def test_runner_rejects_a_symlink_parent(self) -> None:
        outside = self.probe.source / "outside"
        outside.mkdir()
        (outside / "runner").write_text("runner")
        (outside / "runner").chmod(0o755)
        (self.probe.target / "linked").symlink_to(outside, target_is_directory=True)
        messages = self.probe.logs / "messages"
        record = {
            "reason": "compiler-artifact",
            "target": {"name": "oblikovati_head"},
            "profile": {"test": True},
            "executable": "/build/linked/runner",
        }
        messages.write_text(json.dumps(record) + "\n")
        with self.assertRaisesRegex(ValueError, "within"):
            runner_path(messages, self.probe.target)

    def test_runner_rejects_escaping_and_duplicate_artifacts(self) -> None:
        messages = self.probe.logs / "messages"
        record = {
            "reason": "compiler-artifact",
            "target": {"name": "oblikovati_head"},
            "profile": {"test": True},
            "executable": "/build/../outside",
        }
        messages.write_text(json.dumps(record) + "\n")
        with self.assertRaisesRegex(ValueError, "inside /build"):
            runner_path(messages, self.probe.target)
        runner = self.probe.target / "runner"
        runner.write_text("runner")
        runner.chmod(0o755)
        record["executable"] = "/build/runner"
        messages.write_text((json.dumps(record) + "\n") * 2)
        with self.assertRaisesRegex(ValueError, "one regular"):
            runner_path(messages, self.probe.target)


if __name__ == "__main__":
    unittest.main()

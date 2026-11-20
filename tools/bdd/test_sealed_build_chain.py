# SPDX-License-Identifier: GPL-2.0-only
"""Check the in-process link from an observed build to a sealed run."""

import hashlib
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from sandbox_build_probe import BuildProbe
from sealed_build_chain import run_sealed_build
from sealed_execution import file_identity
from test_sealed_execution import SealedFixture


class FakeCommittedBuild:
    """Produce one runner and optionally change it or the sandbox after observation."""

    def __init__(self, change: str = "") -> None:
        self.change = change

    def __call__(self, probe: BuildProbe, repository: Path, revision: str, **options: object) -> dict[str, object]:
        runner = probe.target / "runner"
        runner.write_bytes(b"built runner")
        runner.chmod(0o755)
        digest = hashlib.sha256(b"built runner").hexdigest()
        desktop = probe.target / "desktop"
        desktop.write_bytes(b"built desktop")
        desktop.chmod(0o755)
        desktop_digest = hashlib.sha256(b"built desktop").hexdigest()
        if self.change == "desktop":
            desktop.write_bytes(b"changed desktop")
        sandbox = file_identity(probe.bwrap)["sha256"]
        if self.change == "runner":
            runner.write_bytes(b"changed runner")
        if self.change == "sandbox":
            probe.bwrap.write_bytes(b"changed sandbox")
        return {"runner": str(runner), "runner_sha256": digest, "sandbox_sha256": sandbox, "source_commit": revision, "desktop": str(desktop), "desktop_sha256": desktop_digest, "reusable": True}


class FakePinnedExecution:
    """Record the expected build digest and produce one successful run log."""

    def __init__(self, runner: Path, digest: str, desktop: tuple[Path, str]) -> None:
        self.runner, self.digest = runner, digest
        self.desktop = desktop

    def __call__(self, command: list[str], log: Path) -> int:
        if hashlib.sha256(self.runner.read_bytes()).hexdigest() != self.digest:
            raise ValueError("runner digest: expected build bytes")
        if hashlib.sha256(self.desktop[0].read_bytes()).hexdigest() != self.desktop[1]:
            raise ValueError("desktop digest: expected build bytes")
        log.write_text("executed")
        return 0


@unittest.skipUnless(os.name == "posix" and hasattr(os, "O_NOATIME"), "requires Linux snapshots")
class SealedBuildChainTests(unittest.TestCase):
    def fixture(self, root: Path) -> tuple[BuildProbe, SealedFixture]:
        """Prepare distinct build and execution roots; example: self.fixture(root)."""
        fixture = SealedFixture(root)
        paths = [root / name for name in ("build-source", "vendor", "toolchain", "config", "target", "build-logs")]
        for path in paths:
            path.mkdir()
        system = root / "system"
        system.mkdir()
        return BuildProbe(*paths, fixture.bwrap, system), fixture

    def test_chain_uses_build_artifact_and_never_claims_reuse(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            probe, fixture = self.fixture(Path(folder))
            with patch("sealed_build_chain.run_committed_build_probe", FakeCommittedBuild()), patch("sealed_build_chain.PinnedRunnerProcess", FakePinnedExecution):
                result = run_sealed_build(probe, fixture.request(), probe.source, "a" * 40, fixture.verify)
            self.assertFalse(result["reusable"])
            self.assertEqual(result["execution"]["runner_input"]["sha256"], result["build"]["runner_sha256"])
            self.assertEqual(fixture.log.read_text(), "executed")

    def test_build_output_cannot_overlap_execution_reports(self) -> None:
        from dataclasses import replace

        with tempfile.TemporaryDirectory() as folder:
            probe, fixture = self.fixture(Path(folder))
            request = replace(fixture.request(), reports=probe.logs)
            with patch("sealed_build_chain.run_committed_build_probe", FakeCommittedBuild()):
                with self.assertRaisesRegex(ValueError, "expected disjoint directories"):
                    run_sealed_build(probe, request, probe.source, "a" * 40, fixture.verify)
            self.assertFalse((probe.target / "runner").exists())

    def test_changed_build_artifact_and_sandbox_are_refused(self) -> None:
        for change in ("runner", "sandbox", "desktop"):
            with self.subTest(change=change), tempfile.TemporaryDirectory() as folder:
                probe, fixture = self.fixture(Path(folder))
                with patch("sealed_build_chain.run_committed_build_probe", FakeCommittedBuild(change)):
                    with self.assertRaisesRegex(ValueError, "expected build SHA-256"):
                        run_sealed_build(probe, fixture.request(), probe.source, "a" * 40, fixture.verify)
                self.assertFalse(fixture.log.exists())

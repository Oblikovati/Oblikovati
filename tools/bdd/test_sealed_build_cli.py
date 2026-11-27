# SPDX-License-Identifier: GPL-2.0-only
"""Test the diagnostic CLI's inputs, dispatch and non-reusable result."""

import contextlib
import io
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from sandbox_build_probe import BuildProbe
from sealed_build_cli import main, pinned_scenario_revision
from verify_snapshot import read_snapshot_metadata, restore_snapshot
from sealed_execution import SealedRequest
from test_sealed_execution import SealedFixture


class FakeChainRun:
    """Record requests passed by the CLI and return a selected result."""

    def __init__(self, reusable: bool = False) -> None:
        self.reusable = reusable
        self.calls: list[tuple[BuildProbe, SealedRequest, Path, str, object, dict[str, object]]] = []

    def __call__(self, probe: BuildProbe, request: SealedRequest, repository: Path, revision: str, verify: object, **options: object) -> dict[str, object]:
        self.calls.append((probe, request, repository, revision, verify, options))
        return {"reusable": self.reusable, "diagnostic": "fixture"}


class FakeInventoryReader:
    """Return a selected row whose source exists in the fixture checkout."""

    def __init__(self, checkout: Path) -> None:
        self.checkout = checkout
        path = checkout / "features/application/document-opening/scenario.feature"
        path.parent.mkdir(parents=True)
        path.write_bytes(b"fixture source")

    def __call__(self, repository: Path, revision: str) -> dict[str, object]:
        return {"source_revision": "a" * 40, "scenarios": [{
            "id": "fixture", "area": "application", "feature": "document-opening",
            "scenario": "Open a part", "path": "features/application/document-opening/scenario.feature",
            "source_sha256": hashlib.sha256(b"fixture source").hexdigest(),
        }]}


class FakeReportChecker:
    """Record the report check requested by the CLI."""

    def __init__(self) -> None:
        self.calls: list[tuple[object, ...]] = []

    def __call__(self, request: SealedRequest, row: object, revision: str, capture: str | None) -> dict[str, object]:
        self.calls.append((request, row, revision, capture))
        return {"reusable": False}


@unittest.skipUnless(os.name == "posix", "requires POSIX fixture paths")
class SealedBuildCliTests(unittest.TestCase):
    def arguments(self, root: Path, fixture: SealedFixture) -> list[str]:
        """Make explicit CLI inputs; example: self.arguments(root, fixture)."""
        result: list[str] = []
        for name in ("source", "vendor", "toolchain", "config", "target", "build-logs", "system", "repository"):
            path = root / name
            path.mkdir()
            result.extend([f"--{name}", str(path)])
        for name, path in (("bwrap", fixture.bwrap), ("runtime", fixture.runtime), ("checkout", fixture.checkout), ("reports", fixture.reports), ("run-log", fixture.log)):
            result.extend([f"--{name}", str(path)])
        return result + ["--revision", "a" * 40, "--area", "application", "--feature", "document-opening", "--scenario", "Open a part"]

    def test_cli_dispatches_explicit_inputs_and_never_claims_reuse(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            arguments = self.arguments(root, fixture)
            invoke = FakeChainRun()
            output = io.StringIO()
            report = FakeReportChecker()
            with contextlib.redirect_stdout(output):
                main(arguments, invoke, fixture.verify, FakeInventoryReader(fixture.checkout), report)
            self.assertEqual(len(report.calls), 1)
            self.assertFalse(json.loads(output.getvalue())["reusable"])
            probe, request, repository, revision, verify, options = invoke.calls[0]
            self.assertEqual((probe.target, request.binary), (root / "target", root / "target/build-runner"))
            self.assertEqual((repository, revision), (root / "repository", "a" * 40))
            self.assertEqual(fixture.verify.calls, [fixture.checkout])
            self.assertIsNone(options["archives"])

    def test_cli_passes_explicit_private_proc_request(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            (fixture.runtime / "proc").mkdir()
            arguments = self.arguments(root, fixture) + ["--diagnostic-proc"]
            invoke = FakeChainRun()
            with contextlib.redirect_stdout(io.StringIO()):
                main(arguments, invoke, fixture.verify, FakeInventoryReader(fixture.checkout), FakeReportChecker())
            self.assertTrue(invoke.calls[0][1].diagnostic_proc)

    def test_cli_passes_vendor_check_inputs_to_the_chain(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            arguments = self.arguments(root, fixture)
            archives = root / "archives"
            archives.mkdir()
            arguments += ["--registry-archives", str(archives), "--git-repo", "git+fixture", str(root / "repository")]
            invoke = FakeChainRun()
            report = FakeReportChecker()
            with contextlib.redirect_stdout(io.StringIO()):
                main(arguments, invoke, fixture.verify, FakeInventoryReader(fixture.checkout), report)
            self.assertEqual(len(report.calls), 1)
            options = invoke.calls[0][-1]
            self.assertEqual(options["archives"], archives)
            self.assertEqual(options["git_repositories"], {"git+fixture": root / "repository"})

    def test_cli_rejects_claimed_reusable_output(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            with self.assertRaisesRegex(ValueError, "expected false"):
                main(self.arguments(root, fixture), FakeChainRun(True), fixture.verify, FakeInventoryReader(fixture.checkout), FakeReportChecker())

    def test_zero_exit_without_report_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            arguments = self.arguments(root, fixture)
            with self.assertRaisesRegex(ValueError, "cucumber.json"):
                main(arguments, FakeChainRun(), fixture.verify, FakeInventoryReader(fixture.checkout))

    def test_checkout_verifier_uses_the_pinned_snapshot(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            checkout = Path(folder) / "checkout"
            restore_snapshot(checkout)
            metadata, _ = read_snapshot_metadata()
            self.assertEqual(pinned_scenario_revision(checkout), metadata["snapshot_commit"])
            (checkout / "extra").write_text("untracked")
            with self.assertRaisesRegex(ValueError, "expected clean Git status"):
                pinned_scenario_revision(checkout)

    def test_real_cli_rejects_git_mapping_without_archives(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            fixture = SealedFixture(root)
            cli = Path(__file__).with_name("sealed_build_cli.py")
            command = [sys.executable, "-B", str(cli), *self.arguments(root, fixture), "--git-repo", "git+fixture", str(root)]
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertIn("--git-repo requires --registry-archives", result.stderr)

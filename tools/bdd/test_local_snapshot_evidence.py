# SPDX-License-Identifier: GPL-2.0-only
"""Check feature-scoped local snapshot selection and drift detection."""

from dataclasses import replace
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from local_snapshot_evidence import committed_object, scoped_selection
from test_sealed_execution import SealedFixture
import test_sealed_build_cli as cli_tests
from sealed_build_cli import main


class FakeCommittedSnapshot:
    """Return the snapshot objects as if read from the selected application commit."""
    def __init__(self, metadata: dict[str, object], inventory: dict[str, object]) -> None:
        self.values = {"snapshot.json": metadata, "inventory.json": inventory}

    def __call__(self, repository: Path, revision: str, directory: str, name: str) -> dict[str, object]:
        return self.values[name]


class FakeFeatureReportChecker:
    """Record feature report validation without reading external output."""
    def __init__(self) -> None:
        self.rows: list[dict[str, object]] = []

    def __call__(self, request: object, rows: list[dict[str, object]], revision: str) -> dict[str, object]:
        self.rows = rows
        return {"content_checked": True, "reusable": False}


class ScopedSnapshotTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.fixture = SealedFixture(self.root)
        receipt = Path(__file__).resolve().parents[2] / "architecture/rust-port/receipts/2026-11-20-docking-local-snapshot-3978"
        self.metadata = json.loads((receipt / "snapshot.json").read_text())
        self.inventory = json.loads((receipt / "inventory.json").read_text())
        self.checkout = self.root / "snapshot"
        subprocess.run(["git", "clone", "-q", "--branch", "snapshot", str(receipt / "docking.bundle"), str(self.checkout)], check=True)
        self.request = replace(self.fixture.request(), checkout=self.checkout, feature="docking-system", scenario="Restore the saved panel arrangement after restarting the application")

    def test_reads_only_the_selected_committed_receipt(self) -> None:
        repository = self.root / "repository"
        receipt = repository / "architecture/rust-port/receipts/example"
        receipt.mkdir(parents=True)
        (receipt / "snapshot.json").write_text(json.dumps(self.metadata))
        subprocess.run(["git", "init", "-q", str(repository)], check=True)
        subprocess.run(["git", "-C", str(repository), "add", "."], check=True)
        subprocess.run(["git", "-C", str(repository), "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-q", "-m", "snapshot"], check=True)
        revision = subprocess.check_output(["git", "-C", str(repository), "rev-parse", "HEAD"], text=True).strip()
        (receipt / "snapshot.json").write_text("{}")
        value = committed_object(repository, revision, "architecture/rust-port/receipts/example", "snapshot.json")
        self.assertEqual(value["revision"], self.metadata["revision"])
        with self.assertRaisesRegex(ValueError, "relative receipt directory"):
            committed_object(repository, revision, "architecture/rust-port/receipts/../outside", "snapshot.json")

    def test_matching_snapshot_selects_exact_scenario(self) -> None:
        row, verify = scoped_selection(self.request, self.metadata, self.inventory)
        self.assertEqual(row["scenario"], self.request.scenario)
        self.assertEqual(verify(self.checkout), self.metadata["revision"])

    def test_changed_fixture_or_extra_file_is_rejected(self) -> None:
        row, verify = scoped_selection(self.request, self.metadata, self.inventory)
        self.assertEqual(row["scenario"], self.request.scenario)
        (self.checkout / "fixtures/cad/docking-visible-solid-current.opd").write_text("changed")
        with self.assertRaisesRegex(ValueError, "clean Git status"):
            verify(self.checkout)

    def test_cli_dispatches_the_committed_local_snapshot_as_nonreusable(self) -> None:
        bindir = self.fixture.runtime / "usr/bin"
        bindir.mkdir(parents=True)
        for name in ("python3", "openbox", "import", "compare", "xvfb-run", "Xvfb", "xauth"):
            path = bindir / name
            path.write_bytes(b"fixture")
            path.chmod(0o755)
        arguments = cli_tests.SealedBuildCliTests().arguments(self.root, self.fixture)
        arguments[arguments.index("--checkout") + 1] = str(self.checkout)
        arguments[arguments.index("--feature") + 1] = self.request.feature
        arguments[arguments.index("--scenario") + 1] = self.request.scenario
        arguments += ["--local-snapshot", "architecture/rust-port/receipts/fixture"]
        chain, report = cli_tests.FakeChainRun(), cli_tests.FakeReportChecker()
        output = io.StringIO()
        with patch("sealed_build_cli.committed_object", FakeCommittedSnapshot(self.metadata, self.inventory)):
            with contextlib.redirect_stdout(output):
                main(arguments, chain, self.fixture.verify, report_checker=report)
        self.assertFalse(json.loads(output.getvalue())["reusable"])
        self.assertEqual(len(chain.calls), 1)
        self.assertEqual(len(report.calls), 1)

    def test_cli_dispatches_all_scenarios_from_one_local_feature(self) -> None:
        bindir = self.fixture.runtime / "usr/bin"
        bindir.mkdir(parents=True)
        for name in ("python3", "openbox", "import", "compare", "xvfb-run", "Xvfb", "xauth"):
            path = bindir / name
            path.write_bytes(b"fixture")
            path.chmod(0o755)
        arguments = cli_tests.SealedBuildCliTests().arguments(self.root, self.fixture)
        arguments[arguments.index("--checkout") + 1] = str(self.checkout)
        arguments[arguments.index("--feature") + 1] = self.request.feature
        scenario_index = arguments.index("--scenario")
        del arguments[scenario_index:scenario_index + 2]
        arguments += ["--all-scenarios", "--local-snapshot", "architecture/rust-port/receipts/fixture"]
        chain, report = cli_tests.FakeChainRun(), FakeFeatureReportChecker()
        output = io.StringIO()
        with patch("sealed_build_cli.committed_object", FakeCommittedSnapshot(self.metadata, self.inventory)):
            with patch("sealed_build_cli.check_feature_report", report):
                with contextlib.redirect_stdout(output):
                    main(arguments, chain, self.fixture.verify)
        self.assertFalse(json.loads(output.getvalue())["reusable"])
        self.assertTrue(chain.calls[0][1].all_scenarios)
        self.assertEqual(len(report.rows), 10)

    def test_wrong_scope_or_scenario_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "snapshot scope"):
            scoped_selection(replace(self.request, feature="other"), self.metadata, self.inventory)
        with self.assertRaisesRegex(ValueError, "one exact"):
            scoped_selection(replace(self.request, scenario="missing"), self.metadata, self.inventory)


if __name__ == "__main__":
    unittest.main()

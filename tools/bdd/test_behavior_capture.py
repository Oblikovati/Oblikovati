# SPDX-License-Identifier: GPL-2.0-only
"""Exercise receipt publication with named native-run and application-state fakes."""

import copy
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from behavior_capture import CaptureRequest, capture_receipt
from behavior_evidence import (
    StaleEvidence,
    check_capture_receipt,
    example_capture_name,
    verify_receipt,
)
from capture_cli import (
    find_scenario,
    native_command,
    receipt_destination,
    require_renderer,
)
from test_behavior_evidence import EvidenceFixture
from test_behavior_outline import OutlineFixture


class FakeNativeRun:
    def __init__(self, fixture: EvidenceFixture) -> None:
        self.fixture = fixture
        self.exit_code = 0
        self.dirty_source = False
        self.fail_step = False
        self.no_scenarios = False
        self.wrong_checkout = False
        self.captures = ["First.png"]
        self.environment: dict[str, str] = {}

    def __call__(self, reports: Path, environment: dict[str, str]) -> int:
        self.environment = environment
        report = copy.deepcopy(self.fixture.report)
        if self.no_scenarios:
            report = []
        if self.fail_step:
            report[0]["elements"][0]["steps"][0]["result"]["status"] = "failed"
        (reports / "cucumber.json").write_text(json.dumps(report))
        self.write_source(reports)
        (reports / "junit.xml").write_text("<testsuites/>")
        (reports / "native-test.log").write_text("native run\n")
        for name in self.captures:
            (reports / name).write_bytes(b"\x89PNG\r\n\x1a\nfixture")
        return self.exit_code

    def write_source(self, reports: Path) -> None:
        """Write a fake checkout observation; example: runner.write_source(reports)."""
        checkout = "/other" if self.wrong_checkout else self.environment["OBK_BDD_ROOT"]
        state = {"commit": "source", "dirty": self.dirty_source, "root": checkout}
        (reports / "source-revision.json").write_text(
            json.dumps(
                {
                    "before": state,
                    "after": state,
                    "source_changed": False,
                }
            )
        )


class FakeApplicationState:
    def __init__(self, commit: str) -> None:
        self.states = [(commit, True), (commit, True), (commit, True), (commit, True)]

    def __call__(self, root: Path) -> tuple[str, bool]:
        return self.states.pop(0)


class CaptureTests(unittest.TestCase):
    def reviewed(self, identifier: str) -> dict[str, object]:
        """Make a reviewed scenario; example: self.reviewed('scenario:1')."""
        return {
            "id": identifier,
            "source_revision": "source",
            "source_sha256": "source-hash",
            "milestone": "R4",
            "owner": "desktop",
            "target_module": "crates/head",
            "acceptance_test": "native",
            "issues": ["https://github.com/Oblikovati/Oblikovati/issues/1"],
            "state": "inventoried",
        }

    def fixture(
        self, folder: str
    ) -> tuple[EvidenceFixture, CaptureRequest, FakeNativeRun, FakeApplicationState]:
        root = Path(folder)
        fixture = EvidenceFixture(root)
        fixture.receipt()  # The same locked dependency fixture is used by the verifier.
        fixture.row["feature"] = "a"
        request = CaptureRequest(
            root,
            root / "checkout",
            root / "receipts" / "run",
            fixture.row,
            "source",
            "a" * 40,
            "Linux",
            "Vulkan",
            None,
        )
        return fixture, request, FakeNativeRun(fixture), FakeApplicationState("a" * 40)

    def test_success_publishes_a_checked_historical_receipt(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture, request, runner, state = self.fixture(folder)
            artifact = capture_receipt(request, runner, state)
            check_capture_receipt(request.root, artifact, fixture.row, "source", "a" * 40)
            with self.assertRaisesRegex(StaleEvidence, "build provenance is unproven"):
                verify_receipt(request.root, artifact, fixture.row, "source", "a" * 40)
            self.assertEqual(runner.environment["OBK_BDD_FEATURE"], "a")
            self.assertEqual(runner.environment["OBK_BDD_SCENARIO"], "First")
            self.assertTrue((request.destination / "native-test.log.gz").is_file())

    def test_native_runner_and_filters_are_selected_from_the_scenario_area(
        self,
    ) -> None:
        for area, expected in (
            ("sketch", "sketch_scenarios_use_the_native_window"),
            ("solid-model", "application_scenarios_use_the_native_window"),
        ):
            self.assertIn(expected, native_command(area))
            self.check_area_filter(area)
        with self.assertRaisesRegex(ValueError, "scenario area"):
            native_command("assembly")

    def check_area_filter(self, area: str) -> None:
        """Check that stale BDD settings cannot alter a run; example: self.check_area_filter('sketch')."""
        with tempfile.TemporaryDirectory() as folder:
            fixture, request, runner, state = self.fixture(folder)
            fixture.row["area"] = area
            with patch.dict(
                "os.environ", {"OBK_BDD_DIAGNOSTIC": "1", "OBK_BDD_AREA": "stale"}
            ):
                capture_receipt(request, runner, state)
            self.assertNotIn("OBK_BDD_DIAGNOSTIC", runner.environment)
            self.assertEqual(
                runner.environment.get("OBK_BDD_AREA"),
                "solid-model" if area == "solid-model" else None,
            )

    def test_failure_or_changed_inputs_never_publish_a_receipt(self) -> None:
        for failure in (
            "exit",
            "step",
            "empty",
            "source",
            "checkout",
            "application",
            "publication",
        ):
            with self.subTest(failure=failure), tempfile.TemporaryDirectory() as folder:
                _, request, runner, state = self.fixture(folder)
                self.configure_failure(failure, runner, state)
                with self.assertRaises(ValueError):
                    capture_receipt(request, runner, state)
                self.assertFalse(request.destination.exists())

    def configure_failure(
        self, failure: str, runner: FakeNativeRun, state: FakeApplicationState
    ) -> None:
        """Inject one failure at a time; example: self.configure_failure('exit', runner, state)."""
        runner.exit_code = 1 if failure == "exit" else 0
        runner.fail_step = failure == "step"
        runner.no_scenarios = failure == "empty"
        runner.dirty_source = failure == "source"
        runner.wrong_checkout = failure == "checkout"
        if failure == "application":
            state.states[1] = ("b" * 40, True)
        if failure == "publication":
            state.states[3] = ("b" * 40, True)

    def test_capture_selection_refuses_ambiguity_and_path_escape(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            _, request, runner, state = self.fixture(folder)
            runner.captures = ["First.png", "Other.png"]
            with self.assertRaisesRegex(ValueError, "capture"):
                capture_receipt(request, runner, state)
            self.assertFalse(request.destination.exists())

    def test_scenario_selection_requires_current_reviewed_source(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture, _, _, _ = self.fixture(folder)
            inventory = {"source_revision": "source", "scenarios": [fixture.row]}
            reviewed = self.reviewed(str(fixture.row["id"]))
            classifications = {"version": 1, "scenarios": [reviewed]}
            self.assertEqual(
                find_scenario(inventory, classifications, fixture.row["id"]),
                fixture.row,
            )
            fixture.row["capture_only"] = True
            with self.assertRaisesRegex(ValueError, "non-blocked"):
                find_scenario(inventory, classifications, fixture.row["id"])
            del fixture.row["capture_only"]
            fixture.row["capture_host_only"] = True
            with self.assertRaisesRegex(ValueError, "non-blocked"):
                find_scenario(inventory, classifications, fixture.row["id"])
            del fixture.row["capture_host_only"]
            reviewed["source_sha256"] = "old"
            with self.assertRaisesRegex(ValueError, "source"):
                find_scenario(inventory, classifications, fixture.row["id"])
            reviewed["source_sha256"] = "source-hash"
            reviewed["state"] = "blocked"
            reviewed["reason"] = "a missing kernel capability"
            with self.assertRaisesRegex(ValueError, "non-blocked"):
                find_scenario(inventory, classifications, fixture.row["id"])

    def test_receipt_destination_cannot_escape_the_receipt_directory(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            destination = receipt_destination(
                root, Path("architecture/rust-port/receipts/example")
            )
            self.assertEqual(
                destination, root / "architecture/rust-port/receipts/example"
            )
            for bad in (
                Path("."),
                Path("architecture/rust-port/receipts/../elsewhere"),
            ):
                with self.assertRaisesRegex(ValueError, "receipt destination"):
                    receipt_destination(root, bad)
        with self.assertRaisesRegex(ValueError, "renderer"):
            require_renderer("  ")

    def test_existing_destination_is_never_replaced(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            _, request, runner, state = self.fixture(folder)
            request.destination.mkdir(parents=True)
            (request.destination / "keep").write_text("existing")
            with self.assertRaises(ValueError):
                capture_receipt(request, runner, state)
            self.assertEqual((request.destination / "keep").read_text(), "existing")

    def test_explicit_capture_requires_a_filename_and_selects_only_that_image(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture, request, runner, state = self.fixture(folder)
            runner.captures = ["First.png", "Other.png"]
            selected = CaptureRequest(**{**vars(request), "capture_name": "First.png"})
            artifact = capture_receipt(selected, runner, state)
            check_capture_receipt(request.root, artifact, fixture.row, "source", "a" * 40)
        with tempfile.TemporaryDirectory() as folder:
            _, request, runner, state = self.fixture(folder)
            escaped = CaptureRequest(
                **{**vars(request), "capture_name": "../First.png"}
            )
            with self.assertRaisesRegex(ValueError, "filename"):
                capture_receipt(escaped, runner, state)
            self.assertFalse(request.destination.exists())

    def test_outline_requires_every_example_before_publication(self) -> None:
        for missing in (False, True):
            with self.subTest(missing=missing), tempfile.TemporaryDirectory() as folder:
                fixture, request, runner, state = self.fixture(folder)
                outline = OutlineFixture()
                fixture.row.update(outline.row)
                fixture.report = outline.report
                runner.captures = [
                    example_capture_name(fixture.row, example)
                    for example in fixture.row["examples"]
                ]
                if missing:
                    fixture.report[0]["elements"].pop()
                    with self.assertRaises(ValueError):
                        capture_receipt(request, runner, state)
                    self.assertFalse(request.destination.exists())
                else:
                    result = capture_receipt(request, runner, state)
                    check_capture_receipt(
                        request.root, result, fixture.row, "source", "a" * 40
                    )

    def test_outline_missing_capture_cannot_publish(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture, request, runner, state = self.fixture(folder)
            outline = OutlineFixture()
            fixture.row.update(outline.row)
            fixture.report = outline.report
            runner.captures = [
                example_capture_name(fixture.row, fixture.row["examples"][0])
            ]
            with self.assertRaisesRegex(ValueError, "capture"):
                capture_receipt(request, runner, state)
            self.assertFalse(request.destination.exists())

# SPDX-License-Identifier: GPL-2.0-only
"""Tests for evidence claims against native Cucumber output."""

import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from behavior_evidence import check_scenario, verify_receipt


class EvidenceFixture:
    def __init__(self, root: Path) -> None:
        self.root = root
        self.row = {
            "id": "scenario:1",
            "area": "application",
            "api_revision": "a" * 40,
            "kernel_revision": "b" * 40,
            "path": "features/application/a/a.feature",
            "scenario": "First",
            "rule": None,
            "line": 8,
            "outline": False,
            "source_sha256": "source-hash",
            "background_steps": [],
            "scenario_steps": [
                {"line": 9, "name": "first step"},
                {"line": 10, "name": "second step"},
            ],
        }
        self.report = [
            {
                "uri": "/work/features/application/a/a.feature",
                "elements": [
                    {
                        "type": "scenario",
                        "name": "First",
                        "line": 8,
                        "after": [{"result": {"status": "passed"}}],
                        "steps": [
                            {
                                "line": 9,
                                "name": "first step",
                                "result": {"status": "passed"},
                            },
                            {
                                "line": 10,
                                "name": "second step",
                                "result": {"status": "passed"},
                            },
                        ],
                    }
                ],
            }
        ]

    def store(self, name: str, value: object) -> dict[str, str]:
        path = self.root / name
        path.write_text(json.dumps(value))
        return {"path": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}

    def receipt(self) -> dict[str, str]:
        (self.root / "Cargo.lock").write_text(
            '[[package]]\nname = "oblikovati-api"\nsource = "git+https://example.org/api#'
            + "a" * 40
            + '"\n'
            + '[[package]]\nname = "martensite-tkernel"\nsource = "git+https://example.org/kernel#'
            + "b" * 40
            + '"\n'
        )
        cucumber = self.store("cucumber.json", self.report)
        state = {"commit": "source", "dirty": False, "root": "/scenario-checkout"}
        source = self.store(
            "source.json", {"before": state, "after": state, "source_changed": False}
        )
        capture_path = self.root / "capture.png"
        capture_path.write_bytes(b"\x89PNG\r\n\x1a\nfixture")
        capture = {
            "path": "capture.png",
            "sha256": hashlib.sha256(capture_path.read_bytes()).hexdigest(),
        }
        return self.store(
            "receipt.json",
            {
                "kind": "native-acceptance",
                "scenario_id": self.row["id"],
                "scenario_revision": "source",
                "source_sha256": "source-hash",
                "application_commit": "app",
                "api_revision": "a" * 40,
                "kernel_revision": "b" * 40,
                "cucumber": cucumber,
                "source_report": source,
                "capture": capture,
                "runner": "application_scenarios_use_the_native_window",
                "exit_code": 0,
                "environment": {"os": "Linux", "renderer": "Vulkan"},
            },
        )


class EvidenceTests(unittest.TestCase):
    def test_complete_scenario_receipt_is_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            verify_receipt(
                fixture.root, fixture.receipt(), fixture.row, "source", "app"
            )

    def test_partial_or_failed_steps_and_wrong_source_are_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            artifact = fixture.receipt()
            with self.assertRaisesRegex(ValueError, "scenario_revision"):
                verify_receipt(fixture.root, artifact, fixture.row, "other", "app")
            report = copy.deepcopy(fixture.report)
            report[0]["elements"][0]["steps"].pop()
            with self.assertRaisesRegex(ValueError, "step sequence"):
                check_scenario(report, fixture.row)
            report = copy.deepcopy(fixture.report)
            report[0]["elements"][0]["steps"][1]["result"]["status"] = "failed"
            with self.assertRaisesRegex(ValueError, "status"):
                check_scenario(report, fixture.row)

    def test_background_and_rule_are_checked(self) -> None:
        fixture = EvidenceFixture(Path("."))
        fixture.row["rule"] = "Policy"
        fixture.row["background_steps"] = [{"line": 4, "name": "setup"}]
        scenario = fixture.report[0]["elements"][0]
        scenario["name"] = "Policy First"
        background = {
            "type": "background",
            "name": "Policy First",
            "line": 8,
            "steps": [{"line": 4, "name": "setup", "result": {"status": "passed"}}],
        }
        fixture.report[0]["elements"].insert(0, background)
        check_scenario(fixture.report, fixture.row)
        background["steps"][0]["result"]["status"] = "skipped"
        with self.assertRaisesRegex(ValueError, "status"):
            check_scenario(fixture.report, fixture.row)

    def test_outline_and_tampered_artifacts_are_not_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            fixture.row["outline"] = True
            with self.assertRaisesRegex(ValueError, "outline evidence"):
                check_scenario(fixture.report, fixture.row)
            fixture.row["outline"] = False
            artifact = fixture.receipt()
            (fixture.root / "cucumber.json").write_text("[]")
            with self.assertRaisesRegex(ValueError, "SHA-256"):
                verify_receipt(fixture.root, artifact, fixture.row, "source", "app")


class EvidenceBoundaryTests(unittest.TestCase):
    def test_missing_cleanup_and_extra_source_steps_are_rejected(self) -> None:
        fixture = EvidenceFixture(Path("."))
        fixture.report[0]["elements"][0].pop("after")
        with self.assertRaisesRegex(ValueError, "cleanup"):
            check_scenario(fixture.report, fixture.row)
        fixture = EvidenceFixture(Path("."))
        fixture.row["scenario_steps"].append({"line": 11, "name": "third step"})
        with self.assertRaisesRegex(ValueError, "step sequence"):
            check_scenario(fixture.report, fixture.row)

    def test_runner_source_state_and_capture_are_required(self) -> None:
        from behavior_evidence import check_capture, check_runner, check_source_revision

        fixture = EvidenceFixture(Path("."))
        with self.assertRaisesRegex(ValueError, "runner"):
            check_runner({"runner": "diagnostic"}, fixture.row)
        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            state = {"commit": "source", "dirty": True}
            source = fixture.store(
                "source.json",
                {"before": state, "after": state, "source_changed": False},
            )
            with self.assertRaisesRegex(ValueError, "dirty"):
                check_source_revision(fixture.root, source, "source")
            capture = fixture.store("bad.png", {"not": "a PNG"})
            with self.assertRaisesRegex(ValueError, "expected a PNG"):
                check_capture(fixture.root, capture)


class LockedRevisionTests(unittest.TestCase):
    def test_wrong_or_ambiguous_dependency_revision_is_rejected(self) -> None:
        from behavior_evidence import StaleEvidence, locked_revision

        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            artifact = fixture.receipt()
            lock = fixture.root / "Cargo.lock"
            lock.write_text(lock.read_text().replace("a" * 40, "c" * 40))
            with self.assertRaisesRegex(StaleEvidence, "api_revision"):
                verify_receipt(fixture.root, artifact, fixture.row, "source", "app")
            lock.write_text(
                lock.read_text()
                + '\n[[package]]\nname = "oblikovati-api"\nsource = "git+https://example.org/api#'
                + "a" * 40
                + '"\n'
            )
            with self.assertRaisesRegex(ValueError, "expected one record"):
                locked_revision(fixture.root, "oblikovati-api")


class HistoricalReceiptTests(unittest.TestCase):
    def test_changed_inputs_are_stale_but_corruption_is_an_error(self) -> None:
        from behavior_evidence import StaleEvidence

        with tempfile.TemporaryDirectory() as folder:
            fixture = EvidenceFixture(Path(folder))
            artifact = fixture.receipt()
            with self.assertRaisesRegex(StaleEvidence, "application inputs"):
                verify_receipt(
                    fixture.root,
                    artifact,
                    fixture.row,
                    "source",
                    "app",
                    lambda root, old, current: False,
                )
            (fixture.root / "cucumber.json").write_text("corrupt")
            with self.assertRaisesRegex(ValueError, "SHA-256"):
                verify_receipt(
                    fixture.root,
                    artifact,
                    fixture.row,
                    "source",
                    "app",
                    lambda root, old, current: False,
                )

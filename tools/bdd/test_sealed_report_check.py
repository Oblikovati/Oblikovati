# SPDX-License-Identifier: GPL-2.0-only
"""Adversarial checks for the experimental sealed report gate."""

import copy
import hashlib
import json
import tempfile
import unittest
from dataclasses import replace
from pathlib import Path

from sealed_report_check import check_sealed_report, require_scenario_source, selected_row
from test_behavior_evidence import EvidenceFixture
from test_sealed_execution import SealedFixture


class ReportFixture:
    """Write a selected source row and matching diagnostic artifacts."""

    def __init__(self, root: Path) -> None:
        self.sealed = SealedFixture(root)
        self.evidence = EvidenceFixture(root)
        self.request = replace(self.sealed.request(), area="application", feature="a", scenario="First")
        self.row = self.evidence.row | {"feature": "a"}
        self.revision = "a" * 40
        source = self.sealed.checkout / self.row["path"]
        source.parent.mkdir(parents=True)
        source.write_bytes(b"scenario source")
        self.row["source_sha256"] = hashlib.sha256(source.read_bytes()).hexdigest()
        self.write_report(self.evidence.report)
        state = {"commit": self.revision, "dirty": False, "root": "/scenario"}
        self.write_json("source-revision.json", {"before": state, "after": state, "source_changed": False})
        (self.request.reports / "junit.xml").write_text('<testsuites><testsuite><testcase name="First"/></testsuite></testsuites>')
        (self.request.reports / "capture.png").write_bytes(b"\x89PNG\r\n\x1a\nfixture")

    def write_json(self, name: str, value: object) -> None:
        """Write one report input; example: fixture.write_json('file.json', {})."""
        (self.request.reports / name).write_text(json.dumps(value))

    def write_report(self, report: object) -> None:
        """Replace the Cucumber report; example: fixture.write_report([])."""
        self.write_json("cucumber.json", report)

    def check(self) -> dict[str, object]:
        """Validate the fixture; example: fixture.check()."""
        return check_sealed_report(self.request, self.row, self.revision, None)


class SealedReportTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.fixture = ReportFixture(Path(self.temporary.name))

    def test_matching_report_is_checked_but_not_reusable(self) -> None:
        require_scenario_source(self.fixture.sealed.checkout, self.fixture.row)
        self.assertEqual(self.fixture.check(), {"scenario_id": "scenario:1", "content_checked": True, "reusable": False})

    def test_wrong_scenario_and_extra_scenario_are_rejected(self) -> None:
        for change in ("wrong", "extra"):
            report = copy.deepcopy(self.fixture.evidence.report)
            elements = report[0]["elements"]
            if change == "wrong":
                elements[0]["name"] = "another scenario"
            else:
                elements.append(copy.deepcopy(elements[0]))
            self.fixture.write_report(report)
            with self.subTest(change=change), self.assertRaises(ValueError):
                self.fixture.check()

    def test_failed_step_and_missing_cleanup_are_rejected(self) -> None:
        for change in ("step", "cleanup"):
            report = copy.deepcopy(self.fixture.evidence.report)
            scenario = report[0]["elements"][0]
            if change == "step":
                scenario["steps"][0]["result"]["status"] = "failed"
            else:
                scenario["after"] = []
            self.fixture.write_report(report)
            with self.subTest(change=change), self.assertRaises(ValueError):
                self.fixture.check()

    def test_changed_source_and_dirty_revision_are_rejected(self) -> None:
        (self.fixture.sealed.checkout / self.fixture.row["path"]).write_bytes(b"changed")
        with self.assertRaisesRegex(ValueError, "inventory hash"):
            require_scenario_source(self.fixture.sealed.checkout, self.fixture.row)
        state = {"commit": self.fixture.revision, "dirty": True, "root": "/scenario"}
        self.fixture.write_json("source-revision.json", {"before": state, "after": state, "source_changed": False})
        with self.assertRaisesRegex(ValueError, "dirty"):
            self.fixture.check()

    def test_bad_junit_and_capture_are_rejected(self) -> None:
        junit = self.fixture.request.reports / "junit.xml"
        junit.write_text('<testsuites><testcase><failure/></testcase></testsuites>')
        with self.assertRaisesRegex(ValueError, "JUnit"):
            self.fixture.check()
        junit.write_text('<testsuites><testsuite><testcase/></testsuite></testsuites>')
        (self.fixture.request.reports / "capture.png").write_bytes(b"not png")
        with self.assertRaisesRegex(ValueError, "PNG"):
            self.fixture.check()

    def test_wrong_sealed_source_root_and_invalid_xml_are_rejected(self) -> None:
        state = {"commit": self.fixture.revision, "dirty": False, "root": "/elsewhere"}
        self.fixture.write_json("source-revision.json", {"before": state, "after": state, "source_changed": False})
        with self.assertRaisesRegex(ValueError, "expected /scenario"):
            self.fixture.check()
        state["root"] = "/scenario"
        self.fixture.write_json("source-revision.json", {"before": state, "after": state, "source_changed": False})
        (self.fixture.request.reports / "junit.xml").write_text("<broken")
        with self.assertRaisesRegex(ValueError, "valid XML"):
            self.fixture.check()

    def test_selector_is_exact_and_requires_matching_inventory_revision(self) -> None:
        inventory = {"source_revision": self.fixture.revision, "scenarios": [self.fixture.row]}
        self.assertEqual(selected_row(inventory, self.fixture.request, self.fixture.revision), self.fixture.row)
        with self.assertRaisesRegex(ValueError, "exact"):
            selected_row(inventory, replace(self.fixture.request, scenario="Fir"), self.fixture.revision)
        with self.assertRaisesRegex(ValueError, "revision"):
            selected_row(inventory, self.fixture.request, "b" * 40)


class SealedOutlineTests(unittest.TestCase):
    def test_all_outline_cases_and_captures_are_required(self) -> None:
        from behavior_evidence import example_capture_name
        from test_behavior_outline import OutlineFixture

        with tempfile.TemporaryDirectory() as folder:
            fixture = ReportFixture(Path(folder))
            outline = OutlineFixture()
            fixture.write_report(outline.report)
            (fixture.request.reports / "capture.png").unlink()
            (fixture.request.reports / "junit.xml").write_text('<testsuites><testsuite><testcase/><testcase/></testsuite></testsuites>')
            for example in outline.row["examples"]:
                (fixture.request.reports / example_capture_name(outline.row, example)).write_bytes(b"\x89PNG\r\n\x1a\nfixture")
            result = check_sealed_report(fixture.request, outline.row, fixture.revision, None)
            self.assertFalse(result["reusable"])
            (fixture.request.reports / example_capture_name(outline.row, outline.row["examples"][0])).unlink()
            with self.assertRaisesRegex(ValueError, "expected exactly one"):
                check_sealed_report(fixture.request, outline.row, fixture.revision, None)

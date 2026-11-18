# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for source-matched feature results."""

import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import feature_progress as progress
import inventory


class FeatureProgressTests(unittest.TestCase):
    def setUp(self) -> None:
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.path = "features/application/closing/closing.feature"
        self.feature = self.root / self.path
        self.feature.parent.mkdir(parents=True)
        self.feature.write_text(
            "Feature: Close\n  Scenario: Cancel\n    Given cancel\n  Scenario: Exit\n    Given exit\n"
        )
        self.report = self.root / "run.json"
        self.write_report(["Cancel", "Exit"])
        self.record = {
            "feature_sha256": inventory.feature_record(self.feature)["sha256"],
            "cucumber": "run.json",
            "cucumber_sha256": progress.digest(self.report),
        }

    def write_report(self, names: list[str], status: str = "passed") -> None:
        result = {"result": {"status": status}}
        lines = {"Cancel": (2, 3, "cancel"), "Exit": (4, 5, "exit")}
        scenarios = [
            {
                "type": "scenario",
                "name": name,
                "line": lines[name][0],
                "steps": [{**result, "line": lines[name][1], "name": lines[name][2]}],
                "after": [result],
            }
            for name in names
        ]
        self.report.write_text(json.dumps([{"uri": self.path, "elements": scenarios}]))

    def status(self) -> tuple[str, int]:
        with patch.object(progress, "ROOT", self.root):
            return progress.result_status(
                self.root,
                self.path,
                inventory.feature_record(self.feature),
                self.record,
            )

    def test_complete_report_is_green_and_hash_changes_are_stale(self) -> None:
        self.assertEqual(self.status(), ("green", 2))
        self.feature.write_text(self.feature.read_text() + "# correction\n")
        self.assertEqual(self.status(), ("source_changed", 0))

    def test_missing_skipped_and_altered_results_are_not_green(self) -> None:
        self.write_report(["Cancel"])
        self.record["cucumber_sha256"] = progress.digest(self.report)
        self.assertEqual(self.status(), ("incomplete", 0))
        self.write_report(["Cancel", "Exit"], "skipped")
        self.record["cucumber_sha256"] = progress.digest(self.report)
        self.assertEqual(self.status(), ("incomplete", 0))
        self.report.write_text("[]")
        self.assertEqual(self.status(), ("invalid_result", 0))

    def test_outline_is_not_mistaken_for_complete(self) -> None:
        self.feature.write_text("Feature: X\n  Scenario Outline: case\n")
        self.record["feature_sha256"] = inventory.feature_record(self.feature)["sha256"]
        with patch.object(progress, "source_inventory", return_value=[]):
            self.assertEqual(self.status(), ("needs_outline_review", 0))

    def test_blocker_is_visible_until_source_changes(self) -> None:
        self.record["blocker"] = "Scenario is capture-only"
        self.assertEqual(self.status(), ("blocked", 0))
        self.feature.write_text(self.feature.read_text() + "# updated\n")
        self.assertEqual(self.status(), ("source_changed", 0))

    def test_outline_requires_each_expanded_example(self) -> None:
        row = {
            "id": "outline:1",
            "path": self.path,
            "line": 2,
            "scenario": "Move <axis>",
            "rule": None,
            "outline": True,
            "background_steps": [],
            "scenario_steps": [{"line": 3, "name": "move <axis>"}],
            "examples": [
                {"line": 6, "columns": ["axis"], "values": ["X"]},
                {"line": 7, "columns": ["axis"], "values": ["Y"]},
            ],
        }

        def case(axis: str, line: int) -> dict[str, object]:
            result = {"status": "passed"}
            return {
                "type": "scenario",
                "name": f"Move {axis}",
                "line": line,
                "steps": [{"line": 3, "name": f"move {axis}", "result": result}],
                "after": [{"result": result}],
            }

        report = [{"uri": self.path, "elements": [case("X", 6), case("Y", 7)]}]
        self.assertEqual(progress.count_checked_scenarios(report, [row]), ("green", 1))
        report[0]["elements"].pop()
        self.assertEqual(
            progress.count_checked_scenarios(report, [row]), ("incomplete", 0)
        )

    def test_unrecorded_feature_is_not_run(self) -> None:
        self.assertEqual(
            progress.result_status(
                self.root, self.path, inventory.feature_record(self.feature), None
            ),
            ("not_run", 0),
        )

    def test_changed_or_missing_steps_do_not_count_as_passed(self) -> None:
        report = json.loads(self.report.read_text())
        report[0]["elements"][0]["steps"][0]["name"] = "different"
        self.report.write_text(json.dumps(report))
        self.record["cucumber_sha256"] = progress.digest(self.report)
        self.assertEqual(self.status(), ("incomplete", 1))

    def test_inventory_counts_only_current_green_results(self) -> None:
        with patch.object(progress, "ROOT", self.root):
            summary = progress.report_progress(
                self.root, {self.path: self.record, "removed.feature": {}}
            )
        self.assertEqual(
            (summary["green_features"], summary["passed_scenarios"]), (1, 2)
        )
        self.assertEqual(summary["removed_features"], ["removed.feature"])


if __name__ == "__main__":
    unittest.main()

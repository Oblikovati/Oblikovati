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

    def test_unproven_passing_scenarios_do_not_count_as_green(self) -> None:
        self.record["blocker"] = "Restart was simulated in process"
        self.record["unproven_scenarios"] = {"Exit": "Requires a process restart"}
        self.assertEqual(self.status(), ("blocked", 1))
        self.record["unproven_scenarios"] = {"Unknown": "Missing scenario"}
        self.assertEqual(self.status(), ("needs_source_review", 0))
        self.record["unproven_scenarios"] = {"Exit": ""}
        self.assertEqual(self.status(), ("needs_source_review", 0))
        self.record["unproven_scenarios"] = ["Exit"]
        self.assertEqual(self.status(), ("needs_source_review", 0))

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

    def test_recorded_inventory_requires_its_artifact_hash(self) -> None:
        parsed = self.root / "parsed.json"
        parsed.write_text(json.dumps({"scenarios": [{"path": self.path}]}))
        record = {
            "scenario_inventory": "parsed.json",
            "scenario_inventory_sha256": progress.digest(parsed),
        }
        with patch.object(progress, "ROOT", self.root):
            self.assertEqual(progress.recorded_inventory(record), [{"path": self.path}])
            parsed.write_text("{}")
            self.assertIsNone(progress.recorded_inventory(record))
            record["scenario_inventory"] = "../outside.json"
            self.assertIsNone(progress.recorded_inventory(record))

    def test_blocker_is_visible_until_source_changes(self) -> None:
        self.record["blocker"] = "Scenario is capture-only"
        self.assertEqual(self.status(), ("blocked", 2))
        self.feature.write_text(self.feature.read_text() + "# updated\n")
        self.assertEqual(self.status(), ("source_changed", 0))

    def test_blocked_feature_retains_only_verified_scenario_count(self) -> None:
        report = json.loads(self.report.read_text())
        report[0]["elements"][1]["steps"][0]["result"]["status"] = "failed"
        self.report.write_text(json.dumps(report))
        self.record["cucumber_sha256"] = progress.digest(self.report)
        self.record["blocker"] = "Command details are not confirmed"
        self.assertEqual(self.status(), ("blocked", 1))
        self.report.write_text("[]")
        self.assertEqual(self.status(), ("blocked", 0))

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

    def test_supporting_document_change_invalidates_the_run(self) -> None:
        note = self.feature.parent / "FUNCTIONAL.md"
        note.write_text("Saved part fixture")
        self.record["supporting_files"] = {
            note.relative_to(self.root).as_posix(): progress.digest(note)
        }
        self.assertEqual(self.status(), ("green", 2))
        note.write_text("Changed fixture")
        self.assertEqual(self.status(), ("support_changed", 0))

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

    def argument_fixture(self) -> tuple[dict[str, object], dict[str, object], list[dict[str, object]]]:
        self.feature.write_text(
            "Feature: example\n Background: shared\n  Given setup\n"
            "   | x | y |\n Scenario: case\n  When choose\n   | a | b |\n"
        )
        self.record["feature_sha256"] = inventory.feature_record(self.feature)["sha256"]
        row = {"id": "case", "path": self.path, "line": 5, "scenario": "case",
               "rule": None, "outline": False, "arguments_recorded": True,
               "source_sha256": self.record["feature_sha256"], "source_error": None,
               "background_steps": [{"line": 3, "name": "setup", "table": [["x", "y"]]}],
               "scenario_steps": [{"line": 6, "name": "choose", "table": [["a", "b"]]}]}
        entry = {"path": self.path, "line": 5, "name": "case",
                 "background_steps": row["background_steps"], "scenario_steps": row["scenario_steps"]}
        return row, entry, self.argument_elements()

    def argument_elements(self) -> list[dict[str, object]]:
        success = {"status": "passed"}
        return [{"type": "background", "name": "case", "line": 5,
                 "steps": [{"line": 3, "name": "setup", "result": success}]},
                {"type": "scenario", "name": "case", "line": 5,
                 "steps": [{"line": 6, "name": "choose", "result": success}],
                 "after": [{"result": success}]}]

    def write_argument_report(self, elements: list[dict[str, object]]) -> None:
        self.report.write_text(json.dumps([{"uri": self.path, "elements": elements}]))
        self.record["cucumber_sha256"] = progress.digest(self.report)

    def write_argument_capture(self, entries: list[dict[str, object]]) -> None:
        capture = self.root / "arguments.json"
        capture.write_text(json.dumps({"cucumber_sha256": self.record["cucumber_sha256"],
                                       "scenarios": entries}))
        self.record["scenario_arguments"] = "arguments.json"
        self.record["scenario_arguments_sha256"] = progress.digest(capture)

    def argument_status(self, row: dict[str, object]) -> tuple[str, int]:
        with patch.object(progress, "source_inventory", return_value=[row]):
            return self.status()

    def test_valid_table_and_background_arguments_are_green(self) -> None:
        row, entry, elements = self.argument_fixture()
        self.write_argument_report(elements)
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("green", 1))
        row["source_sha256"] = "incorrect"
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))
        row["source_sha256"] = self.record["feature_sha256"]
        row["arguments_recorded"] = False
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))

    def test_changed_table_and_background_are_not_counted(self) -> None:
        row, entry, elements = self.argument_fixture()
        self.write_argument_report(elements)
        entry["scenario_steps"] = [{"line": 6, "name": "choose", "table": [["a", "changed"]]}]
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("incomplete", 0))
        entry["scenario_steps"] = row["scenario_steps"]
        entry["background_steps"] = [{"line": 3, "name": "setup", "table": [["x"]]}]
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("incomplete", 0))

    def test_duplicate_or_unbound_argument_captures_fail_closed(self) -> None:
        row, entry, elements = self.argument_fixture()
        self.write_argument_report(elements)
        self.write_argument_capture([entry, entry])
        self.assertEqual(self.argument_status(row), ("incomplete", 0))
        self.write_argument_capture([entry])
        self.write_argument_report(elements + [{"type": "scenario", "name": "extra", "line": 9}])
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("incomplete", 0))
        self.record["scenario_arguments"] = "../outside.json"
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))

    def test_missing_cleanup_invalid_report_and_missing_capture_fail(self) -> None:
        row, entry, elements = self.argument_fixture()
        elements[1]["after"] = []
        self.write_argument_report(elements)
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("incomplete", 0))
        self.report.write_text("not JSON")
        self.record["cucumber_sha256"] = progress.digest(self.report)
        self.write_argument_capture([entry])
        self.assertEqual(self.argument_status(row), ("invalid_result", 0))
        self.record["scenario_arguments_sha256"] = "wrong"
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))
        (self.root / "arguments.json").unlink()
        self.assertEqual(self.argument_status(row), ("needs_source_review", 0))


if __name__ == "__main__":
    unittest.main()

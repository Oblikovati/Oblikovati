# SPDX-License-Identifier: GPL-2.0-only
"""Outline receipts must account for all pinned examples."""

import copy
import unittest

from behavior_evidence import check_scenario


class OutlineFixture:
    def __init__(self) -> None:
        self.row = {
            "id": "scenario:outline",
            "path": "features/application/a/a.feature",
            "rule": None,
            "scenario": "Value <x>",
            "line": 3,
            "outline": True,
            "background_steps": [],
            "scenario_steps": [{"line": 4, "name": "select <x>"}],
            "examples": [
                {"line": 7, "columns": ["x"], "values": ["one"]},
                {"line": 8, "columns": ["x"], "values": ["two"]},
            ],
        }
        self.report = [
            {
                "uri": "/checkout/features/application/a/a.feature",
                "elements": [
                    {
                        "type": "scenario",
                        "name": "Value " + value,
                        "line": line,
                        "steps": [
                            {
                                "line": 4,
                                "name": "select " + value,
                                "result": {"status": "passed"},
                            }
                        ],
                        "after": [{"result": {"status": "passed"}}],
                    }
                    for line, value in ((7, "one"), (8, "two"))
                ],
            }
        ]


class OutlineEvidenceTests(unittest.TestCase):
    def test_each_expanded_example_must_pass_with_cleanup(self) -> None:
        fixture = OutlineFixture()
        check_scenario(fixture.report, fixture.row)
        for mutation in ("missing", "duplicate", "failed", "no_cleanup", "wrong_value"):
            report = copy.deepcopy(fixture.report)
            cases = report[0]["elements"]
            if mutation == "missing":
                cases.pop()
            if mutation == "duplicate":
                cases.append(copy.deepcopy(cases[0]))
            if mutation == "failed":
                cases[1]["steps"][0]["result"]["status"] = "skipped"
            if mutation == "no_cleanup":
                cases[1].pop("after")
            if mutation == "wrong_value":
                cases[1]["steps"][0]["name"] = "select one"
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                check_scenario(report, fixture.row)

    def test_incomplete_or_ambiguous_examples_are_rejected(self) -> None:
        fixture = OutlineFixture()
        for examples in (
            [],
            [{"line": 7, "columns": ["x", "x"], "values": ["one", "two"]}],
            [{"line": 7, "columns": ["z"], "values": ["one"]}],
            [{"line": 7, "columns": ["x"], "values": []}],
        ):
            row = dict(fixture.row, examples=examples)
            with self.subTest(examples=examples), self.assertRaises(ValueError):
                check_scenario(fixture.report, row)

    def test_colliding_case_lines_and_unchecked_arguments_are_rejected(self) -> None:
        fixture = OutlineFixture()
        fixture.row["examples"][1]["line"] = 7
        with self.assertRaisesRegex(ValueError, "unique case identities"):
            check_scenario(fixture.report, fixture.row)
        fixture.row["examples"][1]["line"] = 8
        fixture.row["outline_step_arguments"] = True
        with self.assertRaisesRegex(ValueError, "unchecked step arguments"):
            check_scenario(fixture.report, fixture.row)

    def test_each_case_requires_its_own_passing_background(self) -> None:
        fixture = OutlineFixture()
        fixture.row["background_steps"] = [{"line": 2, "name": "setup"}]
        elements = fixture.report[0]["elements"]
        for case in list(elements):
            elements.append(
                {
                    "type": "background",
                    "line": case["line"],
                    "name": case["name"],
                    "steps": [
                        {"line": 2, "name": "setup", "result": {"status": "passed"}}
                    ],
                }
            )
        check_scenario(fixture.report, fixture.row)
        elements[-1]["steps"][0]["result"]["status"] = "failed"
        with self.assertRaisesRegex(ValueError, "status"):
            check_scenario(fixture.report, fixture.row)


if __name__ == "__main__":
    unittest.main()

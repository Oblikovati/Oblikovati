# SPDX-License-Identifier: GPL-2.0-only
"""Check native step-argument evidence."""

import unittest

from feature_arguments import arguments_match


class FeatureArgumentTests(unittest.TestCase):
    def test_docstrings_and_all_rows_must_match(self) -> None:
        row = {
            "path": "features/application/x/x.feature", "line": 4, "scenario": "one",
            "background_steps": [], "scenario_steps": [
                {"line": 5, "name": "send", "docstring": "first\nsecond"},
                {"line": 9, "name": "choose", "table": [["a", "b"], ["1", "2"]]},
            ],
        }
        entry = {"path": row["path"], "line": 4, "name": "one",
                 "background_steps": [], "scenario_steps": row["scenario_steps"]}
        self.assertTrue(arguments_match(row, [entry]))
        self.assertFalse(arguments_match(row, [entry, entry]))
        entry["scenario_steps"] = [
            {"line": 5, "name": "send", "docstring": "first"},
            {"line": 9, "name": "choose", "table": [["a", "b"], ["1", "2"]]},
        ]
        self.assertFalse(arguments_match(row, [entry]))
        entry["scenario_steps"] = [{"line": 5, "name": "send", "docstring": "first\nsecond"}]
        self.assertFalse(arguments_match(row, [entry]))


if __name__ == "__main__":
    unittest.main()

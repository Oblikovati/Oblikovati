# SPDX-License-Identifier: GPL-2.0-only
"""Tests for the milestone-specific scenario work queue."""

import unittest

from behavior_queue import scenario_queue
from behavior_ledger import BASELINE_FIELDS


def scenario(path: str, milestone: str | None, state: str, **extra: object) -> dict[str, object]:
    """Build one minimal ledger row; example: scenario('a.feature', 'R4', 'inventoried')."""
    row: dict[str, object] = {
        "path": path,
        "area": "sketch",
        "milestone": milestone,
        "classified": milestone is not None,
        "state": state,
        "stale": False,
        "issues": [],
    }
    row.update(extra)
    return row


class ScenarioQueueTests(unittest.TestCase):
    def test_separates_milestones_and_preserves_blocked_and_stale_counts(self) -> None:
        rows = [
            scenario("mixed.feature", "R5", "inventoried", issues=["issue-2"]),
            scenario("mixed.feature", "R4", "inventoried", stale=True, issues=["issue-1"]),
            scenario("mixed.feature", "R5", "blocked", issues=["issue-1"]),
            scenario("mixed.feature", "R5", "implemented"),
            scenario("unknown.feature", None, "inventoried"),
        ]
        result = scenario_queue(rows)
        self.assertEqual(result["unclassified"], 1)
        self.assertEqual([r["milestone"] for r in result["rows"]], ["R4", "R5"])
        self.assertEqual(result["rows"][0]["stale"], 1)
        self.assertEqual(result["rows"][1]["blocked"], 1)
        self.assertEqual(result["rows"][1]["issues"], ["issue-1", "issue-2"])
        self.assertEqual(result["rows"][1]["pending"], 2)
        self.assertEqual(result["rows"][1]["implemented"], 1)

    def test_complete_group_is_omitted_but_missing_baseline_stays(self) -> None:
        baseline = {field: "evidence" for field in BASELINE_FIELDS}
        complete = scenario("done.feature", "R3", "verified", **baseline)
        partial = scenario("baseline.feature", "R4", "verified")
        result = scenario_queue([complete, partial])
        self.assertEqual([r["path"] for r in result["rows"]], ["baseline.feature"])
        self.assertEqual(result["rows"][0]["pending"], 0)
        self.assertEqual(result["rows"][0]["missing_baseline"], 1)

    def test_stale_evidence_remains_visible_with_a_complete_baseline(self) -> None:
        baseline = {field: "evidence" for field in BASELINE_FIELDS}
        row = scenario("stale.feature", "R4", "verified", stale=True, **baseline)
        self.assertEqual(scenario_queue([row])["rows"][0]["stale"], 1)

    def test_unclassified_row_is_counted_without_inferred_milestone(self) -> None:
        result = scenario_queue([scenario("x.feature", None, "blocked")])
        self.assertEqual(result, {"unclassified": 1, "rows": []})

    def test_inconsistent_area_for_same_group_is_rejected(self) -> None:
        rows = [scenario("x.feature", "R3", "inventoried"),
                scenario("x.feature", "R3", "inventoried", area="application")]
        with self.assertRaisesRegex(ValueError, "expected one area"):
            scenario_queue(rows)

    def test_unknown_state_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "expected known states"):
            scenario_queue([scenario("x.feature", "R3", "done")])

    def test_bad_issue_entry_is_rejected(self) -> None:
        row = scenario("x.feature", "R3", "inventoried", issues=[42])
        with self.assertRaisesRegex(ValueError, "issues"):
            scenario_queue([row])


if __name__ == "__main__":
    unittest.main()

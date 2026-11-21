# SPDX-License-Identifier: GPL-2.0-only
"""Tests for a feature backlog derived from the scenario ledger."""

import unittest

from behavior_backlog import feature_backlog
from behavior_ledger import BASELINE_FIELDS


FEATURE_SHA256 = "a" * 64


class FeatureBacklogTests(unittest.TestCase):
    def test_mixed_feature_preserves_unassigned_and_blocked_scenarios(self) -> None:
        rows = [
            {
                "path": "features/sketch/line.feature",
                "source_sha256": FEATURE_SHA256,
                "area": "sketch",
                "milestone": "R5",
                "classified": True,
                "state": "verified",
                "stale": False,
                "source_error": None,
            },
            {
                "path": "features/sketch/line.feature",
                "source_sha256": FEATURE_SHA256,
                "area": "sketch",
                "milestone": None,
                "classified": False,
                "state": "blocked",
                "stale": False,
                "source_error": "invalid Gherkin",
            },
            {
                "path": "features/sketch/line.feature",
                "source_sha256": FEATURE_SHA256,
                "area": "sketch",
                "milestone": "R4",
                "classified": True,
                "state": "inventoried",
                "stale": True,
                "source_error": None,
            },
        ]
        result = feature_backlog(rows)
        self.assertEqual(
            result,
            [
                {
                    "path": "features/sketch/line.feature",
                    "source_sha256": FEATURE_SHA256,
                    "area": "sketch",
                    "total": 3,
                    "verified": 1,
                    "blocked": 1,
                    "unclassified": 1,
                    "stale": 1,
                    "missing_baseline": 3,
                    "missing_baseline_fields": {field: 3 for field in BASELINE_FIELDS},
                    "milestones": {"R4": 1, "R5": 1, "unassigned": 1},
                }
            ],
        )

    def test_orders_features_by_earliest_assigned_milestone_then_path(self) -> None:
        def row(path: str, milestone: str | None) -> dict[str, object]:
            return {
                "path": path,
                "source_sha256": FEATURE_SHA256,
                "area": "application",
                "milestone": milestone,
                "classified": milestone is not None,
                "state": "inventoried",
                "stale": False,
                "source_error": None,
            }

        rows = [
            row("z", "R10"),
            row("b", None),
            row("c", "R4"),
            row("a", "R1V"),
            row("d", "R1"),
            row("huge", "R999999999999"),
        ]
        self.assertEqual(
            [entry["path"] for entry in feature_backlog(rows)],
            ["d", "a", "c", "z", "huge", "b"],
        )

    def test_conflicting_areas_for_same_path_are_rejected(self) -> None:
        rows = [{"path": "same", "area": area} for area in ("sketch", "application")]
        with self.assertRaisesRegex(ValueError, "same.*area"):
            feature_backlog(rows)

    def test_combining_two_hashes_for_one_feature_is_rejected(self) -> None:
        first = {"path": "part.feature", "area": "sketch", "source_sha256": FEATURE_SHA256}
        second = dict(first, source_sha256="b" * 64)
        with self.assertRaisesRegex(ValueError, "part.feature.*source_sha256.*expected"):
            feature_backlog([first, second])

    def test_malformed_feature_hash_is_rejected(self) -> None:
        row = {"path": "part.feature", "area": "sketch", "source_sha256": "invalid"}
        with self.assertRaisesRegex(ValueError, "invalid.*expected lowercase SHA-256"):
            feature_backlog([row])

    def test_complete_baseline_is_not_counted_as_missing(self) -> None:
        row = {field: "evidence" for field in BASELINE_FIELDS}
        row.update(source_sha256="a" * 64, baseline_source_sha256="a" * 64)
        row.update(path="part.feature", area="sketch", milestone="R3", classified=True)
        self.assertEqual(feature_backlog([row])[0]["missing_baseline"], 0)

    def test_field_counts_include_only_scenarios_missing_each_field(self) -> None:
        full = {field: "evidence" for field in BASELINE_FIELDS}
        full.update(source_sha256="a" * 64, baseline_source_sha256="a" * 64)
        first = dict(full, path="part.feature", area="sketch", go_reference="")
        second = dict(full, path="part.feature", area="sketch", oracle=None)
        third = dict(full, path="part.feature", area="sketch", go_reference=None)
        result = feature_backlog([first, second, third])[0]
        self.assertEqual(result["missing_baseline"], 3)
        self.assertEqual(
            result["missing_baseline_fields"], {"go_reference": 2, "oracle": 1}
        )

    def test_invalid_milestone_is_rejected(self) -> None:
        row = {"path": "part.feature", "area": "sketch", "milestone": "later", "source_sha256": FEATURE_SHA256}
        with self.assertRaisesRegex(ValueError, "later.*expected R"):
            feature_backlog([row])


if __name__ == "__main__":
    unittest.main()

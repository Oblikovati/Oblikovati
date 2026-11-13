# SPDX-License-Identifier: GPL-2.0-only
"""Describe outstanding scenario work without inferring feature completion."""

import re
from collections import Counter
from typing import Mapping

from behavior_ledger import missing_baseline_fields, require_object, text_field


def milestone_order(value: str) -> tuple[bool, int, str]:
    """Order numbered milestones before unassigned work; for example, milestone_order('R1V')."""
    if value == "unassigned":
        return (True, 0, "")
    match = re.fullmatch(r"R([0-9]+)([A-Z]?)", value)
    if match is None:
        raise ValueError(
            f"milestone {value!r}: expected R followed by a number and optional suffix"
        )
    return (False, int(match.group(1)), match.group(2))


def count_feature(rows: list[Mapping[str, object]]) -> dict[str, object]:
    """Summarize one source file; for example, count_feature([scenario])."""
    first = rows[0]
    path = text_field(first, "path", "scenario")
    area = text_field(first, "area", f"scenario {path!r}")
    if any(row.get("area") != area for row in rows):
        raise ValueError(
            f"feature {path!r} area: expected the same area for every scenario"
        )
    milestones = Counter(str(row.get("milestone") or "unassigned") for row in rows)
    return {
        "path": path,
        "area": area,
        "total": len(rows),
        "verified": sum(row.get("state") == "verified" for row in rows),
        "blocked": sum(row.get("state") == "blocked" for row in rows),
        "unclassified": sum(not row.get("classified") for row in rows),
        "stale": sum(bool(row.get("stale")) for row in rows),
        "missing_baseline": sum(bool(missing_baseline_fields(row)) for row in rows),
        "milestones": dict(
            sorted(milestones.items(), key=lambda item: milestone_order(item[0]))
        ),
    }


def feature_backlog(rows: list[object]) -> list[dict[str, object]]:
    """Group ledger rows by feature in milestone order; for example, feature_backlog(rows)."""
    features: dict[str, list[Mapping[str, object]]] = {}
    for value in rows:
        row = require_object(value, "ledger scenario")
        path = text_field(row, "path", "scenario")
        features.setdefault(path, []).append(row)
    result = [count_feature(group) for group in features.values()]
    return sorted(result, key=feature_order)


def feature_order(feature: Mapping[str, object]) -> tuple[tuple[bool, int, str], str]:
    """Sort by earliest assigned milestone and path; for example, feature_order(feature)."""
    milestones = require_object(feature.get("milestones"), "feature milestones")
    return (min(milestone_order(name) for name in milestones), str(feature["path"]))

# SPDX-License-Identifier: GPL-2.0-only
"""Describe scenario work per milestone without claiming that dependencies are ready."""

from collections import Counter
from typing import Mapping

from behavior_backlog import milestone_order
from behavior_ledger import (
    STATES,
    missing_baseline_fields,
    require_list,
    require_object,
    text_field,
)


def queue_issues(rows: list[Mapping[str, object]]) -> list[str]:
    """Collect issue links; example: queue_issues([{'issues': ['issue']}])."""
    result: set[str] = set()
    for row in rows:
        for issue in require_list(row.get("issues"), "scenario issues"):
            if not isinstance(issue, str) or not issue:
                raise ValueError(f"scenario issues {issue!r}: expected a nonempty string")
            result.add(issue)
    return sorted(result)


def queue_states(rows: list[Mapping[str, object]]) -> Counter[str]:
    """Validate a group's area and states; example: queue_states(rows)."""
    area = text_field(rows[0], "area", "scenario")
    if any(row.get("area") != area for row in rows):
        raise ValueError(f"feature {rows[0]['path']!r}: expected one area")
    states = Counter(text_field(row, "state", "scenario") for row in rows)
    unknown = set(states) - STATES
    if unknown:
        raise ValueError(f"scenario states {sorted(unknown)!r}: expected known states")
    return states


def queue_group(rows: list[Mapping[str, object]]) -> dict[str, object]:
    """Summarize one milestone and feature; example: queue_group([row])."""
    first = rows[0]
    states = queue_states(rows)
    return {
        "milestone": first["milestone"],
        "path": first["path"],
        "area": first["area"],
        "total": len(rows),
        "verified": states["verified"],
        "pending": states["inventoried"] + states["implemented"],
        "implemented": states["implemented"],
        "blocked": states["blocked"],
        "excluded": states["excluded"],
        "stale": sum(bool(row.get("stale")) for row in rows),
        "missing_baseline": sum(bool(missing_baseline_fields(row)) for row in rows),
        "issues": queue_issues(rows),
    }


def scenario_queue(values: list[object]) -> dict[str, object]:
    """Group open work by assigned milestone; example: scenario_queue(ledger_rows)."""
    groups: dict[tuple[str, str], list[Mapping[str, object]]] = {}
    unclassified = 0
    for value in values:
        row = require_object(value, "ledger scenario")
        if not row.get("classified"):
            unclassified += 1
            continue
        milestone = text_field(row, "milestone", "scenario")
        path = text_field(row, "path", "scenario")
        groups.setdefault((milestone, path), []).append(row)
    ordered = sorted(groups, key=lambda key: (milestone_order(key[0]), key[1]))
    summaries = [queue_group(groups[key]) for key in ordered]
    return {
        "unclassified": unclassified,
        "rows": [row for row in summaries if queue_open(row)],
    }


def queue_open(row: Mapping[str, object]) -> bool:
    """Keep unresolved scenarios or baseline evidence; example: queue_open(summary)."""
    return bool(row["pending"] or row["blocked"] or row["stale"] or row["missing_baseline"])

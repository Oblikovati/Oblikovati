#!/usr/bin/env python3
"""Print the whole-project measures a local SonarQube reports. Context, not a gate.

The gate lives in scripts/sonar-newcode.py, because the two numbers that fail a PR are about NEW
code and a local Community server cannot compute those itself (see that file for why).

Kept as a file rather than a heredoc inside sonar-local.sh: an inline `python3 -c` carrying nested
quotes is where the quoting broke twice while this was being built.

    curl .../api/measures/component?...&metricKeys=... | scripts/sonar-gate.py
"""
import json
import sys

LABELS = {
    "coverage": "coverage",
    "duplicated_lines_density": "duplication",
    "bugs": "bugs",
    "vulnerabilities": "vulnerabilities",
    "code_smells": "code smells",
    "ncloc": "lines of code",
}
PERCENT = {"coverage", "duplicated_lines_density"}


def print_measures() -> None:
    """Never raises and never fails the run: this is context, and missing context is not a defect."""
    try:
        measures = json.load(sys.stdin)["component"]["measures"]
    except (json.JSONDecodeError, KeyError):
        print("    (could not read the measures API)", file=sys.stderr)
        return

    for metric in LABELS:  # a fixed order, so two runs are comparable at a glance
        found = next((m for m in measures if m["metric"] == metric), None)
        if found is None:
            continue
        raw = found.get("value", "-")
        suffix = "%" if metric in PERCENT and raw != "-" else ""
        print(f"    {LABELS[metric]:<30} {raw}{suffix}")


if __name__ == "__main__":
    print_measures()

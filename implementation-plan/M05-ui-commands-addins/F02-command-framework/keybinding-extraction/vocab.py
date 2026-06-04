#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Parse the Inventor CommandIDEnum.cs into the canonical operation vocabulary.

The mapping target for every vendor shortcut is one of these enum members (the
user's constraint: map onto the Inventor API enums we already have, never invent
operation IDs). Output is a JSON list of {id, value, summary} where `id` is the
enum member name (e.g. "kCreateExtrudeCommand").

Usage:
    python3 vocab.py > out/vocab.json
"""
import json
import re
import sys
from pathlib import Path

ENUM = (
    Path(__file__).resolve().parents[5]
    / "Oblikovati.Contracts/Oblikovati.Contracts.CSharp/Enums/CommandIDEnum.cs"
)

# A member is `kName = 1234,` optionally preceded by a /// <summary>...</summary>.
_MEMBER = re.compile(r"^\s*(k[A-Za-z0-9_]+)\s*=\s*(\d+)\s*,?\s*$")
_SUMMARY_TEXT = re.compile(r"<summary>\s*(.*?)\s*</summary>", re.DOTALL)


def parse(path: Path) -> list[dict]:
    lines = path.read_text(encoding="utf-8").splitlines()
    members, pending = [], []
    for line in lines:
        stripped = line.strip()
        if stripped.startswith("///"):
            pending.append(stripped.lstrip("/").strip())
            continue
        m = _MEMBER.match(line)
        if m:
            members.append(
                {"id": m.group(1), "value": int(m.group(2)), "summary": _summary(pending)}
            )
        pending = []
    return members


def _summary(doc_lines: list[str]) -> str:
    """Pull the text between <summary> tags out of the accumulated /// lines."""
    blob = " ".join(doc_lines)
    found = _SUMMARY_TEXT.search(blob)
    return re.sub(r"\s+", " ", found.group(1)).strip() if found else ""


if __name__ == "__main__":
    if not ENUM.exists():
        sys.exit(f"CommandIDEnum.cs not found at {ENUM}")
    members = parse(ENUM)
    json.dump(members, sys.stdout, indent=2, ensure_ascii=False)
    print(f"\n# {len(members)} canonical command IDs", file=sys.stderr)

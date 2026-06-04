#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Extract raw {category, action, chord, search} rows from each vendor's PDF.

Three vendors (Siemens NX, Solid Edge, SolidWorks) ship clean tabular layouts that
`pdftotext -layout` preserves as fixed columns; Inventor's PDF is a two-column
sticker sheet handled separately by extract_inventor.py. Output: out/raw/<vendor>.json.

Usage:
    python3 extract.py            # all tabular vendors
"""
import json
import re
import subprocess
import sys
from pathlib import Path

from chord import normalize

HERE = Path(__file__).resolve().parent
PDF_DIR = HERE.parents[4]  # the workspace root, where the source *-shortcuts.pdf live
RAW_OUT = HERE / "generated/raw"

# Each vendor's column layout differs; a parser yields (category, action, chord, search).
# `search` is SolidWorks' extra "Search Shortcut" alias column (kept, not a key chord).


def _pdftotext(pdf: Path) -> list[str]:
    out = subprocess.run(
        ["pdftotext", "-layout", str(pdf), "-"], capture_output=True, text=True, check=True
    )
    return out.stdout.splitlines()


def parse_numbered(lines: list[str]) -> list[dict]:
    """NX / Solid Edge: `No.  Action ....  Chord ....  Section`, action/section multi-word.

    Columns are whitespace-separated with the chord as a middle block; we split on
    runs of 2+ spaces and take [idx, action, chord, section].
    """
    rows = []
    for line in lines:
        if not re.match(r"\s*\d+\s{2,}", line):
            continue
        cols = re.split(r"\s{2,}", line.strip())
        if len(cols) < 4:
            continue
        _, action, chord, section = cols[0], cols[1], cols[2], cols[3]
        rows.append(_row(section, action, chord))
    return rows


# SolidWorks groups every shortcut under one of these menu/section headers; using the
# header as the row key captures both the "Command.." menu rows and the suffix-less
# "Others"/"Search" rows, while rejecting the repeated table header and page footers.
_SW_CATEGORIES = {"File", "Edit", "View", "Insert", "Tools", "Help", "Others", "Search"}


def parse_solidworks(lines: list[str]) -> list[dict]:
    """SolidWorks: `Category  Command[..]  Shortcut(s)  Search`. The chord and the
    search alias are each optional; menu rows suffix the command with '..'."""
    rows = []
    for line in lines:
        cols = re.split(r"\s{2,}", line.strip())
        if len(cols) < 2 or cols[0] not in _SW_CATEGORIES:
            continue
        action = cols[1].rstrip(".").strip()
        chord, search = _sw_cols(cols[2:])
        rows.append(_row(cols[0], action, chord, search))
    return rows


def _sw_cols(rest: list[str]) -> tuple[str, str]:
    """Disambiguate SolidWorks' trailing two columns. A lowercase token with no
    modifier and length>1 (e.g. 'zf', 'bom') is a search alias, not a key chord."""
    chord = search = ""
    for tok in rest:
        if re.fullmatch(r"[a-z]{1,3}", tok):
            search = tok
        else:
            chord = tok
    return chord, search


def _row(category: str, action: str, chord: str, search: str = "") -> dict:
    norm = normalize(chord)
    return {
        "category": category.strip(),
        "action": action.strip(),
        "chord_raw": chord.strip(),
        "chord": norm["chord"] if norm else "",
        "gesture": bool(norm and norm["gesture"]),
        "search": search.strip(),
    }


PARSERS = {
    "siemens-nx": parse_numbered,
    "solidedge": parse_numbered,
    "solidworks": parse_solidworks,
}


def run() -> None:
    RAW_OUT.mkdir(parents=True, exist_ok=True)
    for vendor, parser in PARSERS.items():
        pdf = PDF_DIR / f"{vendor}-shortcuts.pdf"
        rows = parser(_pdftotext(pdf))
        (RAW_OUT / f"{vendor}.json").write_text(json.dumps(rows, indent=2, ensure_ascii=False))
        print(f"{vendor}: {len(rows)} rows", file=sys.stderr)


if __name__ == "__main__":
    run()

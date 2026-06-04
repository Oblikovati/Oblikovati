#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Extract Inventor's shortcut entries from its two-column sticker-sheet PDF.

Unlike the other vendors, Inventor's layout is multi-column art, so row order is
unrecoverable — but every real entry is anchored by the same delimiter that
`pdftotext` emits between the key and the command name: TAB + BEL (\\t\\x07), e.g.
`BA\\t\\x07AUTO BALLOON / Creates ...`. We scan for that anchor globally and pull
(key, name); a few one-key punctuation rows use aligned spaces instead and are
recovered by a second, stricter pass. Category is intentionally dropped here
(columns make it unreliable); the mapping file keys on the command name.

Output: generated/raw/autodesk-inventor.json (kind = "alias" for typed multi-char
aliases, "key" for single keys / named keys / function keys).

Usage:
    python3 extract_inventor.py
"""
import json
import re
import subprocess
from pathlib import Path

from chord import normalize

HERE = Path(__file__).resolve().parent
PDF = HERE.parents[4] / "autodesk-inventor-shortcuts.pdf"  # workspace-root source PDF
OUT = HERE / "generated/raw/autodesk-inventor.json"

# Primary anchor: <key> TAB BEL <NAME> " / ". The key is the non-space token before
# the tab; the name is the uppercase run up to the first " /".
_ANCHOR = re.compile(r"(?P<key>[^\s\t\x07]+)\t\x07\s*(?P<name>[A-Z][A-Z0-9 ,&().\-]*?)\s*/")
# Secondary: a lone punctuation/named key separated from its NAME by aligned spaces
# (e.g. "/            WORK AXIS / ..." and "ESC           CANCEL / ...").
_SPACED = re.compile(r"(?:^|  )(?P<key>[=;/.\]\[]|ESC|END|HOME|DELETE)\s{3,}(?P<name>[A-Z][A-Z0-9 ,&().\-]*?)\s*/", re.M)

_NAMED_KEYS = {"ESC", "END", "HOME", "DELETE", "PAGE UP", "PAGE DOWN", "TAB", "ENTER"}


def _text() -> str:
    out = subprocess.run(["pdftotext", "-layout", str(PDF), "-"], capture_output=True, text=True, check=True)
    return out.stdout


def parse(text: str) -> list[dict]:
    seen: set[tuple[str, str]] = set()
    rows: list[dict] = []
    for pat in (_ANCHOR, _SPACED):
        for m in pat.finditer(text):
            key = m.group("key").strip()
            name = re.sub(r"\s+", " ", m.group("name")).strip()
            if not name or (key, name) in seen:
                continue
            seen.add((key, name))
            rows.append(_row(key, name))
    rows.sort(key=lambda r: (r["kind"], r["action"]))
    return rows


def _row(key: str, name: str) -> dict:
    is_key = bool(re.fullmatch(r"F\d{1,2}", key)) or key in _NAMED_KEYS or len(key) == 1
    norm = normalize(key)
    return {
        "category": "",  # unreliable in two-column layout; mapping keys on action
        "action": name.title(),
        "chord_raw": key,
        "chord": norm["chord"] if (norm and is_key) else "",
        "alias": "" if is_key else key,  # Inventor typed multi-char alias
        "kind": "key" if is_key else "alias",
        "gesture": bool(norm and norm["gesture"]),
        "search": "",
    }


if __name__ == "__main__":
    rows = parse(_text())
    OUT.write_text(json.dumps(rows, indent=2, ensure_ascii=False))
    keys = sum(1 for r in rows if r["kind"] == "key")
    print(f"autodesk-inventor: {len(rows)} rows ({keys} keys, {len(rows) - keys} aliases)")

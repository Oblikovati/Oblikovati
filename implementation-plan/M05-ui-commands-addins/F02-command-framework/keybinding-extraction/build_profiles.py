#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Join raw vendor extractions with the canonical map into per-vendor keybinding profiles.

A profile is the per-profile chord map the future preferences/keybinding system will
load: canonical CommandIDEnum id -> {chord, alias}. We also emit an honest accounting
of what did NOT map (out/unmapped.md) and overall coverage (out/coverage.md), because
the gap (Undo/Redo/Copy/Paste/view orientations/etc. that have no CommandIDEnum member)
is exactly what the implementation phase has to decide about.

Validation: every id in mappings/canonical.json MUST exist in out/vocab.json, else we
abort — this is what enforces "map onto the enums we have, never invent ids".

Usage:
    python3 vocab.py > out/vocab.json   # once, produces the vocabulary
    python3 extract.py                  # tabular vendors -> out/raw/*.json
    python3 extract_inventor.py         # inventor -> out/raw/autodesk-inventor.json
    python3 build_profiles.py           # -> out/profiles/*.json, unmapped.md, coverage.md
"""
import json
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
OUT = HERE / "generated"
VENDORS = ["autodesk-inventor", "solidworks", "siemens-nx", "solidedge"]


def _norm(action: str) -> str:
    """Match key: lowercased, whitespace-collapsed, trailing dots/spaces stripped."""
    return re.sub(r"\s+", " ", action).strip(" .").lower()


def load_index(vocab_ids: set[str]) -> dict[str, str]:
    """Build {normalized action -> canonical id} from the canonical map, rejecting any
    id that is not a real CommandIDEnum member (the no-invented-ids guarantee)."""
    raw = json.loads((HERE / "mappings/canonical.json").read_text())
    index: dict[str, str] = {}
    for cid, synonyms in raw.items():
        if cid.startswith("_"):
            continue
        if cid not in vocab_ids:
            sys.exit(f"canonical.json: id {cid!r} is not in CommandIDEnum.cs (no invented ids)")
        for syn in synonyms:
            index[_norm(syn)] = cid
    return index


def build_profile(rows: list[dict], index: dict[str, str]) -> tuple[dict, list[dict]]:
    """Return (profile, unmapped). profile: cid -> {chord, alias, source}. A row binds a
    chord (keyboard) and/or an alias (typed); the first non-empty wins per field."""
    profile: dict[str, dict] = {}
    unmapped: list[dict] = []
    for row in rows:
        cid = index.get(_norm(row["action"]))
        if cid is None:
            unmapped.append(row)
            continue
        entry = profile.setdefault(cid, {"chord": "", "alias": "", "source": row["action"]})
        if not entry["chord"] and row.get("chord") and not row.get("gesture"):
            entry["chord"] = row["chord"]
        if not entry["alias"] and (row.get("alias") or row.get("search")):
            entry["alias"] = row.get("alias") or row.get("search")
    return profile, unmapped


def run() -> None:
    vocab = json.loads((OUT / "vocab.json").read_text())
    vocab_ids = {m["id"] for m in vocab}
    index = load_index(vocab_ids)
    (OUT / "profiles").mkdir(parents=True, exist_ok=True)

    coverage, unmapped_md, bound_ids = [], [], set()
    for vendor in VENDORS:
        rows = json.loads((OUT / "raw" / f"{vendor}.json").read_text())
        profile, unmapped = build_profile(rows, index)
        bound_ids |= profile.keys()
        (OUT / "profiles" / f"{vendor}.json").write_text(
            json.dumps({"vendor": vendor, "bindings": profile}, indent=2, ensure_ascii=False)
        )
        coverage.append((vendor, len(rows), len(profile), len(unmapped)))
        unmapped_md.append(_unmapped_section(vendor, unmapped))

    _write_coverage(vocab, bound_ids, coverage)
    (OUT / "unmapped.md").write_text("# Unmapped vendor actions\n\n"
        "Actions with no CommandIDEnum counterpart (we do not invent ids). Each is a\n"
        "candidate for either a new operation later or a deliberate 'no Oblikovati op'.\n\n"
        + "\n".join(unmapped_md))
    for v, total, mapped, un in coverage:
        print(f"{v:18} {mapped:3}/{total:<3} mapped, {un:3} unmapped", file=sys.stderr)
    print(f"canonical ids bound by >=1 vendor: {len(bound_ids)}/{len(vocab)}", file=sys.stderr)


def _unmapped_section(vendor: str, unmapped: list[dict]) -> str:
    lines = [f"## {vendor} ({len(unmapped)})", ""]
    for r in sorted(unmapped, key=lambda r: r["action"]):
        chord = r.get("chord") or r.get("alias") or r.get("search") or "—"
        lines.append(f"- `{chord}` {r['action']}")
    return "\n".join(lines) + "\n"


def _write_coverage(vocab: list[dict], bound: set[str], coverage: list[tuple]) -> None:
    lines = ["# Coverage", "", "| Vendor | Rows | Mapped | Unmapped |", "|---|---|---|---|"]
    for v, total, mapped, un in coverage:
        lines.append(f"| {v} | {total} | {mapped} | {un} |")
    unbound = [m["id"] for m in vocab if m["id"] not in bound]
    lines += ["", f"## Canonical commands with no vendor binding ({len(unbound)})", ""]
    lines += [f"- `{cid}`" for cid in unbound]
    (OUT / "coverage.md").write_text("\n".join(lines) + "\n")


if __name__ == "__main__":
    run()

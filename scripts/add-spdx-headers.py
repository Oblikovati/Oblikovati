#!/usr/bin/env python3
"""Insert/verify the GPL-2.0-only SPDX header on every Go file in this repo.

This is the GPL application (module oblikovati) plus its cgo
`head` submodule — all GPL-2.0-only. After the repo split (ADR-0018) the Go sources
live at the repo root rather than under `source/`, so the mapping is simply
"every tracked *.go -> GPL-2.0-only". The Apache-2.0 contract now lives in its own
repo (../Oblikovati.API) with its own checker.

Placement rules (so Go semantics are preserved):
  * The SPDX comment is its own block followed by a blank line, so it never merges
    into a following `// Package ...` doc comment.
  * In files beginning with a build constraint (`//go:build` / `// +build`), the
    header goes AFTER the constraint block (the constraint must stay first).
  * Files that already carry an SPDX-License-Identifier are left untouched.

Usage: python3 scripts/add-spdx-headers.py [--check]
  --check exits non-zero if any file would change (for CI), without writing.
"""
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
IDENTIFIER = "GPL-2.0-only"
# Not part of the GPL Go surface: VCS, tooling, disposable experiments, and the
# vendored third-party C/Go (e.g. head/third_party/imgui, MIT) which keeps its own
# upstream headers.
SKIP_DIRS = {".git", "scripts", "experiments", "third_party"}


def header(identifier: str) -> str:
    return f"// SPDX-License-Identifier: {identifier}\n"


def is_constraint(line: str) -> bool:
    s = line.lstrip()
    return s.startswith("//go:build") or s.startswith("// +build")


def insert_index(lines: list[str]) -> int:
    if lines and is_constraint(lines[0]):
        i = 0
        while i < len(lines) and (is_constraint(lines[i]) or lines[i].strip() == ""):
            i += 1
        return i
    return 0


def patched(text: str, identifier: str) -> str | None:
    if "SPDX-License-Identifier" in text:
        return None
    lines = text.splitlines(keepends=True)
    at = insert_index(lines)
    return "".join(lines[:at] + [header(identifier), "\n"] + lines[at:])


def go_files() -> list[Path]:
    return [
        p
        for p in sorted(ROOT.rglob("*.go"))
        if not any(part in SKIP_DIRS for part in p.relative_to(ROOT).parts)
    ]


def inside_root(path: Path) -> Path:
    """Re-anchor path under the repository root, refusing anything that escapes
    it — a symlinked .go file pointing outside the repo must never be rewritten.
    The returned path is constructed from ROOT plus the validated relative part,
    so every later read/write is provably root-anchored (S2083)."""
    relative = path.resolve().relative_to(ROOT)  # raises ValueError outside ROOT
    return ROOT.joinpath(relative)


def main() -> int:
    check = "--check" in sys.argv[1:]
    changed = []
    for path in go_files():
        path = inside_root(path)
        out = patched(path.read_text(), IDENTIFIER)
        if out is None:
            continue
        changed.append(path.relative_to(ROOT))
        if not check:
            path.write_text(out)
    if check and changed:
        print("missing SPDX header:")
        for p in changed:
            print(f"  {p}")
        return 1
    if not check:
        print(f"added SPDX headers to {len(changed)} files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

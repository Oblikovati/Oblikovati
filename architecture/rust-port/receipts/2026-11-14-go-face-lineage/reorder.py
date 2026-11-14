# SPDX-License-Identifier: GPL-2.0-only
"""Rotate one known fixture's entity records before Go reopens and resaves it."""
from pathlib import Path
import sys

source, destination = map(Path, sys.argv[1:])
text = source.read_text()
start = text.index("          entities:\n")
end = text.index("          constraints:\n", start)
chunks = text[start:end].split("            - id: ")
if len(chunks) != 5 or [part.splitlines()[0] for part in chunks[1:]] != ["6", "7", "8", "9"]:
    raise ValueError("fixture entity records: expected IDs 6, 7, 8, 9 in that order")
rotated = chunks[0] + "".join("            - id: " + part for part in [*chunks[2:], chunks[1]])
destination.write_text(text[:start] + rotated + text[end:])

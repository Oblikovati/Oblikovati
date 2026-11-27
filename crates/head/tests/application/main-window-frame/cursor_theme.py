# SPDX-License-Identifier: GPL-2.0-only
"""Write a private Xcursor theme with identifiable resize cursor buffers."""
from pathlib import Path
from struct import pack
import sys

IMAGE = 0xfffd0002
SIZE = 24


def write_cursor(path: Path, width: int) -> None:
    header = b"Xcur" + pack("<6I", 16, 0x10000, 1, IMAGE, SIZE, 28)
    chunk = pack("<9I", 36, IMAGE, SIZE, 1, width, SIZE, width // 2, SIZE // 2, 0)
    pixels = (0xffffffff if y == SIZE // 2 or x == width // 2 else 0
              for y in range(SIZE) for x in range(width))
    path.write_bytes(header + chunk + pack(f"<{width * SIZE}I", *pixels))


if __name__ == "__main__":
    cursors = Path(sys.argv[1]) / "oblikovati-frame" / "cursors"
    cursors.mkdir(parents=True, exist_ok=True)
    write_cursor(cursors / "default", SIZE - 2)
    for x, horizontal in enumerate(("w", "", "e")):
        for y, vertical in enumerate(("n", "", "s")):
            if vertical or horizontal:
                write_cursor(cursors / f"{vertical}{horizontal}-resize", SIZE + 3 * x + y)

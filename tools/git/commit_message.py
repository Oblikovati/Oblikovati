#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Validate a commit message after Git removes template comments."""

import re
import subprocess
import sys
from pathlib import Path

REFERENCE = re.compile(r"^Refs\s+#[1-9][0-9]*\s*$", re.IGNORECASE)
TRAILER = re.compile(r"^[A-Za-z][A-Za-z0-9-]*:\s*\S")


def message_error(message: str) -> str | None:
    """Return the first error, or None; example: message_error('Fix X\\n\\nExplain why.\\n\\nRefs #1')."""
    lines = message.strip().splitlines()
    if not lines or not lines[0].strip():
        return "expected a nonempty subject"
    if len(lines) < 3 or lines[1].strip():
        return "expected a blank line and an explanatory body after the subject"
    body = [line.strip() for line in lines[2:] if line.strip()]
    if not any(REFERENCE.fullmatch(line) for line in body):
        return "expected a separate Refs #issue line with a positive issue number"
    if not any(
        not REFERENCE.fullmatch(line) and not TRAILER.match(line) for line in body
    ):
        return "expected explanatory body text in addition to the issue reference"
    return None


def cleaned_message(path: Path) -> str:
    """Use Git's comment rules; example: cleaned_message(Path('.git/COMMIT_EDITMSG'))."""
    message = path.read_bytes()
    result = subprocess.run(
        ["git", "stripspace", "--strip-comments"],
        input=message,
        capture_output=True,
        check=True,
    )
    return result.stdout.decode("utf-8")


def main(arguments: list[str]) -> int:
    """Check Git's message file; example: main(['.git/COMMIT_EDITMSG'])."""
    if len(arguments) != 1:
        print(
            f"commit message arguments {arguments!r}: expected one message file",
            file=sys.stderr,
        )
        return 1
    try:
        error = message_error(cleaned_message(Path(arguments[0])))
    except (OSError, UnicodeError, subprocess.CalledProcessError) as failure:
        print(
            f"commit message {arguments[0]!r}: expected readable UTF-8 text ({failure})",
            file=sys.stderr,
        )
        return 1
    if error:
        print(f"commit message {arguments[0]!r}: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

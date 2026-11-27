# SPDX-License-Identifier: GPL-2.0-only
"""Restrict Cargo source configuration used by native capture."""

import os
import stat
import tomllib
from pathlib import Path


ALLOWED_CAPTURE_CONFIG = {
    "alias": {
        "xtask": "run --locked --target-dir target/xtask --package oblikovati-build --",
    },
    "target": {
        'cfg(target_os = "linux")': {
            "rustflags": ["-C", "link-arg=-Wl,-rpath,$ORIGIN/../lib"],
        },
    },
}


def read_source_config_bytes(path: Path) -> bytes:
    """Read one regular config without following links; example: read_source_config_bytes(path)."""
    if not hasattr(os, "O_NOFOLLOW"):
        raise ValueError("Cargo config open: expected no-follow support")
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | getattr(os, "O_NONBLOCK", 0))
    try:
        if not stat.S_ISREG(os.fstat(descriptor).st_mode):
            raise ValueError(f"Cargo config {path}: expected a regular file")
        with os.fdopen(descriptor, "rb", closefd=False) as file:
            content = file.read()
    finally:
        os.close(descriptor)
    return content


def read_capture_source_config(source: Path) -> bytes | None:
    """Read only the supported project config; example: read_capture_source_config(source)."""
    folder = source / ".cargo"
    if folder.is_symlink():
        raise ValueError(f"Cargo config directory {folder}: expected a real directory")
    legacy = folder / "config"
    if legacy.exists() or legacy.is_symlink():
        raise ValueError(f"Cargo config {legacy}: expected no legacy configuration")
    path = folder / "config.toml"
    if path.is_symlink():
        raise ValueError(f"Cargo config {path}: expected a regular file")
    if not path.exists():
        return None
    content = read_source_config_bytes(path)
    config = tomllib.loads(content.decode("utf-8"))
    for key, value in config.items():
        if key not in ALLOWED_CAPTURE_CONFIG or value != ALLOWED_CAPTURE_CONFIG[key]:
            raise ValueError(f"Cargo config {path} setting {key!r}: expected an approved capture setting")
    return content


def require_same_source_config(source: Path, expected: bytes | None) -> None:
    """Reject source config changes after a run; example: require_same_source_config(source, before)."""
    actual = read_capture_source_config(source)
    if actual != expected:
        raise ValueError(f"Cargo config {source / '.cargo'}: expected unchanged capture configuration")

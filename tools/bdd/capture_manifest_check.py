# SPDX-License-Identifier: GPL-2.0-only
"""Reject workspace manifest redirects outside the exported source tree."""

import hashlib
import tomllib
from pathlib import Path, PureWindowsPath
from typing import Mapping


def require_internal_path(value: object, manifest: Path, source: Path) -> None:
    """Refuse external manifest paths; example: require_internal_path('../core', manifest, source)."""
    if not isinstance(value, str) or not value:
        raise ValueError(f"manifest {manifest} path {value!r}: expected a nonempty relative path")
    path = Path(value)
    if path.is_absolute() or PureWindowsPath(value).is_absolute():
        raise ValueError(f"manifest {manifest} path {value!r}: expected a path inside {source}")
    if not (manifest.parent / path).resolve().is_relative_to(source.resolve()):
        raise ValueError(f"manifest {manifest} path {value!r}: expected a path inside {source}")


def check_manifest_values(value: object, manifest: Path, source: Path) -> None:
    """Check path fields in nested TOML values; example: check_manifest_values(document, manifest, source)."""
    if isinstance(value, list):
        for item in value:
            check_manifest_values(item, manifest, source)
    if not isinstance(value, dict):
        return
    for key, item in value.items():
        if key == "path":
            require_internal_path(item, manifest, source)
        check_manifest_values(item, manifest, source)


def read_capture_manifests(source: Path) -> dict[str, str]:
    """Validate and hash exported manifests; example: read_capture_manifests(source)."""
    result: dict[str, str] = {}
    for path in sorted(source.rglob("Cargo.toml")):
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"manifest {path}: expected a regular file")
        content = path.read_bytes()
        document = tomllib.loads(content.decode("utf-8"))
        forbidden = set(document) & {"patch", "replace"}
        if forbidden:
            raise ValueError(f"manifest {path} overrides {sorted(forbidden)!r}: expected no patch or replace")
        check_manifest_values(document, path, source)
        result[path.relative_to(source).as_posix()] = hashlib.sha256(content).hexdigest()
    if not result:
        raise ValueError(f"source {source}: expected at least one Cargo.toml")
    return result


def require_same_manifests(source: Path, expected: Mapping[str, str]) -> None:
    """Reject changed or added workspace manifests; example: require_same_manifests(source, before)."""
    if read_capture_manifests(source) != expected:
        raise ValueError(f"source {source}: expected unchanged capture manifests")

# SPDX-License-Identifier: GPL-2.0-only
"""Check the conventional normalized library target against a locked Git tree."""

import re
import tomllib
from typing import Mapping


def require_conventional_library(source: Mapping[str, object], generated: Mapping[str, object], entries: Mapping[str, str], label: str) -> None:
    """Check only Cargo's conventional inferred library; example: require_conventional_library(src, dst, files, label)."""
    package = source.get("package")
    if not isinstance(package, dict):
        raise ValueError(f"manifest {label} package {package!r}: expected a package table")
    name = package.get("name")
    if not isinstance(name, str) or not re.fullmatch(r"[A-Za-z][A-Za-z0-9_-]*", name):
        raise ValueError(f"manifest {label} name {name!r}: expected a conventional package name")
    if "lib" in source or package.get("autolib", True) is not True:
        raise ValueError(f"manifest {label} source library: expected an autodetected library")
    if "src/lib.rs" not in entries:
        raise ValueError(f"manifest {label} src/lib.rs: expected a tracked library source")
    expected = {"name": name.replace("-", "_"), "path": "src/lib.rs"}
    if generated.get("lib") != expected:
        raise ValueError(f"manifest {label} library {generated.get('lib')!r}: expected {expected!r}")


def require_no_autodiscovery(generated: Mapping[str, object], label: str) -> None:
    """Require Cargo's normalized target autodiscovery switches; example: require_no_autodiscovery(manifest, label)."""
    package = generated.get("package")
    if not isinstance(package, dict):
        raise ValueError(f"manifest {label} package {package!r}: expected a package table")
    for field in ("autolib", "autobins", "autoexamples", "autotests", "autobenches"):
        if package.get(field) is not False:
            raise ValueError(f"manifest {label} {field} {package.get(field)!r}: expected false")


def require_matching_links(source: Mapping[str, object], generated: Mapping[str, object], label: str) -> None:
    """Keep the native-links declaration unchanged; example: require_matching_links(src, dst, label)."""
    original, output = source.get("package"), generated.get("package")
    if not isinstance(original, dict) or not isinstance(output, dict):
        raise ValueError(f"manifest {label} package: expected source and generated package tables")
    expected, actual = original.get("links"), output.get("links")
    if expected is not None and not isinstance(expected, str):
        raise ValueError(f"manifest {label} source links {expected!r}: expected a string or absence")
    if type(actual) is not type(expected) or actual != expected:
        raise ValueError(f"manifest {label} links {actual!r}: expected locked source {expected!r}")


def check_git_manifest_library(source: bytes, generated: bytes, entries: Mapping[str, str], label: str) -> None:
    """Check the conventional library and target switches; example: check_git_manifest_library(src, dst, entries, label)."""
    original = tomllib.loads(source.decode("utf-8"))
    output = tomllib.loads(generated.decode("utf-8"))
    require_conventional_library(original, output, entries, label)
    require_no_autodiscovery(output, label)
    require_matching_links(original, output, label)

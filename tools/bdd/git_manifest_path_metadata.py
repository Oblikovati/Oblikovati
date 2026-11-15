# SPDX-License-Identifier: GPL-2.0-only
"""Check path metadata where Cargo normalization is independent of the filesystem."""

import tomllib
from typing import Mapping

from git_manifest_declarations import declaration, same_toml_value


def explicit_readme(value: object, label: str) -> object:
    """Normalize an explicit README declaration; example: explicit_readme(True, 'crate')."""
    if value is True:
        return "README.md"
    if value is False or isinstance(value, str):
        return value
    raise ValueError(f"manifest {label} readme {value!r}: expected a string or boolean")


def inherited_path(value: object, label: str, name: str) -> bool:
    """Identify an inherited path without guessing its value; example: inherited_path(value, label, 'readme')."""
    if not isinstance(value, dict):
        return False
    if set(value) != {"workspace"} or value["workspace"] is not True:
        raise ValueError(f"manifest {label} {name} {value!r}: expected only workspace = true")
    return True


def compare_path_value(actual: object, expected: object, label: str, name: str) -> None:
    """Reject a changed path declaration; example: compare_path_value(actual, expected, label, 'readme')."""
    if not same_toml_value(actual, expected):
        raise ValueError(f"manifest {label} {name} {actual!r}: expected locked source value {expected!r}")


def check_git_manifest_path_metadata(source: bytes, generated: bytes, label: str) -> None:
    """Check explicit path values only; example: check_git_manifest_path_metadata(src, dst, 'crate')."""
    original = tomllib.loads(source.decode("utf-8")).get("package")
    output = tomllib.loads(generated.decode("utf-8")).get("package")
    if not isinstance(original, dict) or not isinstance(output, dict):
        raise ValueError(f"manifest {label} packages {(original, output)!r}: expected two tables")
    readme = original.get("readme")
    if readme is not None and not inherited_path(readme, label, "readme"):
        compare_path_value(output.get("readme"), explicit_readme(readme, label), label, "readme")
    names = ("license-file", "license_file")
    license_file = declaration(original, names, label)
    if not inherited_path(license_file, label, "license-file"):
        if license_file is not None and not isinstance(license_file, str):
            raise ValueError(f"manifest {label} license-file {license_file!r}: expected a string or absence")
        compare_path_value(declaration(output, names, label), license_file, label, "license-file")

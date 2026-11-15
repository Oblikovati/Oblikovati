# SPDX-License-Identifier: GPL-2.0-only
"""Verify selected Cargo package metadata and lints against locked Git blobs."""

import tomllib
from pathlib import Path
from typing import Mapping

from git_manifest_declarations import declaration, same_toml_value
from git_manifest_workspace import locked_workspace


METADATA = (("description",), ("documentation",), ("homepage",), ("repository",),
            ("license",), ("rust-version", "rust_version"), ("authors",),
            ("keywords",), ("categories",), ("publish",))


def inherited_value(value: object, workspace: Mapping[str, object], section: str, names: tuple[str, ...], label: str) -> object:
    """Resolve an explicit workspace field; example: inherited_value(value, workspace, 'package', ('license',), label)."""
    if not isinstance(value, dict):
        return value
    if set(value) != {"workspace"} or value["workspace"] is not True:
        raise ValueError(f"manifest {label} {names[0]} {value!r}: expected a workspace inheritance marker")
    inherited = workspace.get(section)
    if not isinstance(inherited, dict):
        raise ValueError(f"manifest {label} workspace {section} {inherited!r}: expected a table")
    result = declaration(inherited, names, label)
    if result is None or isinstance(result, dict):
        raise ValueError(f"manifest {label} workspace {names[0]} {result!r}: expected an explicit value")
    return result


def workspace_for_fields(source: Mapping[str, object], repository: Path, tree: Mapping[str, str], root: str, label: str) -> Mapping[str, object]:
    """Load a workspace only when a checked field requests it; example: workspace_for_fields(src, repo, tree, root, label)."""
    package = source.get("package")
    if not isinstance(package, dict) or "workspace" in package:
        raise ValueError(f"manifest {label} package {package!r}: expected an implicit workspace selection")
    values = [declaration(package, names, label) for names in METADATA]
    lints = source.get("lints")
    if any(isinstance(value, dict) for value in values) or (isinstance(lints, dict) and "workspace" in lints):
        return locked_workspace(repository, tree, root, label)
    return {}


def expected_lints(value: object, workspace: Mapping[str, object], label: str) -> object:
    """Resolve direct or inherited lint tables; example: expected_lints(lints, workspace, label)."""
    if value is None:
        return None
    if not isinstance(value, dict):
        raise ValueError(f"manifest {label} lints {value!r}: expected a table")
    if "workspace" not in value:
        return value
    if set(value) != {"workspace"} or value["workspace"] is not True:
        raise ValueError(f"manifest {label} lints {value!r}: expected only workspace = true")
    result = workspace.get("lints")
    if not isinstance(result, dict):
        raise ValueError(f"manifest {label} workspace lints {result!r}: expected a table")
    return result


def check_git_manifest_metadata(source: bytes, generated: bytes, repository: Path, tree: Mapping[str, str], root: str, label: str) -> None:
    """Check independent package fields and lints; example: check_git_manifest_metadata(src, dst, repo, tree, root, label)."""
    original, output = tomllib.loads(source.decode("utf-8")), tomllib.loads(generated.decode("utf-8"))
    workspace = workspace_for_fields(original, repository, tree, root, label)
    package = output.get("package")
    if not isinstance(package, dict):
        raise ValueError(f"manifest {label} generated package {package!r}: expected a table")
    for names in METADATA:
        expected = inherited_value(declaration(original["package"], names, label), workspace, "package", names, label)
        actual = declaration(package, names, label)
        if not same_toml_value(actual, expected):
            raise ValueError(f"manifest {label} {names[0]} {actual!r}: expected locked source value {expected!r}")
    expected = expected_lints(original.get("lints"), workspace, label)
    if not same_toml_value(output.get("lints"), expected):
        raise ValueError(f"manifest {label} lints {output.get('lints')!r}: expected locked source lints {expected!r}")

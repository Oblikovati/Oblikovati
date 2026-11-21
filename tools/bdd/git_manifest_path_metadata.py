# SPDX-License-Identifier: GPL-2.0-only
"""Check path metadata where Cargo normalization is independent of the filesystem."""

import posixpath
import re
import tomllib
from pathlib import Path
from typing import Mapping

from git_manifest_declarations import declaration, same_toml_value
from git_manifest_workspace import locked_workspace_location


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


def relative_workspace_path(value: object, workspace_root: str, package_root: str, label: str, name: str) -> str:
    """Resolve a portable workspace path lexically; example: relative_workspace_path('LICENSE', '.', 'crate', label, 'license-file')."""
    if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9_-]+(?:[./][A-Za-z0-9_-]+)*", value):
        raise ValueError(f"manifest {label} workspace {name} {value!r}: expected a portable relative path")
    return posixpath.relpath(posixpath.join(workspace_root, value), package_root)


def inherited_workspace_values(repository: Path, tree: Mapping[str, str], root: str, label: str) -> tuple[Mapping[str, object], str]:
    """Read inherited path declarations and root; example: inherited_workspace_values(repo, tree, root, label)."""
    workspace, workspace_root = locked_workspace_location(repository, tree, root, label)
    package = workspace.get("package")
    if not isinstance(package, dict):
        raise ValueError(f"manifest {label} workspace package {package!r}: expected a table")
    return package, workspace_root


def locked_highest_priority_readme(tree: Mapping[str, str], root: str) -> str | None:
    """Find a tracked regular highest-priority default; example: locked_highest_priority_readme(tree, 'crate')."""
    path = posixpath.join(root, "README.md") if root != "." else "README.md"
    identity = tree.get(path, "")
    if identity.startswith(("100644:blob:", "100755:blob:")):
        return "README.md"
    return None


def require_missing_default_readme(tree: Mapping[str, str], root: str, label: str) -> bool:
    """Refuse unchecked default files; example: require_missing_default_readme(tree, 'crate', label)."""
    for name in ("README.md", "README.txt", "README"):
        path = posixpath.join(root, name) if root != "." else name
        if path in tree or any(entry.startswith(path + "/") for entry in tree):
            raise ValueError(f"manifest {label} readme {path!r}: expected a supported tracked default")
    return False


def check_inherited_path(actual: object, name: str, workspace: Mapping[str, object], workspace_root: str, root: str, label: str, tree: Mapping[str, str]) -> None:
    """Compare an explicit inherited path; example: check_inherited_path(actual, name, workspace, '.', 'crate', label, tree)."""
    names = ("license-file", "license_file") if name == "license-file" else ("readme",)
    expected = declaration(workspace, names, label)
    if name == "readme" and expected is None:
        expected = locked_highest_priority_readme(tree, workspace_root)
        if expected is None:
            raise ValueError(f"manifest {label} workspace readme: expected a supported declared or default path")
    if name == "readme" and expected is True:
        expected = "README.md"
    path = relative_workspace_path(expected, workspace_root, root, label, name)
    compare_path_value(actual, path, label, name)


def check_workspace_paths(original: Mapping[str, object], output: Mapping[str, object], repository: Path, tree: Mapping[str, str], root: str, label: str) -> None:
    """Check inherited paths from the locked workspace; example: check_workspace_paths(src, dst, repo, tree, root, label)."""
    names = ("license-file", "license_file")
    readme = inherited_path(original.get("readme"), label, "readme")
    license_file = inherited_path(declaration(original, names, label), label, "license-file")
    if not readme and not license_file:
        return
    workspace, workspace_root = inherited_workspace_values(repository, tree, root, label)
    if readme:
        check_inherited_path(output.get("readme"), "readme", workspace, workspace_root, root, label, tree)
    if license_file:
        check_inherited_path(declaration(output, names, label), "license-file", workspace, workspace_root, root, label, tree)


def check_git_manifest_path_metadata(source: bytes, generated: bytes, repository: Path, tree: Mapping[str, str], root: str, label: str) -> None:
    """Check supported explicit and inherited paths; example: check_git_manifest_path_metadata(src, dst, repo, tree, root, label)."""
    original = tomllib.loads(source.decode("utf-8")).get("package")
    output = tomllib.loads(generated.decode("utf-8")).get("package")
    if not isinstance(original, dict) or not isinstance(output, dict):
        raise ValueError(f"manifest {label} packages {(original, output)!r}: expected two tables")
    readme = original.get("readme")
    readme_inherited = inherited_path(readme, label, "readme")
    if readme is None:
        readme = locked_highest_priority_readme(tree, root)
        if readme is None:
            readme = require_missing_default_readme(tree, root, label)
    if readme is not None and not readme_inherited:
        compare_path_value(output.get("readme"), explicit_readme(readme, label), label, "readme")
    names = ("license-file", "license_file")
    license_file = declaration(original, names, label)
    license_inherited = inherited_path(license_file, label, "license-file")
    if not license_inherited:
        if license_file is not None and not isinstance(license_file, str):
            raise ValueError(f"manifest {label} license-file {license_file!r}: expected a string or absence")
        compare_path_value(declaration(output, names, label), license_file, label, "license-file")
    check_workspace_paths(original, output, repository, tree, root, label)

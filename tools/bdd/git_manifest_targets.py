# SPDX-License-Identifier: GPL-2.0-only
"""Compare normalized Cargo targets with locked source declarations and files."""

import posixpath
import tomllib
from pathlib import Path, PurePosixPath
from typing import Mapping

from git_manifest_declarations import same_toml_value
from git_manifest_workspace import locked_workspace


TARGET_KINDS = {"bin": ("src/bin", "autobins"), "example": ("examples", "autoexamples"), "test": ("tests", "autotests"), "bench": ("benches", "autobenches")}


def workspace_edition(repository: Path, tree: Mapping[str, str], root: str, label: str) -> str:
    """Read the closest tracked workspace edition; example: workspace_edition(repo, tree, root, label)."""
    workspace = locked_workspace(repository, tree, root, label)
    package = workspace.get("package")
    if not isinstance(package, dict) or not isinstance(package.get("edition"), str):
        raise ValueError(f"manifest {label} workspace edition: expected an explicit string")
    return package["edition"]


def target_edition(source: Mapping[str, object], repository: Path, tree: Mapping[str, str], root: str, label: str) -> str:
    """Resolve the edition needed for target discovery; example: target_edition(src, repo, tree, root, label)."""
    package = source.get("package")
    if not isinstance(package, dict) or "workspace" in package:
        raise ValueError(f"manifest {label} package workspace: expected implicit workspace selection")
    edition = package.get("edition", "2015")
    if isinstance(edition, dict) and set(edition) == {"workspace"} and edition["workspace"] is True:
        edition = workspace_edition(repository, tree, root, label)
    if edition not in ("2015", "2018", "2021", "2024"):
        raise ValueError(f"manifest {label} edition {edition!r}: expected a supported Cargo edition")
    return str(edition)


def inferred_target(path: str, directory: str) -> tuple[str, str] | None:
    """Recognize a Cargo conventional target path; example: inferred_target('tests/a.rs', 'tests')."""
    parts = PurePosixPath(path).parts
    base = PurePosixPath(directory).parts
    if parts[:len(base)] != base:
        return None
    rest = parts[len(base):]
    if len(rest) == 1 and rest[0].endswith(".rs") and not rest[0].startswith("."):
        return rest[0][:-3], path
    if len(rest) == 2 and rest[1] == "main.rs" and not rest[0].startswith("."):
        return rest[0], path
    return None


def inferred_targets(entries: Mapping[str, str], kind: str, package_name: str) -> dict[str, str]:
    """Build a unique map of tracked conventional targets; example: inferred_targets(files, 'test', name)."""
    directory, _ = TARGET_KINDS[kind]
    pairs = [target for path in entries if (target := inferred_target(path, directory)) is not None]
    if kind == "bin" and "src/main.rs" in entries:
        pairs.append((package_name, "src/main.rs"))
    result: dict[str, str] = {}
    for name, path in pairs:
        if not name or name in result:
            raise ValueError(f"{kind} target {name!r}: expected one conventional source path")
        result[name] = path
    return result


def explicit_targets(value: object, inferred: Mapping[str, str], entries: Mapping[str, str], label: str) -> dict[str, dict[str, object]]:
    """Normalize supported explicit target declarations; example: explicit_targets(rows, inferred, files, label)."""
    if value is None:
        return {}
    if not isinstance(value, list):
        raise ValueError(f"targets {label} {value!r}: expected a list")
    result: dict[str, dict[str, object]] = {}
    for row in value:
        if not isinstance(row, dict) or not isinstance(row.get("name"), str) or not row["name"]:
            raise ValueError(f"target {label} {row!r}: expected a named declaration")
        name = row["name"]
        raw = row.get("path", inferred.get(name))
        if not isinstance(raw, str) or not raw or "\\" in raw:
            raise ValueError(f"target {label} {name!r} path {raw!r}: expected a tracked path")
        path = posixpath.normpath(raw)
        if path not in entries or name in result:
            raise ValueError(f"target {label} {name!r} path {path!r}: expected a unique tracked target")
        result[name] = dict(row) | {"path": path}
    return result


def discovery_enabled(package: Mapping[str, object], flag: str, has_explicit: bool, edition: str, label: str) -> bool:
    """Apply Cargo's edition-dependent target discovery default; example: discovery_enabled(pkg, flag, False, '2024', label)."""
    value = package.get(flag)
    if value is not None and type(value) is not bool:
        raise ValueError(f"manifest {label} {flag} {value!r}: expected a boolean")
    if value is not None:
        return value
    return not has_explicit or edition != "2015"


def expected_targets(source: Mapping[str, object], entries: Mapping[str, str], kind: str, edition: str, label: str) -> dict[str, dict[str, object]]:
    """Derive explicit and auto-discovered target declarations; example: expected_targets(src, files, 'test', '2024', label)."""
    package = source["package"]
    inferred = inferred_targets(entries, kind, package["name"])
    result = explicit_targets(source.get(kind), inferred, entries, label)
    _, flag = TARGET_KINDS[kind]
    if discovery_enabled(package, flag, kind in source, edition, label):
        paths = {target["path"] for target in result.values()}
        for name, path in inferred.items():
            if name not in result and path not in paths:
                result[name] = {"name": name, "path": path}
    return result


def generated_targets(value: object, label: str) -> dict[str, dict[str, object]]:
    """Index normalized generated targets without duplicates; example: generated_targets(rows, label)."""
    if value is None:
        return {}
    if not isinstance(value, list):
        raise ValueError(f"generated targets {label} {value!r}: expected a list")
    result: dict[str, dict[str, object]] = {}
    for row in value:
        if not isinstance(row, dict) or not isinstance(row.get("name"), str) or row["name"] in result:
            raise ValueError(f"generated target {label} {row!r}: expected a unique named target")
        result[row["name"]] = row
    return result


def check_git_manifest_targets(source: bytes, generated: bytes, entries: Mapping[str, str], repository: Path, tree: Mapping[str, str], root: str, label: str) -> None:
    """Compare supported normalized targets with the locked tree; example: check_git_manifest_targets(src, dst, files, repo, tree, root, label)."""
    original, output = tomllib.loads(source.decode("utf-8")), tomllib.loads(generated.decode("utf-8"))
    edition = target_edition(original, repository, tree, root, label)
    package = output.get("package")
    if not isinstance(package, dict) or package.get("edition") != edition:
        raise ValueError(f"manifest {label} generated edition {package!r}: expected locked edition {edition!r}")
    for kind in TARGET_KINDS:
        expected = expected_targets(original, entries, kind, edition, label)
        actual = generated_targets(output.get(kind), label)
        if not same_toml_value(expected, actual):
            names = sorted(set(expected) ^ set(actual))
            if not names:
                names = [name for name in sorted(expected) if not same_toml_value(expected[name], actual[name])]
            name = names[0]
            raise ValueError(f"manifest {label} {kind} target {name!r} {actual.get(name)!r}: expected locked targets {expected.get(name)!r}")

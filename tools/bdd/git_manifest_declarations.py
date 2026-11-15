# SPDX-License-Identifier: GPL-2.0-only
"""Compare selected Git vendor declarations against locked source manifests."""

import tomllib
from typing import Mapping


DEPENDENCY_KEYS = (
    ("dependencies",),
    ("dev-dependencies", "dev_dependencies"),
    ("build-dependencies", "build_dependencies"),
    ("target",),
    ("features",),
)


def declaration(document: Mapping[str, object], names: tuple[str, ...], label: str) -> object:
    """Select one TOML declaration without accepting aliases twice; example: declaration(doc, ('features',), label)."""
    present = [name for name in names if name in document]
    if len(present) > 1:
        raise ValueError(f"manifest {label} declarations {present!r}: expected at most one alias")
    return document[present[0]] if present else None


def same_toml_value(left: object, right: object) -> bool:
    """Compare TOML values without Python's bool/integer coercion; example: same_toml_value(False, 0)."""
    if type(left) is not type(right):
        return False
    if isinstance(left, dict) and isinstance(right, dict):
        return left.keys() == right.keys() and all(same_toml_value(left[key], right[key]) for key in left)
    if isinstance(left, list) and isinstance(right, list):
        return len(left) == len(right) and all(same_toml_value(a, b) for a, b in zip(left, right, strict=True))
    return left == right


def require_matching_declarations(source: Mapping[str, object], generated: Mapping[str, object], label: str) -> None:
    """Reject changed dependencies, targets or features; example: require_matching_declarations(source, generated, label)."""
    for names in DEPENDENCY_KEYS:
        expected = declaration(source, names, label)
        actual = declaration(generated, names, label)
        if not same_toml_value(actual, expected):
            raise ValueError(f"manifest {label} {names[0]} {actual!r}: expected locked source declaration {expected!r}")


def require_no_added_build_script(source: Mapping[str, object], generated: Mapping[str, object], entries: Mapping[str, str], label: str) -> None:
    """Decline unsupported build scripts and reject added ones; example: require_no_added_build_script(source, generated, entries, label)."""
    original = source.get("package")
    output = generated.get("package")
    if not isinstance(original, dict) or not isinstance(output, dict):
        raise ValueError(f"manifest {label} package: expected source and generated package tables")
    if original.get("metabuild") is not None or output.get("metabuild") is not None:
        raise ValueError(f"manifest {label} metabuild: expected no source or generated metabuild")
    build = original.get("build")
    if build is not None and build is not False:
        raise ValueError(f"manifest {label} source build {build!r}: expected no authored build script")
    if build is None and "build.rs" in entries:
        raise ValueError(f"manifest {label} source build.rs: expected no autodetected build script")
    actual = output.get("build")
    if actual is False or (actual is None and "build.rs" not in entries):
        return
    raise ValueError(f"manifest {label} generated build {actual!r}: expected no build script")


def check_git_manifest_declarations(source: bytes, generated: bytes, entries: Mapping[str, str], label: str) -> None:
    """Check selected manifest declarations without claiming full normalization; example: check_git_manifest_declarations(src, dst, entries, label)."""
    original = tomllib.loads(source.decode("utf-8"))
    output = tomllib.loads(generated.decode("utf-8"))
    require_matching_declarations(original, output, label)
    require_no_added_build_script(original, output, entries, label)

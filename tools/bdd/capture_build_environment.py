# SPDX-License-Identifier: GPL-2.0-only
"""Constrain capture build settings without claiming complete provenance."""

from pathlib import Path
from typing import Mapping


CAPTURE_ENVIRONMENT = frozenset({
    "PATH", "HOME", "USER", "LOGNAME", "LANG", "LC_ALL", "TERM", "TMPDIR",
    "RUSTUP_HOME", "DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "XAUTHORITY",
    "VK_ICD_FILENAMES", "VK_DRIVER_FILES", "VK_LAYER_PATH", "VK_INSTANCE_LAYERS",
    "LIBGL_ALWAYS_SOFTWARE", "DRI_PRIME", "EGL_PLATFORM", "WINIT_UNIX_BACKEND",
    "MESA_VK_DEVICE_SELECT", "DBUS_SESSION_BUS_ADDRESS",
    "MESA_LOADER_DRIVER_OVERRIDE", "OBK_BDD_ROOT", "OBK_BDD_REPORTS",
    "OBK_BDD_FEATURE", "OBK_BDD_SCENARIO", "OBK_BDD_AREA",
})


def reject_ancestor_cargo_configs(source: Path) -> None:
    """Reject inherited directory configs; example: reject_ancestor_cargo_configs(source)."""
    for parent in source.parents:
        for name in ("config", "config.toml"):
            path = parent / ".cargo" / name
            if path.exists() or path.is_symlink():
                raise ValueError(
                    f"Cargo configuration {path}: expected no ancestor configuration"
                )


def source_cargo_home(environment: Mapping[str, str]) -> Path:
    """Find the cache source, not its config; example: source_cargo_home({'HOME': '/home/user'})."""
    configured = environment.get("CARGO_HOME")
    if configured:
        return Path(configured).expanduser().absolute()
    return Path(environment.get("HOME", str(Path.home()))) / ".cargo"


def isolate_cargo_home(folder: Path, environment: Mapping[str, str]) -> Path:
    """Expose dependency caches without home configuration; example: isolate_cargo_home(folder, env)."""
    home = folder / "cargo-home"
    home.mkdir()
    source = source_cargo_home(environment)
    registry = home / "registry"
    registry.mkdir()
    index = source / "registry" / "index"
    if index.exists():
        if not index.is_dir():
            raise ValueError(f"Cargo index {index}: expected a directory")
        (registry / "index").symlink_to(index.resolve(), target_is_directory=True)
    git = source / "git"
    if git.exists():
        if not git.is_dir():
            raise ValueError(f"Cargo Git cache {git}: expected a directory")
        (home / "git").symlink_to(git.resolve(), target_is_directory=True)
    return home


def capture_build_environment(folder: Path, environment: Mapping[str, str]) -> dict[str, str]:
    """Set explicit build inputs and retain GUI settings; example: capture_build_environment(folder, env)."""
    result = {key: value for key, value in environment.items() if key in CAPTURE_ENVIRONMENT}
    result.update({
        "CARGO_HOME": str(isolate_cargo_home(folder, environment)),
        "CARGO_TARGET_DIR": str(folder / "target"),
        "CARGO_NET_OFFLINE": "true",
        "CARGO_BUILD_JOBS": "2",
        "CARGO_PROFILE_TEST_DEBUG": "0",
        "CARGO_PROFILE_DEV_DEBUG": "0",
    })
    return result

# SPDX-License-Identifier: GPL-2.0-only
"""A restricted Git environment for dependency and provenance checks."""


def git_environment() -> dict[str, str]:
    """Keep Git configuration out of the export; example: git_environment()."""
    return {
        "PATH": "/usr/bin:/bin",
        "LC_ALL": "C",
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_CONFIG_GLOBAL": "/dev/null",
        "GIT_NO_REPLACE_OBJECTS": "1",
    }


def checkout_git_environment() -> dict[str, str]:
    """Restrict Git overrides for checkout checks; example: checkout_git_environment()."""
    import os

    environment = git_environment()
    if os.name == "nt":
        environment["PATH"] = os.environ.get("PATH", os.defpath)
    environment["GIT_CONFIG_GLOBAL"] = os.devnull
    return environment

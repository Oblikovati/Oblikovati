# SPDX-License-Identifier: GPL-2.0-only
"""Probe an offline native test build in a Linux sandbox; this is not build proof."""

import argparse
import hashlib
import json
import os
import platform
import subprocess
import tomllib
from dataclasses import dataclass
from pathlib import Path
from functools import partial
from typing import Callable, Mapping

from committed_source_check import committed_reference, verify_source_inventory
from git_vendor_check import verify_git_vendor
from registry_vendor_check import (
    read_observed_file,
    verify_registry_vendor,
    verify_vendor_config,
)
from runtime_snapshot import open_flags, snapshot_runtime
from sealed_execution import file_identity, require_disjoint


@dataclass(frozen=True)
class BuildProbe:
    """Locations for one probe; example: BuildProbe(source, vendor, tools, config, target, logs, bwrap, system)."""

    source: Path
    vendor: Path
    toolchain: Path
    config: Path
    target: Path
    logs: Path
    bwrap: Path
    system: Path


ProbeProcess = Callable[[list[str], Path, Path], int]
Snapshot = Callable[[Path], Mapping[str, object]]
SourceCheck = Callable[[Mapping[str, object]], None]
VendorCheck = Callable[[Mapping[str, Mapping[str, object]]], dict[str, object]]


def require_roots(probe: BuildProbe) -> None:
    """Require disjoint real input and output directories; example: require_roots(probe)."""
    roots = (
        probe.source,
        probe.vendor,
        probe.toolchain,
        probe.config,
        probe.target,
        probe.logs,
        probe.system,
    )
    for root in roots:
        if not root.is_absolute() or root.is_symlink() or not root.is_dir():
            raise ValueError(f"build root {root}: expected an absolute real directory")
    for index, left in enumerate(roots):
        for right in roots[index + 1 :]:
            require_disjoint(left.resolve(), right.resolve())
    if any(probe.target.iterdir()) or any(probe.logs.iterdir()):
        raise ValueError(
            f"target {probe.target} and logs {probe.logs}: expected empty directories"
        )


def require_probe(probe: BuildProbe) -> None:
    """Refuse unsupported hosts and inputs; example: require_probe(probe)."""
    if platform.system() != "Linux" or platform.machine() != "x86_64":
        raise ValueError(
            f"platform {platform.system()}/{platform.machine()}: expected Linux x86_64"
        )
    require_roots(probe)
    if (
        not probe.bwrap.is_absolute()
        or probe.bwrap.is_symlink()
        or not probe.bwrap.is_file()
    ):
        raise ValueError(
            f"sandbox {probe.bwrap}: expected an absolute regular executable"
        )
    if not (probe.config / "config.toml").is_file():
        raise ValueError(f"configuration {probe.config}: expected config.toml")


def build_environment() -> dict[str, str]:
    """Fix the environment inside the sandbox; example: build_environment()."""
    return {
        "PATH": "/toolchain/bin:/usr/bin:/bin",
        "HOME": "/tmp",
        "CARGO_HOME": "/tmp/cargo",
        "CARGO_TARGET_DIR": "/build",
        "CARGO_NET_OFFLINE": "true",
        "CARGO_INCREMENTAL": "0",
        "CARGO_PROFILE_TEST_DEBUG": "0",
        "CARGO_PROFILE_DEV_DEBUG": "0",
        "CC": "/usr/bin/gcc",
        "CXX": "/usr/bin/g++",
        "CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER": "/usr/bin/gcc",
        "LANG": "C.UTF-8",
    }


SANDBOX_PREFIX = (
    "--unshare-all",
    "--unshare-user",
    "--disable-userns",
    "--die-with-parent",
    "--new-session",
    "--clearenv",
    "--symlink",
    "usr/bin",
    "/bin",
    "--symlink",
    "usr/lib",
    "/lib",
    "--symlink",
    "usr/lib64",
    "/lib64",
)
BUILD_PROGRAM = (
    "--chdir",
    "/source",
    "/toolchain/bin/cargo",
    "--config",
    "/config/config.toml",
    "test",
    "--locked",
    "--offline",
    "-p",
    "oblikovati-head",
    "--lib",
    "--no-run",
    "--message-format=json-render-diagnostics",
)


def sandbox_build_command(probe: BuildProbe) -> list[str]:
    """Build the fixed offline command; example: sandbox_build_command(probe)."""
    command = [str(probe.bwrap), *SANDBOX_PREFIX]
    for host, guest in (
        (probe.system, "/usr"),
        (probe.source, "/source"),
        (probe.vendor, "/vendor"),
        (probe.toolchain, "/toolchain"),
        (probe.config, "/config"),
    ):
        command.extend(["--ro-bind", str(host), guest])
    command.extend(
        [
            "--bind",
            str(probe.target),
            "/build",
            "--tmpfs",
            "/tmp",
            "--dev",
            "/dev",
            "--proc",
            "/proc",
        ]
    )
    for name, value in build_environment().items():
        command.extend(["--setenv", name, value])
    return command + list(BUILD_PROGRAM)


def sandbox_desktop_command(probe: BuildProbe) -> list[str]:
    """Build the desktop in the same sandbox; example: sandbox_desktop_command(probe)."""
    command = sandbox_build_command(probe)
    return command[:-len(BUILD_PROGRAM)] + [
        "--chdir", "/source", "/toolchain/bin/cargo", "--config", "/config/config.toml",
        "build", "--locked", "--offline", "-p", "oblikovati-head", "--bin", "oblikovati",
        "--message-format=json-render-diagnostics",
    ]


def desktop_path(messages: Path, target: Path) -> Path:
    """Find one production desktop artifact; example: desktop_path(messages, target)."""
    records = (json.loads(line) for line in messages.read_text(encoding="utf-8").splitlines())
    matches = []
    for record in records:
        if (record.get("reason") != "compiler-artifact"
                or record.get("target", {}).get("name") != "oblikovati"
                or record.get("profile", {}).get("test")
                or not record.get("executable")):
            continue
        guest = Path(record["executable"])
        if not guest.is_absolute() or not guest.is_relative_to("/build") or ".." in guest.parts:
            raise ValueError(f"desktop {guest}: expected an absolute path inside /build")
        matches.append(target / guest.relative_to("/build"))
    return require_runner(matches, target)


def run_process(command: list[str], messages: Path, log: Path) -> int:
    """Run the sandbox and retain its output; example: run_process(command, messages, log)."""
    with messages.open("xb") as stdout, log.open("xb") as stderr:
        return subprocess.run(
            command,
            stdin=subprocess.DEVNULL,
            stdout=stdout,
            stderr=stderr,
            env={"PATH": "/usr/bin:/bin", "LC_ALL": "C"},
        ).returncode


def artifact_candidate(record: dict[str, object], target: Path) -> Path | None:
    """Read one matching Cargo artifact; example: artifact_candidate(record, target)."""
    if (
        record.get("reason") != "compiler-artifact"
        or record.get("target", {}).get("name") != "oblikovati_head"
        or not record.get("profile", {}).get("test")
        or not record.get("executable")
    ):
        return None
    guest = Path(record["executable"])
    if (
        not guest.is_absolute()
        or not guest.is_relative_to("/build")
        or ".." in guest.parts
    ):
        raise ValueError(f"runner {guest}: expected an absolute path inside /build")
    return target / guest.relative_to("/build")


def require_runner(matches: list[Path], target: Path) -> Path:
    """Require a single contained executable; example: require_runner(matches, target)."""
    if (
        len(matches) != 1
        or matches[0].is_symlink()
        or not matches[0].is_file()
        or not os.access(matches[0], os.X_OK)
    ):
        raise ValueError(
            f"runner artifacts {matches!r}: expected one regular head test executable"
        )
    result = matches[0]
    if not result.resolve().is_relative_to(target.resolve()):
        raise ValueError(f"runner {result}: expected a file within {target}")
    for parent in result.relative_to(target).parents:
        if (target / parent).is_symlink():
            raise ValueError(f"runner {result}: expected no symlink parents")
    return result


def runner_path(messages: Path, target: Path) -> Path:
    """Find the unique native head test artifact; example: runner_path(messages, target)."""
    records = (
        json.loads(line) for line in messages.read_text(encoding="utf-8").splitlines()
    )
    matches = [
        path
        for record in records
        if (path := artifact_candidate(record, target)) is not None
    ]
    return require_runner(matches, target)


def snapshot_inputs(
    probe: BuildProbe, snapshot: Snapshot
) -> dict[str, Mapping[str, object]]:
    """Inventory mounted build inputs; example: snapshot_inputs(probe, snapshot_runtime)."""
    return {
        name: snapshot(getattr(probe, name))
        for name in ("source", "vendor", "toolchain", "config", "system")
    }


def probe_result(
    runner: Path,
    before: Mapping[str, Mapping[str, object]],
    sandbox: Mapping[str, object],
) -> dict[str, object]:
    """Describe observed inputs without claiming proof; example: probe_result(runner, before, sandbox)."""
    return {
        "runner": str(runner),
        "runner_sha256": file_identity(runner)["sha256"],
        "input_inventory_sha256": {
            name: hashlib.sha256(json.dumps(value, sort_keys=True).encode()).hexdigest()
            for name, value in before.items()
        },
        "sandbox_sha256": sandbox["sha256"],
        "reusable": False,
        "limitations": "System and toolchain origin, kernel, devices and concurrent host writes are not bounded.",
    }


def run_build_probe(
    probe: BuildProbe,
    run: ProbeProcess = run_process,
    snapshot: Snapshot = snapshot_runtime,
    check_source: SourceCheck | None = None,
    check_vendor: VendorCheck | None = None,
) -> dict[str, object]:
    """Run an observed build without claiming reuse; example: run_build_probe(probe)."""
    require_probe(probe)
    before, sandbox = snapshot_inputs(probe, snapshot), file_identity(probe.bwrap)
    if check_source is not None:
        check_source(before["source"])
    vendor_report = check_vendor(before) if check_vendor is not None else None
    messages, log = probe.logs / "cargo.jsonl", probe.logs / "cargo.log"
    code = run(sandbox_build_command(probe), messages, log)
    if code != 0:
        raise ValueError(f"build exit {code}: expected success; inspect {log}")
    desktop_messages, desktop_log = probe.logs / "desktop.jsonl", probe.logs / "desktop.log"
    desktop_code = run(sandbox_desktop_command(probe), desktop_messages, desktop_log)
    if desktop_code != 0:
        raise ValueError(f"desktop build exit {desktop_code}: expected success; inspect {desktop_log}")
    if (
        snapshot_inputs(probe, snapshot) != before
        or file_identity(probe.bwrap) != sandbox
    ):
        raise ValueError(
            "build inputs changed: expected unchanged mounted inputs and sandbox"
        )
    result = probe_result(runner_path(messages, probe.target), before, sandbox)
    desktop = desktop_path(desktop_messages, probe.target)
    result["desktop"] = str(desktop)
    result["desktop_sha256"] = file_identity(desktop)["sha256"]
    if vendor_report is not None:
        key = "vendor_check" if "git" in vendor_report else "registry_check"
        result[key] = vendor_report
    return result


def source_config_bytes(
    probe: BuildProbe, observed: Mapping[str, object]
) -> bytes | None:
    """Read the project's Cargo config from the observed source; example: source_config_bytes(probe, observed)."""
    if ".cargo/config" in observed:
        raise ValueError("source .cargo/config: expected no legacy Cargo configuration")
    if ".cargo/config.toml" not in observed:
        return None
    root = os.open(probe.source, open_flags(True))
    try:
        directory = os.open(".cargo", open_flags(True), dir_fd=root)
        try:
            return read_observed_file(
                directory, "config.toml", observed[".cargo/config.toml"]
            )
        finally:
            os.close(directory)
    finally:
        os.close(root)


def check_source_cargo_config(
    probe: BuildProbe, observed: Mapping[str, object]
) -> None:
    """Refuse project settings that redirect dependencies; example: check_source_cargo_config(probe, observed)."""
    content = source_config_bytes(probe, observed)
    if content is None:
        return
    config = tomllib.loads(content.decode("utf-8"))
    denied = set(config) & {
        "source",
        "registry",
        "registries",
        "patch",
        "replace",
        "paths",
    }
    if denied:
        raise ValueError(
            f"source Cargo configuration {sorted(denied)!r}: expected no source overrides"
        )


def check_registry_inputs(
    probe: BuildProbe, archives: Path, before: Mapping[str, Mapping[str, object]]
) -> dict[str, object]:
    """Check vendored registry packages against the observed lockfile; example: check_registry_inputs(probe, archives, before)."""
    parent = os.open(probe.source, open_flags(True))
    try:
        lock = read_observed_file(
            parent, "Cargo.lock", before["source"].get("Cargo.lock")
        )
    finally:
        os.close(parent)
    config_dir = os.open(probe.config, open_flags(True))
    try:
        config = read_observed_file(
            config_dir, "config.toml", before["config"].get("config.toml")
        )
    finally:
        os.close(config_dir)
    check_source_cargo_config(probe, before["source"])
    verify_vendor_config(lock, config)
    return verify_registry_vendor(lock, probe.vendor, archives, before["vendor"])


def check_git_inputs(
    probe: BuildProbe,
    repositories: Mapping[str, Path],
    before: Mapping[str, Mapping[str, object]],
) -> dict[str, object]:
    """Check Git vendor blobs against locked commits; example: check_git_inputs(probe, repositories, before)."""
    parent = os.open(probe.source, open_flags(True))
    try:
        lock = read_observed_file(
            parent, "Cargo.lock", before["source"].get("Cargo.lock")
        )
    finally:
        os.close(parent)
    return verify_git_vendor(lock, probe.vendor, before["vendor"], repositories)


def check_all_vendor_inputs(
    probe: BuildProbe,
    archives: Path,
    repositories: Mapping[str, Path],
    before: Mapping[str, Mapping[str, object]],
) -> dict[str, object]:
    """Check registry and Git inputs before a build; example: check_all_vendor_inputs(probe, archives, repositories, before)."""
    return {
        "registry": check_registry_inputs(probe, archives, before),
        "git": check_git_inputs(probe, repositories, before),
    }


def require_archives(probe: BuildProbe, archives: Path) -> None:
    """Keep the archive cache separate from build output; example: require_archives(probe, archives)."""
    if not archives.is_absolute() or archives.is_symlink() or not archives.is_dir():
        raise ValueError(
            f"archive cache {archives}: expected an absolute real directory"
        )
    for output in (probe.target, probe.logs):
        require_disjoint(archives.resolve(), output.resolve())


def require_git_repositories(
    probe: BuildProbe, repositories: Mapping[str, Path]
) -> None:
    """Keep Git object stores outside build output; example: require_git_repositories(probe, repositories)."""
    for source, repository in repositories.items():
        if (
            not repository.is_absolute()
            or repository.is_symlink()
            or not repository.is_dir()
        ):
            raise ValueError(
                f"Git repository {source!r}: expected an absolute real directory, got {repository}"
            )
        for output in (probe.target, probe.logs):
            require_disjoint(repository.resolve(), output.resolve())


def run_committed_build_probe(
    probe: BuildProbe,
    repository: Path,
    revision: str,
    run: ProbeProcess = run_process,
    snapshot: Snapshot = snapshot_runtime,
    archives: Path | None = None,
    git_repositories: Mapping[str, Path] | None = None,
) -> dict[str, object]:
    """Check the mounted source against a commit before a probe; example: run_committed_build_probe(probe, repo, sha)."""
    require_probe(probe)
    if archives is not None:
        require_archives(probe, archives)
    if git_repositories is not None and archives is None:
        raise ValueError(
            "Git repositories: expected registry archives for a combined vendor check"
        )
    if git_repositories is not None:
        require_git_repositories(probe, git_repositories)
    reference = committed_reference(repository, revision)
    check = partial(verify_source_inventory, reference=reference)
    vendor_check = None
    if archives is not None:
        vendor_check = partial(check_registry_inputs, probe, archives)
    if archives is not None and git_repositories is not None:
        vendor_check = partial(
            check_all_vendor_inputs, probe, archives, git_repositories
        )
    result = run_build_probe(probe, run, snapshot, check, vendor_check)
    result["source_commit"] = reference.commit
    return result


def main() -> None:
    """Run an experimental build probe; run `python3 sandbox_build_probe.py --help`."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("source", "vendor", "toolchain", "config", "target", "logs", "bwrap", "system"):
        parser.add_argument(name, type=Path)
    parser.add_argument("--repository", type=Path)
    parser.add_argument("--revision")
    parser.add_argument("--registry-archives", type=Path)
    parser.add_argument(
        "--git-repo", nargs=2, action="append", metavar=("SOURCE", "REPO")
    )
    args = parser.parse_args()
    repository, revision, archives = (
        args.repository,
        args.revision,
        args.registry_archives,
    )
    git_entries = args.git_repo
    del args.repository, args.revision, args.registry_archives, args.git_repo
    if (repository is None) != (revision is None):
        parser.error("--repository and --revision must be supplied together")
    if archives is not None and repository is None:
        parser.error("--registry-archives requires --repository and --revision")
    if git_entries is not None and archives is None:
        parser.error("--git-repo requires --registry-archives")
    git_repositories = (
        None
        if git_entries is None
        else {source: Path(path) for source, path in git_entries}
    )
    if git_entries is not None and len(git_repositories) != len(git_entries):
        parser.error("--git-repo: expected each source exactly once")
    probe = BuildProbe(**vars(args))
    result = (
        run_committed_build_probe(
            probe,
            repository,
            revision,
            archives=archives,
            git_repositories=git_repositories,
        )
        if repository is not None and revision is not None
        else run_build_probe(probe)
    )
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main()

# SPDX-License-Identifier: GPL-2.0-only
"""Probe an offline native test build in a Linux sandbox; this is not build proof."""

import argparse
import hashlib
import json
import os
import platform
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Mapping

from runtime_snapshot import snapshot_runtime
from sealed_execution import file_identity, require_disjoint


@dataclass(frozen=True)
class BuildProbe:
    """Locations for one probe; example: BuildProbe(source, vendor, tools, config, target, logs, bwrap)."""

    source: Path
    vendor: Path
    toolchain: Path
    config: Path
    target: Path
    logs: Path
    bwrap: Path


ProbeProcess = Callable[[list[str], Path, Path], int]
Snapshot = Callable[[Path], Mapping[str, object]]


def require_roots(probe: BuildProbe) -> None:
    """Require disjoint real input and output directories; example: require_roots(probe)."""
    roots = (
        probe.source,
        probe.vendor,
        probe.toolchain,
        probe.config,
        probe.target,
        probe.logs,
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
    "--die-with-parent",
    "--new-session",
    "--clearenv",
    "--ro-bind",
    "/usr",
    "/usr",
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
        for name in ("source", "vendor", "toolchain", "config")
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
        "limitations": "Host system libraries, kernel, devices and concurrent host writes are not bounded.",
    }


def run_build_probe(
    probe: BuildProbe,
    run: ProbeProcess = run_process,
    snapshot: Snapshot = snapshot_runtime,
) -> dict[str, object]:
    """Run an observed build without claiming reuse; example: run_build_probe(probe)."""
    require_probe(probe)
    before, sandbox = snapshot_inputs(probe, snapshot), file_identity(probe.bwrap)
    messages, log = probe.logs / "cargo.jsonl", probe.logs / "cargo.log"
    code = run(sandbox_build_command(probe), messages, log)
    if code != 0:
        raise ValueError(f"build exit {code}: expected success; inspect {log}")
    if (
        snapshot_inputs(probe, snapshot) != before
        or file_identity(probe.bwrap) != sandbox
    ):
        raise ValueError(
            "build inputs changed: expected unchanged mounted inputs and sandbox"
        )
    return probe_result(runner_path(messages, probe.target), before, sandbox)


def main() -> None:
    """Run an experimental build probe; run `python3 sandbox_build_probe.py --help`."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("source", "vendor", "toolchain", "config", "target", "logs", "bwrap"):
        parser.add_argument(name, type=Path)
    args = parser.parse_args()
    print(json.dumps(run_build_probe(BuildProbe(**vars(args))), sort_keys=True))


if __name__ == "__main__":
    main()

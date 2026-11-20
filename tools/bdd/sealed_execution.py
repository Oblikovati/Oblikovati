# SPDX-License-Identifier: GPL-2.0-only
"""Run a native scenario with a bounded, read-only filesystem on Linux."""

import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Mapping

from runtime_snapshot import open_flags, snapshot_file, snapshot_runtime


@dataclass(frozen=True)
class SealedRequest:
    """Inputs for one bounded run; example: SealedRequest(runtime, binary, source, reports, log, bwrap, area, feature, scenario)."""

    runtime: Path
    binary: Path
    checkout: Path
    reports: Path
    log: Path
    bwrap: Path
    area: str
    feature: str
    scenario: str
    desktop: Path | None = None


@dataclass(frozen=True)
class SealedInputs:
    """Inputs observed around one run; example: SealedInputs(source, checkout, runtime, runner, sandbox)."""

    source: str
    checkout: Mapping[str, object]
    runtime: Mapping[str, object]
    runner: Mapping[str, object]
    sandbox: Mapping[str, object]
    desktop: Mapping[str, object] | None = None


SealedProcess = Callable[[list[str], Path], int]
SourceVerifier = Callable[[Path], str]


def file_identity(path: Path) -> dict[str, object]:
    """Read one input without following symlinks; example: file_identity(Path('/bin/program'))."""
    parent = os.open(path.parent, open_flags(True))
    try:
        before = os.stat(path.name, dir_fd=parent, follow_symlinks=False)
        return snapshot_file(parent, path.name, str(path), before)
    finally:
        os.close(parent)


def require_slots(runtime: Path) -> None:
    """Require explicit bind targets in the runtime; example: require_slots(runtime)."""
    for name in ("scenario", "reports", "tmp", "dev"):
        path = runtime / name
        if path.is_symlink() or not path.is_dir():
            raise ValueError(f"runtime slot {path}: expected a real directory")
    runner = runtime / "runner"
    if runner.is_symlink() or not runner.is_file():
        raise ValueError(f"runtime slot {runner}: expected a regular file")


def require_desktop_slot(runtime: Path) -> None:
    """Require a mount point for the pinned desktop; example: require_desktop_slot(runtime)."""
    path = runtime / "desktop"
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"runtime slot {path}: expected a regular file for the desktop mount")


def require_request(request: SealedRequest) -> None:
    """Reject invalid filters or occupied output; example: require_request(request)."""
    if sys.platform != "linux":
        raise ValueError(f"platform {sys.platform!r}: expected Linux for sealed runs")
    require_absolute_paths(request)
    require_filter(request)
    require_empty_output(request)
    require_disjoint_paths(request)
    require_slots(request.runtime)
    if request.desktop is not None:
        require_desktop_slot(request.runtime)
    require_docking_tools(request)


def require_docking_tools(request: SealedRequest) -> None:
    """Check the restart runtime before building; example: require_docking_tools(request)."""
    if request.area != "application" or request.feature != "docking-system":
        return
    tools = ("python3", "openbox", "import", "compare", "xvfb-run", "Xvfb", "xauth")
    for name in tools:
        path = request.runtime / "usr" / "bin" / name
        if not path.is_file() or not path.resolve().is_relative_to(request.runtime.resolve()):
            raise ValueError(f"docking runtime tool {path}: expected a file within the runtime")
        if not os.access(path, os.X_OK):
            raise ValueError(f"docking runtime tool {path}: expected an executable file")


def require_absolute_paths(request: SealedRequest) -> None:
    """Require explicit host locations for every mount; example: require_absolute_paths(request)."""
    paths = (
        request.runtime,
        request.binary,
        request.checkout,
        request.reports,
        request.log,
        request.bwrap,
    )
    if request.desktop is not None:
        paths += (request.desktop,)
    for path in paths:
        if not path.is_absolute():
            raise ValueError(f"sealed path {path}: expected an absolute path")


def require_filter(request: SealedRequest) -> None:
    """Check the selected native scenario; example: require_filter(request)."""
    if request.area not in ("application", "sketch", "solid-model"):
        raise ValueError(
            f"area {request.area!r}: expected application, sketch, or solid-model"
        )
    if not re.fullmatch(r"[a-z][a-z-]*", request.feature):
        raise ValueError(
            f"feature {request.feature!r}: expected a lowercase feature folder"
        )
    if not request.scenario or "\0" in request.scenario:
        raise ValueError(
            f"scenario {request.scenario!r}: expected a nonempty scenario name"
        )


def require_empty_output(request: SealedRequest) -> None:
    """Refuse an occupied destination; example: require_empty_output(request)."""
    if (
        request.reports.is_symlink()
        or not request.reports.is_dir()
        or any(request.reports.iterdir())
    ):
        raise ValueError(
            f"reports {request.reports}: expected an empty report directory"
        )
    if request.log.exists() or request.log.is_symlink():
        raise ValueError(
            f"log {request.log}: expected a new file outside the report directory"
        )


def require_disjoint_paths(request: SealedRequest) -> None:
    """Keep mounted inputs and output separate; example: require_disjoint_paths(request)."""
    roots = mounted_roots(request)
    for index, left in enumerate(roots):
        for right in roots[index + 1 :]:
            require_disjoint(left, right)

    log = request.log.resolve()
    if any(log.is_relative_to(root) for root in roots):
        raise ValueError(
            f"log {request.log}: expected a file outside all mounted roots"
        )


def mounted_roots(request: SealedRequest) -> list[Path]:
    """Resolve the three mounted roots; example: mounted_roots(request)."""
    return [
        request.runtime.resolve(),
        request.checkout.resolve(),
        request.reports.resolve(),
    ]


def require_disjoint(left: Path, right: Path) -> None:
    """Reject overlapping mount roots; example: require_disjoint(left, right)."""
    if left == right or left.is_relative_to(right) or right.is_relative_to(left):
        raise ValueError(
            f"sealed paths {left} and {right}: expected disjoint directories"
        )


def scenario_environment(request: SealedRequest) -> list[tuple[str, str]]:
    """Return the complete sandbox environment; example: scenario_environment(request)."""
    result = [
        ("PATH", "/usr/bin:/bin"),
        ("HOME", "/tmp"),
        ("LANG", "C.UTF-8"),
        ("XDG_RUNTIME_DIR", "/tmp"),
        ("VK_ICD_FILENAMES", "/usr/share/vulkan/icd.d/lvp_icd.json"),
        ("GIT_OPTIONAL_LOCKS", "0"),
        ("GIT_CONFIG_NOSYSTEM", "1"),
        ("GIT_CONFIG_GLOBAL", "/dev/null"),
        ("OBK_BDD_ROOT", "/scenario"),
        ("OBK_BDD_REPORTS", "/reports"),
        ("OBK_BDD_FEATURE", request.feature),
        ("OBK_BDD_SCENARIO", request.scenario),
    ]
    if request.desktop is not None:
        result.append(("OBK_BDD_DESKTOP_BINARY", "/desktop"))
    if request.area == "solid-model":
        result.append(("OBK_BDD_AREA", "solid-model"))
    return result


def sandbox_command(request: SealedRequest) -> list[str]:
    """Construct the fixed isolated command; example: sandbox_command(request)."""
    command = sandbox_mounts(request)
    for name, value in scenario_environment(request):
        command.extend(["--setenv", name, value])
    command.extend(sandbox_program(request))
    return command


def sandbox_mounts(request: SealedRequest) -> list[str]:
    """Fix the allowed filesystem mounts; example: sandbox_mounts(request)."""
    command = [
        str(request.bwrap),
        "--unshare-all",
        "--unshare-user",
        "--disable-userns",
        "--clearenv",
        "--die-with-parent",
    ]
    command.extend(["--ro-bind", str(request.runtime), "/"])
    command.extend(["--dev", "/dev", "--tmpfs", "/tmp"])
    command.extend(["--ro-bind", str(request.binary), "/runner"])
    if request.desktop is not None:
        command.extend(["--ro-bind", str(request.desktop), "/desktop"])
    command.extend(["--ro-bind", str(request.checkout), "/scenario"])
    command.extend(["--bind", str(request.reports), "/reports"])
    return command


def sandbox_program(request: SealedRequest) -> list[str]:
    """Choose the native runner inside Xvfb; example: sandbox_program(request)."""
    runner = (
        "sketch_scenarios_use_the_native_window"
        if request.area == "sketch"
        else "application_scenarios_use_the_native_window"
    )
    if request.area == "application" and request.feature == "docking-system":
        return docking_program(runner)
    return ["--chdir", "/tmp", "--", "/usr/bin/xvfb-run", "-a", "-s",
            "-screen 0 1920x1080x24 -noreset", "/runner", runner, "--nocapture"]


def docking_program(runner: str) -> list[str]:
    """Run the docking test with a window manager; example: docking_program('test')."""
    launch = 'openbox >/tmp/openbox.log 2>&1 & wm=$!; trap "kill $wm 2>/dev/null || true" EXIT; "$@"'
    return ["--chdir", "/tmp", "--", "/usr/bin/xvfb-run", "-a", "-s",
            "-screen 0 1440x900x24 -noreset", "/bin/sh", "-c", launch,
            "docking-run", "/runner", runner, "--nocapture"]


def run_process(command: list[str], log: Path) -> int:
    """Execute one command with an exclusively created log; example: run_process(command, log)."""
    with log.open("x", encoding="utf-8") as output:
        return subprocess.run(
            command, stdout=output, stderr=subprocess.STDOUT, check=False, env={}
        ).returncode


def run_sealed(
    request: SealedRequest, process: SealedProcess, verify: SourceVerifier
) -> dict[str, object]:
    """Run and inventory one scenario without authorizing reuse; example: run_sealed(request, run_process, verify)."""
    require_request(request)
    before = collect_inputs(request, verify)
    status = process(sandbox_command(request), request.log)
    if status != 0:
        raise ValueError(f"sealed runner exit {status!r}: expected zero")
    require_same_source(before.source, verify(request.checkout))
    require_stable_inputs(request, before)
    return execution_observation(request, before)


def collect_inputs(request: SealedRequest, verify: SourceVerifier) -> SealedInputs:
    """Inventory the inputs before launching; example: collect_inputs(request, verify)."""
    return SealedInputs(
        verify(request.checkout),
        snapshot_runtime(request.checkout),
        snapshot_runtime(request.runtime),
        file_identity(request.binary),
        file_identity(request.bwrap),
        file_identity(request.desktop) if request.desktop is not None else None,
    )


def require_same_source(before: str, after: str) -> None:
    """Refuse a changed scenario checkout; example: require_same_source(before, after)."""
    if after != before:
        raise ValueError(f"source identity {after!r}: expected unchanged {before!r}")


def execution_observation(
    request: SealedRequest, inputs: SealedInputs
) -> dict[str, object]:
    """Record checked inputs without claiming reuse; example: execution_observation(request, inputs)."""
    return {
        "runtime_coverage": "unbounded",
        "source_identity": inputs.source,
        "source_namespace": inputs.checkout,
        "reusable": False,
        "runtime_inputs": inputs.runtime,
        "runner_input": inputs.runner,
        "desktop_input": inputs.desktop,
        "sandbox_input": inputs.sandbox,
        "area": request.area,
        "feature": request.feature,
        "scenario": request.scenario,
    }


def require_source_namespace(checkout: Path, before: object) -> None:
    """Refuse changes to any mounted scenario file, including Git metadata."""
    if snapshot_runtime(checkout) != before:
        raise ValueError("source namespace: expected unchanged mounted checkout inputs")


def require_stable_inputs(request: SealedRequest, before: SealedInputs) -> None:
    """Refuse changes while the scenario ran; example: require_stable_inputs(request, before)."""
    require_source_namespace(request.checkout, before.checkout)
    if snapshot_runtime(request.runtime) != before.runtime:
        raise ValueError(
            "runtime inputs: expected unchanged mounted files during the run"
        )
    if file_identity(request.binary) != before.runner:
        raise ValueError("runner input: expected the same executable during the run")
    if request.desktop is not None and file_identity(request.desktop) != before.desktop:
        raise ValueError("desktop input: expected the same executable during the run")
    if file_identity(request.bwrap) != before.sandbox:
        raise ValueError("sandbox input: expected the same executable during the run")

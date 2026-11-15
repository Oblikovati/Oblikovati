# SPDX-License-Identifier: GPL-2.0-only
"""Run and validate one native scenario before publishing a reviewable receipt."""

import gzip
import hashlib
import json
import os
import shutil
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Mapping

from behavior_evidence import check_capture_receipt, example_capture_name


@dataclass(frozen=True)
class CaptureRequest:
    """Inputs fixed before a run; for example, CaptureRequest(root, checkout, dest, row, source, commit, 'Linux', 'Vulkan', None)."""

    root: Path
    checkout: Path
    destination: Path
    row: Mapping[str, object]
    source_revision: str
    application_commit: str
    operating_system: str
    renderer: str
    capture_name: str | None


NativeRun = Callable[[Path, dict[str, str]], int]
ApplicationState = Callable[[Path], tuple[str, bool]]


def scenario_environment(request: CaptureRequest, reports: Path) -> dict[str, str]:
    """Set only the selected run's BDD filters; example: scenario_environment(request, reports)."""
    environment = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith("OBK_BDD_")
    }
    row = request.row
    area = scenario_area(row)
    environment.update(
        OBK_BDD_ROOT=str(request.checkout.resolve()), OBK_BDD_REPORTS=str(reports)
    )
    environment.update(
        OBK_BDD_FEATURE=str(row["feature"]), OBK_BDD_SCENARIO=str(row["scenario"])
    )
    if area == "solid-model":
        environment["OBK_BDD_AREA"] = area
    return environment


def scenario_area(row: Mapping[str, object]) -> str:
    """Refuse areas without a native runner; example: scenario_area({'area': 'sketch'})."""
    area = str(row["area"])
    if area not in ("application", "sketch", "solid-model"):
        raise ValueError(
            f"scenario area {area!r}: expected application, sketch, or solid-model"
        )
    return area


def selected_capture(reports: Path, name: str | None) -> Path:
    """Select a reported image without guessing between captures; example: selected_capture(reports, None)."""
    images = sorted(
        path
        for path in reports.glob("*.png")
        if path.is_file() and not path.is_symlink()
    )
    if name is not None:
        if Path(name).name != name:
            raise ValueError(
                f"capture {name!r}: expected a PNG filename without directories"
            )
        images = [path for path in images if path.name == name]
    if len(images) != 1:
        raise ValueError(
            f"capture {name!r}: expected exactly one selected PNG, got {[p.name for p in images]!r}"
        )
    return images[0]


def selected_captures(request: CaptureRequest, reports: Path) -> list[Path]:
    """Select each outline case by source path and line; for example, selected_captures(request, reports)."""
    if not request.row.get("outline"):
        return [selected_capture(reports, request.capture_name)]
    if request.capture_name is not None:
        raise ValueError("outline capture override: expected one capture per example")
    examples = request.row.get("examples")
    if not isinstance(examples, list) or not examples:
        raise ValueError(f"outline examples {examples!r}: expected a nonempty list")
    names = [example_capture_name(request.row, example) for example in examples]
    if len(set(names)) != len(names):
        raise ValueError(
            f"outline captures {names!r}: expected distinct example identities"
        )
    return [selected_capture(reports, name) for name in names]


def artifact(root: Path, path: Path) -> dict[str, str]:
    """Record an inside-repository artifact and its digest; example: artifact(root, receipt)."""
    resolved = path.resolve()
    if not resolved.is_relative_to(root.resolve()) or not resolved.is_file():
        raise ValueError(f"artifact {path}: expected a file inside {root}")
    return {
        "path": resolved.relative_to(root.resolve()).as_posix(),
        "sha256": hashlib.sha256(resolved.read_bytes()).hexdigest(),
    }


def require_application_state(
    request: CaptureRequest, read_state: ApplicationState
) -> None:
    """Refuse changed or dirty application inputs; example: require_application_state(request, application_state)."""
    actual = read_state(request.root)
    expected = (request.application_commit, True)
    if actual != expected:
        raise ValueError(
            f"application state {actual!r}: expected unchanged clean inputs {expected!r}"
        )


def receipt_document(
    request: CaptureRequest, folder: Path, images: list[Path]
) -> dict[str, object]:
    """Make a verifier-compatible receipt; example: receipt_document(request, folder, [image])."""
    identity = receipt_identity(request)
    result = identity | {
        "cucumber": artifact(request.root, folder / "cucumber.json"),
        "source_report": artifact(request.root, folder / "source-revision.json"),
    }
    if request.row.get("outline"):
        result["captures"] = [
            {"line": example["line"], "artifact": artifact(request.root, image)}
            for example, image in zip(request.row["examples"], images, strict=True)
        ]
    else:
        result["capture"] = artifact(request.root, images[0])
    return result


def receipt_identity(request: CaptureRequest) -> dict[str, object]:
    """Record the fixed run inputs; example: receipt_identity(request)."""
    row = request.row
    runner = (
        "sketch_scenarios_use_the_native_window"
        if row["area"] == "sketch"
        else "application_scenarios_use_the_native_window"
    )
    return {
        "kind": "native-acceptance",
        "scenario_id": row["id"],
        "scenario_revision": request.source_revision,
        "source_sha256": row["source_sha256"],
        "application_commit": request.application_commit,
        "api_revision": row["api_revision"],
        "kernel_revision": row["kernel_revision"],
        "runner": runner,
        "exit_code": 0,
        "environment": {"os": request.operating_system, "renderer": request.renderer},
    }


def stage_artifacts(reports: Path, stage: Path, captures: list[Path]) -> None:
    """Copy the required run artifacts into staging; example: stage_artifacts(reports, stage, [image])."""
    for name in ("cucumber.json", "source-revision.json", "junit.xml", *(path.name for path in captures)):
        source = reports / name
        if source.is_symlink() or not source.is_file():
            raise ValueError(f"run artifact {source}: expected a regular file")
        shutil.copyfile(source, stage / name)
    log = reports / "native-test.log"
    if log.is_symlink() or not log.is_file():
        raise ValueError(f"run log {log}: expected a regular file")
    with (stage / "native-test.log.gz").open("wb") as destination:
        with gzip.GzipFile(
            filename="", mode="wb", fileobj=destination, mtime=0
        ) as output:
            output.write(log.read_bytes())


def write_receipt(request: CaptureRequest, folder: Path, capture_names: list[str]) -> Path:
    """Write a receipt for the files in its folder; example: write_receipt(request, stage, ['scene.png'])."""
    receipt = receipt_path(request, folder)
    receipt.write_text(
        json.dumps(
            receipt_document(request, folder, [folder / name for name in capture_names]),
            indent=2,
        )
        + "\n"
    )
    return receipt


def publish_receipt(
    request: CaptureRequest,
    stage: Path,
    capture_names: list[str],
    read_state: ApplicationState,
) -> dict[str, str]:
    """Publish without replacing an existing directory; example: publish_receipt(request, stage, ['scene.png'], application_state)."""
    destination = request.destination.resolve()
    destination.mkdir()
    created: list[Path] = []
    try:
        return copy_and_verify(
            request, stage, destination, capture_names, read_state, created
        )
    except BaseException:
        rollback_publication(destination, created)
        raise


def rollback_publication(destination: Path, created: list[Path]) -> None:
    """Remove only this publication's files; example: rollback_publication(destination, created)."""
    for path in created:
        path.unlink(missing_ok=True)
    try:
        destination.rmdir()
    except OSError:
        pass


def copy_and_verify(
    request: CaptureRequest,
    stage: Path,
    destination: Path,
    capture_names: list[str],
    read_state: ApplicationState,
    created: list[Path],
) -> dict[str, str]:
    """Copy and recheck the final files; example: copy_and_verify(request, stage, dest, name, state, [])."""
    copy_staged_files(request, stage, destination, created)
    receipt = receipt_path(request, destination)
    created.append(receipt)
    write_receipt(request, destination, capture_names)
    result = artifact(request.root, receipt)
    verify_stored_receipt(request, result)
    require_application_state(request, read_state)
    return result


def copy_staged_files(
    request: CaptureRequest, stage: Path, destination: Path, created: list[Path]
) -> None:
    """Copy checked artifacts without the staging receipt; example: copy_staged_files(request, stage, dest, [])."""
    for path in stage.iterdir():
        if path.name != receipt_path(request, stage).name:
            target = destination / path.name
            created.append(target)
            shutil.copyfile(path, target)


def verify_stored_receipt(request: CaptureRequest, result: dict[str, str]) -> None:
    """Check historical capture content; example: verify_stored_receipt(request, result)."""
    check_capture_receipt(
        request.root,
        result,
        request.row,
        request.source_revision,
        request.application_commit,
    )


def receipt_path(request: CaptureRequest, folder: Path) -> Path:
    """Use the declaration ID for the receipt filename; example: receipt_path(request, folder)."""
    return folder / ("receipt-" + str(request.row["id"]).split(":")[-1][:16] + ".json")


def require_new_destination(request: CaptureRequest) -> Path:
    """Refuse overwrites and paths outside the workspace; example: require_new_destination(request)."""
    root, destination = request.root.resolve(), request.destination.resolve()
    if (
        not destination.is_relative_to(root)
        or destination == root
        or destination.exists()
    ):
        raise ValueError(
            f"receipt destination {destination}: expected a new directory inside {root}"
        )
    return destination


def capture_receipt(
    request: CaptureRequest, run_native: NativeRun, read_state: ApplicationState
) -> dict[str, str]:
    """Run one declaration and publish only checked evidence; example: capture_receipt(request, runner, application_state)."""
    destination = require_new_destination(request)
    require_application_state(request, read_state)
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="oblikovati-receipt-run-") as run_folder:
        reports = Path(run_folder)
        status = run_native(reports, scenario_environment(request, reports))
        if type(status) is not int or status != 0:
            raise ValueError(f"native runner exit {status!r}: expected zero")
        require_application_state(request, read_state)
        return stage_and_publish(request, reports, destination, read_state)


def stage_and_publish(
    request: CaptureRequest,
    reports: Path,
    destination: Path,
    read_state: ApplicationState,
) -> dict[str, str]:
    """Validate artifacts before publication; example: stage_and_publish(request, reports, destination, state)."""
    require_report_checkout(request, reports)
    captures = selected_captures(request, reports)
    capture_names = [capture.name for capture in captures]
    with tempfile.TemporaryDirectory(
        prefix=".receipt-staging-", dir=destination.parent
    ) as staged:
        stage = Path(staged)
        stage_artifacts(reports, stage, captures)
        receipt = write_receipt(request, stage, capture_names)
        verify_stored_receipt(request, artifact(request.root, receipt))
        require_application_state(request, read_state)
        return publish_receipt(request, stage, capture_names, read_state)


def require_report_checkout(request: CaptureRequest, reports: Path) -> None:
    """Match the recorded source root to the requested checkout; example: require_report_checkout(request, reports)."""
    path = reports / "source-revision.json"
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"source report {path}: expected a regular file")
    report = json.loads(path.read_text())
    expected = str(request.checkout.resolve())
    for phase in ("before", "after"):
        state = report.get(phase) if isinstance(report, dict) else None
        actual = state.get("root") if isinstance(state, dict) else None
        if actual != expected:
            raise ValueError(
                f"source report {phase} checkout {actual!r}: expected {expected!r}"
            )

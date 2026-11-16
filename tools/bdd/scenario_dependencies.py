# SPDX-License-Identifier: GPL-2.0-only
"""Describe conservative per-scenario dependencies without inventing isolation."""

import re
import subprocess
from pathlib import Path
from typing import Mapping

from git_environment import git_environment
from runtime_snapshot import snapshot_runtime

VERSION = 1
RUNTIME_COVERAGE = "unbounded"


def require_checkout(root: Path, expected: str) -> None:
    """Bind the requested revision to this worktree; example: require_checkout(repo, sha)."""
    if not re.fullmatch(r"[0-9a-f]{40}", expected):
        raise ValueError(f"commit {expected!r}: expected a full Git SHA-1")
    actual_root = git_checkout_value(root, "--show-toplevel")
    if Path(actual_root).resolve() != root.resolve():
        raise ValueError(f"worktree {actual_root!r}: expected {str(root.resolve())!r}")
    actual = git_checkout_value(root, "--verify", "HEAD^{commit}")
    if actual != expected:
        raise ValueError(f"HEAD {actual!r}: expected commit {expected!r}")


def git_checkout_value(root: Path, *args: str) -> str:
    """Read a Git checkout value without inherited overrides; example: git_checkout_value(repo, '--show-toplevel')."""
    output = subprocess.check_output(
        ["git", "-C", str(root), "rev-parse", *args], env=git_environment(), text=True
    )
    return output.removesuffix("\n")


def dependency_checkout_state(root: Path) -> tuple[str, bool]:
    """Read HEAD and cleanliness within this worktree; example: dependency_checkout_state(repo)."""
    commit = git_checkout_value(root, "--verify", "HEAD^{commit}")
    require_checkout(root, commit)
    clean = worktree_clean(root)
    require_checkout(root, commit)
    return commit, clean


def git_status_command(root: Path, *args: str) -> list[str]:
    """Disable stateful Git extensions for the dependency check; example: git_status_command(repo, 'status')."""
    return [
        "git", "-C", str(root), "-c", "core.fsmonitor=false", "-c",
        "core.untrackedCache=false", *args,
    ]


def git_tree(root: Path, commit: str) -> dict[str, str]:
    """Read tracked paths and object identities; example: git_tree(repo, commit)."""
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError(f"commit {commit!r}: expected a full Git SHA-1")
    output = subprocess.check_output(
        ["git", "-C", str(root), "ls-tree", "-rz", "--full-tree", commit],
        env=git_environment(),
    )
    return dict(parse_tree_entry(row) for row in output.split(b"\0") if row)


def parse_tree_entry(row: bytes) -> tuple[str, str]:
    """Keep file mode and Git identity, including symlinks; example: parse_tree_entry(row)."""
    metadata, name = row.split(b"\t", 1)
    mode, kind, object_id = metadata.decode("ascii").split(" ")
    path = name.decode("utf-8", "surrogateescape")
    if kind not in ("blob", "commit") or not path:
        raise ValueError(
            f"Git tree entry {row!r}: expected a tracked file or submodule"
        )
    return path, f"{mode}:{kind}:{object_id}"


def require_no_submodules(tree: Mapping[str, str]) -> None:
    """Refuse untracked submodule contents; example: require_no_submodules(git_tree(repo, sha))."""
    submodules = sorted(
        path for path, entry in tree.items() if entry.startswith("160000:commit:")
    )
    if submodules:
        raise ValueError(
            f"submodules {submodules!r}: expected fully inventoried inputs"
        )


def runner_for(area: object) -> str:
    """Select the native runner; example: runner_for('sketch')."""
    if area == "sketch":
        return "sketch_scenarios_use_the_native_window"
    if area in ("application", "solid-model"):
        return "application_scenarios_use_the_native_window"
    raise ValueError(
        f"scenario area {area!r}: expected application, sketch, or solid-model"
    )


def scenario_record(row: Mapping[str, object]) -> dict[str, object]:
    """Describe the source and shared runner; example: scenario_record(row)."""
    identifier, path, digest = row.get("id"), row.get("path"), row.get("source_sha256")
    if not isinstance(identifier, str) or not identifier:
        raise ValueError(f"scenario id {identifier!r}: expected a nonempty string")
    if (
        not isinstance(path, str)
        or not path.startswith("features/")
        or ".." in Path(path).parts
    ):
        raise ValueError(f"scenario path {path!r}: expected a relative feature path")
    if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
        raise ValueError(f"scenario source hash {digest!r}: expected a SHA-256 digest")
    return {
        "source": {"path": path, "sha256": digest},
        "runner": runner_for(row.get("area")),
        "shared": "application_inputs",
    }


def scenario_records(inventory: Mapping[str, object]) -> dict[str, dict[str, object]]:
    """Index declarations without merging duplicates; example: scenario_records(inventory)."""
    rows = inventory.get("scenarios")
    if not isinstance(rows, list):
        raise ValueError(f"scenarios {rows!r}: expected a list")
    result: dict[str, dict[str, object]] = {}
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError(f"scenario {row!r}: expected an object")
        record = scenario_record(row)
        identifier = str(row["id"])
        if identifier in result:
            raise ValueError(f"scenario id {identifier!r}: expected a unique id")
        result[identifier] = record
    return result


def worktree_clean(root: Path) -> bool:
    """Detect tracked and untracked changes; example: worktree_clean(repo)."""
    output = subprocess.check_output(
        git_status_command(root, "status", "--porcelain", "-z", "--untracked-files=all"),
        env=git_environment() | {"GIT_OPTIONAL_LOCKS": "0"},
    )
    if output:
        return False
    flags = subprocess.check_output(
        git_status_command(root, "ls-files", "-v", "-z"),
        env=git_environment() | {"GIT_OPTIONAL_LOCKS": "0"},
    )
    return all(entry.startswith(b"H ") for entry in flags.split(b"\0") if entry)


def build_map(
    root: Path,
    inventory: Mapping[str, object],
    commit: str,
    runtime_scenario: str | None = None,
    runtime_root: Path | None = None,
) -> dict[str, object]:
    """Create a conservative map for a Git revision; example: build_map(repo, inventory, commit)."""
    require_checkout(root, commit)
    clean = worktree_clean(root)
    result = collect_map(root, inventory, commit, clean, runtime_scenario, runtime_root)
    require_checkout(root, commit)
    if worktree_clean(root) != clean:
        raise ValueError("application worktree: expected unchanged cleanliness while mapping")
    return result


def collect_map(
    root: Path, inventory: Mapping[str, object], commit: str, clean: bool,
    runtime_scenario: str | None, runtime_root: Path | None,
) -> dict[str, object]:
    """Collect tracked and optional runtime inputs; example: collect_map(repo, inventory, sha, True, None, None)."""
    application_inputs = git_tree(root, commit)
    require_no_submodules(application_inputs)
    return {
        "version": VERSION,
        "application_commit": commit,
        "source_revision": source_revision(inventory),
        "runtime_coverage": RUNTIME_COVERAGE,
        "worktree_clean": clean,
        "application_inputs": application_inputs,
        "runtime_scenario": runtime_scenario,
        "runtime_inputs": snapshot_runtime(runtime_root) if runtime_root else None,
        "scenarios": records_with_runtime(inventory, runtime_scenario, runtime_root),
    }


def source_revision(inventory: Mapping[str, object]) -> str:
    """Require the source commit identity; example: source_revision(inventory)."""
    revision = inventory.get("source_revision")
    if not isinstance(revision, str) or not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError(f"source revision {revision!r}: expected a full Git SHA-1")
    return revision


def records_with_runtime(
    inventory: Mapping[str, object],
    scenario: str | None,
    root: Path | None,
) -> dict[str, dict[str, object]]:
    """Bind an optional runtime to exactly one scenario; example: records_with_runtime(rows, id, root)."""
    records = scenario_records(inventory)
    if (scenario is None) != (root is None):
        raise ValueError("runtime inputs: expected both a scenario and a runtime root")
    if scenario is not None:
        if scenario not in records:
            raise ValueError(f"scenario {scenario!r}: expected a mapped scenario")
        records[scenario]["runtime"] = "runtime_inputs"
    return records


def validate_map(
    mapping: Mapping[str, object], inventory: Mapping[str, object]
) -> None:
    """Refuse an unsupported or mismatched scenario map; example: validate_map(mapping, inventory)."""
    if mapping.get("version") != VERSION:
        raise ValueError(
            f'dependency version {mapping.get("version")!r}: expected {VERSION}'
        )
    if mapping.get("runtime_coverage") != RUNTIME_COVERAGE:
        raise ValueError(
            f'runtime coverage {mapping.get("runtime_coverage")!r}: expected {RUNTIME_COVERAGE!r}'
        )
    if mapping.get("source_revision") != inventory.get("source_revision"):
        raise ValueError("source revision: expected the inventoried scenario revision")
    validate_runtime_record(mapping, inventory)


def validate_runtime_record(
    mapping: Mapping[str, object],
    inventory: Mapping[str, object],
) -> None:
    """Check scenario/runtime links; example: validate_runtime_record(mapping, inventory)."""
    runtime_scenario = mapping.get("runtime_scenario")
    runtime_inputs = mapping.get("runtime_inputs")
    if runtime_scenario is not None and not isinstance(runtime_scenario, str):
        raise ValueError("runtime scenario: expected a scenario id")
    if runtime_scenario is None and runtime_inputs is not None:
        raise ValueError("runtime inputs: expected a mapped scenario")
    if runtime_scenario is not None and not isinstance(runtime_inputs, dict):
        raise ValueError("runtime inputs: expected a recorded runtime tree")
    runtime_marker = Path(".") if runtime_scenario is not None else None
    if mapping.get("scenarios") != records_with_runtime(
        inventory, runtime_scenario, runtime_marker
    ):
        raise ValueError(
            "scenario dependencies: expected every inventoried declaration"
        )


def require_ancestor(root: Path, tested: str, current: str) -> None:
    """Reject divergent histories; example: require_ancestor(repo, tested, current)."""
    for commit in (tested, current):
        if not re.fullmatch(r"[0-9a-f]{40}", commit):
            raise ValueError(f"commit {commit!r}: expected a full Git SHA-1")
    result = subprocess.run(
        ["git", "-C", str(root), "merge-base", "--is-ancestor", tested, current],
        capture_output=True,
        env=git_environment(),
    )
    if result.returncode:
        raise ValueError(
            f"application commit {tested!r}: expected an ancestor of {current!r}"
        )


def changed_dependencies(
    root: Path,
    mapping: Mapping[str, object],
    inventory: Mapping[str, object],
    scenario: str,
    current: str,
) -> list[str]:
    """List conservative tracked-input changes; example: changed_dependencies(repo, mapping, id, head)."""
    require_checkout(root, current)
    validate_map(mapping, inventory)
    scenarios = mapping.get("scenarios")
    if not isinstance(scenarios, dict) or scenario not in scenarios:
        raise ValueError(f"scenario {scenario!r}: expected a mapped scenario")
    after = git_tree(root, current)
    require_no_submodules(after)
    before = checked_application_tree(root, mapping, current)
    changes = changed_paths(before, after)
    require_checkout(root, current)
    if not worktree_clean(root):
        raise ValueError("application worktree: expected clean state after comparison")
    return changes


def checked_application_tree(
    root: Path,
    mapping: Mapping[str, object],
    current: str,
) -> dict[str, str]:
    """Verify the recorded shared input tree; example: checked_application_tree(root, mapping, head)."""
    if mapping.get("worktree_clean") is not True or not worktree_clean(root):
        raise ValueError(
            "application inputs: expected clean recorded and current worktrees"
        )
    tested = str(mapping.get("application_commit"))
    require_ancestor(root, tested, current)
    before = git_tree(root, tested)
    require_no_submodules(before)
    if mapping.get("application_inputs") != before:
        raise ValueError(
            "application input map: expected the complete recorded Git tree"
        )
    return before


def changed_paths(
    before: Mapping[str, object], after: Mapping[str, object]
) -> list[str]:
    """List every added, removed, or modified input; example: changed_paths(old, new)."""
    return sorted(
        path
        for path in before.keys() | after.keys()
        if before.get(path) != after.get(path)
    )


def changed_runtime_dependencies(
    mapping: Mapping[str, object],
    scenario: str,
    root: Path,
) -> list[str]:
    """Compare one recorded runtime tree; example: changed_runtime_dependencies(mapping, id, root)."""
    before = mapping.get("runtime_inputs")
    if mapping.get("runtime_scenario") != scenario or not isinstance(before, dict):
        raise ValueError(f"scenario {scenario!r}: expected its recorded runtime")
    after = snapshot_runtime(root)
    return changed_paths(before, after)


def reuse_decision(
    mapping: Mapping[str, object],
    changes: list[str],
    runtime_changes: list[str] | None = None,
) -> dict[str, object]:
    """Refuse unproven reuse; example: reuse_decision(mapping, [])."""
    if mapping.get("runtime_coverage") != RUNTIME_COVERAGE:
        raise ValueError("runtime coverage: expected the supported conservative scope")
    if changes:
        return {"reusable": False, "reason": "tracked application inputs changed"}
    if runtime_changes:
        return {"reusable": False, "reason": "runtime inputs changed"}
    return {"reusable": False, "reason": "runtime inputs are not bounded"}

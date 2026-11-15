# SPDX-License-Identifier: GPL-2.0-only
"""Connect one experimental committed build to one pinned sealed execution."""

from dataclasses import replace
from pathlib import Path
from typing import Mapping

from pinned_runner import PinnedRunnerProcess
from sandbox_build_probe import BuildProbe, require_runner, run_committed_build_probe
from sealed_execution import SealedRequest, SourceVerifier, file_identity, require_disjoint, run_sealed


def require_chain_paths(probe: BuildProbe, request: SealedRequest) -> None:
    """Keep runner inputs separate from the build output; example: require_chain_paths(probe, request)."""
    if probe.bwrap.resolve() != request.bwrap.resolve():
        raise ValueError(f"sandbox {request.bwrap}: expected build sandbox {probe.bwrap}")
    for output in (probe.target, probe.logs):
        for root in (request.runtime, request.checkout, request.reports):
            require_disjoint(output.resolve(), root.resolve())
        if request.log.resolve().is_relative_to(output.resolve()):
            raise ValueError(f"execution log {request.log}: expected a path outside build output {output}")


def checked_build_runner(probe: BuildProbe, request: SealedRequest, built: Mapping[str, object], revision: str) -> tuple[Path, str, str]:
    """Check that the build observation names its committed runner and sandbox; example: checked_build_runner(probe, request, built, sha)."""
    if built.get("source_commit") != revision:
        raise ValueError(f"build source {built.get('source_commit')!r}: expected commit {revision}")
    runner = require_runner([Path(str(built["runner"]))], probe.target)
    digest = str(built["runner_sha256"])
    sandbox_digest = str(built["sandbox_sha256"])
    if file_identity(request.bwrap)["sha256"] != sandbox_digest:
        raise ValueError(f"sandbox {request.bwrap}: expected build SHA-256 {sandbox_digest}")
    return runner, digest, sandbox_digest


def observation_digest(execution: Mapping[str, object], key: str) -> object:
    """Read one observed file digest; example: observation_digest(run, 'runner_input')."""
    value = execution.get(key)
    if not isinstance(value, dict):
        raise ValueError(f"execution {key} {value!r}: expected a file observation")
    return value.get("sha256")


def require_built_execution(execution: Mapping[str, object], runner: Path, digest: str, sandbox_digest: str) -> None:
    """Reject an execution observation for different files; example: require_built_execution(run, path, sha, sandbox_sha)."""
    if observation_digest(execution, "sandbox_input") != sandbox_digest:
        raise ValueError(f"sandbox observation: expected build SHA-256 {sandbox_digest}")
    if observation_digest(execution, "runner_input") != digest:
        raise ValueError(f"runner {runner}: expected build SHA-256 {digest}")


def run_sealed_build(
    probe: BuildProbe,
    request: SealedRequest,
    repository: Path,
    revision: str,
    verify: SourceVerifier,
    archives: Path | None = None,
    git_repositories: Mapping[str, Path] | None = None,
) -> dict[str, object]:
    """Run one committed build then its pinned executable; example: run_sealed_build(probe, request, repo, sha, verify)."""
    require_chain_paths(probe, request)
    built = run_committed_build_probe(
        probe, repository, revision, archives=archives, git_repositories=git_repositories,
    )
    runner, digest, sandbox_digest = checked_build_runner(probe, request, built, revision)
    execution = run_sealed(replace(request, binary=runner), PinnedRunnerProcess(runner, digest), verify)
    require_built_execution(execution, runner, digest, sandbox_digest)
    return {"build": built, "execution": execution, "reusable": False}

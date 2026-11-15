# SPDX-License-Identifier: GPL-2.0-only
"""Test the deliberately non-reusable offline build probe."""

import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from sandbox_build_probe import (
    BuildProbe,
    build_environment,
    require_probe,
    run_build_probe,
    runner_path,
    sandbox_build_command,
)


class FakeBuildProcess:
    """Write a Cargo artifact response for a fixed test binary."""

    def __init__(self, target: Path, code: int = 0) -> None:
        self.target, self.code = target, code
        self.command: list[str] = []

    def __call__(self, command: list[str], messages: Path, log: Path) -> int:
        self.command = command
        runner = self.target / "debug/deps/head-test"
        runner.parent.mkdir(parents=True)
        runner.write_bytes(b"test runner")
        runner.chmod(0o755)
        messages.write_text(
            json.dumps(
                {
                    "reason": "compiler-artifact",
                    "target": {"name": "oblikovati_head"},
                    "profile": {"test": True},
                    "executable": "/build/debug/deps/head-test",
                }
            )
            + "\n"
        )
        log.write_text("build log")
        return self.code


class FakeInputSnapshot:
    """Return stable input identities or simulate a source change."""

    def __init__(self, changed: bool = False) -> None:
        self.count = 0
        self.changed = changed

    def __call__(self, path: Path) -> dict[str, object]:
        self.count += 1
        return {
            "path": str(path),
            "version": 2 if self.changed and self.count > 4 else 1,
        }


class SandboxBuildProbeTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        root = Path(self.temporary.name)
        roots = [
            root / name
            for name in ("source", "vendor", "toolchain", "config", "target", "logs")
        ]
        for path in roots:
            path.mkdir()
        (roots[3] / "config.toml").write_text("[source.vendored-sources]\n")
        bwrap = root / "bwrap"
        bwrap.write_bytes(b"sandbox executable")
        self.probe = BuildProbe(*roots, bwrap)

    def test_build_observes_inputs_and_never_claims_reuse(self) -> None:
        process = FakeBuildProcess(self.probe.target)
        result = run_build_probe(self.probe, process, FakeInputSnapshot())
        self.assertFalse(result["reusable"])
        self.assertEqual(len(result["runner_sha256"]), 64)
        self.assertEqual(
            set(result["input_inventory_sha256"]),
            {"source", "vendor", "toolchain", "config"},
        )
        self.assertEqual(len(result["sandbox_sha256"]), 64)
        self.assertIn("--unshare-all", process.command)
        self.assertIn("--clearenv", process.command)
        self.assertIn("--offline", process.command)
        self.assertNotIn("RUSTFLAGS", build_environment())
        self.assertIn("/config/config.toml", sandbox_build_command(self.probe))

    def test_changed_inputs_and_failed_build_are_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "inputs changed"):
            run_build_probe(
                self.probe, FakeBuildProcess(self.probe.target), FakeInputSnapshot(True)
            )
        for path in (self.probe.target, self.probe.logs):
            for child in path.iterdir():
                if child.is_dir():
                    import shutil

                    shutil.rmtree(child)
                else:
                    child.unlink()
        with self.assertRaisesRegex(ValueError, "build exit 3"):
            run_build_probe(
                self.probe, FakeBuildProcess(self.probe.target, 3), FakeInputSnapshot()
            )

    def test_refuses_overlap_occupied_output_and_unsupported_host(self) -> None:
        overlap = BuildProbe(
            self.probe.source,
            self.probe.source,
            self.probe.toolchain,
            self.probe.config,
            self.probe.target,
            self.probe.logs,
            self.probe.bwrap,
        )
        with self.assertRaisesRegex(ValueError, "disjoint"):
            require_probe(overlap)
        (self.probe.logs / "existing").write_text("keep")
        with self.assertRaisesRegex(ValueError, "empty directories"):
            require_probe(self.probe)
        with patch("sandbox_build_probe.platform.machine", return_value="arm64"):
            with self.assertRaisesRegex(ValueError, "Linux x86_64"):
                require_probe(self.probe)

    def test_runner_rejects_a_symlink_parent(self) -> None:
        outside = self.probe.source / "outside"
        outside.mkdir()
        (outside / "runner").write_text("runner")
        (outside / "runner").chmod(0o755)
        (self.probe.target / "linked").symlink_to(outside, target_is_directory=True)
        messages = self.probe.logs / "messages"
        record = {
            "reason": "compiler-artifact",
            "target": {"name": "oblikovati_head"},
            "profile": {"test": True},
            "executable": "/build/linked/runner",
        }
        messages.write_text(json.dumps(record) + "\n")
        with self.assertRaisesRegex(ValueError, "within"):
            runner_path(messages, self.probe.target)

    def test_runner_rejects_escaping_and_duplicate_artifacts(self) -> None:
        messages = self.probe.logs / "messages"
        record = {
            "reason": "compiler-artifact",
            "target": {"name": "oblikovati_head"},
            "profile": {"test": True},
            "executable": "/build/../outside",
        }
        messages.write_text(json.dumps(record) + "\n")
        with self.assertRaisesRegex(ValueError, "inside /build"):
            runner_path(messages, self.probe.target)
        runner = self.probe.target / "runner"
        runner.write_text("runner")
        runner.chmod(0o755)
        record["executable"] = "/build/runner"
        messages.write_text((json.dumps(record) + "\n") * 2)
        with self.assertRaisesRegex(ValueError, "one regular"):
            runner_path(messages, self.probe.target)


if __name__ == "__main__":
    unittest.main()

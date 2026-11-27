# SPDX-License-Identifier: GPL-2.0-only
"""Test immutable runner pinning and the Bubblewrap mount boundary."""

import hashlib
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

from pinned_runner import PinnedRunnerProcess, seal_runner


class ReplaceRunnerLauncher:
    """Replace the path just before the real sandbox starts."""

    def __init__(self, runner: Path) -> None:
        self.runner = runner

    def __call__(self, command: list[str], **options: object) -> subprocess.CompletedProcess[str]:
        self.runner.write_bytes(b"forged runner")
        return subprocess.run(command, **options)


@unittest.skipUnless(os.name == "posix" and hasattr(os, "memfd_create"), "requires Linux memfd")
class PinnedRunnerTests(unittest.TestCase):
    def test_sealed_bytes_cannot_be_modified(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "runner"
            path.write_bytes(b"original")
            path.chmod(0o755)
            fd = seal_runner(path, hashlib.sha256(b"original").hexdigest())
            try:
                with self.assertRaises(PermissionError):
                    os.write(fd, b"forged")
                self.assertEqual(os.read(fd, 8), b"original")
            finally:
                os.close(fd)

    def test_changed_or_symlinked_runner_is_refused(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "runner"
            path.write_bytes(b"forged")
            path.chmod(0o755)
            with self.assertRaisesRegex(ValueError, "expected build SHA-256"):
                seal_runner(path, hashlib.sha256(b"original").hexdigest())
            link = Path(folder) / "link"
            link.symlink_to(path)
            with self.assertRaises(OSError):
                seal_runner(link, hashlib.sha256(b"forged").hexdigest())

    def test_missing_runner_mount_is_refused_before_log_creation(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path = root / "runner"
            path.write_bytes(b"original")
            path.chmod(0o755)
            process = PinnedRunnerProcess(path, hashlib.sha256(b"original").hexdigest())
            with self.assertRaisesRegex(ValueError, "expected exactly one matching bind"):
                process(["bwrap", "--", "/runner"], root / "log")
            self.assertFalse((root / "log").exists())

    def test_desktop_is_sealed_and_requires_its_mount(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            runner, desktop = root / "runner", root / "desktop"
            runner.write_bytes(b"runner")
            desktop.write_bytes(b"desktop")
            runner.chmod(0o755)
            desktop.chmod(0o755)
            digest = lambda value: hashlib.sha256(value).hexdigest()
            process = PinnedRunnerProcess(runner, digest(b"runner"), desktop=(desktop, digest(b"desktop")))
            command = ["bwrap", "--ro-bind", str(runner), "/runner"]
            with self.assertRaisesRegex(ValueError, "expected exactly one matching bind"):
                process(command, root / "log")
            self.assertFalse((root / "log").exists())
            desktop.write_bytes(b"changed")
            command.extend(["--ro-bind", str(desktop), "/desktop"])
            with self.assertRaisesRegex(ValueError, "expected build SHA-256"):
                process(command, root / "log")
            self.assertFalse((root / "log").exists())

    @unittest.skipUnless(shutil.which("bwrap"), "requires Bubblewrap")
    def test_real_sandbox_executes_pinned_desktop_after_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            runner, desktop = root / "runner", root / "desktop"
            for path in (runner, desktop):
                shutil.copyfile("/usr/bin/true", path)
                path.chmod(0o755)
            digest = hashlib.sha256(runner.read_bytes()).hexdigest()
            command = [shutil.which("bwrap"), "--unshare-all", "--unshare-user", "--disable-userns", "--ro-bind", "/usr", "/usr", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib64", "/lib64", "--ro-bind", str(runner), "/runner", "--ro-bind", str(desktop), "/desktop", "--", "/desktop"]
            process = PinnedRunnerProcess(runner, digest, ReplaceRunnerLauncher(desktop), desktop=(desktop, digest))
            self.assertEqual(process(command, root / "log"), 0, (root / "log").read_text())

    @unittest.skipUnless(shutil.which("bwrap") and shutil.which("rustc"), "requires Bubblewrap and rustc")
    def test_real_rust_test_binary_runs_from_the_pinned_copy(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            source, runner = root / "fixture.rs", root / "runner"
            source.write_text('#[test] fn fixture() { assert_eq!(2 + 2, 4); }\n')
            subprocess.run(["rustc", "--test", str(source), "-o", str(runner)], check=True)
            digest = hashlib.sha256(runner.read_bytes()).hexdigest()
            command = [shutil.which("bwrap"), "--unshare-all", "--unshare-user", "--disable-userns", "--ro-bind", "/usr", "/usr", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib64", "/lib64", "--ro-bind", str(runner), "/runner", "--", "/runner", "--nocapture"]
            baseline = subprocess.run(command, capture_output=True, text=True)
            if baseline.returncode != 0:
                self.skipTest(f"Bubblewrap execution unavailable: {baseline.stderr.strip()}")
            result = PinnedRunnerProcess(runner, digest, ReplaceRunnerLauncher(runner))(command, root / "log")
            self.assertEqual(result, 0, (root / "log").read_text())
            self.assertIn("1 passed", (root / "log").read_text())

    @unittest.skipUnless(shutil.which("bwrap") and Path("/usr/bin/true").is_file(), "requires Bubblewrap")
    def test_real_sandbox_executes_pinned_bytes_after_path_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            runner = root / "runner"
            shutil.copyfile("/usr/bin/true", runner)
            runner.chmod(0o755)
            digest = hashlib.sha256(runner.read_bytes()).hexdigest()
            command = [shutil.which("bwrap"), "--unshare-all", "--unshare-user", "--disable-userns", "--ro-bind", "/usr", "/usr", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib64", "/lib64", "--ro-bind", str(runner), "/runner", "--", "/runner"]
            baseline = subprocess.run(command, capture_output=True, text=True)
            if baseline.returncode != 0:
                self.skipTest(f"Bubblewrap execution unavailable: {baseline.stderr.strip()}")
            result = PinnedRunnerProcess(runner, digest, ReplaceRunnerLauncher(runner))(command, root / "log")
            self.assertEqual(result, 0, (root / "log").read_text())
            self.assertEqual(runner.read_bytes(), b"forged runner")
            baseline = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(baseline.returncode, 0)

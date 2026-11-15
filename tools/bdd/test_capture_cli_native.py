# SPDX-License-Identifier: GPL-2.0-only
"""Exercise the native runner's report ownership boundary."""

import contextlib
import io
import subprocess
import os
import shutil
import tempfile
import unittest
from pathlib import Path

from capture_cli import NativeCargoRun


class FakeCargoProcess:
    """A native runner that refuses a report directory containing prior artifacts."""

    def __init__(self, command: list[str], **options: object) -> None:
        environment = options["env"]
        assert isinstance(environment, dict)
        self.reports = Path(environment["OBK_BDD_REPORTS"])
        source = Path(str(options["cwd"]))
        assert (source / "tracked.txt").read_text() == "committed"
        assert not (source / "ignored.txt").exists()
        target = Path(environment["CARGO_TARGET_DIR"])
        assert target.is_absolute() and not target.exists()
        assert target != source and not target.is_relative_to(source)
        if list(self.reports.iterdir()):
            raise ValueError(f"reports {self.reports}: expected an empty directory")
        self.stdout = iter(["native output\n"])

    def __enter__(self) -> "FakeCargoProcess":
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def wait(self) -> int:
        return 0


class FakeFailingCargoProcess(FakeCargoProcess):
    def wait(self) -> int:
        return 7


class FakeConcurrentLogProcess(FakeCargoProcess):
    def wait(self) -> int:
        (self.reports / "native-test.log").write_text("concurrent evidence\n")
        return 0


class NativeCargoRunTests(unittest.TestCase):
    def setUp(self) -> None:
        self.repo = tempfile.TemporaryDirectory()
        self.addCleanup(self.repo.cleanup)
        root = Path(self.repo.name)
        subprocess.run(["git", "init", "-q", str(root)], check=True)
        (root / "tracked.txt").write_text("committed")
        (root / ".gitignore").write_text("ignored.txt\n")
        self.git("add", "tracked.txt", ".gitignore")
        self.git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture")
        self.commit = self.git("rev-parse", "HEAD").strip()
        (root / "ignored.txt").write_text("untracked build input")
        (root / "tracked.txt").write_text("uncommitted change")

    def git(self, *arguments: str) -> str:
        result = subprocess.run(["git", "-C", self.repo.name, *arguments], check=True, capture_output=True, text=True)
        return result.stdout

    def test_child_sees_empty_reports_and_log_is_retained(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                status = NativeCargoRun(Path(self.repo.name), "application", self.commit, FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(status, 0)
            self.assertEqual((reports / "native-test.log").read_text(), "native output\n")

    def ignored_include_commit(self) -> str:
        root = Path(self.repo.name)
        (root / "Cargo.toml").write_text(
            '[package]\nname="oblikovati-head"\nversion="0.1.0"\nedition="2021"\n'
        )
        (root / "Cargo.lock").write_text(
            'version = 3\n\n[[package]]\nname = "oblikovati-head"\nversion = "0.1.0"\n'
        )
        (root / "src").mkdir()
        (root / "src/lib.rs").write_text('const INPUT: &str = include_str!("../ignored.txt");\n')
        self.git("add", "Cargo.toml", "Cargo.lock", "src/lib.rs")
        self.git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture crate")
        return self.git("rev-parse", "HEAD").strip()

    @unittest.skipUnless(shutil.which("cargo"), "requires Cargo")
    def test_real_cargo_cannot_read_ignored_checkout_input(self) -> None:
        root = Path(self.repo.name)
        commit = self.ignored_include_commit()
        with tempfile.TemporaryDirectory() as folder, contextlib.redirect_stdout(io.StringIO()):
            reports = Path(folder)
            env = dict(os.environ, CARGO_TARGET_DIR=str(root / "user-target"))
            status = NativeCargoRun(root, "application", commit)(reports, env)
            self.assertNotEqual(status, 0)
            self.assertIn("ignored.txt", (reports / "native-test.log").read_text())
            self.assertFalse((root / "user-target").exists())

    @unittest.skipUnless(shutil.which("cargo"), "requires Cargo")
    def test_real_cargo_runs_with_exported_runtime_asset(self) -> None:
        root = Path(self.repo.name)
        self.ignored_include_commit()
        (root / "src/lib.rs").write_text(
            '#[test] fn application_scenarios_use_the_native_window() { '
            'let dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR")); '
            'assert_eq!(std::fs::read_to_string(dir.join("tracked.txt")).unwrap(), "committed"); '
            'assert!(std::env::current_exe().unwrap().is_file()); }\n'
        )
        self.git("add", "src/lib.rs")
        self.git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "runtime asset")
        commit = self.git("rev-parse", "HEAD").strip()
        with tempfile.TemporaryDirectory() as folder, contextlib.redirect_stdout(io.StringIO()):
            reports = Path(folder)
            status = NativeCargoRun(root, "application", commit)(reports, dict(os.environ))
            self.assertEqual(status, 0, (reports / "native-test.log").read_text())
            self.assertIn("1 passed", (reports / "native-test.log").read_text())

    def test_failed_child_output_is_still_recorded(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                status = NativeCargoRun(Path(self.repo.name), "application", self.commit, FakeFailingCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(status, 7)
            self.assertEqual((reports / "native-test.log").read_text(), "native output\n")

    def test_concurrent_log_is_not_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                with self.assertRaises(FileExistsError):
                    NativeCargoRun(Path(self.repo.name), "application", self.commit, FakeConcurrentLogProcess)(
                        reports, {"OBK_BDD_REPORTS": str(reports)}
                    )
            self.assertEqual((reports / "native-test.log").read_text(), "concurrent evidence\n")

    def test_other_existing_artifacts_are_left_untouched(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            existing = reports / "existing.json"
            existing.write_text("prior evidence\n")
            with self.assertRaisesRegex(ValueError, "expected an empty directory"):
                NativeCargoRun(Path(self.repo.name), "application", self.commit, FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(existing.read_text(), "prior evidence\n")
            self.assertFalse((reports / "native-test.log").exists())

    def test_existing_log_is_not_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            log = reports / "native-test.log"
            log.write_text("prior evidence\n")
            with self.assertRaises((OSError, ValueError)):
                NativeCargoRun(Path(self.repo.name), "application", self.commit, FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(log.read_text(), "prior evidence\n")


if __name__ == "__main__":
    unittest.main()

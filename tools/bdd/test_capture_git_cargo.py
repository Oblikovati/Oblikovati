# SPDX-License-Identifier: GPL-2.0-only
"""Exercise isolated Cargo Git checkouts against a forged shared checkout."""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

from capture_inventory import isolated_inventory


class FakeGitCargo:
    """Create a local Git dependency and populate one disposable Cargo cache."""

    def __init__(self, root: Path, selector: str = "rev") -> None:
        self.selector = selector
        self.root = root
        self.home = root / "cargo"
        self.home.mkdir()
        self.dependency = root / "dependency"
        self.dependency.mkdir()
        self.populate_dependency()
        self.tool = root / "tool"
        self.tool.mkdir()
        self.populate_tool()
        self.environment = dict(os.environ, CARGO_HOME=str(self.home))
        self.environment.pop("RUSTFLAGS", None)
        self.environment.pop("RUSTC_WRAPPER", None)
        subprocess.run(["cargo", "generate-lockfile"], cwd=self.tool, env=self.environment, check=True, stdout=subprocess.DEVNULL)
        self.tool_commit = self.commit(self.tool)

    def commit(self, repo: Path) -> str:
        """Commit a fixture tree; example: fixture.commit(repo)."""
        subprocess.run(["git", "init", "-q", str(repo)], check=True)
        subprocess.run(["git", "-C", str(repo), "add", "."], check=True)
        subprocess.run(["git", "-C", str(repo), "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture"], check=True)
        return subprocess.check_output(["git", "-C", str(repo), "rev-parse", "HEAD"], text=True).strip()

    def populate_dependency(self) -> None:
        """Create the dependency's committed source; example: fixture.populate_dependency()."""
        (self.dependency / "src").mkdir()
        (self.dependency / "Cargo.toml").write_text('[package]\nname="witness"\nversion="1.0.0"\nedition="2021"\n')
        (self.dependency / "src/lib.rs").write_text('pub fn value() -> u32 { 42 }\n')
        self.dependency_commit = self.commit(self.dependency)
        if self.selector in ("tag", "branch"):
            subprocess.run(["git", "-C", str(self.dependency), self.selector, "test-ref"], check=True)

    def populate_tool(self) -> None:
        """Create the inventory tool with a locked local dependency; example: fixture.populate_tool()."""
        (self.tool / "src").mkdir()
        value = self.dependency_commit if self.selector == "rev" else "test-ref"
        (self.tool / "Cargo.toml").write_text(
            '[package]\nname="oblikovati-build"\nversion="0.1.0"\nedition="2021"\n'
            f'[dependencies]\nwitness={{git="{self.dependency.as_uri()}",{self.selector}="{value}"}}\n'
        )
        (self.tool / "src/main.rs").write_text('fn main() { println!("{}", witness::value()); }\n')

    def poison_shared_checkout(self) -> None:
        """Replace the disposable cached source; example: fixture.poison_shared_checkout()."""
        files = list((self.home / "git/checkouts").glob("*/*/src/lib.rs"))
        if len(files) != 1:
            raise ValueError(f"cached source {files!r}: expected exactly one dependency")
        files[0].write_text('compile_error!("forged shared Git checkout");\n')


class CaptureGitCargoTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("cargo"), "requires Cargo")
    def test_real_cargo_ignores_forged_shared_git_checkout(self) -> None:
        for selector in ("rev", "tag", "branch"):
            with self.subTest(selector=selector), tempfile.TemporaryDirectory() as directory:
                fixture = FakeGitCargo(Path(directory), selector)
                fixture.poison_shared_checkout()
                result = isolated_inventory(
                    fixture.tool, fixture.tool, "1" * 40, fixture.tool_commit, fixture.environment,
                )
                self.assertEqual(result, "42\n")

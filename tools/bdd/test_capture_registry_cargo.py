# SPDX-License-Identifier: GPL-2.0-only
"""Prove Cargo ignores forged extracted registry sources during capture."""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

from capture_inventory import isolated_inventory


class FakeRegistryCargo:
    """Construct a cache with a valid archive and a forged extracted crate."""

    def __init__(self, root: Path, actual: Path) -> None:
        self.home = root / "cargo"
        self.namespace = "index.crates.io-1949cf8c6b5b557f"
        cache = self.home / "registry/cache" / self.namespace
        cache.mkdir(parents=True)
        shutil.copyfile(actual / "registry/cache" / self.namespace / "itoa-1.0.18.crate", cache / "itoa-1.0.18.crate")
        (self.home / "registry/index").symlink_to(actual / "registry/index", target_is_directory=True)
        forged = self.home / "registry/src" / self.namespace / "itoa-1.0.18"
        (forged / "src").mkdir(parents=True)
        (forged / ".cargo-ok").write_text('{"v":1}')
        (forged / "Cargo.toml").write_text('[package]\nname="itoa"\nversion="1.0.18"\n')
        (forged / "src/lib.rs").write_text('compile_error!("forged extracted crate");')


class RegistryCargoTests(unittest.TestCase):
    def fixture_repository(self, repo: Path) -> str:
        """Create a tool that requires the locked itoa crate; example: self.fixture_repository(repo)."""
        subprocess.run(["git", "init", "-q", str(repo)], check=True)
        (repo / "src").mkdir()
        (repo / "Cargo.toml").write_text('[package]\nname="oblikovati-build"\nversion="0.1.0"\nedition="2021"\n[dependencies]\nitoa="=1.0.18"\n')
        (repo / "Cargo.lock").write_text('version = 3\n[[package]]\nname = "oblikovati-build"\nversion = "0.1.0"\ndependencies = ["itoa"]\n[[package]]\nname = "itoa"\nversion = "1.0.18"\nsource = "registry+https://github.com/rust-lang/crates.io-index"\nchecksum = "8f42a60cbdf9a97f5d2305f08a87dc4e09308d1276d28c869c684d7777685682"\n')
        (repo / "src/main.rs").write_text('fn main() { let mut b = itoa::Buffer::new(); println!("{}", b.format(42)); }')
        subprocess.run(["git", "-C", str(repo), "add", "."], check=True)
        subprocess.run(["git", "-C", str(repo), "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture"], check=True)
        return subprocess.check_output(["git", "-C", str(repo), "rev-parse", "HEAD"], text=True).strip()

    def test_real_cargo_extracts_verified_archive_not_forged_source(self) -> None:
        actual = Path.home() / ".cargo"
        archive = actual / "registry/cache/index.crates.io-1949cf8c6b5b557f/itoa-1.0.18.crate"
        if not shutil.which("cargo") or not archive.is_file():
            self.skipTest("requires Cargo and the cached itoa 1.0.18 registry package")
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            repo = root / "repo"
            repo.mkdir()
            commit = self.fixture_repository(repo)
            fake = FakeRegistryCargo(root, actual)
            environment = dict(os.environ, CARGO_HOME=str(fake.home))
            self.assertEqual(isolated_inventory(repo, repo, "1" * 40, commit, environment), "42\n")

# SPDX-License-Identifier: GPL-2.0-only
"""Check the explicit build environment and Cargo configuration boundary."""

import tempfile
import unittest
from pathlib import Path

from capture_build_environment import capture_build_environment, reject_ancestor_cargo_configs


class CaptureBuildEnvironmentTests(unittest.TestCase):
    def test_environment_keeps_display_and_drops_unknown_overrides(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            environment = {
                "HOME": str(root), "PATH": "/bin", "DISPLAY": ":7",
                "XDG_RUNTIME_DIR": "/run/user/7", "VK_ICD_FILENAMES": "/gpu.json",
                "OBK_BDD_FEATURE": "create", "RUSTFLAGS": "--cfg injected",
                "CARGO_BUILD_RUSTC_WRAPPER": "/bad", "ARBITRARY_BUILD_FLAG": "bad",
            }
            result = capture_build_environment(root, environment)
            for name in ("DISPLAY", "XDG_RUNTIME_DIR", "VK_ICD_FILENAMES", "OBK_BDD_FEATURE"):
                self.assertEqual(result[name], environment[name])
            for name in ("RUSTFLAGS", "CARGO_BUILD_RUSTC_WRAPPER", "ARBITRARY_BUILD_FLAG"):
                self.assertNotIn(name, result)
            self.assertEqual(result["CARGO_NET_OFFLINE"], "true")
            self.assertNotEqual(result["CARGO_HOME"], str(root / ".cargo"))

    def test_only_dependency_caches_are_exposed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            external, folder = root / "external", root / "build"
            (external / "registry").mkdir(parents=True)
            folder.mkdir()
            (external / "config.toml").write_text('[build]\nrustc-wrapper="/bad"\n')
            result = capture_build_environment(folder, {"CARGO_HOME": str(external)})
            home = Path(result["CARGO_HOME"])
            self.assertEqual((home / "registry").resolve(), (external / "registry").resolve())
            self.assertFalse((home / "config.toml").exists())

    def test_ancestor_config_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "build" / "source"
            source.mkdir(parents=True)
            (root / ".cargo").mkdir()
            (root / ".cargo" / "config.toml").write_text("[build]\n")
            with self.assertRaisesRegex(ValueError, "expected no ancestor configuration"):
                reject_ancestor_cargo_configs(source)


if __name__ == "__main__":
    unittest.main()

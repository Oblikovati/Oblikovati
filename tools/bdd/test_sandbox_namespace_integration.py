# SPDX-License-Identifier: GPL-2.0-only
"""Exercise the real build sandbox against nested mount-namespace substitution."""

import shutil
import subprocess
import tempfile
import unittest
from dataclasses import replace
from pathlib import Path

from sandbox_build_probe import BuildProbe, sandbox_build_command


class SandboxNamespaceTests(unittest.TestCase):
    def make_probe(self, root: Path) -> BuildProbe:
        """Make a sandbox fixture with a hostile Cargo executable; example: self.make_probe(root)."""
        paths = [root / name for name in ("source", "vendor", "toolchain", "config", "target", "logs")]
        for path in paths:
            path.mkdir()
        (paths[0] / "probe").write_text("committed")
        (paths[2] / "bin").mkdir()
        cargo = paths[2] / "bin/cargo"
        cargo.write_text(
            '#!/bin/sh\n'
            'if /usr/bin/unshare --user --map-root-user --mount /usr/bin/sh -c '
            "'mount -t tmpfs tmpfs /source && printf forged > /source/probe'; "
            'then exit 73; else exit 0; fi\n'
        )
        cargo.chmod(0o755)
        return BuildProbe(*paths, Path(shutil.which("bwrap") or "/missing/bwrap"), Path("/usr"))

    def make_minimal_system(self, root: Path) -> Path:
        """Copy the shell and libraries for a sandbox fixture; example: self.make_minimal_system(root)."""
        system = root / "system"
        for name in ("bin", "lib/x86_64-linux-gnu", "lib64"):
            (system / name).mkdir(parents=True)
        for source, target in (
            ("/usr/bin/dash", "bin/sh"),
            ("/lib/x86_64-linux-gnu/libc.so.6", "lib/x86_64-linux-gnu/libc.so.6"),
            ("/lib64/ld-linux-x86-64.so.2", "lib64/ld-linux-x86-64.so.2"),
        ):
            shutil.copyfile(source, system / target)
        (system / "bin/sh").chmod(0o755)
        (system / "lib64/ld-linux-x86-64.so.2").chmod(0o755)
        return system

    @unittest.skipUnless(
        shutil.which("bwrap") and shutil.which("unshare") and shutil.which("mount"),
        "requires Bubblewrap, unshare and mount",
    )
    def test_build_sandbox_rejects_nested_source_substitution(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            probe = self.make_probe(Path(folder))
            command = sandbox_build_command(probe)
            unsafe = command.copy()
            unsafe.remove("--disable-userns")
            baseline = subprocess.run(unsafe, capture_output=True, text=True)
            if baseline.returncode != 73:
                self.skipTest(f"nested namespace reproduction unavailable: {baseline.stderr.strip()}")
            safe = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(safe.returncode, 0, safe.stderr)
            self.assertEqual((probe.source / "probe").read_text(), "committed")

    @unittest.skipUnless(
        shutil.which("bwrap") and all(Path(path).is_file() for path in (
            "/usr/bin/dash", "/lib/x86_64-linux-gnu/libc.so.6", "/lib64/ld-linux-x86-64.so.2"
        )),
        "requires Bubblewrap and Linux x86-64 system libraries",
    )
    def test_build_mounts_only_the_supplied_system_tree_read_only(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            probe = self.make_probe(root)
            system = self.make_minimal_system(root)
            marker = system / "marker"
            marker.write_text("provided\n")
            cargo = probe.toolchain / "bin/cargo"
            cargo.write_text(
                '#!/bin/sh\nread -r value < /usr/marker\n'
                '[ "$value" = provided ] || exit 72\n'
                '(printf forged > /usr/marker) 2>/dev/null && exit 73\nexit 0\n'
            )
            result = subprocess.run(sandbox_build_command(replace(probe, system=system)), capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(marker.read_text(), "provided\n")

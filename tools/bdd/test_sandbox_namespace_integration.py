# SPDX-License-Identifier: GPL-2.0-only
"""Exercise the real build sandbox against nested mount-namespace substitution."""

import shutil
import subprocess
import tempfile
import unittest
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
        return BuildProbe(*paths, Path(shutil.which("bwrap") or "/missing/bwrap"))

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

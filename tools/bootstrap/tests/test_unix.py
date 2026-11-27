# SPDX-License-Identifier: GPL-2.0-only
"""Run bootstrap process boundaries in a disposable checkout with named fake tools."""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
BASH = shutil.which("bash")
FAKE_COMMAND = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["FAKE_LOG"], "a") as log:
    log.write(json.dumps([name, *args]) + "\n")
if name == "id": print(os.environ.get("FAKE_UID", "0"))
elif name == "uname": print("Linux" if args == ["-s"] else "x86_64")
elif name == "rustup" and args[:2] == ["component", "list"]:
    print("rustfmt-x86_64-unknown-linux-gnu\nclippy-x86_64-unknown-linux-gnu\nllvm-tools-x86_64-unknown-linux-gnu")
elif name == "kache": print("kache 2.0.0")
elif name == "cargo" and args in (["xtask", "build"], ["xtask", "gui"]):
    if os.environ.get("FAKE_EDIT_DURING_BUILD"): pathlib.Path("during-build.rs").write_text("changed during build")
    if os.environ.get("FAKE_BUILD_FAIL"): sys.exit(19)
    output = pathlib.Path(os.environ["CARGO_TARGET_DIR"]) / "debug" / "oblikovati"
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("#!/usr/bin/env python3\nimport json,os,sys\nwith open(os.environ['FAKE_LOG'],'a') as f: f.write(json.dumps(['gui',*sys.argv[1:]])+'\\n')\nsys.exit(int(os.environ.get('FAKE_GUI_EXIT','0')))\n")
    output.chmod(0o755)
elif name == "curl":
    target = pathlib.Path(args[args.index("--output") + 1])
    target.write_bytes(os.environ.get("FAKE_DOWNLOAD", "payload").encode())
'''


class FakeUnixHost:
    """Own the fake executables, source checkout and process log for one test."""

    def __init__(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="oblikovati-bootstrap-test-")
        self.root = Path(self.temporary.name) / "checkout with spaces"
        self.bin = Path(self.temporary.name) / "fake tools"
        self.root.mkdir()
        self.bin.mkdir()
        self.log = Path(self.temporary.name) / "commands.jsonl"
        self.log.touch()
        self._copy_sources()
        self._create_tools()
        self._create_git_history()
        self.environment = self._environment()

    def _copy_sources(self) -> None:
        shutil.copy(ROOT / "build.sh", self.root)
        shutil.copytree(ROOT / "tools/bootstrap", self.root / "tools/bootstrap")
        shutil.copy(ROOT / "rust-toolchain.toml", self.root)
        (self.root / "tools/build/src").mkdir(parents=True)
        shutil.copy(ROOT / "tools/build/src/kache.rs", self.root / "tools/build/src/kache.rs")
        (self.root / ".gitignore").write_text("target/\n.tools/\n")

    def _create_tools(self) -> None:
        names = ("id", "uname", "cargo", "rustup", "kache", "pkg-config", "cmake", "ninja", "cc", "c++", "ar", "make", "curl", "Xvfb", "apt-get", "apt", "dnf", "yum", "zypper", "pacman", "apk", "xbps-install", "eopkg", "emerge", "nix", "nix-env", "microdnf", "rpm-ostree")
        for name in names:
            executable = self.bin / name
            executable.write_text(FAKE_COMMAND)
            executable.chmod(0o755)

    def _create_git_history(self) -> None:
        for arguments in (("init", "-q"), ("add", "."), ("-c", "user.name=Bootstrap test", "-c", "user.email=bootstrap@example.invalid", "commit", "-qm", "Fixture")):
            subprocess.run(["git", *arguments], cwd=self.root, check=True, capture_output=True)

    def _environment(self) -> dict[str, str]:
        environment = os.environ.copy()
        environment.update(PATH=f"{self.bin}:{environment['PATH']}", CARGO_HOME=str(self.bin.parent / "cargo home"), FAKE_LOG=str(self.log), DISPLAY=":89")
        for key in ("CARGO_TARGET_DIR", "CARGO_BUILD_TARGET", "RUSTC_WRAPPER", "WAYLAND_DISPLAY"):
            environment.pop(key, None)
        return environment

    def invoke(self, *arguments: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run([BASH, str(self.root / "build.sh"), *arguments], cwd=self.root.parent, env=self.environment, text=True, capture_output=True, timeout=30)

    def function(self, program: str) -> subprocess.CompletedProcess[str]:
        script = 'set -euo pipefail\nBOOTSTRAP_ROOT=$1\nsource "$BOOTSTRAP_ROOT/tools/bootstrap/unix.sh"\n' + program
        return subprocess.run([BASH, "-c", script, "bootstrap-test", str(self.root)], env=self.environment, text=True, capture_output=True, timeout=30)

    def calls(self) -> list[list[str]]:
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def close(self) -> None:
        self.temporary.cleanup()


class UnixBootstrapTests(unittest.TestCase):
    def setUp(self) -> None:
        self.host = FakeUnixHost()
        self.addCleanup(self.host.close)

    def test_build_uses_existing_rust_build_system_and_prebuilt_run_skips_it(self) -> None:
        result = self.host.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.host.log.write_text("")
        result = self.host.invoke("run", "--", "path with spaces.obk", "--frames", "3")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([call for call in self.host.calls() if call[0] in ("cargo", "gui")], [["gui", "path with spaces.obk", "--frames", "3"]])

    def test_missing_gui_builds_then_launches_and_propagates_exit_status(self) -> None:
        self.host.environment["FAKE_GUI_EXIT"] = "7"
        result = self.host.invoke("run", "file with spaces.obk")
        self.assertEqual(result.returncode, 7, result.stderr)
        calls = [call for call in self.host.calls() if call[0] in ("cargo", "gui")]
        self.assertEqual(calls, [["cargo", "xtask", "gui"], ["gui", "file with spaces.obk"]])

    def test_failed_build_never_launches_an_old_binary(self) -> None:
        self.assertEqual(self.host.invoke().returncode, 0)
        (self.host.root / "Cargo.toml").write_text("changed source")
        self.host.log.write_text("")
        self.host.environment["FAKE_BUILD_FAIL"] = "1"
        result = self.host.invoke("run")
        self.assertEqual(result.returncode, 19, result.stderr)
        self.assertFalse(any(call[0] == "gui" for call in self.host.calls()))

    def test_absolute_custom_target_directory_with_spaces_is_respected(self) -> None:
        self.host.environment["CARGO_TARGET_DIR"] = str(self.host.root.parent / "output with spaces")
        result = self.host.invoke("run", "--", "--frames", "1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((Path(self.host.environment["CARGO_TARGET_DIR"]) / "debug/oblikovati").is_file())

    def test_modified_binary_and_new_untracked_source_invalidate_the_stamp(self) -> None:
        self.assertEqual(self.host.invoke().returncode, 0)
        (self.host.root / "new.rs").write_text("new source")
        self.host.log.write_text("")
        self.assertEqual(self.host.invoke("run").returncode, 0)
        self.assertIn(["cargo", "xtask", "gui"], self.host.calls())
        binary = self.host.root / "target/debug/oblikovati"
        binary.write_text(binary.read_text() + "# changed binary\n")
        self.host.log.write_text("")
        self.assertEqual(self.host.invoke("run").returncode, 0)
        self.assertIn(["cargo", "xtask", "gui"], self.host.calls())

    def test_wayland_without_xwayland_cannot_launch_or_install(self) -> None:
        self.host.environment.pop("DISPLAY")
        self.host.environment["WAYLAND_DISPLAY"] = "wayland-0"
        result = self.host.invoke("run")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("X11/XWayland", result.stderr)
        self.assertFalse(any(call[0] in ("cargo", "gui", "apt-get") for call in self.host.calls()))

    def test_check_and_invalid_arguments_have_no_install_or_build_side_effect(self) -> None:
        self.assertEqual(self.host.invoke("--check").returncode, 0)
        self.assertEqual(self.host.invoke("unknown").returncode, 2)
        self.assertFalse(any(call[0] in ("cargo", "apt-get", "gui") for call in self.host.calls()))

    def test_download_requires_the_locked_digest_before_publication(self) -> None:
        payload = b"trusted fixture"
        lock = self.host.root / "tools/bootstrap/downloads.lock"
        lock.write_text(f"fixture {hashlib.sha256(payload).hexdigest()} https://example.invalid/fixture\n")
        self.host.environment["FAKE_DOWNLOAD"] = payload.decode()
        result = self.host.function('bootstrap_download fixture "$BOOTSTRAP_ROOT/artifact"')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.host.root / "artifact").read_bytes(), payload)
        (self.host.root / "artifact").unlink()
        self.host.environment["FAKE_DOWNLOAD"] = "corrupted bytes"
        result = self.host.function('bootstrap_download fixture "$BOOTSTRAP_ROOT/artifact"')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256", result.stderr)
        self.assertFalse((self.host.root / "artifact").exists())

    def test_all_linux_manager_commands_are_selected_without_host_mutation(self) -> None:
        managers = ("apt-get", "apt", "dnf", "microdnf", "yum", "zypper", "pacman", "apk", "xbps-install", "eopkg", "emerge", "rpm-ostree", "nix", "nix-env")
        for manager in managers:
            with self.subTest(manager=manager):
                self.host.log.write_text("")
                script = f'bootstrap_manager() {{ printf "%s\\n" "{manager}"; }}\nbootstrap_install_linux'
                result = self.host.function(script)
                if manager == "rpm-ostree":
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("reboot", result.stderr)
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                self.assertTrue(any(call[0] == manager for call in self.host.calls()))

    def test_relative_custom_target_is_normalized_and_remains_fresh(self) -> None:
        self.host.environment["CARGO_TARGET_DIR"] = "./output with spaces/../custom target"
        result = self.host.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.host.log.write_text("")
        result = self.host.invoke("run")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any(call[0] == "cargo" for call in self.host.calls()))

    def test_sources_changed_during_build_are_not_blessed_or_launched(self) -> None:
        self.host.environment["FAKE_EDIT_DURING_BUILD"] = "1"
        result = self.host.invoke("run")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("sources changed during", result.stderr)
        self.assertFalse(any(call[0] == "gui" for call in self.host.calls()))

    def test_first_homebrew_install_output_does_not_corrupt_manager_selection(self) -> None:
        installer = '#!/bin/bash\nprintf "Installer progress\\n"\nprintf "#!/bin/bash\\nexit 0\\n" > "$FAKE_BIN/brew"\nchmod +x "$FAKE_BIN/brew"\n'
        digest = hashlib.sha256(installer.encode()).hexdigest()
        (self.host.root / "tools/bootstrap/downloads.lock").write_text(f"homebrew {digest} https://example.invalid/install.sh\n")
        self.host.environment.update(FAKE_DOWNLOAD=installer, FAKE_BIN=str(self.host.bin))
        program = 'BOOTSTRAP_TEMP=$BOOTSTRAP_ROOT\nbootstrap_has() { if [[ $1 = brew || $1 = port ]]; then [[ -x "$FAKE_BIN/$1" ]]; else command -v "$1" >/dev/null; fi; }\nbootstrap_macos_manager'
        result = self.host.function(program)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "brew")
        self.assertIn("Installer progress", result.stderr)

    def test_insecure_or_unknown_downloads_never_invoke_curl(self) -> None:
        (self.host.root / "tools/bootstrap/downloads.lock").write_text("bad " + "a" * 64 + " http://example.invalid/file\n")
        for key in ("bad", "unknown"):
            result = self.host.function(f'bootstrap_download {key} "$BOOTSTRAP_ROOT/artifact"')
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("expected a locked HTTPS URL", result.stderr)
        self.assertFalse(any(call[0] == "curl" for call in self.host.calls()))

    def test_missing_sha256_tools_produce_an_explicit_error(self) -> None:
        result = self.host.function('PATH=/not-installed\nprintf bytes | bootstrap_digest')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256 tools missing", result.stderr)


if __name__ == "__main__":
    unittest.main()

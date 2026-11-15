# SPDX-License-Identifier: GPL-2.0-only
"""Check independent Git manifest declaration comparisons."""

import unittest

from git_manifest_declarations import check_git_manifest_declarations, same_toml_value


class GitManifestDeclarationsTests(unittest.TestCase):
    def test_matching_selected_declarations_are_accepted(self) -> None:
        source = b'[package]\nname="demo"\nversion="1"\n[dependencies]\nfoo={version="1",default-features=false}\n[features]\ndefault=["foo"]\n'
        generated = b'[package]\nname="demo"\nversion="1"\nbuild=false\n[dependencies.foo]\nversion="1"\ndefault-features=false\n[features]\ndefault=["foo"]\n'
        check_git_manifest_declarations(source, generated, {"Cargo.toml": "blob"}, "demo")

    def test_bool_and_integer_are_not_treated_as_equal(self) -> None:
        self.assertFalse(same_toml_value(False, 0))
        source = b'[package]\nname="demo"\n[dependencies]\nfoo={version="1",default-features=false}\n'
        generated = b'[package]\nname="demo"\n[dependencies]\nfoo={version="1",default-features=0}\n'
        with self.assertRaisesRegex(ValueError, "expected locked source declaration"):
            check_git_manifest_declarations(source, generated, {}, "demo")

    def test_authored_or_autodetected_build_scripts_are_declined(self) -> None:
        generated = b'[package]\nname="demo"\nbuild=false\n'
        for source, entries in (
            (b'[package]\nname="demo"\nbuild="build.rs"\n', {}),
            (b'[package]\nname="demo"\n', {"build.rs": "blob"}),
        ):
            with self.subTest(source=source, entries=entries):
                with self.assertRaisesRegex(ValueError, "expected no"):
                    check_git_manifest_declarations(source, generated, entries, "demo")

    def test_disabled_build_rs_cannot_be_reenabled_by_omission(self) -> None:
        source = b'[package]\nname="demo"\nbuild=false\n'
        generated = b'[package]\nname="demo"\n'
        with self.assertRaisesRegex(ValueError, "expected no build script"):
            check_git_manifest_declarations(source, generated, {"build.rs": "blob"}, "demo")

    def test_unsupported_workspace_inheritance_is_declined(self) -> None:
        source = b'[package]\nname="demo"\n[dependencies]\nfoo={workspace=true}\n'
        generated = b'[package]\nname="demo"\n[dependencies]\nfoo="1"\n'
        with self.assertRaisesRegex(ValueError, "expected locked source declaration"):
            check_git_manifest_declarations(source, generated, {}, "demo")

    def test_duplicate_dependency_alias_is_declined(self) -> None:
        source = b'[package]\nname="demo"\n'
        generated = b'[package]\nname="demo"\n[dev-dependencies]\n[dev_dependencies]\n'
        with self.assertRaisesRegex(ValueError, "expected at most one alias"):
            check_git_manifest_declarations(source, generated, {}, "demo")

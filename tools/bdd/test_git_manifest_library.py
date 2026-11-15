# SPDX-License-Identifier: GPL-2.0-only
"""Test conservative checks for conventional Git vendor library targets."""

import unittest

from git_manifest_library import check_git_manifest_library


SOURCE = b'[package]\nname="demo-crate"\nversion="1"\n'
GENERATED = (
    b'[package]\nname="demo-crate"\nversion="1"\n'
    b'autolib=false\nautobins=false\nautoexamples=false\nautotests=false\nautobenches=false\n'
    b'[lib]\nname="demo_crate"\npath="src/lib.rs"\n'
)
ENTRIES = {"src/lib.rs": "100644:blob:fixture"}


class GitManifestLibraryTests(unittest.TestCase):
    def test_conventional_library_is_accepted(self) -> None:
        check_git_manifest_library(SOURCE, GENERATED, ENTRIES, "demo")

    def test_added_proc_macro_and_crate_type_are_rejected(self) -> None:
        for field in (b'proc-macro=true\n', b'crate-type=["cdylib"]\n', b'test=false\n'):
            with self.subTest(field=field):
                with self.assertRaisesRegex(ValueError, "expected"):
                    check_git_manifest_library(SOURCE, GENERATED + field, ENTRIES, "demo")

    def test_library_path_name_and_autodiscovery_are_checked(self) -> None:
        for changed in (
            GENERATED.replace(b'src/lib.rs', b'../outside.rs'),
            GENERATED.replace(b'demo_crate', b'forged'),
            GENERATED.replace(b'autobins=false', b'autobins=true'),
            GENERATED.replace(b'autolib=false', b'autolib=0'),
        ):
            with self.subTest(changed=changed):
                with self.assertRaisesRegex(ValueError, "expected"):
                    check_git_manifest_library(SOURCE, changed, ENTRIES, "demo")

    def test_explicit_library_or_missing_source_is_declined(self) -> None:
        for source, entries in ((SOURCE + b'[lib]\npath="src/lib.rs"\n', ENTRIES), (SOURCE, {})):
            with self.subTest(source=source, entries=entries):
                with self.assertRaisesRegex(ValueError, "expected"):
                    check_git_manifest_library(source, GENERATED, entries, "demo")

    def test_links_must_match_locked_source(self) -> None:
        forged = GENERATED.replace(b'autolib=false', b'links="forged"\nautolib=false')
        with self.assertRaisesRegex(ValueError, "expected locked source"):
            check_git_manifest_library(SOURCE, forged, ENTRIES, "demo")

# SPDX-License-Identifier: GPL-2.0-only
"""Test the filesystem-independent subset of Cargo path metadata."""

import unittest

from git_manifest_path_metadata import check_git_manifest_path_metadata, explicit_readme, inherited_path, relative_workspace_path
from test_git_manifest_metadata import MetadataRepository
from git_object_inventory import locked_git_tree
from pathlib import Path
import tempfile


class PathMetadataTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.fixture = MetadataRepository(Path(self.temp.name))

    def check(self, source: str, generated: str) -> None:
        """Compare package fields; example: self.check('readme=false', 'readme=false')."""
        check_git_manifest_path_metadata(('[package]\n' + source).encode(),
                                         ('[package]\n' + generated).encode(),
                                         self.fixture.repository, self.fixture.tree, 'crate', 'fixture')

    def test_explicit_readme_and_license_file(self) -> None:
        for source, expected in (('true', '"README.md"'), ('false', 'false'), ('"doc.txt"', '"doc.txt"')):
            with self.subTest(source=source):
                self.check(f'readme={source}\nlicense-file="LICENSE"\n',
                           f'readme={expected}\nlicense-file="LICENSE"\n')
                with self.assertRaisesRegex(ValueError, 'readme.*expected locked'):
                    self.check(f'readme={source}\n', 'readme="forged"\n')

    def test_license_file_changes_and_additions_are_refused(self) -> None:
        for source, generated in (('', 'license-file="forged"\n'),
                                  ('license_file="LICENSE"\n', 'license-file="other"\n')):
            with self.subTest(source=source):
                with self.assertRaisesRegex(ValueError, 'license-file.*expected locked'):
                    self.check(source, generated)

    def test_default_and_inherited_paths_remain_unchecked(self) -> None:
        self.check('', 'readme="unknown"\n')
        self.check('readme.workspace=true\n', 'readme="unknown"\n')

    def test_inherited_paths_use_locked_workspace_and_reject_forgeries(self) -> None:
        workspace = self.fixture.repository / 'Cargo.toml'
        workspace.write_text('[workspace]\n[workspace.package]\nreadme="doc/README.md"\nlicense-file="LICENSE"\n')
        self.fixture.git.git('add', '.')
        self.fixture.git.git('commit', '-qm', 'path metadata')
        self.fixture.tree = locked_git_tree(self.fixture.repository, self.fixture.git.git('rev-parse', 'HEAD'))
        source = 'readme.workspace=true\nlicense-file.workspace=true\n'
        generated = 'readme="../doc/README.md"\nlicense-file="../LICENSE"\n'
        self.check(source, generated)
        workspace.write_text('[workspace]\n')
        self.check(source, generated)
        for name in ('doc/README.md', 'LICENSE'):
            with self.subTest(name=name):
                with self.assertRaisesRegex(ValueError, 'expected locked source'):
                    self.check(source, generated.replace(name, 'forged'))

    def test_workspace_path_resolution_and_rejections(self) -> None:
        self.assertEqual(relative_workspace_path('doc/README.md', '.', 'crates/a', 'fixture', 'readme'), '../../doc/README.md')
        for path in ('../LICENSE', '/LICENSE', 'C:/LICENSE', 'foo\\bar', 'a//b'):
            with self.subTest(path=path):
                with self.assertRaisesRegex(ValueError, 'expected a portable relative path'):
                    relative_workspace_path(path, '.', 'crate', 'fixture', 'readme')

    def test_invalid_inheritance_and_readme_types_are_refused(self) -> None:
        for value in ({'workspace': 1}, {'workspace': True, 'extra': True}):
            with self.subTest(value=value):
                with self.assertRaisesRegex(ValueError, 'expected only workspace = true'):
                    inherited_path(value, 'fixture', 'readme')
        with self.assertRaisesRegex(ValueError, 'expected a string or boolean'):
            explicit_readme(1, 'fixture')


if __name__ == '__main__':
    unittest.main()

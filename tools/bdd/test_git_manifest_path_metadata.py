# SPDX-License-Identifier: GPL-2.0-only
"""Test the filesystem-independent subset of Cargo path metadata."""

import unittest

from git_manifest_path_metadata import check_git_manifest_path_metadata, explicit_readme, inherited_path


class PathMetadataTests(unittest.TestCase):
    def check(self, source: str, generated: str) -> None:
        """Compare package fields; example: self.check('readme=false', 'readme=false')."""
        check_git_manifest_path_metadata(('[package]\n' + source).encode(),
                                         ('[package]\n' + generated).encode(), 'fixture')

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
        self.check('readme.workspace=true\nlicense-file.workspace=true\n',
                   'readme="unknown"\nlicense-file="unknown"\n')

    def test_invalid_inheritance_and_readme_types_are_refused(self) -> None:
        for value in ({'workspace': 1}, {'workspace': True, 'extra': True}):
            with self.subTest(value=value):
                with self.assertRaisesRegex(ValueError, 'expected only workspace = true'):
                    inherited_path(value, 'fixture', 'readme')
        with self.assertRaisesRegex(ValueError, 'expected a string or boolean'):
            explicit_readme(1, 'fixture')


if __name__ == '__main__':
    unittest.main()

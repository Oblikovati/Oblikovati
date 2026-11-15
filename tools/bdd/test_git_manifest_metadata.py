# SPDX-License-Identifier: GPL-2.0-only
"""Test metadata and lint checks against committed workspace declarations."""

import tempfile
import unittest
from pathlib import Path

from git_manifest_metadata import check_git_manifest_metadata, expected_lints, inherited_value
from git_object_inventory import locked_git_tree
from test_committed_build_source import SourceRepository


class MetadataRepository:
    """A workspace with a committed package and independently generated manifest."""

    def __init__(self, root: Path) -> None:
        self.repository = root / "repo"
        self.repository.mkdir()
        self.git = SourceRepository(self.repository)
        (self.repository / "Cargo.toml").write_text(
            '[workspace]\nmembers=["crate"]\n[workspace.package]\n'
            'repository="https://example.org/source"\nlicense="MIT"\n'
            '[workspace.lints.rust]\nunsafe_code="forbid"\n'
        )
        (self.repository / "crate").mkdir()
        self.source = ('[package]\nname="sample"\nversion="1.0.0"\n'
                       'repository.workspace=true\nlicense.workspace=true\n'
                       '[lints]\nworkspace=true\n').encode()
        (self.repository / "crate/Cargo.toml").write_bytes(self.source)
        self.git.git("add", ".")
        self.git.git("commit", "-qm", "workspace")
        self.tree = locked_git_tree(self.repository, self.git.git("rev-parse", "HEAD"))
        self.generated = ('[package]\nname="sample"\nversion="1.0.0"\n'
                          'repository="https://example.org/source"\nlicense="MIT"\n'
                          '[lints.rust]\nunsafe_code="forbid"\n').encode()

    def verify(self, source: bytes | None = None, generated: bytes | None = None) -> None:
        """Check package declarations; example: fixture.verify()."""
        check_git_manifest_metadata(source or self.source, generated or self.generated,
                                    self.repository, self.tree, "crate", "sample")


class MetadataCheckTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.fixture = MetadataRepository(Path(self.temp.name))

    def test_inherited_values_and_lints_match_locked_workspace(self) -> None:
        self.fixture.verify()
        (self.fixture.repository / "Cargo.toml").write_text('[workspace]\n')
        self.fixture.verify()

    def test_forged_inherited_metadata_and_lints_fail(self) -> None:
        for old, new in ((b'MIT', b'Apache-2.0'), (b'forbid', b'allow')):
            with self.subTest(old=old):
                with self.assertRaisesRegex(ValueError, 'expected locked source'):
                    self.fixture.verify(generated=self.fixture.generated.replace(old, new))

    def test_direct_values_and_type_sensitive_arrays(self) -> None:
        src = b'[package]\nauthors=["A"]\npublish=false\n[lints.rust]\nunsafe_code="deny"\n'
        self.fixture.verify(source=src, generated=src)
        forged = src.replace(b'publish=false', b'publish=0')
        with self.assertRaisesRegex(ValueError, 'publish'):
            self.fixture.verify(source=src, generated=forged)

    def test_invalid_inheritance_fails_closed(self) -> None:
        for value in ({"workspace": 1}, {"workspace": True, "extra": True}):
            with self.subTest(value=value):
                with self.assertRaisesRegex(ValueError, 'expected a workspace inheritance marker'):
                    inherited_value(value, {}, 'package', ('license',), 'sample')
        with self.assertRaisesRegex(ValueError, 'expected only workspace = true'):
            expected_lints({'workspace': False}, {}, 'sample')
        with self.assertRaisesRegex(ValueError, 'workspace lints'):
            expected_lints({'workspace': True}, {}, 'sample')

    def test_conflicting_aliases_and_explicit_workspace_fail(self) -> None:
        for source in (b'[package]\nrust-version="1.0"\nrust_version="1.0"\n',
                       b'[package]\nworkspace="../other"\n'):
            with self.subTest(source=source):
                with self.assertRaises(ValueError):
                    self.fixture.verify(source=source)


if __name__ == '__main__':
    unittest.main()

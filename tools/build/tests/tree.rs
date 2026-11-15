// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(crate) struct FixtureTree(pub(crate) tempfile::TempDir);

impl FixtureTree {
    pub(crate) fn new() -> Self {
        Self(tempfile::tempdir().unwrap())
    }
    pub(crate) fn root(&self) -> &Path {
        self.0.path()
    }
    pub(crate) fn put(&self, relative: &str, text: &str) {
        write_text(&self.root().join(relative), text).unwrap();
    }
}

#[test]
fn discovery_ignores_outputs_but_keeps_independent_fixture_crates() {
    let tree = FixtureTree::new();
    for file in [
        "Cargo.toml",
        "tests/plugin/Cargo.toml",
        "target/old/Cargo.toml",
        "node_modules/x/Cargo.toml",
    ] {
        tree.put(file, "manifest");
    }
    let files = source_files(tree.root())
        .unwrap()
        .iter()
        .map(|path| relative_name(tree.root(), path).unwrap())
        .collect::<Vec<_>>();
    assert_eq!(files, ["Cargo.toml", "tests/plugin/Cargo.toml"]);
    assert_eq!(
        read_text(&tree.root().join("Cargo.toml")).unwrap(),
        "manifest"
    );
    assert!(read_text(&tree.root().join("missing")).is_err());
    assert!(relative_name(tree.root(), Path::new("outside")).is_err());
}

#[test]
fn source_walk_excludes_only_worktree_storage() {
    let tree = FixtureTree::new();
    for name in [
        ".slim/worktrees/other/src/invalid.rs",
        ".slim/worktrees/other/Cargo.toml",
        ".slim/settings.md",
        ".slim/worktrees-extra/keep.rs",
        "fixtures/worktrees/keep.rs",
        "fixtures/nested/src/lib.rs",
        "fixtures/nested/Cargo.toml",
    ] {
        tree.put(name, "fixture");
    }
    let found = source_files(tree.root())
        .unwrap()
        .iter()
        .map(|path| relative_name(tree.root(), path).unwrap())
        .collect::<Vec<_>>();
    assert_eq!(
        found,
        [
            ".slim/settings.md",
            ".slim/worktrees-extra/keep.rs",
            "fixtures/nested/Cargo.toml",
            "fixtures/nested/src/lib.rs",
            "fixtures/worktrees/keep.rs"
        ]
    );
    assert_eq!(all_files(tree.root()).unwrap().len(), 7);
}

#[cfg(unix)]
#[test]
fn worktree_storage_symlink_is_not_silently_accepted() {
    let tree = FixtureTree::new();
    tree.put(".slim/keep", "fixture");
    std::os::unix::fs::symlink(tree.root(), tree.root().join(".slim/worktrees")).unwrap();
    let error = source_files(tree.root()).unwrap_err();
    assert!(error.contains(".slim/worktrees") && error.contains("regular file or directory"));
}

#[test]
fn active_checkout_inside_worktree_storage_is_still_scanned() {
    let tree = FixtureTree::new();
    tree.put(".slim/worktrees/active/src/lib.rs", "fixture");
    tree.put(
        ".slim/worktrees/active/fixtures/nested/Cargo.toml",
        "fixture",
    );
    let active = tree.root().join(".slim/worktrees/active");
    let files = source_files(&active).unwrap();
    assert_eq!(files.len(), 2);
    assert!(files.iter().any(|path| path.ends_with("src/lib.rs")));
    assert!(
        files
            .iter()
            .any(|path| path.ends_with("fixtures/nested/Cargo.toml"))
    );
}

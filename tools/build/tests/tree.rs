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

// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::tree::source_files;
use crate::{platform::platform, process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn archive_inputs_and_linuxdeploy_arguments_are_checked() {
    let tree = FixtureTree::new();
    let runner = FakeRunner::default();
    let output = tree.root().join("package.zip");
    assert!(archive(tree.root(), &runner, tree.root(), &output).is_err());
    tree.put("input/binary", "archive input");
    archive(tree.root(), &runner, &tree.root().join("input"), &output).unwrap();
    let arguments = linuxdeploy_arguments(Path::new("app dir"), Path::new("app.desktop"));
    assert_eq!(
        arguments
            .iter()
            .filter(|value| *value == "--executable")
            .count(),
        2
    );
}

#[test]
fn cli_archive_removes_the_gui_but_preserves_runtime_resources() {
    let tree = FixtureTree::new();
    tree.put("input/bin/oblikovati-head", "head");
    tree.put("input/bin/oblikovati-cli", "cli");
    tree.put("input/lib/runtime.so", "library");
    tree.put(
        "dist/oblikovati-rust-cli-0.1.2.3-linux-amd64.tar.gz",
        "archive",
    );
    let runner = FakeRunner::default();
    cli_archive(
        tree.root(),
        &runner,
        platform("x86_64-unknown-linux-gnu").unwrap(),
        &tree.root().join("input"),
        "0.1.2.3",
    )
    .unwrap();
    let stage = tree
        .root()
        .join("target/cli-package/x86_64-unknown-linux-gnu");
    assert!(!stage.join("bin/oblikovati-head").exists());
    assert_eq!(source_files(&stage).unwrap().len(), 2);
}

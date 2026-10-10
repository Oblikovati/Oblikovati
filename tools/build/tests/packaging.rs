// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, release_config::tests::config, tree::tests::FixtureTree};

pub(crate) fn resources_fixture(tree: &FixtureTree) {
    for path in [
        "icons/oblikovati.png",
        "icons/Oblikovati.icns",
        "fonts/font",
        "shaders/shader",
        "templates/part",
        "catalogs/materials",
        "NOTICES",
    ] {
        tree.put(&format!("packaging/resources/{path}"), "fixture bytes");
    }
}

#[test]
fn complete_resources_are_mandatory_and_paths_cannot_escape() {
    let tree = FixtureTree::new();
    let mut config = config();
    assert!(
        resources(tree.root(), &config)
            .unwrap_err()
            .contains("resource")
    );
    resources_fixture(&tree);
    let source = resources(tree.root(), &config).unwrap();
    copy_tree(&source, &tree.root().join("copied")).unwrap();
    assert_eq!(source_files(&tree.root().join("copied")).unwrap().len(), 7);
    config.product.resources = "../outside".into();
    assert!(resources(tree.root(), &config).is_err());
}

#[test]
fn stage_contains_real_binaries_and_required_metadata() {
    let tree = FixtureTree::new();
    let platform = platform("x86_64-unknown-linux-gnu").unwrap();
    resources_fixture(&tree);
    for file in [
        "LICENSE",
        "target/release-build.json",
        "target/x86_64-unknown-linux-gnu/release/oblikovati-cli",
        "target/x86_64-unknown-linux-gnu/release/oblikovati-head",
    ] {
        tree.put(file, "fixture bytes");
    }
    let stage = stage_product(tree.root(), &config(), platform).unwrap();
    assert!(stage.join("bin/oblikovati-cli").is_file());
    assert!(stage.join("release-manifest.json").is_file());
    stage_product(tree.root(), &config(), platform).unwrap();
    let runner = FakeRunner::default();
    assert!(smoke(tree.root(), &runner, platform, &stage, "0.1.2.3").is_err());
    runner.replies.borrow_mut().push_back(Ok("0.1.2.3".into()));
    assert!(
        smoke(tree.root(), &runner, platform, &stage, "0.1.2.3")
            .unwrap_err()
            .contains("smoke test")
    );
}

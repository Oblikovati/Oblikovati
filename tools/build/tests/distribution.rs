// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, release_config::tests::config, tree::tests::FixtureTree};

#[test]
fn collection_requires_all_platforms_and_detects_tampering() {
    let tree = FixtureTree::new();
    let version = "0.0015400.1.0";
    let runner = FakeRunner::default();
    runner
        .replies
        .borrow_mut()
        .extend([Ok("a".repeat(40)), Ok("rustc 1.99.0".into())]);
    let manifest = new_manifest(tree.root(), &runner, config(), version).unwrap();
    for name in artifact_names(version).unwrap() {
        tree.put(&format!("input/{name}"), "test package bytes");
    }
    let output = tree.root().join("output");
    collect(&tree.root().join("input"), &output, manifest).unwrap();
    let receipt = verify(&output, version, &"a".repeat(40)).unwrap();
    assert_eq!(receipt.artifacts.len(), 9);
    assert!(verify(&output, version, &"b".repeat(40)).is_err());
    tree.put(
        &format!("output/{}", receipt.artifacts.keys().next().unwrap()),
        "changed",
    );
    assert!(
        verify(&output, version, &"a".repeat(40))
            .unwrap_err()
            .contains("checksum")
    );
}

#[test]
fn incomplete_duplicate_or_staging_artifacts_are_rejected() {
    let tree = FixtureTree::new();
    let names = artifact_names("0.1.2.3").unwrap();
    tree.put("input/unsigned.dylib", "staging");
    assert!(distribution_files(&tree.root().join("input"), &names).is_err());
    tree.put(&format!("twice/a/{}", names[0]), "package");
    tree.put(&format!("twice/b/{}", names[0]), "package");
    assert!(
        distribution_files(&tree.root().join("twice"), &names)
            .unwrap_err()
            .contains("duplicates")
    );
    tree.put("empty", "");
    assert!(sha256(&tree.root().join("empty")).is_err());
    let manifest = ReleaseManifest {
        format: 1,
        version: "0.1.2.3".into(),
        application_commit: "a".repeat(40),
        toolchain: "test".into(),
        dependencies: config(),
        artifacts: BTreeMap::new(),
    };
    assert!(
        collect(
            &tree.root().join("twice/a"),
            &tree.root().join("out"),
            manifest
        )
        .unwrap_err()
        .contains("missing")
    );
}

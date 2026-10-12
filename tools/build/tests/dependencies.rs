// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, release_config::tests::config, tree::tests::FixtureTree};

#[test]
fn application_lock_consumes_the_pinned_kernel_modeling_release() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    let pin = crate::release_config::load(tree.root()).unwrap().kernel;
    let names = locked_crates(&tree.root().join("Cargo.lock"), &pin).unwrap();
    assert_eq!(pin.version, "0.2.1");
    assert_eq!(pin.revision, "193572dc3665598c4948777c87375bad7dc8114a");
    for name in [
        "rustycad-tkprim",
        "rustycad-tkbrep",
        "rustycad-tkgeombase",
        "rustycad-tktopalgo",
    ] {
        assert!(names.iter().any(|locked| locked == name), "missing {name}");
    }
}

#[test]
fn published_archive_and_tag_must_both_match_the_pin() {
    let tree = FixtureTree::new();
    let mut dependency = config().api;
    dependency.revision = "a".repeat(40);
    let runner = FakeRunner::default();
    runner.replies.borrow_mut().extend([Ok(r#"{"draft":false,"prerelease":false,"assets":[{"name":"oblikovati-api-0.154.0.crate"}]}"#.into()), Ok(format!("{}\trefs/tags/v0.154.0\n", dependency.revision))]);
    verify_release(
        tree.root(),
        &runner,
        &dependency,
        &["oblikovati-api".into()],
    )
    .unwrap();
    assert_eq!(
        release_commit(
            "tagobject\trefs/tags/v1\ncommit\trefs/tags/v1^{}",
            "refs/tags/v1"
        )
        .unwrap(),
        "commit"
    );
    assert!(release_commit("", "missing").is_err());
    assert!(
        require_assets(
            &UpstreamRelease {
                draft: false,
                prerelease: false,
                assets: vec![]
            },
            &dependency,
            &["oblikovati-api".into()]
        )
        .is_err()
    );
    assert!(
        require_assets(
            &UpstreamRelease {
                draft: true,
                prerelease: false,
                assets: vec![]
            },
            &dependency,
            &[]
        )
        .is_err()
    );
    assert!(
        verify_release(
            tree.root(),
            &runner,
            &dependency,
            &["oblikovati-api".into()]
        )
        .is_err()
    );
}

#[test]
fn lockfile_cannot_substitute_a_local_or_different_dependency() {
    let tree = FixtureTree::new();
    let mut dependency = config().api;
    dependency.revision = "a".repeat(40);
    let source = format!(
        "git+https://github.com/{}.git?tag=v{}#{}",
        dependency.repository, dependency.version, dependency.revision
    );
    tree.put(
        "Cargo.lock",
        &format!(
            "[[package]]\nname = 'oblikovati-api'\nversion = '0.154.0'\nsource = '{source}'\n"
        ),
    );
    assert_eq!(
        locked_crates(&tree.root().join("Cargo.lock"), &dependency).unwrap(),
        ["oblikovati-api"]
    );
    let package = LockedPackage {
        name: "oblikovati-api".into(),
        version: "0.154.0".into(),
        source: None,
    };
    assert!(verify_locked(&package, &dependency).is_err());
    tree.put("Cargo.lock", "package = []");
    assert!(locked_crates(&tree.root().join("Cargo.lock"), &dependency).is_err());
    tree.put("Cargo.lock", "malformed");
    assert!(locked_crates(&tree.root().join("Cargo.lock"), &dependency).is_err());
    assert!(owned_package("rustycad-tkbrep", &config().kernel));
    assert!(!owned_package("serde", &config().kernel));
}

#[test]
fn application_lock_consumes_the_api_release_named_by_packaging() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    let pin = crate::release_config::load(tree.root()).unwrap().api;
    let names = locked_crates(&tree.root().join("Cargo.lock"), &pin).unwrap();
    assert_eq!(names, ["oblikovati-api"]);
    assert_eq!(pin.version, "0.158.0");
    assert_eq!(pin.revision, "32ff38398ab6d6a7df0ca838654d5b8bc8d8b9cd");
}

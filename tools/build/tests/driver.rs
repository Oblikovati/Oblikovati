// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn publication_is_blocked_before_build_or_network_activity() {
    let tree = FixtureTree::new();
    tree.put("release.toml", include_str!("../../../release.toml"));
    let runner = FakeRunner::default();
    assert!(
        preflight(tree.root(), &runner, true)
            .unwrap_err()
            .contains("R9")
    );
    assert!(runner.calls.borrow().is_empty());
    assert!(release_request(tree.root(), &runner, "nightly").is_err());
    assert!(verify_release(tree.root(), &runner, "0.1.2.3", tree.root()).is_err());
    assert!(
        package(
            tree.root(),
            &runner,
            "x86_64-unknown-linux-gnu",
            "0.1.2.3",
            true
        )
        .is_err()
    );
}

#[test]
fn released_dependency_preflight_builds_all_native_package_plans() {
    use crate::{
        platform::PLATFORMS,
        release_fixture::{ReleaseFixture, VERSION},
    };
    for platform in PLATFORMS {
        let fixture = ReleaseFixture::new(platform.target);
        let root = fixture.tree.root();
        preflight(root, &fixture, true).unwrap();
        package(root, &fixture, platform.target, VERSION, false).unwrap();
        let files = crate::tree::all_files(&root.join("dist")).unwrap();
        assert!(!files.is_empty(), "{}", platform.target);
        assert!(files.iter().all(|path| {
            path.file_name()
                .unwrap()
                .to_string_lossy()
                .starts_with("oblikovati-rust-")
        }));
        assert!(
            fixture
                .calls
                .borrow()
                .iter()
                .any(|call| call.starts_with("bash "))
        );
        assert!(
            !fixture
                .calls
                .borrow()
                .iter()
                .any(|call| call.contains("cargo test") || call.contains("cargo publish"))
        );
    }
}

#[test]
fn release_request_and_verified_artifacts_keep_exact_dependency_identity() {
    use crate::{
        platform::artifact_names,
        release_fixture::{ReleaseFixture, VERSION},
    };
    let fixture = ReleaseFixture::new("x86_64-unknown-linux-gnu");
    let root = fixture.tree.root();
    release_request(root, &fixture, "stable").unwrap();
    let request = crate::tree::read_text(&root.join("target/release-request.json")).unwrap();
    assert!(request.contains(VERSION));
    for name in artifact_names(VERSION).unwrap() {
        fixture
            .tree
            .put(&format!("packages/{name}"), "fixture final package");
    }
    collect(
        root,
        &fixture,
        VERSION,
        &root.join("packages"),
        &root.join("published"),
    )
    .unwrap();
    verify_release(root, &fixture, VERSION, &root.join("published")).unwrap();
    let mut changed = fixture.config.clone();
    changed.product.c_abi = "changed-abi".into();
    fixture
        .tree
        .put("release.toml", &toml::to_string(&changed).unwrap());
    assert!(
        verify_release(root, &fixture, VERSION, &root.join("published"))
            .unwrap_err()
            .contains("exact reviewed")
    );
}

#[test]
fn universal_app_plan_checks_merged_package_and_preserves_source_metadata() {
    use crate::{
        packaging::copy_tree,
        release_fixture::{ReleaseFixture, VERSION},
    };
    let fixture = ReleaseFixture::new("aarch64-apple-darwin");
    let root = fixture.tree.root();
    package(root, &fixture, fixture.platform.target, VERSION, false).unwrap();
    let package = root.join("target/package").join(fixture.platform.target);
    for suffix in ["macos-amd64", "macos-arm64"] {
        copy_tree(&package, &root.join("stage").join(suffix)).unwrap();
    }
    mac_bundle(root, &fixture, VERSION, false).unwrap();
    assert!(
        root.join("dist/Oblikovati Rust.app/Contents/Info.plist")
            .is_file()
    );
    assert!(
        fixture
            .calls
            .borrow()
            .iter()
            .any(|call| call.starts_with("lipo -verify_arch"))
    );
    assert!(
        mac_bundle(root, &fixture, VERSION, false)
            .unwrap_err()
            .contains("new app bundle")
    );
}

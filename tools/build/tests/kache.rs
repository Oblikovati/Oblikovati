// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};
#[test]
fn platform_assets_pin_both_matching_binaries_and_official_checksums() {
    let mut names = std::collections::BTreeSet::new();
    for os in ["linux", "macos", "windows"] {
        for arch in ["x86_64", "aarch64"] {
            let asset = kache_asset(os, arch).unwrap();
            assert!(names.insert(asset.name.clone()));
            assert_platform_asset(os, arch, &asset);
        }
    }
    assert!(
        kache_asset("other", "x86_64")
            .unwrap_err()
            .contains("other/x86_64")
    );
}
fn assert_platform_asset(os: &str, arch: &str, asset: &KacheAsset) {
    assert!(crate::release_config::hex_digest(&asset.digest, 64));
    assert!(asset.name.contains(arch));
    assert_eq!(asset.name.ends_with(".zip"), os == "windows");
    let binaries = if os == "windows" {
        ["kache.exe", "kache-engine.exe"]
    } else {
        ["kache", "kache-engine"]
    };
    assert_eq!(asset.binaries, binaries);
}
#[test]
fn extraction_is_limited_to_the_client_and_its_matching_engine() {
    let tree = FixtureTree::new();
    let runner = FakeRunner::default();
    let asset = kache_asset("windows", "aarch64").unwrap();
    extract_pair(tree.root(), &runner, &asset, &tree.root().join("tool.zip")).unwrap();
    let calls = runner.calls.borrow();
    assert_eq!(calls.len(), 1);
    assert_eq!(&calls[0][5..], &["kache.exe", "kache-engine.exe"]);
}

#[test]
fn pinned_pair_install_checks_the_archive_and_removes_only_that_download() {
    let tree = FixtureTree::new();
    tree.put(".tools/kache-fixture.tar.gz", "fixture");
    let archive = tree.root().join(".tools/kache-fixture.tar.gz");
    let asset = KacheAsset {
        name: "kache-fixture.tar.gz".into(),
        digest: crate::distribution::sha256(&archive).unwrap(),
        binaries: ["kache", "kache-engine"],
    };
    let runner = FakeRunner::default();
    install_archive(tree.root(), &runner, &asset).unwrap();
    assert!(!archive.exists());
    let calls = runner.calls.borrow();
    assert_eq!(calls[0][0], "curl");
    assert_eq!(calls[1][0], "tar");
    assert_eq!(&calls[1][5..], &["kache", "kache-engine"]);
}
#[test]
fn a_bad_archive_never_reaches_extraction() {
    let tree = FixtureTree::new();
    tree.put(".tools/kache-bad.tar.gz", "bad archive");
    let asset = KacheAsset {
        name: "kache-bad.tar.gz".into(),
        digest: "0".repeat(64),
        binaries: ["kache", "kache-engine"],
    };
    let runner = FakeRunner::default();
    let failure = install_archive(tree.root(), &runner, &asset).unwrap_err();
    assert!(failure.contains("SHA-256") && failure.contains(&asset.digest));
    assert_eq!(runner.calls.borrow().len(), 1);
}

#[cfg(unix)]
#[path = "kache/make_environment.rs"]
mod make_environment;

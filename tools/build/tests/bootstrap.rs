// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn tool_archives_are_pinned_for_all_six_hosts() {
    for os in ["linux", "macos", "windows"] {
        for arch in ["x86_64", "aarch64"] {
            let (_, checksum) = actionlint_asset(os, arch).unwrap();
            assert!(crate::release_config::hex_digest(checksum, 64));
        }
    }
    assert!(actionlint_asset("other", "x86_64").is_err());
}

#[test]
fn downloads_check_hashes_before_extraction() {
    let tree = FixtureTree::new();
    tree.put("archive", "fixture");
    let archive = tree.root().join("archive");
    let runner = FakeRunner::default();
    download(
        tree.root(),
        &runner,
        "https://example.test/tool",
        &sha256(&archive).unwrap(),
        &archive,
    )
    .unwrap();
    assert!(
        download(
            tree.root(),
            &runner,
            "https://example.test/tool",
            &"a".repeat(64),
            &archive
        )
        .is_err()
    );
    assert!(
        download(
            tree.root(),
            &runner,
            "http://example.test/tool",
            "",
            &archive
        )
        .is_err()
    );
}

#[derive(serde::Deserialize)]
struct CargoAliases {
    alias: std::collections::BTreeMap<String, String>,
}

#[test]
fn xtask_output_is_separate_from_the_workspace_it_builds() {
    let config: CargoAliases = toml::from_str(include_str!("../../../.cargo/config.toml")).unwrap();
    let arguments: Vec<&str> = config.alias["xtask"].split_whitespace().collect();
    let output = arguments
        .windows(2)
        .find(|pair| pair[0] == "--target-dir")
        .unwrap()[1];
    let output = std::path::Path::new(output);
    assert!(
        output.starts_with("target"),
        "bootstrap artifacts must remain ignored"
    );
    assert_ne!(
        output,
        std::path::Path::new("target"),
        "Windows locks the running executable"
    );
    assert!(
        arguments
            .windows(2)
            .any(|pair| pair == ["--package", "oblikovati-build"])
    );
}

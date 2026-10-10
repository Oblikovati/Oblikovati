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

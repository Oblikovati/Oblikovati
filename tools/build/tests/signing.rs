// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn apple_rejection_stops_publication_and_nested_code_is_signed_first() {
    let tree = FixtureTree::new();
    tree.put("app/Contents/MacOS/oblikovati-cli", "cli");
    tree.put("app/Contents/Frameworks/libMoltenVK.dylib", "dylib");
    let runner = FakeRunner::default();
    let identity = SigningIdentity {
        developer_id: "Developer ID".into(),
        keychain: "temporary.keychain".into(),
    };
    sign_app(
        tree.root(),
        &runner,
        &tree.root().join("app"),
        &tree.root().join("app.zip"),
        &identity,
    )
    .unwrap();
    let calls = runner.calls.borrow();
    assert!(calls[0].last().unwrap().ends_with("libMoltenVK.dylib"));
    assert!(calls[5].contains(&"notarytool".into()));
    assert!(calls[6].contains(&"staple".into()));
    drop(calls);
    runner
        .replies
        .borrow_mut()
        .push_back(Err("Apple rejected".into()));
    assert!(
        sign_app(
            tree.root(),
            &runner,
            &tree.root().join("app"),
            &tree.root().join("app.zip"),
            &identity
        )
        .is_err()
    );
}

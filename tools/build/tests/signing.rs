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

#[test]
fn standalone_gui_and_cli_executables_are_both_signed() {
    let tree = FixtureTree::new();
    tree.put("bin/oblikovati", "gui");
    tree.put("bin/oblikovati-cli", "cli");
    let runner = FakeRunner::default();
    let identity = SigningIdentity {
        developer_id: "Developer ID".into(),
        keychain: "temporary.keychain".into(),
    };
    sign_tree(tree.root(), &runner, &tree.root().join("bin"), &identity).unwrap();
    let calls = runner.calls.borrow();
    assert_eq!(calls.len(), 2, "expected GUI and CLI signatures: {calls:?}");
    assert!(
        calls
            .iter()
            .any(|call| call.last().unwrap().ends_with("oblikovati"))
    );
    assert!(
        calls
            .iter()
            .any(|call| call.last().unwrap().ends_with("oblikovati-cli"))
    );
}

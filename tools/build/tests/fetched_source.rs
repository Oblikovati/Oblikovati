// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

fn source_record(name: &str) -> serde_json::Value {
    serde_json::json!({"name": name, "version": "0.7.0", "source": "git+https://example.test/sdk#reviewed", "manifest_path": "/sdk/crates/toolkit/Cargo.toml"})
}

#[test]
fn fetched_source_is_selected_through_locked_cargo_metadata() {
    let root = FixtureTree::new();
    let runner = FakeRunner::default();
    let output = serde_json::json!({"packages": [source_record("other"), source_record("wanted")]})
        .to_string();
    runner.replies.borrow_mut().push_back(Ok(output));
    let package = locked_source_package(root.root(), &runner, "wanted").unwrap();
    assert_eq!(package.name, "wanted");
    assert_eq!(
        package.manifest_path,
        Path::new("/sdk/crates/toolkit/Cargo.toml")
    );
    assert!(runner.calls.borrow()[0].contains(&"--locked".into()));
}

#[test]
fn missing_ambiguous_and_malformed_metadata_are_named_declines() {
    for output in [
        "invalid".into(),
        serde_json::json!({"packages": []}).to_string(),
        serde_json::json!({"packages": [source_record("wanted"), source_record("wanted")]})
            .to_string(),
    ] {
        let message = unique_source_package(&output, "wanted").unwrap_err();
        assert!(message.contains("wanted") && message.contains("expected"));
    }
}

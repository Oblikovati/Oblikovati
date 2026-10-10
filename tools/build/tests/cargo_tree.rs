// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

pub(crate) fn package(tree: &FixtureTree) -> Package {
    Package {
        id: "fixture".into(),
        name: "fixture".into(),
        manifest_path: tree.root().join("Cargo.toml"),
        license: Some("GPL-2.0-only".into()),
        targets: vec![],
        dependencies: vec![],
    }
}

#[test]
fn separate_workspaces_are_discovered_and_member_manifests_deduplicate() {
    let tree = FixtureTree::new();
    for path in [
        "Cargo.toml",
        "Cargo.lock",
        "member/Cargo.toml",
        "fixtures/addin/Cargo.toml",
        "fixtures/addin/Cargo.lock",
    ] {
        tree.put(path, "");
    }
    let runner = FakeRunner::default();
    for relative in ["", "fixtures/addin", ""] {
        runner.replies.borrow_mut().push_back(Ok(serde_json::json!({"workspace_root": tree.root().join(relative), "workspace_members": [], "packages": []}).to_string()));
    }
    assert_eq!(inventory(tree.root(), &runner).unwrap().len(), 2);
    assert_eq!(runner.calls.borrow().len(), 3);
    assert!(
        runner
            .calls
            .borrow()
            .iter()
            .all(|command| command.contains(&"--locked".into()))
    );
}

#[test]
fn missing_manifests_lockfiles_and_bad_metadata_fail() {
    let tree = FixtureTree::new();
    let runner = FakeRunner::default();
    assert!(inventory(tree.root(), &runner).is_err());
    tree.put("Cargo.toml", "");
    assert!(
        inventory(tree.root(), &runner)
            .unwrap_err()
            .contains("metadata")
    );
    let mut workspace = Metadata {
        workspace_root: tree.root().into(),
        workspace_members: vec![],
        packages: vec![],
    };
    assert!(
        validate_workspace(tree.root(), &workspace)
            .unwrap_err()
            .contains("lockfile")
    );
    workspace.workspace_root = tree.root().parent().unwrap().into();
    assert!(validate_workspace(tree.root(), &workspace).is_err());
}

#[test]
fn dependency_boundaries_and_head_display_are_required() {
    let tree = FixtureTree::new();
    tree.put(
        "Cargo.toml",
        "[package.metadata.oblikovati]\nrequires-display = true\n",
    );
    let mut member = package(&tree);
    assert!(requires_display(&member).unwrap());
    member.dependencies.push(Dependency {
        name: "rustycad-tkbrep".into(),
        source: Some("git+https://example.test/kernel".into()),
        path: None,
    });
    assert!(
        validate_package(tree.root(), &member)
            .unwrap_err()
            .contains("boundary")
    );
    member.name = "oblikovati-martensite".into();
    assert!(validate_package(tree.root(), &member).is_ok());
    member.dependencies[0].source = None;
    assert!(validate_package(tree.root(), &member).is_err());
    member.dependencies[0].path = Some(tree.root().parent().unwrap().into());
    assert!(
        validate_package(tree.root(), &member)
            .unwrap_err()
            .contains("sibling")
    );
    member.license = None;
    assert!(
        validate_package(tree.root(), &member)
            .unwrap_err()
            .contains("license")
    );
}

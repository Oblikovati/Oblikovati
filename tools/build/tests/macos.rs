// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn dependency_paths_keep_spaces_and_distinguish_system_libraries() {
    assert_eq!(
        linked_paths("binary:\n  /tmp/a b.dylib (compatibility version 1.0.0)\n"),
        ["/tmp/a b.dylib"]
    );
    assert!(!bundled_dependency("/usr/lib/libSystem.B.dylib"));
    assert!(!bundled_dependency(
        "/System/Library/Frameworks/Metal.framework/Metal"
    ));
    assert!(bundled_dependency("/opt/homebrew/lib/libvulkan.1.dylib"));
    let info = plist("0.0015400.1.0-nightly.26101012", "1791633600").unwrap();
    assert!(info.contains("org.oblikovati.rust"));
    assert!(info.contains("OblikovatiProductVersion"));
    assert!(plist("0.1.2.3", "bad").is_err());
    assert!(plist("0.1.2.3", "999999999999").is_err());
}

#[test]
fn universal_merge_refuses_missing_architecture_or_different_resources() {
    let tree = FixtureTree::new();
    let runner = FakeRunner::default();
    for architecture in ["intel", "arm"] {
        tree.put(&format!("{architecture}/bin/oblikovati-head"), architecture);
        tree.put(&format!("{architecture}/LICENSE"), "GPL");
    }
    merge(
        tree.root(),
        &runner,
        &tree.root().join("intel"),
        &tree.root().join("arm"),
        &tree.root().join("universal"),
    )
    .unwrap();
    assert_eq!(runner.calls.borrow().len(), 2);
    tree.put("arm/LICENSE", "different");
    assert!(
        merge(
            tree.root(),
            &runner,
            &tree.root().join("intel"),
            &tree.root().join("arm"),
            &tree.root().join("out")
        )
        .is_err()
    );
    tree.put("arm/lib/extra.dylib", "extra");
    assert!(
        merge(
            tree.root(),
            &runner,
            &tree.root().join("intel"),
            &tree.root().join("arm"),
            &tree.root().join("out")
        )
        .unwrap_err()
        .contains("inventories")
    );
}

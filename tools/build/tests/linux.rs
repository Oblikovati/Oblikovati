// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn linux_packages_bundle_dependencies_but_use_host_glibc() {
    let tree = FixtureTree::new();
    tree.put("bin/app", "executable");
    tree.put("provided/libvulkan.so.1", "loader");
    let runner = FakeRunner::default();
    runner.replies.borrow_mut().push_back(Ok(format!(
        "libvulkan.so.1 => {} (0x1234)\nlibc.so.6 => /lib/libc.so.6 (0x5678)\n",
        tree.root().join("provided/libvulkan.so.1").display()
    )));
    bundle(tree.root(), &runner, tree.root()).unwrap();
    assert!(tree.root().join("lib/libvulkan.so.1").is_file());
    assert!(!tree.root().join("lib/libc.so.6").exists());
    assert!(linked_libraries("missing.so => not found").is_err());
    assert!(linked_libraries("relative.so => relative/path (0x123)").is_err());
    assert!(
        linked_libraries("linux-vdso.so.1 (0x1234)")
            .unwrap()
            .is_empty()
    );
}

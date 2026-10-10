// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{platform::platform, process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn non_system_dlls_are_bundled_recursively_and_missing_ones_fail() {
    let tree = FixtureTree::new();
    tree.put("stage/bin/app.exe", "executable");
    let platform = platform("x86_64-pc-windows-msvc").unwrap();
    tree.put("target/x86_64-pc-windows-msvc/release/gui.dll", "DLL");
    let runner = FakeRunner::default();
    runner.replies.borrow_mut().extend([
        Ok("sysroot".into()),
        Ok("Name: KERNEL32.dll\nName: gui.dll\n".into()),
        Ok("Name: gui.dll".into()),
    ]);
    bundle(tree.root(), &runner, platform, &tree.root().join("stage")).unwrap();
    assert!(tree.root().join("stage/bin/gui.dll").is_file());
    assert!(!system_dll("vcruntime140.dll"));
    assert!(system_dll("api-ms-win-core.dll"));
    assert!(system_dll("vulkan-1.dll"));
    runner
        .replies
        .borrow_mut()
        .extend([Ok("sysroot".into()), Ok("Name: missing.dll".into())]);
    assert!(bundle(tree.root(), &runner, platform, &tree.root().join("stage")).is_err());
}

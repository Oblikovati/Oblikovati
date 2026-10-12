// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#[allow(dead_code)]
#[path = "../../addin-host/tests/typed_workflow/compiler.rs"]
mod compiler;
#[allow(dead_code)]
#[path = "../../addin-host/tests/support/resources.rs"]
mod resources;
use std::{
    path::{Path, PathBuf},
    process::Command,
};
#[test]
fn executable_starts_real_go_and_rust_addins_and_skips_a_broken_library() {
    let directory = tempfile::tempdir().unwrap();
    let selected = directory.path().join("selected addins");
    std::fs::create_dir(&selected).unwrap();
    let root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../addin-host/tests/typed_workflow");
    let compiler = compiler::WorkflowCompiler::from_fixture_root(&root);
    prepare_startup_libraries(&compiler, directory.path(), &selected);
    let output = Command::new(env!("CARGO_BIN_EXE_oblikovati-head"))
        .args(["--frames", "3"])
        .env("OBK_ADDINS_DIR", &selected)
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(directory.path().join("rust-startup.opd").is_file());
    assert!(directory.path().join("go-startup.opd").is_file());
}
fn prepare_startup_libraries(
    compiler: &compiler::WorkflowCompiler,
    directory: &Path,
    selected: &Path,
) {
    let rust = compiler.rust_workflow(&directory.join("rust-startup.opd"), true);
    let go = compiler.go_workflow(&directory.join("go-startup.opd"), true);
    std::fs::copy(rust, selected.join("a-rust.so")).unwrap();
    std::fs::copy(go, selected.join("b-go.so")).unwrap();
    std::fs::write(selected.join("0-broken.so"), b"not a shared library").unwrap();
}

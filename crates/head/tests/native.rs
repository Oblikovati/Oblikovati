// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use std::process::{Command, Output};
fn invoke(arguments: &[&str], display: Option<&str>) -> Output {
    let mut command = Command::new(env!("CARGO_BIN_EXE_oblikovati-head"));
    command.args(arguments);
    if let Some(display) = display {
        command.env("DISPLAY", display);
    }
    command.output().unwrap()
}
#[test]
fn native_head_smoke_and_library_root_complete_on_local_linux() {
    let summary =
        oblikovati_head::run_native(oblikovati_head::HeadOptions { frames: Some(3) }).unwrap();
    assert_eq!(summary.attempts, 3);
    assert!(summary.completed > 0);
    let output = invoke(&["--frames", "3"], None);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
}
#[test]
fn non_window_commands_and_argument_errors_need_no_display() {
    for arguments in [&["--help"][..], &["--version"][..]] {
        assert!(invoke(arguments, Some(":65530")).status.success());
    }
    assert!(
        !invoke(&["--frames", "bad"], Some(":65530"))
            .status
            .success()
    );
    let failed = invoke(&["--frames", "1"], Some(":65530"));
    assert!(!failed.status.success());
    assert!(!failed.stderr.is_empty());
}

#[test]
fn executable_opens_startup_paths_with_spaces_and_reports_missing_files() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("startup part.opd");
    let bytes = b"schemaVersion: 2\ndocumentType: 1\n";
    std::fs::write(&path, bytes).unwrap();
    let output = invoke(&["--frames", "1", path.to_str().unwrap()], None);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(std::fs::read(path).unwrap(), bytes);
    verify_missing_startup_path(directory.path());
}
fn verify_missing_startup_path(directory: &std::path::Path) {
    let missing = directory.join("missing.opd");
    let output = invoke(&["--frames=1", missing.to_str().unwrap()], None);
    assert!(!output.status.success());
    assert!(String::from_utf8_lossy(&output.stderr).contains("missing.opd"));
}

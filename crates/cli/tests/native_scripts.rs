// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use std::{
    path::Path,
    process::{Command, Output},
};
fn cli(arguments: &[&str]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .args(arguments)
        .output()
        .unwrap()
}
fn write_source(directory: &Path, source: &str) -> std::path::PathBuf {
    let path = directory.join("model.lua");
    std::fs::write(&path, source).unwrap();
    path
}
#[test]
fn executable_script_run_creates_a_real_native_document_that_info_can_read() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"oblikovati.documents.create{type="part",name="Lua native"}; oblikovati.parameters.add{name="width",expression="4 cm"}; print("done")"#,
    );
    let saved = directory.path().join("lua.opd");
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    verify_native_result(&output, &saved);
}
fn verify_native_result(output: &Output, saved: &Path) {
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("done"));
    assert!(saved.is_file());
    let info = cli(&["info", saved.to_str().unwrap()]);
    assert!(info.status.success());
    assert!(String::from_utf8_lossy(&info.stdout).contains("Lua native"));
}

#[test]
fn failed_script_returns_nonzero_and_never_writes_the_requested_save() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"oblikovati.documents.create{type="part",name="script part"}; error("intentional failure value")"#,
    );
    let saved = directory.path().join("failed.opd");
    verify_failed_native_save(&script, &saved);
}
fn verify_failed_native_save(script: &Path, saved: &Path) {
    let output = cli(&[
        "script",
        "run",
        script.to_str().unwrap(),
        "--save",
        saved.to_str().unwrap(),
    ]);
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr).contains("intentional failure value"),
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(!saved.exists());
}

#[test]
fn native_source_io_refuses_missing_non_utf8_and_oversize_files() {
    verify_invalid_native_sources();
}
fn verify_invalid_native_sources() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("missing.lua");
    verify_source_decline(&path);
    std::fs::write(&path, [0xff]).unwrap();
    verify_source_decline(&path);
    std::fs::write(&path, vec![b' '; (1 << 20) + 1]).unwrap();
    verify_source_decline(&path);
}

fn verify_source_decline(path: &Path) {
    assert!(
        !cli(&["script", "run", path.to_str().unwrap()])
            .status
            .success()
    );
}

#[test]
fn executable_lua_reads_physical_and_visual_catalogs_without_a_document() {
    let directory = tempfile::tempdir().unwrap();
    let script = write_source(
        directory.path(),
        r#"local m=oblikovati.materials.list().materials;assert(#m==126);local a=oblikovati.appearances.list().appearances;assert(#a==94);local s=oblikovati.materials.get{id='steel'};assert(s.isotropyClass=='isotropic');assert(s.density==7.87);local v=oblikovati.appearances.get{id=s.appearanceId};assert(v.id==s.appearanceId);assert(#oblikovati.documents.list().documents==0);print('catalog-ready')"#,
    );
    let output = cli(&["script", "run", script.to_str().unwrap()]);
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(String::from_utf8_lossy(&output.stdout).contains("catalog-ready"));
}

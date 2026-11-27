// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::{
    load::{NativePackageReader, read_parameter_package},
    package::DocumentPackage,
};
use std::{
    path::Path,
    process::{Command, Output},
};

fn create(kind: &str, path: &Path, flags: &[&str]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg("new")
        .arg(kind)
        .arg(path)
        .args(flags)
        .output()
        .unwrap()
}
fn inspect(path: &Path) -> serde_json::Value {
    let result = Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg("info")
        .arg(path)
        .arg("--json")
        .output()
        .unwrap();
    assert!(result.status.success(), "{:?}", result);
    serde_json::from_slice(&result.stdout).unwrap()
}
#[test]
fn native_creation_and_info_round_trip_all_four_document_kinds() {
    let directory = tempfile::tempdir().unwrap();
    for (kind, extension) in [
        ("part", ".opd"),
        ("assembly", ".oad"),
        ("drawing", ".odd"),
        ("presentation", ".ord"),
    ] {
        let result = create(kind, &directory.path().join("Part Ω"), &[]);
        assert!(result.status.success(), "{result:?}");
        assert!(result.stderr.is_empty());
        let destination = directory.path().join(format!("Part Ω{extension}"));
        let manifest = inspect(&destination);
        assert_eq!(manifest["type"], kind);
        assert_eq!(manifest["name"], "Part Ω");
        assert_full_load(&destination);
    }
}
fn assert_full_load(path: &Path) {
    let loaded = read_parameter_package(
        path,
        "test",
        &mut SystemGuidSource,
        &mut NativePackageReader,
    )
    .unwrap();
    assert_eq!(
        loaded
            .package
            .identity_record()
            .unwrap()
            .unwrap()
            .save_counter,
        1
    );
    assert!(loaded.work_geometry.program.features().is_empty());
}
#[test]
fn native_save_failure_and_seed_refusal_have_nonzero_exit_without_success_output() {
    let directory = tempfile::tempdir().unwrap();
    let missing = create("part", &directory.path().join("missing/Part"), &[]);
    assert_eq!(missing.status.code(), Some(1));
    assert!(missing.stdout.is_empty());
    assert!(!missing.stderr.is_empty());
    let seeded = create("part", &directory.path().join("seeded"), &["--seed"]);
    assert_eq!(seeded.status.code(), Some(1));
    assert!(seeded.stdout.is_empty());
    assert!(!directory.path().join("seeded.opd").exists());
}
#[test]
fn native_non_utf8_filename_keeps_its_bytes_through_atomic_save() {
    use std::{ffi::OsString, os::unix::ffi::OsStringExt};
    let directory = tempfile::tempdir().unwrap();
    let requested = directory
        .path()
        .join(OsString::from_vec(b"part-\xff".to_vec()));
    let result = create("part", &requested, &[]);
    assert!(result.status.success(), "{result:?}");
    let destination = directory
        .path()
        .join(OsString::from_vec(b"part-\xff.opd".to_vec()));
    assert_full_load(&destination);
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
}
#[test]
fn new_replaces_old_bytes_with_one_complete_saved_document() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("replace.OPD");
    std::fs::write(&destination, b"old package").unwrap();
    let result = create("part", &destination, &[]);
    assert!(result.status.success(), "{result:?}");
    let bytes = std::fs::read(&destination).unwrap();
    assert!(DocumentPackage::decode(&bytes).is_ok());
    assert_full_load(&destination);
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
}

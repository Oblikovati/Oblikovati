// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_persistence::package::DocumentPackage;
use std::{
    ffi::OsStr,
    path::Path,
    process::{Command, Output},
};

fn command(verb: &str, operands: &[&OsStr]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_oblikovati-cli"))
        .arg(verb)
        .args(operands)
        .output()
        .unwrap()
}
fn successful(verb: &str, operands: &[&OsStr]) -> String {
    let result = command(verb, operands);
    assert!(result.status.success(), "{result:?}");
    assert!(result.stderr.is_empty());
    String::from_utf8(result.stdout).unwrap()
}
fn package(path: &Path) -> DocumentPackage {
    DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap()
}
#[test]
fn native_new_open_save_as_reopen_preserves_kind_and_saved_identity() {
    let directory = tempfile::tempdir().unwrap();
    for (kind, extension) in [
        ("part", ".opd"),
        ("assembly", ".oad"),
        ("drawing", ".odd"),
        ("presentation", ".ord"),
    ] {
        assert_native_round_trip(directory.path(), kind, extension);
    }
}
fn assert_native_round_trip(directory: &Path, kind: &str, extension: &str) {
    let source = directory.join(format!("source{extension}"));
    successful("new", &[OsStr::new(kind), source.as_os_str()]);
    let before = std::fs::read(&source).unwrap();
    let opened = successful("open", &[source.as_os_str()]);
    assert!(opened.starts_with(&format!("opened {kind} \"source\"")));
    let requested = directory.join("copy Ω");
    successful("save-as", &[source.as_os_str(), requested.as_os_str()]);
    let destination = directory.join(format!("copy Ω{extension}"));
    assert!(successful("open", &[destination.as_os_str()]).contains("\"copy Ω\""));
    assert_eq!(std::fs::read(&source).unwrap(), before);
    assert_saved_transition(&package(&source), &package(&destination));
}
fn assert_saved_transition(source: &DocumentPackage, copied: &DocumentPackage) {
    assert_eq!(
        copied.manifest().document_type,
        source.manifest().document_type
    );
    let before = source.identity_record().unwrap().unwrap();
    let after = copied.identity_record().unwrap().unwrap();
    assert_eq!(after.internal_name, before.internal_name);
    assert_eq!(after.save_counter, before.save_counter + 1);
    assert_ne!(after.revision_id, before.revision_id);
    assert_eq!(after.database_revision_id, before.database_revision_id);
    assert_eq!(after.model_digest, before.model_digest);
}
#[test]
fn native_failed_reconstruction_keeps_an_existing_destination() {
    let directory = tempfile::tempdir().unwrap();
    let source = directory.path().join("bad.opd");
    let destination = directory.path().join("existing.opd");
    std::fs::write(
        &source,
        b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  features: [{legacy: true}]\n",
    )
    .unwrap();
    std::fs::write(&destination, b"old destination").unwrap();
    let result = command("save-as", &[source.as_os_str(), destination.as_os_str()]);
    assert_eq!(result.status.code(), Some(1));
    assert!(result.stdout.is_empty());
    assert!(
        String::from_utf8(result.stderr)
            .unwrap()
            .contains("legacy feature import")
    );
    assert_eq!(std::fs::read(&destination).unwrap(), b"old destination");
}
#[test]
fn native_non_utf8_source_and_destination_paths_keep_original_bytes() {
    use std::{ffi::OsString, os::unix::ffi::OsStringExt};
    let directory = tempfile::tempdir().unwrap();
    let source = directory
        .path()
        .join(OsString::from_vec(b"source-\xff.opd".to_vec()));
    let requested = directory
        .path()
        .join(OsString::from_vec(b"copy-\xfe".to_vec()));
    successful("new", &[OsStr::new("part"), source.as_os_str()]);
    successful("open", &[source.as_os_str()]);
    successful("save-as", &[source.as_os_str(), requested.as_os_str()]);
    let destination = directory
        .path()
        .join(OsString::from_vec(b"copy-\xfe.opd".to_vec()));
    successful("open", &[destination.as_os_str()]);
    assert_saved_transition(&package(&source), &package(&destination));
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 2);
}
#[test]
fn native_same_path_save_as_replaces_a_complete_package_once() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("part.OPD");
    successful("new", &[OsStr::new("part"), path.as_os_str()]);
    let before = package(&path);
    successful("save-as", &[path.as_os_str(), path.as_os_str()]);
    assert_saved_transition(&before, &package(&path));
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
}
#[test]
fn native_missing_parent_fails_save_without_changing_source() {
    let directory = tempfile::tempdir().unwrap();
    let source = directory.path().join("source.opd");
    successful("new", &[OsStr::new("part"), source.as_os_str()]);
    let before = std::fs::read(&source).unwrap();
    let destination = directory.path().join("missing/copy.opd");
    let result = command("save-as", &[source.as_os_str(), destination.as_os_str()]);
    assert_eq!(result.status.code(), Some(1));
    assert!(result.stdout.is_empty());
    assert!(!result.stderr.is_empty());
    assert_eq!(std::fs::read(&source).unwrap(), before);
    assert!(!destination.exists());
}

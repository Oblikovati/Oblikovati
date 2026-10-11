// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_cli::run_with_files;
use oblikovati_model::identity::{DocumentGuid, GuidSource, IdentityTransitionError};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::DocumentFiles,
    load::PackageFileReader,
    package::DocumentPackage,
};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    ffi::OsString,
    io::{self, Write},
    path::{Path, PathBuf},
    rc::Rc,
};
#[path = "new_command/fake.rs"]
mod fake;
use fake::{RecordedFiles, RecordedStore, SequenceGuids};

fn args(values: &[&str]) -> Vec<OsString> {
    values.iter().map(OsString::from).collect()
}
fn execute(
    values: &[&str],
    files: RecordedFiles,
    guids: &mut SequenceGuids,
) -> Result<Vec<u8>, String> {
    let mut output = Vec::new();
    run_with_files(&args(values), files, guids, &mut output)?;
    Ok(output)
}
fn saved(store: &Rc<RefCell<RecordedStore>>, path: &str) -> DocumentPackage {
    let bytes = store.borrow().files[Path::new(path)].clone();
    DocumentPackage::decode(&bytes).unwrap()
}
#[test]
fn all_document_kinds_have_complete_saved_identity_and_reopenable_recipes() {
    for (kind, extension, numeric) in document_cases() {
        let files = RecordedFiles::default();
        let store = files.store.clone();
        let output = execute(
            &["new", kind, "/fixtures/Bracket"],
            files,
            &mut SequenceGuids::default(),
        )
        .unwrap();
        let path = format!("/fixtures/Bracket{extension}");
        assert_eq!(
            String::from_utf8(output).unwrap(),
            format!("created {kind} \"Bracket\" at {path}\n")
        );
        assert_saved_package(&saved(&store, &path), numeric);
    }
}
fn assert_saved_package(package: &DocumentPackage, numeric: u32) {
    assert_eq!(package.manifest().document_type, numeric);
    let identity = package.identity_record().unwrap().unwrap();
    assert_eq!(identity.save_counter, 1);
    assert!(!identity.internal_name.is_empty() && !identity.revision_id.is_empty());
    assert!(!identity.database_revision_id.is_empty() && !identity.model_digest.is_empty());
    assert_eq!(identity.version_created, identity.version_saved);
    assert!(package.restore_document_parameters().is_ok());
    assert!(package.restore_work_geometry().is_ok());
}
#[test]
fn case_matching_dotfiles_and_wrong_extensions_follow_go_stamping() {
    for (input, expected) in [
        ("/fixtures/Gear.OPD", "/fixtures/Gear.OPD"),
        ("/fixtures/gear.old", "/fixtures/gear.old.opd"),
        ("/fixtures/.opd", "/fixtures/.opd"),
        ("/fixtures/.hidden", "/fixtures/.hidden.opd"),
    ] {
        let files = RecordedFiles::default();
        let store = files.store.clone();
        execute(
            &["new", " PART ", input],
            files,
            &mut SequenceGuids::default(),
        )
        .unwrap();
        assert!(store.borrow().files.contains_key(Path::new(expected)));
        assert_eq!(store.borrow().files.len(), 1);
    }
}
#[test]
fn rejected_arguments_and_seeded_parts_do_not_touch_storage_or_identity() {
    for values in [
        vec!["new"],
        vec!["new", "wrong", "file"],
        vec!["new", "part", "file", "extra"],
        vec!["new", "--seed", "part", "file"],
        vec!["new", "part", "file", "--bad"],
    ] {
        let files = RecordedFiles::default();
        let store = files.store.clone();
        let mut guids = SequenceGuids::default();
        assert!(execute(&values, files, &mut guids).is_err());
        assert!(store.borrow().events.is_empty());
        assert_eq!(guids.issued, 0);
    }
}
#[test]
fn ignored_seed_and_last_false_flag_match_go_behavior() {
    for values in [
        vec!["new", "assembly", "/fixtures/test", "--seed"],
        vec!["new", "--seed", "part", "/fixtures/test", "--seed=false"],
    ] {
        let output = execute(
            &values,
            RecordedFiles::default(),
            &mut SequenceGuids::default(),
        )
        .unwrap();
        if values.contains(&"assembly") {
            assert!(output.starts_with(b"note: --seed applies only to parts; ignored\n"));
        } else {
            assert!(output.starts_with(b"created part"));
        }
    }
}
#[test]
fn storage_failures_keep_old_bytes_and_never_report_creation() {
    for phase in ["path", "stage", "write", "sync", "commit"] {
        let files = RecordedFiles::with_failure(phase);
        let store = files.store.clone();
        store
            .borrow_mut()
            .files
            .insert("/fixtures/test.opd".into(), b"old package".to_vec());
        let mut output = Vec::new();
        let mut guids = SequenceGuids::default();
        let arguments = args(&["new", "part", "/fixtures/test"]);
        let error = run_with_files(&arguments, files, &mut guids, &mut output).unwrap_err();
        assert!(error.contains(phase), "{error}");
        assert!(output.is_empty());
        assert_old_file(&store);
    }
}
#[test]
fn identity_failure_at_creation_or_save_cannot_replace_an_existing_file() {
    for fail_at in [1, 2] {
        let files = RecordedFiles::default();
        let store = files.store.clone();
        store
            .borrow_mut()
            .files
            .insert("/fixtures/test.opd".into(), b"old package".to_vec());
        let mut guids = SequenceGuids {
            fail_at: Some(fail_at),
            ..Default::default()
        };
        assert!(execute(&["new", "part", "/fixtures/test"], files, &mut guids).is_err());
        assert_old_file(&store);
        assert!(!store.borrow().events.contains(&"stage"));
    }
}
struct RefusingNewOutput;
impl Write for RefusingNewOutput {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        Err(io::Error::other("CLI output closed"))
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}
#[test]
fn output_failure_after_save_keeps_a_valid_committed_package() {
    let files = RecordedFiles::default();
    let store = files.store.clone();
    let error = run_with_files(
        &args(&["new", "part", "/fixtures/test"]),
        files,
        &mut SequenceGuids::default(),
        &mut RefusingNewOutput,
    )
    .unwrap_err();
    assert!(error.contains("CLI output closed"));
    assert_saved_package(&saved(&store, "/fixtures/test.opd"), 1);
}
#[test]
fn readonly_dispatch_uses_the_injected_file_access() {
    let files = RecordedFiles::default();
    files.store.borrow_mut().files.insert(
        "/fixtures/test.opd".into(),
        b"schemaVersion: 2\ndocumentType: 1\n".to_vec(),
    );
    let store = files.store.clone();
    let output = execute(
        &["info", "/fixtures/test.opd", "--json"],
        files,
        &mut SequenceGuids::default(),
    )
    .unwrap();
    assert_eq!(
        serde_json::from_slice::<serde_json::Value>(&output).unwrap()["type"],
        "part"
    );
    assert_eq!(store.borrow().events, ["read"]);
}

fn document_cases() -> [(&'static str, &'static str, u32); 4] {
    [
        ("part", ".opd", 1),
        ("assembly", ".oad", 2),
        ("drawing", ".odd", 3),
        ("presentation", ".ord", 4),
    ]
}
fn assert_old_file(store: &Rc<RefCell<RecordedStore>>) {
    assert_eq!(
        store.borrow().files[Path::new("/fixtures/test.opd")],
        b"old package"
    );
}

#[cfg(target_os = "linux")]
#[test]
fn non_utf8_dash_prefixed_flags_are_refused_before_storage() {
    use std::os::unix::ffi::OsStringExt;
    let arguments = vec![
        "new".into(),
        "part".into(),
        OsString::from_vec(b"--bad\xff".to_vec()),
    ];
    let files = RecordedFiles::default();
    let store = files.store.clone();
    let mut guids = SequenceGuids::default();
    let mut output = Vec::new();
    let error = run_with_files(&arguments, files, &mut guids, &mut output).unwrap_err();
    assert!(error.contains("flag") && error.contains("expected --seed"));
    assert!(store.borrow().events.is_empty() && output.is_empty());
    assert_eq!(guids.issued, 0);
}

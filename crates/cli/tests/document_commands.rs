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

fn arguments(values: &[&str]) -> Vec<OsString> {
    values.iter().map(OsString::from).collect()
}
fn execute(
    values: &[&str],
    files: RecordedFiles,
    guids: &mut SequenceGuids,
) -> Result<Vec<u8>, String> {
    let mut output = Vec::new();
    run_with_files(&arguments(values), files, guids, &mut output)?;
    Ok(output)
}
fn put_source(files: &RecordedFiles, kind: &str, name: &str, model: &str) {
    let encoded = format!("schemaVersion: 2\ndocumentType: {kind}\ndisplayName: {name}\n{model}");
    files
        .store
        .borrow_mut()
        .files
        .insert("/source.opd".into(), encoded.into_bytes());
}
fn saved(store: &Rc<RefCell<RecordedStore>>, path: &str) -> DocumentPackage {
    DocumentPackage::decode(&store.borrow().files[Path::new(path)]).unwrap()
}
#[test]
fn open_reports_each_reconstructed_kind_without_writing_source() {
    for (numeric, kind) in [
        ("1", "part"),
        ("2", "assembly"),
        ("3", "drawing"),
        ("4", "presentation"),
    ] {
        assert_open_kind(numeric, kind);
    }
}
fn assert_open_kind(numeric: &str, kind: &str) {
    let files = RecordedFiles::default();
    put_source(&files, numeric, "Authored Ω", "");
    let store = files.store.clone();
    let before = store.borrow().files.clone();
    let output = execute(
        &["open", "/source.opd"],
        files,
        &mut SequenceGuids::default(),
    )
    .unwrap();
    assert_eq!(
        String::from_utf8(output).unwrap(),
        format!("opened {kind} \"Authored Ω\" from /source.opd\n")
    );
    assert_eq!(store.borrow().files, before);
    assert_eq!(store.borrow().events, ["canonical", "read"]);
}
#[test]
fn save_as_stamps_the_source_kind_and_retains_original_bytes() {
    for (numeric, extension) in [("1", ".opd"), ("2", ".oad"), ("3", ".odd"), ("4", ".ord")] {
        assert_save_as_kind(numeric, extension);
    }
}
fn assert_save_as_kind(numeric: &str, extension: &str) {
    let files = RecordedFiles::default();
    put_source(&files, numeric, "source", "");
    let store = files.store.clone();
    let before = store.borrow().files[Path::new("/source.opd")].clone();
    let output = execute(
        &["save-as", "/source.opd", "/renamed"],
        files,
        &mut SequenceGuids::default(),
    )
    .unwrap();
    let destination = format!("/renamed{extension}");
    assert_eq!(
        String::from_utf8(output).unwrap(),
        format!("saved \"renamed\" as {destination}\n")
    );
    let package = saved(&store, &destination);
    assert_eq!(package.manifest().display_name, "renamed");
    assert_eq!(store.borrow().files[Path::new("/source.opd")], before);
}
#[test]
fn save_as_keeps_authored_names_and_current_recipe_sections() {
    let files = RecordedFiles::default();
    put_source(
        &files,
        "1",
        "Authored Ω",
        "model:\n  parameters: [{name: width, kind: user, expression: '2'}]\n  features: []\n  vendorState: retained\ndata: {addin/state: AP8=}\nbodyNames: {face:key: name}\n",
    );
    let store = files.store.clone();
    let original = saved(&store, "/source.opd");
    execute(
        &["save-as", "/source.opd", "/copy.OPD"],
        files,
        &mut SequenceGuids::default(),
    )
    .unwrap();
    let copied = saved(&store, "/copy.OPD");
    assert_retained_sections(&copied, &original);
    assert_parameter_recipe(&copied);
}
fn assert_retained_sections(copied: &DocumentPackage, original: &DocumentPackage) {
    assert_eq!(copied.manifest().display_name, "Authored Ω");
    assert_eq!(copied.streams(), original.streams());
    assert_eq!(
        copied.section_yaml("bodyNames").unwrap(),
        original.section_yaml("bodyNames").unwrap()
    );
    assert!(
        String::from_utf8(copied.section_yaml("model").unwrap().unwrap())
            .unwrap()
            .contains("vendorState: retained")
    );
}
fn assert_parameter_recipe(copied: &DocumentPackage) {
    let restored = copied.restore_document_parameters().unwrap();
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
}

#[test]
fn invalid_operand_counts_do_not_access_files_or_issue_identity() {
    for values in [
        vec!["open"],
        vec!["open", "a", "b"],
        vec!["save-as"],
        vec!["save-as", "a"],
        vec!["save-as", "a", "b", "c"],
    ] {
        let files = RecordedFiles::default();
        let store = files.store.clone();
        let mut guids = SequenceGuids::default();
        let error = execute(&values, files, &mut guids).unwrap_err();
        assert!(error.contains("expected") && error.contains("operand"));
        assert!(store.borrow().events.is_empty());
        assert_eq!(guids.issued, 0);
    }
}
#[test]
fn malformed_recipes_are_refused_before_identity_or_destination_write() {
    for model in [
        "model:\n  parameters: [{name: bad, kind: user, expression: '('}]\n",
        "model:\n  workPlanes: {version: 999}\n",
        "model:\n  features: [{legacy: retained}]\n",
    ] {
        let files = RecordedFiles::default();
        put_source(&files, "1", "source", model);
        let store = files.store.clone();
        let mut guids = SequenceGuids::default();
        assert!(execute(&["save-as", "/source.opd", "/copy"], files, &mut guids).is_err());
        assert_eq!(guids.issued, 0);
        assert_eq!(store.borrow().events, ["canonical", "read"]);
    }
}
#[test]
fn every_storage_failure_preserves_source_and_existing_destination() {
    for phase in [
        "canonical",
        "read",
        "path",
        "stage",
        "write",
        "sync",
        "commit",
    ] {
        assert_storage_failure(phase);
    }
}
fn assert_storage_failure(phase: &'static str) {
    let files = RecordedFiles::with_failure(phase);
    put_source(&files, "1", "source", "");
    let store = files.store.clone();
    store
        .borrow_mut()
        .files
        .insert("/copy.opd".into(), b"old destination".to_vec());
    let before = store.borrow().files.clone();
    let error = execute(
        &["save-as", "/source.opd", "/copy"],
        files,
        &mut SequenceGuids::default(),
    )
    .unwrap_err();
    assert!(error.contains(phase), "{phase}: {error}");
    assert_eq!(store.borrow().files, before);
}
#[test]
fn identity_failures_do_not_start_atomic_storage() {
    for fail_at in [1, 2, 3] {
        let files = RecordedFiles::default();
        put_source(&files, "1", "source", "");
        let store = files.store.clone();
        let before = store.borrow().files.clone();
        let mut guids = SequenceGuids {
            fail_at: Some(fail_at),
            ..Default::default()
        };
        assert!(execute(&["save-as", "/source.opd", "/copy"], files, &mut guids).is_err());
        assert_eq!(store.borrow().files, before);
        assert!(!store.borrow().events.contains(&"stage"));
    }
}
struct RefusingDocumentOutput;
impl Write for RefusingDocumentOutput {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        Err(io::Error::other("document output closed"))
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}
#[test]
fn reporting_failure_keeps_successfully_committed_save_as() {
    let files = RecordedFiles::default();
    put_source(&files, "1", "source", "");
    let store = files.store.clone();
    let error = run_with_files(
        &arguments(&["save-as", "/source.opd", "/copy"]),
        files,
        &mut SequenceGuids::default(),
        &mut RefusingDocumentOutput,
    )
    .unwrap_err();
    assert!(error.contains("document output closed"));
    let identity = saved(&store, "/copy.opd")
        .identity_record()
        .unwrap()
        .unwrap();
    assert_eq!(identity.save_counter, 1);
}
#[test]
fn missing_source_and_open_output_failure_propagate() {
    let missing = execute(
        &["open", "/missing.opd"],
        RecordedFiles::default(),
        &mut SequenceGuids::default(),
    )
    .unwrap_err();
    assert!(missing.contains("missing.opd"));
    let files = RecordedFiles::default();
    put_source(&files, "1", "source", "");
    let error = run_with_files(
        &arguments(&["open", "/source.opd"]),
        files,
        &mut SequenceGuids::default(),
        &mut RefusingDocumentOutput,
    )
    .unwrap_err();
    assert!(error.contains("document output closed"));
}

#[test]
fn unknown_document_kinds_cannot_open_or_write_a_destination() {
    for values in [
        vec!["open", "/source.opd"],
        vec!["save-as", "/source.opd", "/copy"],
    ] {
        let files = RecordedFiles::default();
        put_source(&files, "42", "source", "");
        let store = files.store.clone();
        let before = store.borrow().files.clone();
        let error = execute(&values, files, &mut SequenceGuids::default()).unwrap_err();
        assert!(error.contains("document type 42") && error.contains("expected Part"));
        assert_eq!(store.borrow().files, before);
        assert_eq!(store.borrow().events, ["canonical", "read"]);
    }
}

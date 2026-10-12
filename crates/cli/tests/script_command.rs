// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_cli::{ScriptSources, run_script_with_files};
use oblikovati_model::identity::{DocumentGuid, GuidSource, IdentityTransitionError};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::DocumentFiles,
    load::PackageFileReader,
};
use oblikovati_script::{IsolatedLuaEngine, WORKER_ARGUMENT};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    ffi::OsString,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};
#[path = "new_command/fake.rs"]
mod fake;
use fake::{RecordedFiles, SequenceGuids};
struct FakeScriptSources {
    text: String,
    reads: Vec<PathBuf>,
    fail: bool,
}
impl FakeScriptSources {
    fn new(text: &str) -> Self {
        Self {
            text: text.into(),
            reads: Vec::new(),
            fail: false,
        }
    }
}
impl ScriptSources for FakeScriptSources {
    fn read_source(&mut self, path: &Path, maximum: usize) -> Result<String, String> {
        self.reads.push(path.into());
        if self.fail {
            return Err(format!("source {path:?}: injected read failure"));
        }
        assert!(self.text.len() <= maximum);
        Ok(self.text.clone())
    }
}
fn run_script(args: &[&str], source: &str, files: RecordedFiles) -> Result<String, String> {
    let mut sources = FakeScriptSources {
        text: source.into(),
        reads: Vec::new(),
        fail: false,
    };
    execute(args, files, &mut sources)
}
fn execute(
    args: &[&str],
    files: RecordedFiles,
    sources: &mut FakeScriptSources,
) -> Result<String, String> {
    execute_with_guids(args, files, sources, &mut SequenceGuids::default())
}
fn execute_with_guids(
    args: &[&str],
    files: RecordedFiles,
    sources: &mut FakeScriptSources,
    guids: &mut SequenceGuids,
) -> Result<String, String> {
    let args: Vec<OsString> = args.iter().map(OsString::from).collect();
    let engine = IsolatedLuaEngine::new(
        Path::new(env!("CARGO_BIN_EXE_oblikovati-cli")),
        vec![WORKER_ARGUMENT.into()],
    );
    let mut output = Vec::new();
    run_script_with_files(&args, files, guids, &mut output, sources, &engine)?;
    Ok(String::from_utf8(output).unwrap())
}
#[test]
fn script_builds_edits_undoes_and_saves_through_the_existing_owner() {
    let files = RecordedFiles::default();
    let store = files.store.clone();
    let source = r#"oblikovati.documents.create{type="part",name="script part"}
oblikovati.parameters.add{name="width",expression="20 mm"}
oblikovati.parameters.set{name="width",expression="35 mm"}
oblikovati.transaction.undo{}
assert(oblikovati.parameters.get{name="width"}.expression=="20 mm")
oblikovati.transaction.redo{}
assert(oblikovati.parameters.get{name="width"}.expression=="35 mm")
print("built")"#;
    let output = run_script(
        &["run", "program.lua", "--save", "/models/built.opd"],
        source,
        files,
    )
    .unwrap();
    assert_eq!(
        output,
        "built\nsaved active document to /models/built.opd\n"
    );
    assert!(
        store
            .borrow()
            .files
            .contains_key(Path::new("/models/built.opd"))
    );
    assert!(store.borrow().events.contains(&"commit"));
}
#[test]
fn script_failure_keeps_explicit_saves_but_does_not_apply_the_save_flag() {
    let files = RecordedFiles::default();
    let store = files.store.clone();
    let source = r#"local doc=oblikovati.documents.create{type="part",name="script part"}
oblikovati.documents.saveAs{document=doc.id,newFullDocumentName="/models/committed.opd"}
error("later failure")"#;
    let error = run_script(
        &["run", "program.lua", "--save", "/models/after.opd"],
        source,
        files,
    )
    .unwrap_err();
    assert!(error.contains("later failure"));
    assert!(
        store
            .borrow()
            .files
            .contains_key(Path::new("/models/committed.opd"))
    );
    assert!(
        !store
            .borrow()
            .files
            .contains_key(Path::new("/models/after.opd"))
    );
}
#[test]
fn doc_flag_opens_before_source_and_save_flag_uses_the_reopened_owner() {
    let (store, mut guids, mut sources) = seed_script_document();
    let output = edit_reopened_script(store.clone(), &mut guids, &mut sources);
    assert!(output.contains("opened"));
    assert!(
        store
            .borrow()
            .files
            .contains_key(Path::new("/models/edited.opd"))
    );
}
fn seed_script_document() -> (
    Rc<RefCell<fake::RecordedStore>>,
    SequenceGuids,
    FakeScriptSources,
) {
    let files = RecordedFiles::default();
    let store = files.store.clone();
    let mut guids = SequenceGuids::default();
    let mut sources =
        FakeScriptSources::new(r#"oblikovati.documents.create{type="part",name="script part"}"#);
    execute_with_guids(
        &["run", "program.lua", "--save", "/models/input.opd"],
        files,
        &mut sources,
        &mut guids,
    )
    .unwrap();
    (store, guids, sources)
}
fn edit_reopened_script(
    store: Rc<RefCell<fake::RecordedStore>>,
    guids: &mut SequenceGuids,
    sources: &mut FakeScriptSources,
) -> String {
    let mut files = RecordedFiles::default();
    files.store = store;
    sources.text =
        r#"oblikovati.parameters.add{name="edited",expression="4 cm"}; print("opened")"#.into();
    execute_with_guids(
        &[
            "run",
            "program.lua",
            "--doc=/models/input.opd",
            "--save=/models/edited.opd",
        ],
        files,
        sources,
        guids,
    )
    .unwrap()
}

#[test]
fn malformed_operands_and_flags_fail_before_reading_source() {
    for args in [
        vec![],
        vec!["check"],
        vec!["run"],
        vec!["run", "--doc"],
        vec!["run", "file.lua", "--save"],
        vec!["run", "file.lua", "--unknown", "x"],
        vec!["run", "file.lua", "extra", "x"],
    ] {
        let mut sources = FakeScriptSources {
            text: String::new(),
            reads: Vec::new(),
            fail: false,
        };
        assert!(execute(&args, RecordedFiles::default(), &mut sources).is_err());
        assert!(sources.reads.is_empty());
    }
}
#[test]
fn source_open_and_save_failures_are_named_without_fake_success() {
    verify_source_read_failure();
    verify_owner_file_failures();
}
fn verify_source_read_failure() {
    let mut sources = FakeScriptSources::new("");
    sources.fail = true;
    let error = execute(
        &["run", "missing.lua"],
        RecordedFiles::default(),
        &mut sources,
    )
    .unwrap_err();
    assert!(error.contains("read failure"));
}
fn verify_owner_file_failures() {
    for flag in ["--doc", "--save"] {
        assert!(
            run_script(
                &["run", "program.lua", flag, "/models/missing.opd"],
                "",
                RecordedFiles::default()
            )
            .is_err()
        );
    }
    let source = r#"oblikovati.documents.create{type="part",name="script part"}"#;
    let failure = run_script(
        &["run", "program.lua", "--save", "/models/blocked.opd"],
        source,
        RecordedFiles::with_failure("write"),
    )
    .unwrap_err();
    assert!(failure.contains("write refused"));
}

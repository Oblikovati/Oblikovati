// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::{history::DocumentSnapshotTarget, identity::DocumentSaveTarget};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    package::{DocumentPackage, ParameterPackageTarget, SnapshotPackageError},
};
use std::{cell::RefCell, io, path::Path, rc::Rc};

#[derive(Default)]
struct MemoryFiles {
    contents: Rc<RefCell<Vec<u8>>>,
    stages: usize,
    refuse: bool,
    refuse_commit: bool,
}
struct MemoryStage {
    contents: Vec<u8>,
    destination: Rc<RefCell<Vec<u8>>>,
    refuse_commit: bool,
}
impl AtomicFiles for MemoryFiles {
    type Stage = MemoryStage;
    fn stage(&mut self, _: &Path) -> io::Result<Self::Stage> {
        self.stages += 1;
        if self.refuse {
            return Err(io::Error::other("storage unavailable"));
        }
        Ok(MemoryStage {
            contents: Vec::new(),
            destination: Rc::clone(&self.contents),
            refuse_commit: self.refuse_commit,
        })
    }
}
impl StagedFile for MemoryStage {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        self.contents = contents.to_vec();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn commit(self, _: &Path) -> io::Result<()> {
        if self.refuse_commit {
            return Err(io::Error::other("replacement refused"));
        }
        *self.destination.borrow_mut() = self.contents;
        Ok(())
    }
}

#[test]
fn failed_final_replacement_preserves_old_file_and_registered_identity() {
    let (mut workspace, id) = edited_workspace();
    let reserved = workspace
        .commands(id)
        .unwrap()
        .run("Reserve table", |_, ids| ids.allocate_derived_table())
        .unwrap();
    assert_eq!(reserved, 1);
    let mut package = package();
    let before = package.clone();
    let original = workspace.identity(id).unwrap().clone();
    let mut files = MemoryFiles {
        refuse_commit: true,
        ..Default::default()
    };
    *files.contents.borrow_mut() = b"old file".to_vec();
    let error = save_package(
        &mut workspace,
        id,
        &mut package,
        &mut SaveGuidSource::default(),
        &mut files,
    )
    .unwrap_err();
    assert!(error.to_string().contains("replacement refused"));
    assert_eq!(&*files.contents.borrow(), b"old file");
    assert_eq!(package, before);
    assert_eq!(workspace.identity(id).unwrap(), &original);
    assert!(workspace.needs_save(id).unwrap());
    assert!(!workspace.session(id).unwrap().history_entries()[0].saved);
    let next = workspace
        .commands(id)
        .unwrap()
        .run("Reserve next table", |_, ids| ids.allocate_derived_table())
        .unwrap();
    assert_eq!(next, 2);
}
struct SaveGuidSource {
    next: u64,
    fail: bool,
    calls: usize,
}
impl Default for SaveGuidSource {
    fn default() -> Self {
        Self {
            next: 1000,
            fail: false,
            calls: 0,
        }
    }
}
impl GuidSource for SaveGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.calls += 1;
        if self.fail {
            return Err(IdentityTransitionError::Randomness("no entropy".into()));
        }
        self.next += 1;
        Ok(guid(self.next))
    }
}
fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\ndata: {opaque: keep}\n").unwrap()
}
fn edited_workspace() -> (TestWorkspace, u64) {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let id = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    workspace
        .commands(id)
        .unwrap()
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    (workspace, id)
}
fn save_package(
    workspace: &mut TestWorkspace,
    id: u64,
    package: &mut DocumentPackage,
    source: &mut SaveGuidSource,
    files: &mut MemoryFiles,
) -> Result<(), WorkspaceSaveError<SnapshotPackageError>> {
    let mut target = ParameterPackageTarget::new(package, Path::new("part.obk"), files);
    workspace.save(id, "beta", source, &mut target)
}

#[test]
fn successful_save_adopts_same_recipe_identity_and_baseline_as_written() {
    let (mut workspace, id) = edited_workspace();
    let mut package = package();
    let retained = package.section_yaml("data").unwrap();
    let mut source = SaveGuidSource::default();
    let mut files = MemoryFiles::default();
    save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap();
    let loaded = DocumentPackage::decode(&files.contents.borrow()).unwrap();
    assert_eq!(loaded, package);
    assert!(workspace.session(id).unwrap().history_entries()[0].saved);
    assert_eq!(loaded.section_yaml("data").unwrap(), retained);
    let parameters = loaded.restore_document_parameters().unwrap().parameters;
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        2.0
    );
    assert_eq!(
        loaded.restore_parameter_properties(&parameters).unwrap(),
        workspace.session(id).unwrap().snapshot().properties
    );
    let current = workspace.identity(id).unwrap();
    assert_eq!(
        loaded
            .identity_record()
            .unwrap()
            .unwrap()
            .restore(current)
            .unwrap(),
        *current
    );
    assert!(!workspace.needs_save(id).unwrap());
    assert_eq!(
        workspace.observer().notices.last(),
        Some(&WorkspaceNotice::Saved(id))
    );
}

#[test]
fn undo_after_save_is_dirty_and_redo_returns_to_saved_baseline() {
    let (mut workspace, id) = edited_workspace();
    let mut package = package();
    let mut source = SaveGuidSource::default();
    let mut files = MemoryFiles::default();
    save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap();
    let saved = workspace.identity(id).unwrap().clone();
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(workspace.needs_save(id).unwrap());
    workspace.commands(id).unwrap().redo().unwrap();
    assert!(!workspace.needs_save(id).unwrap());
    save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap();
    assert_eq!(
        workspace.identity(id).unwrap().database_revision_id,
        saved.database_revision_id
    );
    assert_ne!(
        workspace.identity(id).unwrap().revision_id,
        saved.revision_id
    );
}

#[test]
fn failed_storage_preserves_package_identity_and_dirty_state_then_retries() {
    let (mut workspace, id) = edited_workspace();
    let mut package = package();
    let before = package.clone();
    let identity = workspace.identity(id).unwrap().clone();
    let notices = workspace.observer().notices.len();
    let mut source = SaveGuidSource::default();
    let mut files = MemoryFiles {
        refuse: true,
        ..Default::default()
    };
    let error =
        save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap_err();
    assert!(error.to_string().contains("storage unavailable"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(package, before);
    assert_eq!(workspace.identity(id).unwrap(), &identity);
    assert!(workspace.needs_save(id).unwrap());
    assert_eq!(workspace.observer().notices.len(), notices + 1);
    assert_eq!(
        workspace.observer().notices.last(),
        Some(&WorkspaceNotice::BeforeSave(id))
    );
    files.refuse = false;
    save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap();
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn preparation_failure_never_generates_identity_or_writes_a_file() {
    let (mut workspace, id) = edited_workspace();
    let mut malformed = DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\nmodel: {units: {workingScale: '-1'}}\n",
    )
    .unwrap();
    let mut source = SaveGuidSource::default();
    let mut files = MemoryFiles::default();
    let error =
        save_package(&mut workspace, id, &mut malformed, &mut source, &mut files).unwrap_err();
    assert!(matches!(error, WorkspaceSaveError::Preparation(_)));
    assert!(error.to_string().contains("-1"), "{error}");
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(source.calls, 0);
    assert_eq!(files.stages, 0);
    assert!(workspace.needs_save(id).unwrap());
}

#[test]
fn identity_failure_and_open_or_missing_documents_never_write() {
    let (mut workspace, id) = edited_workspace();
    let mut package = package();
    let mut source = SaveGuidSource {
        fail: true,
        ..Default::default()
    };
    let mut files = MemoryFiles::default();
    let error =
        save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap_err();
    assert!(error.to_string().contains("no entropy"));
    workspace.commands(id).unwrap().begin("preview").unwrap();
    let error =
        save_package(&mut workspace, id, &mut package, &mut source, &mut files).unwrap_err();
    assert!(matches!(error, WorkspaceSaveError::OpenTransaction(_)));
    assert!(error.to_string().contains("bounded transactions closed"));
    assert!(std::error::Error::source(&error).is_none());
    let error =
        save_package(&mut workspace, 99, &mut package, &mut source, &mut files).unwrap_err();
    assert!(error.to_string().contains("document 99"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(source.calls, 1);
    assert_eq!(files.stages, 0);
}

#[test]
fn collision_identity_is_clean_only_after_a_successful_save() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let _first = open(&mut workspace, 1, OpenActivation::Activate);
    let copy = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    assert!(workspace.needs_save(copy).unwrap());
    save_package(
        &mut workspace,
        copy,
        &mut package(),
        &mut SaveGuidSource::default(),
        &mut MemoryFiles::default(),
    )
    .unwrap();
    assert!(!workspace.needs_save(copy).unwrap());
    assert_eq!(workspace.owner(guid(99)), Some(copy));
}

#[test]
fn stale_property_preparation_clears_prior_candidate_and_cannot_persist() {
    let mut package = package();
    let mut files = MemoryFiles::default();
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("part.obk"), &mut files);
    target.prepare(&recipe(), 0, 1, Default::default()).unwrap();
    let mut stale = recipe();
    stale.properties.remove(USER_DEFINED, "width");
    let error = target
        .prepare(&stale, 0, 1, Default::default())
        .unwrap_err();
    assert!(matches!(error, SnapshotPackageError::StaleProperties));
    assert!(error.to_string().contains("synchronized properties"));
    assert!(std::error::Error::source(&error).is_none());
    let error = target.persist(&identity(1)).unwrap_err();
    assert!(matches!(error, SnapshotPackageError::Unprepared));
    assert!(
        error
            .to_string()
            .contains("successful snapshot preparation")
    );
    assert_eq!(files.stages, 0);
}

#[test]
fn save_veto_stops_identity_preparation_and_file_io() {
    let observer = RecordingWorkspace {
        deny_save: Some(1),
        ..Default::default()
    };
    let mut workspace = TestWorkspace::new(observer);
    let id = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let original = workspace.identity(id).unwrap().clone();
    let mut source = SaveGuidSource::default();
    let mut files = MemoryFiles::default();
    let error =
        save_package(&mut workspace, id, &mut package(), &mut source, &mut files).unwrap_err();
    assert!(error.to_string().contains("file policy locked"));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(source.calls, 0);
    assert_eq!(files.stages, 0);
    assert_eq!(workspace.identity(id).unwrap(), &original);
    assert_eq!(
        workspace.observer().notices.last(),
        Some(&WorkspaceNotice::BeforeSave(id))
    );
}

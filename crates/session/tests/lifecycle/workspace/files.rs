// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::{DocumentFiles, NativeDocumentFiles},
    load::PackageFileReader,
    package::DocumentPackage,
};
use oblikovati_session::files::{FileOpenOutcome, FileWorkspace};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};

#[derive(Default)]
struct WorkspaceFiles {
    contents: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
    aliases: BTreeMap<PathBuf, PathBuf>,
    reads: usize,
    fail_read: bool,
    fail_write: bool,
}
struct WorkspaceStage {
    path: PathBuf,
    bytes: Vec<u8>,
    files: Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>,
}
impl PackageFileReader for WorkspaceFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.reads += 1;
        if self.fail_read {
            return Err(io::Error::other("read denied"));
        }
        self.contents
            .borrow()
            .get(path)
            .cloned()
            .ok_or_else(|| io::Error::other("file absent"))
    }
}
impl AtomicFiles for WorkspaceFiles {
    type Stage = WorkspaceStage;
    fn stage(&mut self, path: &Path) -> io::Result<Self::Stage> {
        if self.fail_write {
            return Err(io::Error::other("write denied"));
        }
        Ok(WorkspaceStage {
            path: path.into(),
            bytes: Vec::new(),
            files: Rc::clone(&self.contents),
        })
    }
}
impl StagedFile for WorkspaceStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> {
        self.bytes = bytes.to_vec();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn commit(self, path: &Path) -> io::Result<()> {
        assert_eq!(path, self.path);
        self.files.borrow_mut().insert(self.path, self.bytes);
        Ok(())
    }
}
impl DocumentFiles for WorkspaceFiles {
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        Ok(self
            .aliases
            .get(requested)
            .cloned()
            .unwrap_or_else(|| requested.into()))
    }
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        if let Some(path) = self.aliases.get(requested) {
            return Ok(path.clone());
        }
        if self.contents.borrow().contains_key(requested) {
            return Ok(requested.into());
        }
        Err(io::Error::other("path absent"))
    }
}
struct SequenceGuids(u64);
impl GuidSource for SequenceGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.0 += 1;
        Ok(guid(self.0))
    }
}
type Host = FileWorkspace<RecordingObserver, RecordingWorkspace, WorkspaceFiles>;
fn path(name: &str) -> PathBuf {
    std::env::temp_dir()
        .join("oblikovati-file-fixtures")
        .join(name)
}
fn fixture(number: u64) -> Vec<u8> {
    format!("schemaVersion: 2\ndocumentType: 1\nidentity: {{internalName: {}}}\ndata: {{marker: AQID}}\nmodel:\n  parameters: [{{name: width, kind: user, expression: '1 cm', export: true}}]\n", guid(number)).into_bytes()
}
fn host(observer: RecordingWorkspace) -> Host {
    let files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    files
        .contents
        .borrow_mut()
        .insert(path("b.obk"), fixture(2));
    FileWorkspace::new(observer, files)
}
fn open_file(
    host: &mut Host,
    name: &str,
    source: &mut SequenceGuids,
    activation: OpenActivation,
) -> u64 {
    host.open(
        &path(name),
        RecordingObserver::default(),
        "alpha",
        source,
        activation,
    )
    .unwrap()
    .session_id()
}

#[test]
fn reopening_cached_path_keeps_unsaved_state_and_does_not_read_or_mint_ids() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let second = open_file(&mut host, "b.obk", &mut source, OpenActivation::Activate);
    host.commands(first)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let count = source.0;
    assert_eq!(
        open_file(&mut host, "a.obk", &mut source, OpenActivation::Background),
        first
    );
    assert_eq!(host.workspace().active_document(), Some(second));
    assert_eq!(host.files().reads, 2);
    assert_eq!(source.0, count);
    assert_eq!(
        width(host.workspace().session(first).unwrap().snapshot()),
        2.0
    );
    host.activate(first).unwrap();
    assert_eq!(host.workspace().active_document(), Some(first));
}

#[test]
fn aliases_resolve_to_one_live_document() {
    let mut files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    files
        .aliases
        .insert(PathBuf::from("alias.obk"), path("a.obk"));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let outcome = host
        .open(
            Path::new("alias.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap();
    assert_eq!(outcome, FileOpenOutcome::Existing(id));
    assert_eq!(host.path(id).unwrap(), path("a.obk"));
    assert_eq!(host.files().reads, 1);
}

#[test]
fn save_uses_only_the_bound_package_and_destination() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let _other = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    host.save(id, "beta", &mut source).unwrap();
    let files = host.files().contents.borrow();
    let written = DocumentPackage::decode(&files[&path("a.obk")]).unwrap();
    assert_eq!(
        written
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(files[&path("b.obk")], fixture(2));
    assert!(!host.workspace().needs_save(id).unwrap());
}

#[test]
fn malformed_recipe_claims_no_path_identity_or_session_and_can_retry() {
    let mut host = host(RecordingWorkspace::default());
    host.files().contents.borrow_mut().insert(
        path("a.obk"),
        b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: invalid}\n".to_vec(),
    );
    let mut source = SequenceGuids(1000);
    let error = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("invalid"), "{error}");
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(source.0, 1000);
    assert!(host.workspace().documents().is_empty());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    assert_eq!(
        open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate),
        1
    );
}

#[test]
fn open_veto_happens_before_reading_or_identity_generation() {
    let mut host = host(RecordingWorkspace {
        deny_open: true,
        ..Default::default()
    });
    let mut source = SequenceGuids(1000);
    let error = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("project locked"));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(host.files().reads, 0);
    assert_eq!(source.0, 1000);
    assert!(host.workspace().documents().is_empty());
}

#[test]
fn close_releases_path_and_reopen_gets_new_session_but_same_guid() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.close_discarding_changes(first).unwrap();
    assert!(host.path(first).is_err());
    assert!(host.workspace().owner(guid(1)).is_none());
    let next = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!(next > first);
    assert_eq!(
        host.workspace().identity(next).unwrap().internal_name,
        guid(1)
    );
}

#[test]
fn close_veto_keeps_the_cached_file_binding() {
    let mut host = host(RecordingWorkspace {
        deny_close: Some(1),
        ..Default::default()
    });
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!(host.close_discarding_changes(id).is_err());
    assert_eq!(
        open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate),
        id
    );
    assert_eq!(host.files().reads, 1);
}

#[test]
fn path_read_and_save_failures_preserve_ownership() {
    let mut files = WorkspaceFiles {
        fail_read: true,
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    files
        .aliases
        .insert(PathBuf::from("bad-alias"), PathBuf::from("relative"));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let relative = host
        .open(
            Path::new("bad-alias"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(relative.to_string().contains("absolute canonical"));
    let missing = host
        .open(
            &path("missing"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(missing.to_string().contains("path absent"));
    assert!(std::error::Error::source(&missing).is_some());
    let read = host
        .open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(read.to_string().contains("read denied"));
    assert!(host.workspace().documents().is_empty());
    assert!(
        host.save(9, "beta", &mut source)
            .unwrap_err()
            .to_string()
            .contains("owned file binding")
    );
}

#[test]
fn native_open_edit_save_close_and_reopen_uses_the_real_atomic_writer() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("part.obk");
    std::fs::write(&destination, fixture(1)).unwrap();
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .open(
            &destination,
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "4 cm")
        })
        .unwrap();
    host.save(id, "beta", &mut source).unwrap();
    let saved_identity = host.workspace().identity(id).unwrap().clone();
    host.close_discarding_changes(id).unwrap();
    let reopened = host
        .open(
            &destination,
            RecordingObserver::default(),
            "beta",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(
        width(host.workspace().session(reopened).unwrap().snapshot()),
        4.0
    );
    assert_eq!(
        host.workspace().identity(reopened).unwrap(),
        &saved_identity
    );
    assert!(!host.workspace().needs_save(reopened).unwrap());
}

#[test]
fn failed_bound_save_keeps_current_recipe_dirty_and_file_unchanged() {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let identity = host.workspace().identity(id).unwrap().clone();
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let error = host.save(id, "beta", &mut source).unwrap_err();
    assert!(error.to_string().contains("write denied"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(host.workspace().identity(id).unwrap(), &identity);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
    assert!(host.workspace().needs_save(id).unwrap());
}

#[test]
fn distinct_paths_with_one_guid_get_distinct_dirty_identity_owners() {
    let mut host = host(RecordingWorkspace::default());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("b.obk"), fixture(1));
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let outcome = host
        .open(
            &path("b.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Background,
        )
        .unwrap();
    assert!(matches!(
        outcome,
        FileOpenOutcome::Opened(RegisteredDocument {
            identity: IdentityRegistration::Reassigned { .. },
            ..
        })
    ));
    let second = outcome.session_id();
    assert_ne!(first, second);
    assert!(host.workspace().needs_save(second).unwrap());
    assert_eq!(host.workspace().owner(guid(1)), Some(first));
    assert_eq!(host.workspace().active_document(), Some(first));
}

#[path = "files/transitions.rs"]
mod transitions;

#[path = "files/copies.rs"]
mod copies;

#[path = "files/batch.rs"]
mod batch;

#[path = "files/create.rs"]
mod create;

#[path = "files/template.rs"]
mod template;

#[path = "files/project.rs"]
mod project;

#[path = "files/unbound.rs"]
mod unbound;

// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DOCUMENT_UNKNOWN};
use oblikovati_persistence::project::{ProjectFileLookup, ProjectFileProbe};

impl ProjectFileProbe for WorkspaceFiles {
    fn is_regular_file(&mut self, requested: &Path) -> io::Result<bool> {
        if self.fail_read {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "template metadata denied",
            ));
        }
        let resolved = self
            .aliases
            .get(requested)
            .map_or(requested, PathBuf::as_path);
        Ok(self.contents.borrow().contains_key(resolved))
    }
}

fn template_project() -> ProjectFileLookup {
    ProjectFileLookup::new(None, Vec::new(), Some(path("templates"))).unwrap()
}
fn new_metadata() -> NewFileMetadata {
    NewFileMetadata {
        file_name: path("new.obk").to_str().unwrap().into(),
        display_name: "From project".into(),
        ..Default::default()
    }
}

#[test]
fn project_default_template_creates_a_dirty_independent_document() {
    let mut host = host(RecordingWorkspace::default());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("templates/part.opd"), fixture(7));
    let mut source = SequenceGuids(1000);
    let created = host
        .create_from_project_template(
            &template_project(),
            DOCUMENT_PART,
            &new_metadata(),
            RecordingObserver::default(),
            "beta",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_eq!(host.files().reads, 1);
    assert!(host.workspace().needs_save(created).unwrap());
    assert_ne!(
        host.workspace().identity(created).unwrap().internal_name,
        guid(7)
    );
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("new.obk"))
    );
    host.save(created, "beta", &mut source).unwrap();
    let saved = DocumentPackage::decode(&host.files().contents.borrow()[&path("new.obk")]).unwrap();
    assert_eq!(saved.manifest().display_name, "From project");
    assert_eq!(
        host.files().contents.borrow()[&path("templates/part.opd")],
        fixture(7)
    );
}

#[test]
fn explicit_template_overrides_the_project_default_without_fallback_on_failure() {
    let mut host = host(RecordingWorkspace::default());
    let mut metadata = new_metadata();
    metadata.template_file = path("a.obk").to_str().unwrap().into();
    let created = host
        .create_from_project_template(
            &template_project(),
            DOCUMENT_PART,
            &metadata,
            RecordingObserver::default(),
            "beta",
            &mut SequenceGuids(1000),
        )
        .unwrap()
        .session_id;
    assert_eq!(host.workspace().documents(), [created]);
    host.close_discarding_changes(created).unwrap();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("templates/part.opd"), fixture(7));
    metadata.template_file = path("absent.obk").to_str().unwrap().into();
    assert!(
        host.create_from_project_template(
            &template_project(),
            DOCUMENT_PART,
            &metadata,
            RecordingObserver::default(),
            "beta",
            &mut SequenceGuids(1000)
        )
        .is_err()
    );
    assert!(host.workspace().documents().is_empty());
    assert_eq!(host.files().reads, 1);
}

#[test]
fn missing_template_and_unknown_kind_refuse_before_read_or_identity_generation() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    for kind in [DOCUMENT_PART, DOCUMENT_UNKNOWN] {
        let error = host
            .create_from_project_template(
                &template_project(),
                kind,
                &new_metadata(),
                RecordingObserver::default(),
                "beta",
                &mut source,
            )
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
    }
    assert_eq!(host.files().reads, 0);
    assert_eq!(source.0, 1000);
    assert!(host.workspace().observer().notices.is_empty());
}

#[test]
fn mismatched_template_kind_is_not_published() {
    let mut host = host(RecordingWorkspace::default());
    let metadata = NewFileMetadata {
        template_file: path("a.obk").to_str().unwrap().into(),
        ..new_metadata()
    };
    let mut source = SequenceGuids(1000);
    let error = host
        .create_from_project_template(
            &template_project(),
            DOCUMENT_ASSEMBLY,
            &metadata,
            RecordingObserver::default(),
            "beta",
            &mut source,
        )
        .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("template document type 1: expected requested document type 2")
    );
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(source.0, 1000);
    assert!(host.workspace().documents().is_empty());
}

#[test]
fn project_probe_errors_remain_visible_and_do_not_start_creation() {
    let files = WorkspaceFiles {
        fail_read: true,
        ..Default::default()
    };
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let error = host
        .create_from_project_template(
            &template_project(),
            DOCUMENT_PART,
            &new_metadata(),
            RecordingObserver::default(),
            "beta",
            &mut SequenceGuids(1000),
        )
        .unwrap_err();
    assert!(error.to_string().contains("template metadata denied"));
    assert!(std::error::Error::source(&error).is_some());
    assert!(host.workspace().documents().is_empty());
    assert_eq!(host.files().reads, 0);
}

#[test]
fn native_project_template_create_save_reopen_keeps_the_template_unchanged() {
    let directory = tempfile::tempdir().unwrap();
    let templates = directory.path().join("Templates Ω");
    std::fs::create_dir(&templates).unwrap();
    let template = templates.join("part.opd");
    std::fs::write(&template, fixture(7)).unwrap();
    let destination = directory.path().join("new.opd");
    let project = ProjectFileLookup::new(None, Vec::new(), Some(templates)).unwrap();
    let metadata = NewFileMetadata {
        file_name: destination.to_str().unwrap().into(),
        ..new_metadata()
    };
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .create_from_project_template(
            &project,
            DOCUMENT_PART,
            &metadata,
            RecordingObserver::default(),
            "beta",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert!(!destination.exists());
    host.close_saving_changes(id, "beta", &mut source).unwrap();
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
    assert!(!host.workspace().needs_save(reopened).unwrap());
    assert_eq!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .parameters
            .ids(),
        [1]
    );
    assert_eq!(std::fs::read(template).unwrap(), fixture(7));
}

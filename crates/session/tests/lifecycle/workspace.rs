// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::identity::{
    DocumentGuid, FileIdentity, GuidSource, IdentityRegistration, IdentityTransitionError,
};
use oblikovati_session::workspace::*;

#[derive(Debug, PartialEq)]
enum WorkspaceNotice {
    Created(u64, IdentityRegistration),
    PathChanged(u64, std::path::PathBuf, std::path::PathBuf),
    BeforeOpen(std::path::PathBuf),
    BeforeSave(u64),
    Saved(u64),
    Opened(u64, IdentityRegistration),
    Activated(u64),
    BeforeClose(u64),
    Closed(u64, Option<u64>),
}
#[derive(Default)]
struct RecordingWorkspace {
    present_clients: Vec<String>,
    interest_notices: Vec<(u64, String, Vec<String>)>,
    deny_open: bool,
    notices: Vec<WorkspaceNotice>,
    deny_close: Option<u64>,
    deny_save: Option<u64>,
}
impl WorkspaceObserver for RecordingWorkspace {
    fn interest_client_present(&self, client: &str) -> bool {
        self.present_clients.iter().any(|present| present == client)
    }
    fn unavailable_interest_clients(&mut self, document: u64, name: &str, clients: &[String]) {
        self.interest_notices
            .push((document, name.into(), clients.to_vec()));
    }

    fn created(&mut self, id: u64, registration: IdentityRegistration) {
        self.notices
            .push(WorkspaceNotice::Created(id, registration));
    }
    fn path_changed(&mut self, id: u64, previous: &std::path::Path, current: &std::path::Path) {
        self.notices.push(WorkspaceNotice::PathChanged(
            id,
            previous.into(),
            current.into(),
        ));
    }
    fn before_open(&mut self, path: &std::path::Path) -> Result<(), String> {
        self.notices.push(WorkspaceNotice::BeforeOpen(path.into()));
        if self.deny_open {
            return Err("project locked".into());
        }
        Ok(())
    }
    fn before_save(&mut self, id: u64, _: &DocumentSnapshot) -> Result<(), String> {
        self.notices.push(WorkspaceNotice::BeforeSave(id));
        if self.deny_save == Some(id) {
            return Err("file policy locked".into());
        }
        Ok(())
    }
    fn saved(&mut self, id: u64) {
        self.notices.push(WorkspaceNotice::Saved(id));
    }
    fn opened(&mut self, id: u64, registration: IdentityRegistration) {
        self.notices.push(WorkspaceNotice::Opened(id, registration));
    }
    fn activated(&mut self, id: u64) {
        self.notices.push(WorkspaceNotice::Activated(id));
    }
    fn before_close(&mut self, id: u64, _: &DocumentSnapshot) -> Result<(), String> {
        self.notices.push(WorkspaceNotice::BeforeClose(id));
        if self.deny_close == Some(id) {
            return Err("tool owns document".into());
        }
        Ok(())
    }
    fn closed(&mut self, id: u64, active: Option<u64>) {
        self.notices.push(WorkspaceNotice::Closed(id, active));
    }
}
struct WorkspaceGuidSource {
    result: Result<DocumentGuid, IdentityTransitionError>,
    calls: usize,
}
impl GuidSource for WorkspaceGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.calls += 1;
        self.result.clone()
    }
}
type TestWorkspace = WorkspaceSession<RecordingObserver, RecordingWorkspace>;
fn guid(number: u64) -> DocumentGuid {
    format!("00000000-0000-4000-8000-{number:012x}")
        .parse()
        .unwrap()
}
fn identity(number: u64) -> FileIdentity {
    FileIdentity {
        internal_name: guid(number),
        revision_id: Some(guid(100)),
        database_revision_id: Some(guid(101)),
        save_counter: 1,
        version_created: "alpha".into(),
        version_saved: "alpha".into(),
        model_digest: None,
    }
}
fn history_for_open() -> DocumentEditHistory {
    DocumentEditHistory::new(recipe(), RecipeIdAllocator::restore(Some(1), &[1]).unwrap()).unwrap()
}
fn open(
    workspace: &mut TestWorkspace,
    number: u64,
    activation: OpenActivation,
) -> RegisteredDocument {
    let mut source = WorkspaceGuidSource {
        result: Ok(guid(99)),
        calls: 0,
    };
    workspace
        .register(
            history_for_open(),
            identity(number),
            RecordingObserver::default(),
            &mut source,
            activation,
        )
        .unwrap()
}

#[test]
fn registration_order_background_focus_and_unknown_activation_are_explicit() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    assert!(matches!(
        workspace.active_commands(),
        Err(WorkspaceError::NoActive)
    ));
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let second = open(&mut workspace, 2, OpenActivation::Background).session_id;
    assert_eq!(workspace.documents(), [first, second]);
    assert_eq!(workspace.active_document(), Some(first));
    assert!(!workspace.needs_save(first).unwrap());
    let count = workspace.observer().notices.len();
    assert_eq!(workspace.activate(99), Err(WorkspaceError::Missing(99)));
    assert_eq!(workspace.observer().notices.len(), count);
    workspace.activate(second).unwrap();
    assert_eq!(workspace.active_document(), Some(second));
}

#[test]
fn named_transaction_does_not_follow_focus_and_document_histories_are_isolated() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let second = open(&mut workspace, 2, OpenActivation::Background).session_id;
    workspace
        .commands(first)
        .unwrap()
        .begin("first batch")
        .unwrap();
    workspace.activate(second).unwrap();
    workspace
        .active_commands()
        .unwrap()
        .run("second edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "3 cm")
        })
        .unwrap();
    workspace
        .commands(first)
        .unwrap()
        .run("first edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    workspace.commands(first).unwrap().end().unwrap();
    assert_eq!(
        workspace.session(first).unwrap().undo_labels(),
        ["first batch"]
    );
    assert_eq!(
        workspace.session(second).unwrap().undo_labels(),
        ["second edit"]
    );
    assert!(workspace.needs_save(first).unwrap());
    workspace.commands(first).unwrap().undo().unwrap();
    assert!(!workspace.needs_save(first).unwrap());
    assert_eq!(width(workspace.session(second).unwrap().snapshot()), 3.0);
    workspace.commands(first).unwrap().redo().unwrap();
    assert_eq!(width(workspace.session(first).unwrap().snapshot()), 2.0);
}

#[test]
fn aborted_named_group_restores_loaded_dirty_state() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let id = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    workspace.commands(id).unwrap().begin("discard").unwrap();
    workspace
        .commands(id)
        .unwrap()
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    assert!(workspace.needs_save(id).unwrap());
    workspace.commands(id).unwrap().abort().unwrap();
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn collision_requires_identity_save_even_after_undo() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let second = open(&mut workspace, 1, OpenActivation::Activate);
    assert_eq!(
        second.identity,
        IdentityRegistration::Reassigned {
            previous: guid(1),
            current: guid(99)
        }
    );
    assert_eq!(workspace.owner(guid(1)), Some(first));
    assert_eq!(workspace.owner(guid(99)), Some(second.session_id));
    assert_eq!(
        workspace.identity(second.session_id).unwrap().revision_id,
        Some(guid(100))
    );
    workspace
        .commands(second.session_id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    workspace
        .commands(second.session_id)
        .unwrap()
        .undo()
        .unwrap();
    assert!(workspace.needs_save(second.session_id).unwrap());
    assert!(
        workspace
            .observer()
            .notices
            .contains(&WorkspaceNotice::Opened(second.session_id, second.identity))
    );
}

#[test]
fn collision_failure_keeps_existing_ownership_and_consumes_no_reusable_session_id() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let mut source = WorkspaceGuidSource {
        result: Err(IdentityTransitionError::Randomness("offline".into())),
        calls: 0,
    };
    let error = workspace
        .register(
            history_for_open(),
            identity(1),
            RecordingObserver::default(),
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(error.to_string().contains("offline"));
    assert!(std::error::Error::source(&error).is_some());
    assert_eq!(source.calls, 1);
    assert_eq!(workspace.documents(), [first]);
    assert_eq!(workspace.active_document(), Some(first));
    assert_eq!(workspace.owner(guid(1)), Some(first));
    assert_eq!(workspace.observer().notices.len(), 2);
    assert_eq!(
        open(&mut workspace, 2, OpenActivation::Background).session_id,
        3
    );
}

#[test]
fn close_veto_preserves_document_history_identity_and_focus() {
    let observer = RecordingWorkspace {
        deny_close: Some(1),
        ..Default::default()
    };
    let mut workspace = TestWorkspace::new(observer);
    let id = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    workspace
        .commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let error = match workspace.close_discarding_changes(id) {
        Err(error) => error,
        Ok(_) => panic!("expected veto"),
    };
    assert!(error.to_string().contains("tool owns document"));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(workspace.documents(), [id]);
    assert_eq!(workspace.owner(guid(1)), Some(id));
    assert_eq!(workspace.active_document(), Some(id));
    assert_eq!(workspace.session(id).unwrap().undo_labels(), ["edit"]);
}

#[test]
fn discard_close_releases_claim_and_selects_first_remaining_document() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let second = open(&mut workspace, 2, OpenActivation::Activate).session_id;
    let observer = workspace.close_discarding_changes(second).unwrap();
    assert_eq!(workspace.active_document(), Some(first));
    assert_eq!(workspace.owner(guid(2)), None);
    assert_eq!(workspace.documents(), [first]);
    assert!(
        matches!(&observer.observations[0], Observation::After(payload, _) if payload.type_ == EVENT_TRANSACTION_DELETED && payload.document == second)
    );
    assert_eq!(
        workspace.observer().notices.last(),
        Some(&WorkspaceNotice::Closed(second, Some(first)))
    );
    let reopened = open(&mut workspace, 2, OpenActivation::Background);
    assert_eq!(reopened.identity, IdentityRegistration::Preserved);
    assert!(reopened.session_id > second);
    workspace
        .close_discarding_changes(reopened.session_id)
        .unwrap();
    assert_eq!(workspace.active_document(), Some(first));
    workspace.close_discarding_changes(first).unwrap();
    assert_eq!(workspace.active_document(), None);
}

#[test]
fn missing_document_errors_never_emit_lifecycle_notices() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    assert!(workspace.session(9).is_err());
    assert!(workspace.commands(9).is_err());
    assert!(workspace.identity(9).is_err());
    assert_eq!(workspace.needs_save(9), Err(WorkspaceError::Missing(9)));
    assert!(workspace.close_discarding_changes(9).is_err());
    assert!(
        WorkspaceError::Missing(9)
            .to_string()
            .contains("document 9")
    );
    assert!(WorkspaceError::NoActive.to_string().contains("expected"));
    assert!(workspace.observer().notices.is_empty());
}

#[path = "workspace/save.rs"]
mod save;

#[path = "workspace/files.rs"]
mod files;

#[path = "workspace/open_verification.rs"]
mod open_verification;

#[test]
fn focus_identity_tracks_accepted_switches_and_active_close_only() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let first = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let focus = workspace.focus_identity();
    let second = open(&mut workspace, 2, OpenActivation::Background).session_id;
    assert!(workspace.retains_focus(&focus));
    workspace.activate(first).unwrap();
    assert!(workspace.activate(999).is_err());
    assert!(workspace.retains_focus(&focus));
    workspace.close_discarding_changes(second).unwrap();
    assert!(workspace.retains_focus(&focus));
    let second = open(&mut workspace, 3, OpenActivation::Background).session_id;
    workspace.activate(second).unwrap();
    workspace.activate(first).unwrap();
    assert!(!workspace.retains_focus(&focus));
    let current = workspace.focus_identity();
    workspace.close_discarding_changes(first).unwrap();
    assert!(!workspace.retains_focus(&current));
    assert_eq!(workspace.active_document(), Some(second));
}

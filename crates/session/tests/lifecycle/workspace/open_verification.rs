// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::HistoryError;
#[test]
fn open_history_is_refused_before_identity_claim_and_session_reservation() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let mut source = WorkspaceGuidSource {
        result: Ok(guid(99)),
        calls: 0,
    };
    let (error, token) = refuse_open_history(&mut workspace, &mut source);
    assert_eq!(
        error,
        WorkspaceError::InitialVerification(HistoryError::Open(token))
    );
    assert_refused_owner(&workspace, &source, &error);
    assert_eq!(
        open(&mut workspace, 1, OpenActivation::Activate).session_id,
        1
    );
}
fn assert_refused_owner(
    workspace: &TestWorkspace,
    source: &WorkspaceGuidSource,
    error: &WorkspaceError,
) {
    assert!(workspace.documents().is_empty());
    assert_eq!(workspace.active_document(), None);
    assert_eq!(workspace.owner(guid(1)), None);
    assert!(workspace.observer().notices.is_empty());
    assert_eq!(source.calls, 0);
    assert!(error.to_string().contains("initial sketch verification"));
    assert!(std::error::Error::source(error).is_some());
}
#[test]
fn existing_document_focus_and_identity_survive_refused_initial_preparation() {
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let existing = open(&mut workspace, 1, OpenActivation::Activate).session_id;
    let notices = workspace.observer().notices.len();
    let mut source = WorkspaceGuidSource {
        result: Ok(guid(99)),
        calls: 0,
    };
    let (error, _) = refuse_open_history(&mut workspace, &mut source);
    assert!(matches!(error, WorkspaceError::InitialVerification(_)));
    assert_eq!(workspace.documents(), [existing]);
    assert_eq!(workspace.active_document(), Some(existing));
    assert_eq!(workspace.owner(guid(1)), Some(existing));
    assert_eq!(workspace.owner(guid(2)), None);
    assert_eq!(workspace.observer().notices.len(), notices);
}

fn refuse_open_history(
    workspace: &mut TestWorkspace,
    source: &mut WorkspaceGuidSource,
) -> (WorkspaceError, oblikovati_model::history::TransactionToken) {
    let mut history = history_for_open();
    let token = history.begin("Uncommitted").unwrap();
    let error = workspace
        .register(
            history,
            identity(2),
            RecordingObserver::default(),
            source,
            OpenActivation::Activate,
        )
        .unwrap_err();
    (error, token)
}

// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{types::*, wire::*};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, EditError},
    identity::RecipeIdAllocator,
    parameter::{ParameterAnnotations, ParameterCollection},
    property::{DocumentProperties, USER_DEFINED},
    units::UnitsOfMeasure,
};
use oblikovati_session::{
    CommandError, DocumentSession, MutationError, SessionError, TransactionObserver,
};

#[derive(Debug, PartialEq)]
enum Observation {
    Before(TransactionEventPayload, f64),
    After(TransactionEventPayload, f64),
    Changed(u64, u64),
}
#[derive(Default)]
struct RecordingObserver {
    observations: Vec<Observation>,
    veto: Option<&'static str>,
}
impl TransactionObserver for RecordingObserver {
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), String> {
        self.observations
            .push(Observation::Before(event.clone(), width(snapshot)));
        if self.veto == Some(event.type_.as_str()) {
            return Err("export locked".into());
        }
        Ok(())
    }
    fn after(&mut self, event: &TransactionEventPayload, snapshot: &DocumentSnapshot) {
        self.observations
            .push(Observation::After(event.clone(), width(snapshot)));
    }
    fn changed(&mut self, document: u64, generation: u64) {
        self.observations
            .push(Observation::Changed(document, generation));
    }
}
fn width(snapshot: &DocumentSnapshot) -> f64 {
    snapshot
        .parameters
        .get(1)
        .unwrap()
        .current_value()
        .unwrap()
        .value
}
fn recipe() -> DocumentSnapshot {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    parameters
        .set_annotations(
            1,
            ParameterAnnotations {
                exposed_as_property: true,
                ..Default::default()
            },
        )
        .unwrap();
    let mut properties = DocumentProperties::default();
    properties.sync_exposed_parameters(&parameters).unwrap();
    DocumentSnapshot {
        attributes: Default::default(),
        interests: Default::default(),
        parameters,
        properties,
    }
}
fn session(veto: Option<&'static str>) -> DocumentSession<RecordingObserver> {
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(recipe(), ids).unwrap();
    let observer = RecordingObserver {
        veto,
        ..Default::default()
    };
    DocumentSession::new(17, history, observer)
}
fn edit_width(session: &mut DocumentSession<RecordingObserver>, source: &str) {
    session
        .edit(|snapshot, _| snapshot.parameters.set_expression(1, source))
        .unwrap();
}
fn commit_width(session: &mut DocumentSession<RecordingObserver>) {
    session.begin("Edit Parameters").unwrap();
    edit_width(session, "2 cm");
    session.end().unwrap();
}
fn event(kind: &str, label: &str, point: TransactionPoint) -> TransactionEventPayload {
    TransactionEventPayload {
        type_: kind.into(),
        document: 17,
        label: label.into(),
        point,
    }
}

#[test]
fn commit_observations_use_canonical_payload_and_coalesce_nested_edits() {
    let mut session = session(None);
    session.begin("Edit Parameters").unwrap();
    session.begin("nested label").unwrap();
    edit_width(&mut session, "2 cm");
    session.end().unwrap();
    assert!(session.observer().observations.is_empty());
    assert!(session.in_transaction());
    session.end().unwrap();
    let payload = event(
        EVENT_TRANSACTION_COMMITTED,
        "Edit Parameters",
        TRANSACTION_POINT_CURRENT,
    );
    assert_eq!(
        session.observer().observations,
        [
            Observation::Before(payload.clone(), 2.0),
            Observation::Changed(17, 1),
            Observation::After(payload, 2.0)
        ]
    );
    assert_eq!(session.undo_labels(), ["Edit Parameters"]);
    assert!(!session.in_transaction());
}

#[test]
fn navigation_observes_state_before_and_after_and_refreshes_properties() {
    let mut session = session(None);
    commit_width(&mut session);
    session.undo().unwrap();
    let payload = event(
        EVENT_TRANSACTION_UNDONE,
        "Edit Parameters",
        TRANSACTION_POINT_PREVIOUS,
    );
    assert_eq!(
        &session.observer().observations[3..],
        [
            Observation::Before(payload.clone(), 2.0),
            Observation::Changed(17, 2),
            Observation::After(payload, 1.0)
        ]
    );
    assert_eq!(
        session
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .value,
        VariantValue::String("10 mm".into())
    );
    assert_eq!(session.redo_labels(), ["Edit Parameters"]);
    session.redo().unwrap();
    let payload = event(
        EVENT_TRANSACTION_REDONE,
        "Edit Parameters",
        TRANSACTION_POINT_NEXT,
    );
    assert_eq!(
        &session.observer().observations[6..],
        [
            Observation::Before(payload.clone(), 1.0),
            Observation::Changed(17, 3),
            Observation::After(payload, 2.0)
        ]
    );
    assert_eq!(
        session
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .unwrap()
            .value,
        VariantValue::String("20 mm".into())
    );
}

#[test]
fn commit_veto_rolls_back_whole_group_and_emits_only_abort_after() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    session.begin("batch").unwrap();
    session
        .edit(|snapshot, ids| {
            snapshot.parameters.insert_expression(
                ids.allocate().unwrap(),
                "temporary",
                USER_PARAM,
                "3",
            )
        })
        .unwrap();
    edit_width(&mut session, "2 cm");
    assert_eq!(
        session.end(),
        Err(SessionError::Veto("export locked".into()))
    );
    let commit = event(
        EVENT_TRANSACTION_COMMITTED,
        "batch",
        TRANSACTION_POINT_CURRENT,
    );
    let abort = event(
        EVENT_TRANSACTION_ABORTED,
        "batch",
        TRANSACTION_POINT_CURRENT,
    );
    assert_eq!(
        session.observer().observations,
        [
            Observation::Before(commit, 2.0),
            Observation::After(abort, 1.0)
        ]
    );
    assert_eq!(session.snapshot().parameters.ids(), [1]);
    assert_eq!(session.high_water(), 2);
    assert_eq!(session.generation(), 3);
    assert!(session.undo_labels().is_empty());
}

#[test]
fn undo_veto_does_not_move_state_or_emit_changed_or_after() {
    let mut session = session(Some(EVENT_TRANSACTION_UNDONE));
    commit_width(&mut session);
    assert!(
        session
            .undo()
            .unwrap_err()
            .to_string()
            .contains("export locked")
    );
    assert_eq!(session.generation(), 1);
    assert_eq!(width(session.snapshot()), 2.0);
    assert_eq!(session.observer().observations.len(), 4);
    assert_eq!(session.undo_labels(), ["Edit Parameters"]);
}

#[test]
fn redo_veto_preserves_redo_step() {
    let mut session = session(Some(EVENT_TRANSACTION_REDONE));
    commit_width(&mut session);
    session.undo().unwrap();
    assert!(session.redo().is_err());
    assert_eq!(session.generation(), 2);
    assert_eq!(width(session.snapshot()), 1.0);
    assert_eq!(session.observer().observations.len(), 7);
    assert_eq!(session.redo_labels(), ["Edit Parameters"]);
}

#[test]
fn abort_cancels_all_nested_levels_with_one_observation() {
    let mut session = session(None);
    session.begin("outer").unwrap();
    edit_width(&mut session, "2 cm");
    session.begin("inner").unwrap();
    edit_width(&mut session, "3 cm");
    session.abort().unwrap();
    assert!(!session.in_transaction());
    assert!(session.undo_labels().is_empty());
    let payload = event(
        EVENT_TRANSACTION_ABORTED,
        "outer",
        TRANSACTION_POINT_CURRENT,
    );
    assert_eq!(
        session.observer().observations,
        [Observation::After(payload, 1.0)]
    );
    assert!(session.end().is_err());
}

#[test]
fn invalid_navigation_emits_no_events_and_invalid_edit_never_calls_command() {
    let mut session = session(None);
    assert!(session.undo().unwrap_err().to_string().contains("empty"));
    assert!(session.redo().is_err());
    assert_eq!(session.abort(), Err(SessionError::NoGroup));
    let error = session
        .edit(|_, _| -> Result<(), std::io::Error> { panic!("no open group") })
        .unwrap_err();
    assert!(error.to_string().contains("expected an open"));
    session.begin("open").unwrap();
    assert!(session.undo().is_err());
    assert!(session.redo().is_err());
    assert!(session.observer().observations.is_empty());
}

#[test]
fn failed_command_is_atomic_and_unhealthy_projection_can_be_published() {
    let mut session = session(None);
    session.begin("failed").unwrap();
    let error = session
        .edit(|snapshot, _| snapshot.parameters.set_expression(1, "("))
        .unwrap_err();
    assert!(matches!(
        error,
        CommandError::Edit(EditError::Mutation(MutationError::Command(_)))
    ));
    assert!(!error.to_string().is_empty());
    assert_eq!(session.generation(), 0);
    session
        .edit(|snapshot, _| snapshot.parameters.set_expression(1, "missing"))
        .unwrap();
    assert!(
        session
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .is_none()
    );
    assert!(
        session
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_none()
    );
    session.abort().unwrap();
    assert_eq!(width(session.snapshot()), 1.0);
    assert!(
        session
            .snapshot()
            .properties
            .get(USER_DEFINED, "width")
            .is_some()
    );
}

#[test]
fn close_consumes_stream_and_reports_deleted_without_veto() {
    let mut session = session(Some(EVENT_TRANSACTION_DELETED));
    commit_width(&mut session);
    let observer = session.close();
    let payload = event(EVENT_TRANSACTION_DELETED, "", TRANSACTION_POINT_UNKNOWN);
    assert_eq!(
        observer.observations.last(),
        Some(&Observation::After(payload, 2.0))
    );
    assert_eq!(observer.observations.len(), 4);
}

#[test]
fn new_commit_replaces_redo_through_the_same_session_path() {
    let mut session = session(None);
    commit_width(&mut session);
    session.undo().unwrap();
    session.begin("replacement").unwrap();
    session
        .edit(|snapshot, _| snapshot.parameters.set_expression(1, "3 cm"))
        .unwrap();
    session.end().unwrap();
    assert!(session.redo_labels().is_empty());
    assert_eq!(session.undo_labels(), ["replacement"]);
}

#[path = "lifecycle/commands.rs"]
mod commands;

#[path = "lifecycle/workspace.rs"]
mod workspace;

#[path = "lifecycle/jumps.rs"]
mod jumps;

#[path = "lifecycle/attributes.rs"]
mod attributes;

#[path = "lifecycle/interests.rs"]
mod interests;

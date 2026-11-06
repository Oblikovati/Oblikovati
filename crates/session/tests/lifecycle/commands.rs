// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn automatic_command_records_one_step_and_returns_its_result() {
    let mut session = session(None);
    let result = session
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")?;
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(42)
        })
        .unwrap();
    assert_eq!(result, 42);
    assert_eq!(session.undo_labels(), ["width"]);
    assert_eq!(session.observer().observations.len(), 3);
    assert!(!session.in_transaction());
    session.undo().unwrap();
    assert_eq!(width(session.snapshot()), 1.0);
}

#[test]
fn automatic_command_joins_caller_group_without_closing_it() {
    let mut session = session(None);
    session.begin("batch").unwrap();
    session
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    assert!(session.in_transaction());
    assert!(session.observer().observations.is_empty());
    assert!(session.undo_labels().is_empty());
    session.end().unwrap();
    assert_eq!(session.undo_labels(), ["batch"]);
}

#[test]
fn failed_automatic_command_closes_only_its_own_group() {
    let mut session = session(None);
    let error = session
        .run("bad", |snapshot, ids| {
            ids.allocate().unwrap();
            snapshot.parameters.set_expression(1, "(")
        })
        .unwrap_err();
    assert!(matches!(error, CommandError::Edit(_)));
    assert!(!session.in_transaction());
    assert_eq!(session.high_water(), 2);
    assert!(session.observer().observations.is_empty());
    session.begin("caller").unwrap();
    assert!(
        session
            .run("bad", |snapshot, _| snapshot
                .parameters
                .set_expression(1, "("))
            .is_err()
    );
    assert!(session.in_transaction());
    assert!(session.observer().observations.is_empty());
}

#[test]
fn command_commit_veto_is_reported_after_rollback() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    let error = session
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap_err();
    assert!(matches!(
        error,
        CommandError::Session(SessionError::Veto(_))
    ));
    assert!(!session.in_transaction());
    assert_eq!(width(session.snapshot()), 1.0);
    assert!(session.undo_labels().is_empty());
    assert_eq!(session.observer().observations.len(), 2);
}

#[test]
fn empty_same_value_and_reverted_groups_preserve_redo_and_emit_nothing() {
    let mut session = session(None);
    commit_width(&mut session);
    session.undo().unwrap();
    let count = session.observer().observations.len();
    session.begin("empty").unwrap();
    session.end().unwrap();
    session
        .run("same", |snapshot, _| {
            snapshot.parameters.set_expression(1, "1 cm")
        })
        .unwrap();
    session.begin("reverted").unwrap();
    edit_width(&mut session, "3 cm");
    edit_width(&mut session, "1 cm");
    session.end().unwrap();
    assert_eq!(session.observer().observations.len(), count);
    assert_eq!(session.redo_labels(), ["Edit Parameters"]);
    assert!(session.undo_labels().is_empty());
    session.redo().unwrap();
    assert_eq!(width(session.snapshot()), 2.0);
}

#[test]
fn allocation_without_recipe_change_is_reserved_without_an_undo_step() {
    let mut session = session(None);
    let id = session.run("reserve", |_, ids| ids.allocate()).unwrap();
    assert_eq!(id, 2);
    assert_eq!(session.high_water(), 2);
    assert!(session.undo_labels().is_empty());
    assert!(session.observer().observations.is_empty());
}

#[test]
fn property_metadata_edits_are_not_filtered_out() {
    let mut session = session(None);
    session
        .run("title", |snapshot, _| {
            snapshot
                .properties
                .put(USER_DEFINED, "Title", VariantValue::String("part".into()))
        })
        .unwrap();
    assert_eq!(session.undo_labels(), ["title"]);
    session.undo().unwrap();
    assert!(
        session
            .snapshot()
            .properties
            .get(USER_DEFINED, "Title")
            .is_none()
    );
}

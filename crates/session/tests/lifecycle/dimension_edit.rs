// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchDimensionEdit;
fn edit(session: &mut DocumentSession<RecordingObserver>, input: &SketchDimensionEdit) {
    session
        .edit_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), input)
        .unwrap();
}
fn changes() -> SketchDimensionEdit {
    SketchDimensionEdit {
        target_expression: Some("4 cm".into()),
        reference: Some(true),
        drive_limits: Some(SketchDimensionLimits {
            lower: 2.0,
            upper: 3.0,
        }),
    }
}
fn radius(session: &DocumentSession<RecordingObserver>) -> f64 {
    session.snapshot().parameters.model_value(2).unwrap().value
}
fn reference(session: &DocumentSession<RecordingObserver>) -> bool {
    session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .dimension_is_reference(SketchConstraintId(1))
        .unwrap()
}
fn measured(session: &DocumentSession<RecordingObserver>) -> f64 {
    session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .measure_dimension(SketchConstraintId(1))
        .unwrap()
        .value
}
#[test]
fn reference_drive_changes_target_without_moving_measured_geometry() {
    let mut session = dimension_session(None);
    let original = measured(&session);
    edit(&mut session, &changes());
    assert!(reference(&session));
    assert_eq!(radius(&session), 3.0);
    assert_eq!(measured(&session), original);
    let count = session.undo_labels().len();
    edit(&mut session, &changes());
    assert_eq!(session.undo_labels().len(), count);
}
#[test]
fn returning_to_driving_recomputes_geometry_with_the_retained_target() {
    let mut session = dimension_session(None);
    edit(&mut session, &changes());
    set_role(&mut session, false);
    assert!(!reference(&session));
    assert!((measured(&session) - 3.0).abs() < 1e-9);
    assert_eq!(
        session
            .analyze_sketch(DocumentSketchId(1))
            .unwrap()
            .degrees_of_freedom,
        2
    );
}
#[test]
fn undo_redo_and_group_abort_restore_targets_modes_and_limits_together() {
    let mut session = dimension_session(None);
    let original = session.snapshot().clone();
    edit(&mut session, &changes());
    session.undo().unwrap();
    assert_original(&session, &original);
    session.redo().unwrap();
    assert!(reference(&session));
    assert_eq!(radius(&session), 3.0);
    assert_group_abort(&mut session);
}
fn assert_original(session: &DocumentSession<RecordingObserver>, original: &DocumentSnapshot) {
    assert!(
        session
            .snapshot()
            .sketches
            .same_evaluated_state(&original.sketches)
    );
    assert_eq!(
        session.snapshot().parameters.get(2),
        original.parameters.get(2)
    );
}
#[test]
fn rejected_reply_cannot_adopt_any_dimension_edit() {
    let mut session = dimension_session(None);
    let original = session.snapshot().clone();
    let result: Result<(), CommandError<String>> = session.edit_sketch_dimension_projected(
        DocumentSketchId(1),
        SketchConstraintId(1),
        &changes(),
        |reply| {
            reply.unwrap();
            Err("reply rejected".into())
        },
    );
    assert!(result.unwrap_err().to_string().contains("reply rejected"));
    assert_original(&session, &original);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn commit_veto_cannot_adopt_any_dimension_edit() {
    let mut session = dimension_session(Some(EVENT_TRANSACTION_COMMITTED));
    let original = session.snapshot().clone();
    assert!(
        session
            .edit_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), &changes())
            .is_err()
    );
    assert_original(&session, &original);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn reference_mode_ignores_a_failed_target_and_restoring_driving_reports_it() {
    let mut session = dimension_session(None);
    set_role(&mut session, true);
    session
        .run("Fail target", |snapshot, _| {
            snapshot.parameters.set_expression(2, "missing")
        })
        .unwrap();
    assert!(session.analyze_sketch(DocumentSketchId(1)).is_ok());
    set_role(&mut session, false);
    assert!(session.analyze_sketch(DocumentSketchId(1)).is_err());
    assert_eq!(measured(&session), 1.0);
}

fn assert_group_abort(session: &mut DocumentSession<RecordingObserver>) {
    session.begin("Drive group").unwrap();
    edit(
        session,
        &SketchDimensionEdit {
            reference: Some(false),
            target_expression: Some("2 cm".into()),
            ..Default::default()
        },
    );
    session.abort().unwrap();
    assert!(reference(session));
    assert_eq!(radius(session), 3.0);
}

fn set_role(session: &mut DocumentSession<RecordingObserver>, reference: bool) {
    let input = SketchDimensionEdit {
        reference: Some(reference),
        ..Default::default()
    };
    edit(session, &input);
}

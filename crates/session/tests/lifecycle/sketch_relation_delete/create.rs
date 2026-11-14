// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::SketchGeometricCreationReport, sketch::SketchGeometricConstraintCreation,
};
fn input() -> SketchGeometricConstraintCreation {
    SketchGeometricConstraintCreation::Fix {
        point: SketchEntityId(1),
    }
}
#[test]
fn prepared_response_rejection_preserves_geometry_but_retires_the_attempted_id() {
    let mut session = constrained_session();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let result: Result<(), CommandError<String>> = session
        .create_sketch_geometric_constraint_projected(DocumentSketchId(1), &input(), |report| {
            assert_eq!(report.unwrap().constraint.id, SketchConstraintId(2));
            Err("response rejected".into())
        });
    assert_response_rejected(result);
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    let next = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap();
    assert_eq!(next.constraint.id, SketchConstraintId(3));
}
#[test]
fn aborted_group_preserves_permanent_creation_marks_outside_history() {
    let mut session = constrained_session();
    session.begin("Constraint batch").unwrap();
    let first = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap();
    session.abort().unwrap();
    assert_eq!(
        session.snapshot().sketches.sketches()[0]
            .geometry()
            .constraints()
            .len(),
        1
    );
    let second = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap();
    assert_eq!(second.constraint.id.0, first.constraint.id.0 + 1);
}
#[test]
fn committed_group_coalesces_creation_and_undo_redo_restores_both_relations() {
    let mut session = constrained_session();
    session.begin("Constraint batch").unwrap();
    let first = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap();
    let second = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap();
    session.end().unwrap();
    assert_eq!(session.undo_labels(), ["Constraint batch"]);
    session.undo().unwrap();
    assert_restored_relation(&session);
    session.redo().unwrap();
    assert_created_group(&session, &first, &second);
}
fn assert_created_group(
    session: &DocumentSession<RecordingObserver>,
    first: &SketchGeometricCreationReport,
    second: &SketchGeometricCreationReport,
) {
    let rows = session.snapshot().sketches.sketches()[0]
        .geometry()
        .constraints();
    assert_eq!(rows.len(), 3);
    assert_eq!(rows[1], first.constraint);
    assert_eq!(rows[2], second.constraint);
    assert_eq!(second.geometric_index, 2);
}
#[test]
fn commit_veto_discards_relation_and_keeps_the_attempted_reservation() {
    let mut session = vetoed_session();
    let before = session.snapshot().sketches.clone();
    let error = session
        .create_sketch_geometric_constraint(DocumentSketchId(1), &input())
        .unwrap_err();
    assert!(error.to_string().contains("locked"));
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.begin("Inspect reservation").unwrap();
    let next = session
        .edit(|_, ids| ids.allocate_sketch_constraint(DocumentSketchId(1)))
        .unwrap();
    assert_eq!(next, SketchConstraintId(3));
    session.abort().unwrap();
}
fn vetoed_session() -> DocumentSession<RecordingObserver> {
    let snapshot = constrained_session().snapshot().clone();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(
        17,
        history,
        RecordingObserver {
            veto: Some(EVENT_TRANSACTION_COMMITTED),
            ..Default::default()
        },
    )
}

fn assert_response_rejected(result: Result<(), CommandError<String>>) {
    let cause = result.unwrap_err().to_string();
    assert!(cause.contains("response rejected"), "{cause}");
}

#[test]
fn preferred_creation_projects_before_commit_and_rolls_back_on_reply_error() {
    let mut session = constrained_session();
    let before = session.snapshot().sketches.clone();
    let result: Result<(), CommandError<String>> = session
        .create_sketch_geometric_constraint_preferred_projected(
            DocumentSketchId(1),
            &input(),
            SketchEntityId(1),
            |report| {
                assert!(report.unwrap().preference.is_some());
                Err("preferred reply rejected".into())
            },
        );
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("preferred reply rejected")
    );
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn preferred_creation_undo_redo_restores_relation() {
    let mut session = constrained_session();
    let report = session
        .create_sketch_geometric_constraint_preferred_projected(
            DocumentSketchId(1),
            &input(),
            SketchEntityId(1),
            |result| result,
        )
        .unwrap();
    assert!(report.preference.is_some());
    session.undo().unwrap();
    assert_restored_relation(&session);
    session.redo().unwrap();
    assert_eq!(
        session.snapshot().sketches.sketches()[0]
            .geometry()
            .constraints()
            .len(),
        2
    );
}

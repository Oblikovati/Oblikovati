// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_dimension_delete::{dimension_session, dimension_session_from_snapshot};
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};

pub(super) fn selection_session(veto: Option<&'static str>) -> DocumentSession<RecordingObserver> {
    let mut snapshot = dimension_session(None).snapshot().clone();
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    append_surviving_circle(&mut geometry);
    append_center_fix(&mut geometry);
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    dimension_session_from_snapshot(snapshot, veto)
}
fn append_surviving_circle(geometry: &mut Sketch2D) {
    geometry
        .add_circle(SketchEntityId(3), SketchEntityId(1), 1.0)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(3),
            PlanarConstraint::Radius {
                curve: SketchEntityId(3),
                parameter: 2,
            },
        )
        .unwrap();
}
fn mixed_selection() -> SketchDeletionSelection {
    SketchDeletionSelection {
        entities: vec![SketchEntityId(2)],
        dimensions: vec![SketchConstraintId(1), SketchConstraintId(3)],
        geometric_constraints: vec![SketchConstraintId(2)],
    }
}
pub(super) fn assert_mixed_cascade(session: &DocumentSession<RecordingObserver>) {
    let snapshot = session.snapshot();
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(
        geometry.entity_ids(),
        &[SketchEntityId(1), SketchEntityId(3)]
    );
    assert!(geometry.constraints().is_empty());
    assert!(snapshot.parameters.get(2).is_none());
}
#[test]
fn mixed_selection_is_one_history_step_and_abort_restores_the_entire_recipe() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let report = session
        .delete_sketch_selection(DocumentSketchId(1), &mixed_selection())
        .unwrap();
    assert_mixed_report(&report);
    assert_eq!(session.undo_labels(), ["Delete sketch selection"]);
    assert_mixed_cascade(&session);
    session.undo().unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(session.snapshot().parameters.get(2).is_some());
    session.redo().unwrap();
    assert_mixed_cascade(&session);
    session.undo().unwrap();
    assert_group_abort(&mut session, &before);
}
fn assert_group_abort(
    session: &mut DocumentSession<RecordingObserver>,
    before: &oblikovati_model::document_sketch::DocumentSketches,
) {
    session.begin("Delete group").unwrap();
    session
        .delete_sketch_selection(DocumentSketchId(1), &mixed_selection())
        .unwrap();
    session.abort().unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(session.snapshot().parameters.get(2).is_some());
}
#[test]
fn rejected_reply_and_commit_veto_publish_no_mixed_selection() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let result = session.delete_sketch_selection_projected(
        DocumentSketchId(1),
        &mixed_selection(),
        |reply| {
            assert_eq!(reply.unwrap().constraints.len(), 3);
            Err::<(), _>("reply refused")
        },
    );
    assert!(result.is_err());
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(session.snapshot().parameters.get(2).is_some());
    assert_eq!(session.generation(), generation);
    assert_veto();
}
fn assert_veto() {
    let mut session = selection_session(Some(EVENT_TRANSACTION_COMMITTED));
    let before = session.snapshot().sketches.clone();
    assert!(
        session
            .delete_sketch_selection(DocumentSketchId(1), &mixed_selection())
            .is_err()
    );
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(session.snapshot().parameters.get(2).is_some());
    assert!(session.undo_labels().is_empty());
}
#[test]
fn invalid_selection_is_refused_before_any_dimension_source_is_removed() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let mut selected = mixed_selection();
    selected.geometric_constraints.push(SketchConstraintId(1));
    assert!(
        session
            .delete_sketch_selection(DocumentSketchId(1), &selected)
            .is_err()
    );
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(session.snapshot().parameters.get(2).is_some());
    assert!(session.undo_labels().is_empty());
}
#[test]
fn single_dimension_deletion_retains_a_shared_model_source_until_its_last_owner() {
    let mut session = selection_session(None);
    session
        .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    assert!(session.snapshot().parameters.get(2).is_some());
    assert_remaining_dimension(&session);
    session
        .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(3))
        .unwrap();
    assert!(session.snapshot().parameters.get(2).is_none());
    session.undo().unwrap();
    assert!(session.snapshot().parameters.get(2).is_some());
}

fn append_center_fix(geometry: &mut Sketch2D) {
    geometry
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Fix {
                point: SketchEntityId(1),
                position: [0.0, 0.0],
            },
        )
        .unwrap();
}

fn assert_mixed_report(report: &SketchEntityDeletion) {
    let ids: Vec<u64> = report.constraints.iter().map(|row| row.id.0).collect();
    assert_eq!(ids, vec![1, 3, 2]);
}

fn assert_remaining_dimension(session: &DocumentSession<RecordingObserver>) {
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(
        geometry.dimension_source(SketchConstraintId(3)).unwrap().0,
        2
    );
}

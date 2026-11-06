// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
pub(super) fn dimension_session(veto: Option<&'static str>) -> DocumentSession<RecordingObserver> {
    let mut snapshot = recipe();
    snapshot
        .parameters
        .insert_expression(2, "radius", MODEL_PARAM, "1 cm")
        .unwrap();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Circle",
            SketchPlane::xy(),
            dimension_geometry(),
        )
        .unwrap();
    dimension_session_from_snapshot(snapshot, veto)
}
pub(super) fn dimension_session_from_snapshot(
    snapshot: DocumentSnapshot,
    veto: Option<&'static str>,
) -> DocumentSession<RecordingObserver> {
    let history = DocumentEditHistory::new(
        snapshot,
        RecipeIdAllocator::restore(Some(2), &[1, 2]).unwrap(),
    )
    .unwrap();
    DocumentSession::new(
        17,
        history,
        RecordingObserver {
            veto,
            ..Default::default()
        },
    )
}
pub(super) fn assert_owned_dimension(session: &DocumentSession<RecordingObserver>, present: bool) {
    assert_eq!(session.snapshot().parameters.get(2).is_some(), present);
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(
        geometry.dimensional_constraints().count(),
        usize::from(present)
    );
}
#[test]
fn dimension_and_parameter_restore_together_on_undo_redo_and_group_abort() {
    let mut session = dimension_session(None);
    assert_eq!(
        session
            .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
            .unwrap(),
        3
    );
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    assert_owned_dimension(&session, true);
    session.redo().unwrap();
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    verify_dimension_group_abort(&mut session);
}
fn verify_dimension_group_abort(session: &mut DocumentSession<RecordingObserver>) {
    session.begin("Delete group").unwrap();
    session
        .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    session.abort().unwrap();
    assert_owned_dimension(session, true);
}
#[test]
fn reply_failure_and_commit_veto_restore_the_entire_cascade() {
    let mut session = dimension_session(None);
    let generation = session.generation();
    let result: Result<(), CommandError<String>> = session.delete_sketch_dimension_projected(
        DocumentSketchId(1),
        SketchConstraintId(1),
        |reply| {
            assert_eq!(reply.unwrap(), 3);
            Err("reply rejected".into())
        },
    );
    assert!(result.unwrap_err().to_string().contains("reply rejected"));
    assert_owned_dimension(&session, true);
    assert_eq!(session.generation(), generation);
}
#[test]
fn commit_veto_restores_the_dimension_cascade() {
    let mut session = dimension_session(Some(EVENT_TRANSACTION_COMMITTED));
    assert!(
        session
            .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
            .is_err()
    );
    assert_owned_dimension(&session, true);
    assert!(session.undo_labels().is_empty());
}

fn dimension_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Radius {
                curve: SketchEntityId(2),
                parameter: 2,
            },
        )
        .unwrap();
    geometry
}

#[test]
fn deletion_does_not_reuse_parameter_or_constraint_ids() {
    let mut session = dimension_session(None);
    session
        .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    let (parameter, constraint) = session
        .run("Reserve after deletion", |_, ids| -> Result<_, String> {
            Ok((
                ids.allocate().unwrap(),
                ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap(),
            ))
        })
        .unwrap();
    assert_eq!(parameter, 3);
    assert_eq!(constraint, SketchConstraintId(2));
}

#[path = "dimension_placement.rs"]
mod placement;

#[path = "dimension_edit.rs"]
mod edit;

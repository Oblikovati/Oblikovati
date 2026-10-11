// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane},
};
fn constrained_session() -> DocumentSession<RecordingObserver> {
    let mut snapshot = recipe();
    let geometry = constrained_geometry();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Delete", SketchPlane::xy(), geometry)
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
#[test]
fn projected_reply_failure_cannot_delete_geometry_or_advance_history() {
    let mut session = constrained_session();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let result: Result<(), CommandError<String>> = session
        .delete_sketch_geometric_constraint_projected(
            DocumentSketchId(1),
            SketchConstraintId(1),
            |reply| {
                assert_eq!(reply.unwrap().id, SketchConstraintId(1));
                Err("reply rejected".into())
            },
        );
    assert!(result.unwrap_err().to_string().contains("reply rejected"));
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn deletion_recomputes_and_undo_redo_preserve_the_relation_identity() {
    let mut session = constrained_session();
    let removed = session
        .delete_sketch_geometric_constraint(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    assert_eq!(removed.id, SketchConstraintId(1));
    assert_eq!(
        session
            .analyze_sketch(DocumentSketchId(1))
            .unwrap()
            .degrees_of_freedom,
        4
    );
    assert_deletion_navigation(&mut session, removed);
}
#[test]
fn aborted_group_restores_relation_but_reserves_its_identity_forever() {
    let mut session = constrained_session();
    session.begin("Delete group").unwrap();
    session
        .delete_sketch_geometric_constraint(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    session.abort().unwrap();
    let next = session
        .run("Reserve after deletion", |_, ids| {
            ids.allocate_sketch_constraint(DocumentSketchId(1))
        })
        .unwrap();
    assert_eq!(next, SketchConstraintId(2));
    assert_restored_relation(&session);
}
#[test]
fn caught_invalid_deletion_does_not_damage_an_open_edit() {
    let mut session = constrained_session();
    session
        .run("Caught failure", |snapshot, _| -> Result<(), String> {
            assert_invalid_deletions(snapshot);
            Ok(())
        })
        .unwrap();
}

fn constrained_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(2), [4.0, 1.0]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::LineHorizontal {
                line: SketchEntityId(3),
            },
        )
        .unwrap();
    geometry
}

fn assert_deletion_navigation(
    session: &mut DocumentSession<RecordingObserver>,
    removed: oblikovati_model::sketch::SketchConstraint,
) {
    session.undo().unwrap();
    assert_eq!(
        session.snapshot().sketches.sketches()[0]
            .geometry()
            .constraints()[0],
        removed
    );
    session.redo().unwrap();
    assert!(
        session.snapshot().sketches.sketches()[0]
            .geometry()
            .constraints()
            .is_empty()
    );
}
fn assert_invalid_deletions(snapshot: &mut DocumentSnapshot) {
    let before = snapshot.sketches.clone();
    for (sketch, constraint) in [(1, 0), (1, 99), (99, 1)] {
        let error = snapshot
            .delete_sketch_geometric_constraint(
                DocumentSketchId(sketch),
                SketchConstraintId(constraint),
            )
            .unwrap_err();
        let offending = if sketch == 99 { sketch } else { constraint };
        assert!(error.to_string().contains(&offending.to_string()));
        assert!(before.same_evaluated_state(&snapshot.sketches));
    }
}
#[test]
fn committed_deletion_never_releases_the_constraint_reservation() {
    let mut session = constrained_session();
    session
        .delete_sketch_geometric_constraint(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    let next = session
        .run("Reserve after commit", |_, ids| {
            ids.allocate_sketch_constraint(DocumentSketchId(1))
        })
        .unwrap();
    assert_eq!(next, SketchConstraintId(2));
}

fn assert_restored_relation(session: &DocumentSession<RecordingObserver>) {
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(geometry.constraints().len(), 1);
    assert_eq!(geometry.constraints()[0].id, SketchConstraintId(1));
}

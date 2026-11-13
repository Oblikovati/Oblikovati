// SPDX-License-Identifier: GPL-2.0-only
//! Solved transform previews share commit semantics and never change the live document.
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane,
        SketchSelectionTransform,
    },
};

fn session_with_line() -> DocumentSession<RecordingObserver> {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    geometry.add_point(SketchEntityId(2), [4., 0.]).unwrap();
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
    let mut snapshot = recipe();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Preview", SketchPlane::xy(), geometry)
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
#[test]
fn preview_matches_committed_solve_without_history_or_identity_changes() {
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let transform = SketchSelectionTransform::Move([2., 3.]);
    let preview = session
        .preview_sketch_transform(DocumentSketchId(1), &[SketchEntityId(3)], transform)
        .unwrap();
    assert_eq!(preview.members, [SketchEntityId(3)]);
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    session
        .transform_sketch_entities_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            transform,
            |result| result,
        )
        .unwrap();
    let committed = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert!(preview.geometry.same_recipe(committed));
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
}
#[test]
fn rejected_preview_preserves_the_live_sketch_and_history() {
    let session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    for (entities, vector) in [
        (vec![SketchEntityId(99)], [1., 2.]),
        (vec![SketchEntityId(3)], [f64::NAN, 2.]),
    ] {
        assert!(
            session
                .preview_sketch_transform(
                    DocumentSketchId(1),
                    &entities,
                    SketchSelectionTransform::Move(vector)
                )
                .is_err()
        );
    }
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
}

#[test]
fn copy_preview_matches_commit_and_reserves_no_live_ids() {
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let preview = session
        .preview_sketch_copy(DocumentSketchId(1), &[SketchEntityId(3)], [2., 3.])
        .unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    let report = session
        .copy_sketch_entities(DocumentSketchId(1), &[SketchEntityId(3)], [2., 3.])
        .unwrap();
    assert_eq!(preview.members, report.copies);
    let committed = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert!(preview.geometry.same_recipe(committed));
    assert_eq!(committed.lines().len(), 2);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(
        preview.geometry.same_recipe(
            session
                .snapshot()
                .sketches
                .get(DocumentSketchId(1))
                .unwrap()
                .geometry()
        )
    );
}

#[test]
fn rejected_copy_preview_does_not_change_live_history() {
    let session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    assert!(
        session
            .preview_sketch_copy(DocumentSketchId(1), &[SketchEntityId(99)], [1., 2.])
            .is_err()
    );
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
}

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

fn session_with_linked_lines(fixed: bool) -> DocumentSession<RecordingObserver> {
    let mut snapshot = session_with_line().snapshot().clone();
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry.add_point(SketchEntityId(4), [4., 0.]).unwrap();
    geometry.add_point(SketchEntityId(5), [7., 2.]).unwrap();
    geometry
        .add_line(SketchEntityId(6), [SketchEntityId(4), SketchEntityId(5)])
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(2), SketchEntityId(4)],
            },
        )
        .unwrap();
    if fixed {
        geometry
            .add_constraint(
                SketchConstraintId(3),
                PlanarConstraint::Fix {
                    point: SketchEntityId(1),
                    position: [0., 0.],
                },
            )
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}

#[test]
fn moving_a_linked_line_holds_the_selected_points_at_their_targets() {
    let mut session = session_with_linked_lines(false);
    let transform = SketchSelectionTransform::Move([2., 3.]);
    let preview = session
        .preview_sketch_transform(DocumentSketchId(1), &[SketchEntityId(3)], transform)
        .unwrap();
    assert_eq!(
        preview.geometry.point(SketchEntityId(1)).unwrap().position,
        [2., 3.]
    );
    assert_eq!(
        preview.geometry.point(SketchEntityId(2)).unwrap().position,
        [6., 3.]
    );
    let linked = preview.geometry.point(SketchEntityId(4)).unwrap().position;
    assert!(
        (linked[0] - 6.).abs() < 1e-7 && (linked[1] - 3.).abs() < 1e-7,
        "{linked:?}"
    );
    session
        .transform_sketch_entities_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            transform,
            |result| result,
        )
        .unwrap();
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
fn moving_a_fixed_line_refuses_the_candidate_and_preserves_history() {
    let mut session = session_with_linked_lines(true);
    let before = session.snapshot().sketches.clone();
    let history: Vec<String> = session
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect();
    let transform = SketchSelectionTransform::Move([2., 3.]);
    assert!(
        session
            .preview_sketch_transform(DocumentSketchId(1), &[SketchEntityId(3)], transform)
            .is_err()
    );
    assert!(
        session
            .transform_sketch_entities_projected(
                DocumentSketchId(1),
                &[SketchEntityId(3)],
                transform,
                |result| result
            )
            .is_err()
    );
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(history, session.undo_labels());
}

#[test]
fn rotated_copy_is_one_undoable_history_step() {
    use oblikovati_model::sketch::SketchCopyOperation;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let operation = SketchCopyOperation::Rotate {
        center: [0.; 2],
        radians: std::f64::consts::FRAC_PI_2,
    };
    let generation = session.generation();
    let preview = session
        .preview_selected_sketch_copy(DocumentSketchId(1), &[SketchEntityId(3)], operation)
        .unwrap();
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let report = session
        .copy_selected_sketch_entities_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            operation,
            |result| result,
        )
        .unwrap();
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(preview.members, report.copies);
    assert!(preview.geometry.same_recipe(geometry));
    let line = geometry.line(report.copies[0]).unwrap();
    let end = geometry.point(line.endpoints[1]).unwrap().position;
    assert!(end[0].abs() < 1e-12 && (end[1] - 4.).abs() < 1e-12);
    let after = session.snapshot().sketches.clone();
    assert_eq!(session.undo_labels(), ["Rotate copied sketch entities"]);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn invalid_rotated_copy_leaves_history_and_geometry_unchanged() {
    use oblikovati_model::sketch::SketchCopyOperation;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let operation = SketchCopyOperation::Rotate {
        center: [0.; 2],
        radians: f64::NAN,
    };
    let generation = session.generation();
    assert!(
        session
            .preview_selected_sketch_copy(DocumentSketchId(1), &[SketchEntityId(3)], operation)
            .is_err()
    );
    let result = session.copy_selected_sketch_entities_projected(
        DocumentSketchId(1),
        &[SketchEntityId(3)],
        operation,
        |result| result,
    );
    assert_eq!(session.generation(), generation);
    assert!(result.is_err());
    assert!(session.undo_labels().is_empty());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
}

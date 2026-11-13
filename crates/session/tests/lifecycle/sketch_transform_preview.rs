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

#[test]
fn batch_copy_preview_matches_one_undoable_commit() {
    use oblikovati_model::sketch::SketchCopyOperation;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let operations = [
        SketchCopyOperation::Translate([2., 3.]),
        SketchCopyOperation::Rotate {
            center: [0.; 2],
            radians: std::f64::consts::FRAC_PI_2,
        },
    ];
    let generation = session.generation();
    let preview = session
        .preview_selected_sketch_copies(DocumentSketchId(1), &[SketchEntityId(3)], &operations)
        .unwrap();
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let reports = session
        .copy_selected_sketch_entities_batch_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            &operations,
            |result| result,
        )
        .unwrap();
    assert_eq!(
        preview.members,
        reports
            .into_iter()
            .flat_map(|report| report.copies)
            .collect::<Vec<_>>()
    );
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
    let after = session.snapshot().sketches.clone();
    assert_eq!(session.undo_labels(), ["Copy sketch instances"]);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn failed_batch_keeps_history_unchanged_and_retires_allocated_ids() {
    use oblikovati_model::sketch::SketchCopyOperation;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let first = SketchCopyOperation::Translate([2., 3.]);
    let preview = session
        .preview_selected_sketch_copies(DocumentSketchId(1), &[SketchEntityId(3)], &[first])
        .unwrap();
    let operations = [
        first,
        SketchCopyOperation::Rotate {
            center: [0.; 2],
            radians: f64::NAN,
        },
    ];
    assert!(
        session
            .preview_selected_sketch_copies(DocumentSketchId(1), &[SketchEntityId(3)], &operations)
            .is_err()
    );
    assert!(
        session
            .copy_selected_sketch_entities_batch_projected(
                DocumentSketchId(1),
                &[SketchEntityId(3)],
                &operations,
                |result| result
            )
            .is_err()
    );
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
    let reports = session
        .copy_selected_sketch_entities_batch_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            &[first],
            |result| result,
        )
        .unwrap();
    assert!(reports[0].copies[0].0 > preview.members[0].0);
}

#[test]
fn rectangular_layout_preview_and_commit_share_fitted_suppression_and_history() {
    use oblikovati_model::sketch::SketchRectangularPatternLayout;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let layout = SketchRectangularPatternLayout {
        counts: [3, 2],
        vectors: [[-10., 0.], [0., 5.]],
        fitted: true,
        suppressed: BTreeSet::from([[1, 1]]),
    };
    let operations: Vec<_> = layout
        .occurrences()
        .unwrap()
        .into_iter()
        .map(|item| item.operation)
        .collect();
    let preview = session
        .preview_selected_sketch_copies(DocumentSketchId(1), &[SketchEntityId(3)], &operations)
        .unwrap();
    let positions: Vec<_> = preview
        .members
        .iter()
        .map(|id| {
            let line = preview.geometry.line(*id).unwrap();
            preview.geometry.point(line.endpoints[0]).unwrap().position
        })
        .collect();
    assert_eq!(positions, [[0., 5.], [-5., 0.], [-10., 0.], [-10., 5.]]);
    let reports = session
        .copy_selected_sketch_entities_batch_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            &operations,
            |result| result,
        )
        .unwrap();
    assert_eq!(
        preview.members,
        reports
            .into_iter()
            .flat_map(|report| report.copies)
            .collect::<Vec<_>>()
    );
    let committed = session.snapshot().sketches.clone();
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(committed.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn linked_copy_preview_matches_commit_and_one_undo_redo() {
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let offsets = [[10., 0.], [0., 5.]];
    let preview = session
        .preview_associative_translations(DocumentSketchId(1), &[SketchEntityId(3)], &offsets)
        .unwrap();
    assert_eq!(preview.geometry.pattern_point_links().len(), 4);
    assert!(session.undo_labels().is_empty());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let reports = session
        .copy_associative_translations_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            &offsets,
            |result| result,
        )
        .unwrap();
    assert_eq!(
        preview.members,
        reports
            .into_iter()
            .flat_map(|report| report.copies)
            .collect::<Vec<_>>()
    );
    let committed = session.snapshot().sketches.clone();
    assert!(
        preview
            .geometry
            .same_recipe(committed.get(DocumentSketchId(1)).unwrap().geometry())
    );
    assert_eq!(session.undo_labels(), ["Copy linked sketch instances"]);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(committed.same_evaluated_state(&session.snapshot().sketches));
}

#[test]
fn parameter_driven_linked_copy_recomputes_and_undoes_spacing_edits() {
    use oblikovati_model::sketch::SketchPatternOffsetTerm;
    let mut session = session_with_line();
    let terms = vec![vec![SketchPatternOffsetTerm {
        parameter: 1,
        coefficient: [2., 0.],
    }]];
    let preview = session
        .preview_parametric_translations(DocumentSketchId(1), &[SketchEntityId(3)], &terms)
        .unwrap();
    let reports = session
        .copy_parametric_translations_projected(
            DocumentSketchId(1),
            &[SketchEntityId(3)],
            &terms,
            |result| result,
        )
        .unwrap();
    assert_eq!(preview.members, reports[0].copies);
    let copy = reports[0].copies[0];
    let offset = |session: &DocumentSession<RecordingObserver>| {
        let geometry = session
            .snapshot()
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry();
        let original = geometry.line(SketchEntityId(3)).unwrap();
        let member = geometry.line(copy).unwrap();
        let source = geometry.point(original.endpoints[0]).unwrap().position;
        let target = geometry.point(member.endpoints[0]).unwrap().position;
        [target[0] - source[0], target[1] - source[1]]
    };
    assert!((offset(&session)[0] - 2.).abs() < 1e-7);
    session
        .run("Edit spacing", |snapshot, _| {
            snapshot.parameters.set_expression(1, "3 cm")
        })
        .unwrap();
    assert!((offset(&session)[0] - 6.).abs() < 1e-7);
    session.undo().unwrap();
    assert!((offset(&session)[0] - 2.).abs() < 1e-7);
    session.redo().unwrap();
    assert!((offset(&session)[0] - 6.).abs() < 1e-7);
}

#[test]
fn rectangular_pattern_preview_commit_and_history_keep_record_identity() {
    use oblikovati_model::sketch::SketchRectangularPatternDefinition;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let before = session.snapshot().sketches.clone();
    let definition = SketchRectangularPatternDefinition {
        seed: vec![SketchEntityId(3)],
        counts: [3, 2],
        directions: [[-1., 0.], [0., 1.]],
        spacing: [1, 1],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let preview = session
        .preview_rectangular_pattern(DocumentSketchId(1), definition.clone())
        .unwrap();
    assert!(session.undo_labels().is_empty());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let pattern = session
        .create_rectangular_pattern_projected(DocumentSketchId(1), definition, |result| result)
        .unwrap();
    assert_eq!(
        preview.members,
        pattern
            .members
            .iter()
            .flat_map(|member| member.entities.iter().copied())
            .collect::<Vec<_>>()
    );
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert!(preview.geometry.same_recipe(geometry));
    assert_eq!(geometry.rectangular_patterns(), &[pattern.clone()]);
    assert!(
        session
            .snapshot()
            .sketches
            .entity_ids()
            .contains(&pattern.id.0)
    );
    let after = session.snapshot().sketches.clone();
    assert_eq!(session.undo_labels(), ["Create rectangular sketch pattern"]);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
}

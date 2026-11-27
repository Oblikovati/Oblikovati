// SPDX-License-Identifier: GPL-2.0-only
//! Linked copy and rectangular layout preview tests.
use super::*;

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

// SPDX-License-Identifier: GPL-2.0-only
//! Rectangular pattern preview and editing tests.
use super::*;

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
    assert_eq!(
        geometry.rectangular_patterns(),
        std::slice::from_ref(&pattern)
    );
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

#[test]
fn rectangular_pattern_edit_preview_cancel_and_undo_redo_keep_member_identity() {
    use oblikovati_model::sketch::SketchRectangularPatternDefinition;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let mut definition = SketchRectangularPatternDefinition {
        seed: vec![SketchEntityId(3)],
        counts: [3, 2],
        directions: [[-1., 0.], [0., 1.]],
        spacing: [1, 1],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let original = session
        .create_rectangular_pattern_projected(DocumentSketchId(1), definition.clone(), |result| {
            result
        })
        .unwrap();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    definition.counts = [4, 2];
    let preview = session
        .preview_rectangular_pattern_edit(DocumentSketchId(1), original.id, definition.clone())
        .unwrap();
    assert_eq!(preview.members.len(), 7);
    assert_eq!(session.generation(), generation);
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let edit = session
        .edit_rectangular_pattern_projected(
            DocumentSketchId(1),
            original.id,
            definition,
            |result| result,
        )
        .unwrap();
    assert_eq!(
        preview.members,
        edit.pattern
            .members
            .iter()
            .flat_map(|member| member.entities.iter().copied())
            .collect::<Vec<_>>()
    );
    for member in &original.members {
        assert_eq!(
            edit.pattern
                .members
                .iter()
                .find(|item| item.index == member.index)
                .unwrap(),
            member
        );
    }
    let after = session.snapshot().sketches.clone();
    assert!(
        preview
            .geometry
            .same_recipe(after.get(DocumentSketchId(1)).unwrap().geometry())
    );
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(after.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .rectangular_patterns()[0]
            .definition
            .counts,
        [4, 2]
    );
}

#[test]
fn expression_pattern_preview_and_commit_share_parameters_without_preview_reservations() {
    use oblikovati_model::history::SketchRectangularPatternRequest;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let request = SketchRectangularPatternRequest {
        seed: vec![SketchEntityId(3)],
        counts: [3, 2],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["width".into(), "5 mm".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let before = session.snapshot().clone();
    let preview = session
        .preview_pattern_from_expressions(DocumentSketchId(1), request.clone())
        .unwrap();
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    let created = session
        .create_pattern_from_expressions_projected(DocumentSketchId(1), request, |result| result)
        .unwrap();
    assert_eq!(
        preview.members,
        created
            .members
            .iter()
            .flat_map(|member| member.entities.iter().copied())
            .collect::<Vec<_>>()
    );
    let after = session.snapshot();
    assert!(
        preview
            .geometry
            .same_recipe(after.sketches.get(DocumentSketchId(1)).unwrap().geometry())
    );
    assert_eq!(
        after
            .parameters
            .get(created.definition.spacing[0])
            .unwrap()
            .source,
        "width"
    );
    session.undo().unwrap();
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
}

#[test]
fn invalid_pattern_expression_does_not_publish_preview_or_parameters() {
    use oblikovati_model::history::SketchRectangularPatternRequest;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let request = SketchRectangularPatternRequest {
        seed: vec![SketchEntityId(3)],
        counts: [2, 1],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["width".into(), "missing".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let before = session.snapshot().clone();
    assert!(
        session
            .preview_pattern_from_expressions(DocumentSketchId(1), request.clone())
            .is_err()
    );
    assert!(
        session
            .create_pattern_from_expressions_projected(DocumentSketchId(1), request, |result| {
                result
            })
            .is_err()
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(session.undo_labels().is_empty());
}

#[test]
fn editing_pattern_expressions_previews_without_mutation_and_undoes_together() {
    use oblikovati_model::history::SketchRectangularPatternRequest;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let mut request = SketchRectangularPatternRequest {
        seed: vec![SketchEntityId(3)],
        counts: [3, 2],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["10 mm".into(), "5 mm".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let original = session
        .create_pattern_from_expressions_projected(DocumentSketchId(1), request.clone(), |result| {
            result
        })
        .unwrap();
    let before = session.snapshot().clone();
    request.counts[0] = 4;
    request.spacing[0] = "20 mm".into();
    let preview = session
        .preview_pattern_edit_from_expressions(DocumentSketchId(1), original.id, request.clone())
        .unwrap();
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    let edit = session
        .edit_pattern_from_expressions_projected(
            DocumentSketchId(1),
            original.id,
            request,
            |result| result,
        )
        .unwrap();
    assert_eq!(
        preview.members,
        edit.pattern
            .members
            .iter()
            .flat_map(|member| member.entities.iter().copied())
            .collect::<Vec<_>>()
    );
    assert_eq!(edit.pattern.definition.spacing, original.definition.spacing);
    assert_eq!(
        session
            .snapshot()
            .parameters
            .get(original.definition.spacing[0])
            .unwrap()
            .source,
        "20 mm"
    );
    let after = session.snapshot().clone();
    session.undo().unwrap();
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    session.redo().unwrap();
    assert!(after.parameters.same_recipe(&session.snapshot().parameters));
    assert!(
        after
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
}

#[test]
fn failed_pattern_expression_edit_keeps_the_record_and_sources() {
    use oblikovati_model::history::SketchRectangularPatternRequest;
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let mut request = SketchRectangularPatternRequest {
        seed: vec![SketchEntityId(3)],
        counts: [2, 1],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["10 mm".into(), "5 mm".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let created = session
        .create_pattern_from_expressions_projected(DocumentSketchId(1), request.clone(), |result| {
            result
        })
        .unwrap();
    let before = session.snapshot().clone();
    let labels = session
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    request.spacing[1] = "missing + 1 mm".into();
    assert!(
        session
            .preview_pattern_edit_from_expressions(DocumentSketchId(1), created.id, request.clone())
            .is_err()
    );
    assert!(
        session
            .edit_pattern_from_expressions_projected(
                DocumentSketchId(1),
                created.id,
                request,
                |result| result
            )
            .is_err()
    );
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    assert_eq!(labels, session.undo_labels());
}

#[test]
fn editing_a_pattern_does_not_rewrite_shared_user_parameters() {
    use oblikovati_model::{
        history::SketchRectangularPatternRequest, sketch::SketchRectangularPatternDefinition,
    };
    use std::collections::BTreeSet;
    let mut session = session_with_line();
    let definition = SketchRectangularPatternDefinition {
        seed: vec![SketchEntityId(3)],
        counts: [2, 1],
        directions: [[1., 0.], [0., 1.]],
        spacing: [1, 1],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    let pattern = session
        .create_rectangular_pattern_projected(DocumentSketchId(1), definition, |result| result)
        .unwrap();
    let mut request = SketchRectangularPatternRequest {
        seed: vec![SketchEntityId(3)],
        counts: [3, 1],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["1 cm".into(), "1 cm".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    session
        .edit_pattern_from_expressions_projected(
            DocumentSketchId(1),
            pattern.id,
            request.clone(),
            |result| result,
        )
        .unwrap();
    let before = session.snapshot().clone();
    request.spacing = ["2 cm".into(), "2 cm".into()];
    assert!(
        session
            .edit_pattern_from_expressions_projected(
                DocumentSketchId(1),
                pattern.id,
                request,
                |result| result
            )
            .is_err()
    );
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
}

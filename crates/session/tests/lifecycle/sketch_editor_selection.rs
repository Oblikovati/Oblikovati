// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_deletion_selection::{assert_mixed_cascade, selection_session};
use super::sketch_dimension_delete::dimension_session_from_snapshot;
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
use oblikovati_session::selection::{
    Selectable, SelectionFilter, SelectionKind, SelectionSet, SketchPick, SketchPickTarget,
};
const EDITOR_SKETCH: DocumentSketchId = DocumentSketchId(1);
fn editor_picks(session: &DocumentSession<RecordingObserver>) -> SelectionSet<SketchPick> {
    let mut picks = SelectionSet::default();
    for target in [
        SketchPickTarget::Entity(SketchEntityId(2)),
        SketchPickTarget::Dimension(SketchConstraintId(1)),
        SketchPickTarget::Dimension(SketchConstraintId(3)),
        SketchPickTarget::GeometricConstraint(SketchConstraintId(2)),
    ] {
        let pick = session.sketch_pick(EDITOR_SKETCH, target).unwrap();
        assert!(!picks.contains(&pick));
        assert!(picks.add(pick));
        assert!(!picks.add(pick));
    }
    picks
}
#[test]
fn checked_picks_preserve_the_three_editor_categories() {
    let session = selection_session(None);
    for (target, expected) in [
        (
            SketchPickTarget::Entity(SketchEntityId(2)),
            SelectionKind::SketchEntity,
        ),
        (
            SketchPickTarget::Dimension(SketchConstraintId(1)),
            SelectionKind::SketchDimension,
        ),
        (
            SketchPickTarget::GeometricConstraint(SketchConstraintId(2)),
            SelectionKind::SketchConstraint,
        ),
    ] {
        assert_pick_kind(&session, target, expected);
    }
}
#[test]
fn missing_entities_and_wrong_relation_categories_cannot_mint_picks() {
    let session = selection_session(None);
    for target in [
        SketchPickTarget::Entity(SketchEntityId(99)),
        SketchPickTarget::Dimension(SketchConstraintId(2)),
        SketchPickTarget::GeometricConstraint(SketchConstraintId(1)),
    ] {
        assert!(session.sketch_pick(EDITOR_SKETCH, target).is_err());
    }
}
#[test]
fn missing_sketch_cannot_mint_a_pick() {
    let session = selection_session(None);
    assert!(
        session
            .sketch_pick(
                DocumentSketchId(99),
                SketchPickTarget::Entity(SketchEntityId(2))
            )
            .is_err()
    );
}
fn assert_pick_kind(
    session: &DocumentSession<RecordingObserver>,
    target: SketchPickTarget,
    expected: SelectionKind,
) {
    let picked = session.sketch_pick(EDITOR_SKETCH, target).unwrap();
    assert_eq!(picked.target(), target);
    assert_eq!(picked.selection_kind(), expected);
}
#[test]
fn editor_delete_clears_picks_retains_filter_and_records_one_reversible_edit() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let mut picks = editor_picks(&session);
    picks.set_filter(SelectionFilter::None);
    let report = delete_editor(&mut session, &mut picks);
    assert_deleted_report(&report);
    assert!(picks.items().is_empty());
    assert_eq!(picks.filter(), &SelectionFilter::None);
    assert_eq!(session.undo_labels(), ["Delete Sketch Geometry"]);
    assert_mixed_cascade(&session);
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    session.redo().unwrap();
    assert_mixed_cascade(&session);
}
#[test]
fn no_editor_active_tool_and_empty_selection_do_not_publish_history_or_clear_picks() {
    let mut session = selection_session(None);
    let mut picks = editor_picks(&session);
    let before = picks.items().to_vec();
    let generation = session.generation();
    for (sketch, tool) in [(None, false), (Some(EDITOR_SKETCH), true)] {
        let report = session
            .delete_selected_sketch_picks(sketch, tool, &mut picks)
            .unwrap();
        assert_eq!(report, SketchEntityDeletion::default());
        assert_eq!(picks.items(), before);
    }
    picks.clear();
    let report = session
        .delete_selected_sketch_picks(Some(DocumentSketchId(99)), false, &mut picks)
        .unwrap();
    assert_eq!(report, SketchEntityDeletion::default());
    assert_eq!(session.generation(), generation);
    assert_no_editor_history(&session);
}
#[test]
fn observer_veto_retains_original_selection_and_model_without_history() {
    let mut session = selection_session(Some(EVENT_TRANSACTION_COMMITTED));
    let mut picks = editor_picks(&session);
    let selected = picks.items().to_vec();
    let before = session.snapshot().sketches.clone();
    assert!(
        session
            .delete_selected_sketch_picks(Some(EDITOR_SKETCH), false, &mut picks)
            .is_err()
    );
    assert_eq!(picks.items(), selected);
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert!(session.snapshot().parameters.get(2).is_some());
    assert!(session.undo_labels().is_empty());
}
#[test]
fn a_different_active_sketch_refuses_the_entire_selection_before_mutation() {
    let mut session = two_sketch_owner();
    let mut picks = editor_picks(&session);
    let before = session.snapshot().sketches.clone();
    assert!(
        session
            .delete_selected_sketch_picks(Some(DocumentSketchId(4)), false, &mut picks)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(picks.items().len(), 4);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn a_foreign_document_with_identical_local_ids_cannot_delete_selected_geometry() {
    let source = selection_session(None);
    let mut picks = editor_picks(&source);
    let mut recipient = foreign_owner(&source);
    let before = recipient.snapshot().sketches.clone();
    let error = recipient
        .delete_selected_sketch_picks(Some(EDITOR_SKETCH), false, &mut picks)
        .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected current document, generation and sketch")
    );
    assert!(recipient.snapshot().sketches.same_recipe(&before));
    assert_eq!(picks.items().len(), 4);
    assert!(recipient.undo_labels().is_empty());
}
#[test]
fn unrelated_edit_invalidates_old_picks_even_when_entity_ids_still_exist() {
    let mut session = selection_session(None);
    let mut picks = editor_picks(&session);
    session
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    assert!(
        session
            .delete_selected_sketch_picks(Some(EDITOR_SKETCH), false, &mut picks)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
    assert_eq!(session.undo_labels(), ["Width"]);
    assert_eq!(picks.items().len(), 4);
}
#[test]
fn undo_restores_geometry_without_revalidating_old_presentation_picks() {
    let mut session = selection_session(None);
    let mut old_picks = editor_picks(&session);
    let mut deleted = old_picks.clone();
    delete_editor(&mut session, &mut deleted);
    session.undo().unwrap();
    let generation = session.generation();
    assert!(
        session
            .delete_selected_sketch_picks(Some(EDITOR_SKETCH), false, &mut old_picks)
            .is_err()
    );
    assert_eq!(session.generation(), generation);
    assert_eq!(old_picks.items().len(), 4);
    let mut fresh = editor_picks(&session);
    delete_editor(&mut session, &mut fresh);
    assert_mixed_cascade(&session);
}
#[test]
fn group_abort_restores_recipes_but_does_not_recreate_transient_selection() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let mut picks = editor_picks(&session);
    session.begin("Editor group").unwrap();
    delete_editor(&mut session, &mut picks);
    assert!(picks.items().is_empty());
    session.abort().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert!(picks.items().is_empty());
    assert!(session.undo_labels().is_empty());
}

fn delete_editor(
    session: &mut DocumentSession<RecordingObserver>,
    picks: &mut SelectionSet<SketchPick>,
) -> SketchEntityDeletion {
    session
        .delete_selected_sketch_picks(Some(EDITOR_SKETCH), false, picks)
        .unwrap()
}
fn assert_deleted_report(report: &SketchEntityDeletion) {
    assert_eq!(report.entities, [SketchEntityId(2)]);
    let ids: Vec<u64> = report.constraints.iter().map(|row| row.id.0).collect();
    assert_eq!(ids, [1, 3, 2]);
}
fn assert_no_editor_history(session: &DocumentSession<RecordingObserver>) {
    assert!(session.undo_labels().is_empty());
    assert!(session.observer().observations.is_empty());
}
fn foreign_owner(
    source: &DocumentSession<RecordingObserver>,
) -> DocumentSession<RecordingObserver> {
    let history = DocumentEditHistory::new(
        source.snapshot().clone(),
        RecipeIdAllocator::restore(Some(3), &[1, 2, 3]).unwrap(),
    )
    .unwrap();
    DocumentSession::new(18, history, RecordingObserver::default())
}

fn two_sketch_owner() -> DocumentSession<RecordingObserver> {
    let mut snapshot = selection_session(None).snapshot().clone();
    snapshot
        .sketches
        .add(
            DocumentSketchId(4),
            "Other",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
    dimension_session_from_snapshot(snapshot, None)
}

#[test]
fn editor_history_names_single_category_and_mixed_deletions_truthfully() {
    for (target, label) in SINGLE_PICK_LABELS {
        let mut session = selection_session(None);
        let mut picks = SelectionSet::default();
        picks.add(session.sketch_pick(EDITOR_SKETCH, target).unwrap());
        delete_editor(&mut session, &mut picks);
        assert_eq!(session.undo_labels(), [label]);
    }
}

const SINGLE_PICK_LABELS: [(SketchPickTarget, &str); 3] = [
    (
        SketchPickTarget::Entity(SketchEntityId(2)),
        "Delete Sketch Entities",
    ),
    (
        SketchPickTarget::Dimension(SketchConstraintId(1)),
        "Delete Sketch Dimensions",
    ),
    (
        SketchPickTarget::GeometricConstraint(SketchConstraintId(2)),
        "Delete Sketch Constraints",
    ),
];

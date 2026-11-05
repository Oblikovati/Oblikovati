// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_deletion_selection::selection_session;
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchEntityId},
};
use oblikovati_session::selection::{
    SelectionClickMode::{self, Replace, Toggle},
    SelectionSet, SketchPick, SketchPickTarget,
};
const CLICK_SKETCH: DocumentSketchId = DocumentSketchId(1);
fn curve_pick(session: &DocumentSession<RecordingObserver>) -> SketchPick {
    session
        .sketch_pick(CLICK_SKETCH, SketchPickTarget::Entity(SketchEntityId(2)))
        .unwrap()
}
fn apply_click(
    session: &DocumentSession<RecordingObserver>,
    picked: Option<SketchPick>,
    mode: SelectionClickMode,
    picks: &mut SelectionSet<SketchPick>,
) -> Result<Option<usize>, oblikovati_model::sketch::SketchError> {
    session.apply_sketch_click(CLICK_SKETCH, picked, mode, false, picks)
}
#[test]
fn typed_clicks_build_a_mixed_selection_that_the_editor_deletes_in_one_step() {
    let mut session = selection_session(None);
    let mut picks = SelectionSet::default();
    assert_eq!(
        apply_click(&session, Some(curve_pick(&session)), Replace, &mut picks).unwrap(),
        Some(1)
    );
    select_dimension(&session, &mut picks);
    assert_no_model_edit(&session);
    let report = session
        .delete_selected_sketch_picks(Some(CLICK_SKETCH), false, &mut picks)
        .unwrap();
    assert_eq!(report.entities, [SketchEntityId(2)]);
    assert_eq!(report.constraints.len(), 2);
    assert!(picks.items().is_empty());
    assert_eq!(session.undo_labels(), ["Delete Sketch Geometry"]);
}
fn assert_no_model_edit(session: &DocumentSession<RecordingObserver>) {
    assert_eq!(session.generation(), 0);
    assert!(session.undo_labels().is_empty());
    assert!(session.observer().observations.is_empty());
}
#[test]
fn stale_candidate_refusal_retains_current_selection_and_fresh_replace_recovers() {
    let mut session = selection_session(None);
    let stale = curve_pick(&session);
    change_width(&mut session);
    let fresh = curve_pick(&session);
    let mut picks = SelectionSet::default();
    picks.add(fresh);
    assert!(apply_click(&session, Some(stale), Replace, &mut picks).is_err());
    assert_eq!(picks.items(), &[fresh]);
    assert_eq!(
        apply_click(&session, Some(fresh), Replace, &mut picks).unwrap(),
        Some(1)
    );
    assert_eq!(session.undo_labels(), ["Width"]);
}
fn change_width(session: &mut DocumentSession<RecordingObserver>) {
    session
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
}
#[test]
fn modified_click_cannot_merge_old_generation_picks_but_plain_click_replaces_them() {
    let mut session = selection_session(None);
    let mut picks = SelectionSet::default();
    let stale = curve_pick(&session);
    picks.add(stale);
    change_width(&mut session);
    let fresh = curve_pick(&session);
    assert!(apply_click(&session, Some(fresh), Toggle, &mut picks).is_err());
    assert_eq!(picks.items(), &[stale]);
    assert_eq!(
        apply_click(&session, Some(fresh), Replace, &mut picks).unwrap(),
        Some(1)
    );
    assert_eq!(picks.items(), &[fresh]);
}
#[test]
fn foreign_document_candidate_refuses_without_replacing_the_receiving_selection() {
    let source = selection_session(None);
    let history = DocumentEditHistory::new(
        source.snapshot().clone(),
        RecipeIdAllocator::restore(Some(3), &[1, 2, 3]).unwrap(),
    )
    .unwrap();
    let recipient = DocumentSession::new(18, history, RecordingObserver::default());
    let local = curve_pick(&recipient);
    let mut picks = SelectionSet::default();
    picks.add(local);
    assert!(apply_click(&recipient, Some(curve_pick(&source)), Replace, &mut picks).is_err());
    assert_eq!(picks.items(), &[local]);
    assert_no_model_edit(&recipient);
}
#[test]
fn active_tool_bypasses_selection_validation_and_does_not_clear_or_toggle_picks() {
    let mut session = selection_session(None);
    let stale = curve_pick(&session);
    change_width(&mut session);
    let mut picks = SelectionSet::default();
    picks.add(stale);
    for candidate in [None, Some(stale)] {
        assert_eq!(
            session
                .apply_sketch_click(CLICK_SKETCH, candidate, Replace, true, &mut picks)
                .unwrap(),
            None
        );
        assert_eq!(picks.items(), &[stale]);
    }
    assert_eq!(session.undo_labels(), ["Width"]);
}
#[test]
fn empty_click_can_clear_stale_picks_but_modified_empty_click_keeps_them() {
    let mut session = selection_session(None);
    let mut picks = SelectionSet::default();
    let stale = curve_pick(&session);
    picks.add(stale);
    change_width(&mut session);
    assert_eq!(
        apply_click(&session, None, Toggle, &mut picks).unwrap(),
        None
    );
    assert_eq!(picks.items(), &[stale]);
    assert_eq!(
        apply_click(&session, None, Replace, &mut picks).unwrap(),
        Some(0)
    );
    assert!(picks.items().is_empty());
    assert_eq!(session.undo_labels(), ["Width"]);
}
#[test]
fn missing_active_sketch_refuses_input_before_membership_changes() {
    let session = selection_session(None);
    let mut picks = SelectionSet::default();
    let picked = curve_pick(&session);
    picks.add(picked);
    assert!(
        session
            .apply_sketch_click(DocumentSketchId(99), None, Replace, false, &mut picks)
            .is_err()
    );
    assert_eq!(picks.items(), &[picked]);
    assert_no_model_edit(&session);
}

fn select_dimension(
    session: &DocumentSession<RecordingObserver>,
    picks: &mut SelectionSet<SketchPick>,
) {
    let dimension = session
        .sketch_pick(
            CLICK_SKETCH,
            SketchPickTarget::Dimension(SketchConstraintId(3)),
        )
        .unwrap();
    assert_eq!(
        apply_click(session, Some(dimension), Toggle, picks).unwrap(),
        Some(2)
    );
}

// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_deletion_selection::selection_session;
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::selection::{
    Selectable,
    SelectionClickMode::{Replace, Toggle},
    SelectionKind, SelectionSet,
};
const BROWSER_SKETCH: DocumentSketchId = DocumentSketchId(1);

#[test]
fn browser_pick_is_a_whole_sketch_and_selection_creates_no_history() {
    let session = selection_session(None);
    let pick = session.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    assert_eq!(pick.sketch(), BROWSER_SKETCH);
    assert_eq!(pick.selection_kind(), SelectionKind::Sketch);
    let mut picks = SelectionSet::default();
    assert_eq!(
        session
            .apply_browser_sketch_click(Some(pick), Replace, false, &mut picks)
            .unwrap(),
        Some(1)
    );
    assert_eq!(picks.items(), &[pick]);
    assert_eq!(session.generation(), 0);
    assert!(session.undo_labels().is_empty());
    assert!(session.observer().observations.is_empty());
}
#[test]
fn browser_identity_survives_parameter_edits_and_can_toggle_off() {
    let mut session = selection_session(None);
    let pick = session.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    let mut picks = SelectionSet::default();
    picks.add(pick);
    session
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    assert_eq!(session.browser_sketch_pick(BROWSER_SKETCH).unwrap(), pick);
    assert_eq!(
        session
            .apply_browser_sketch_click(Some(pick), Toggle, false, &mut picks)
            .unwrap(),
        Some(0)
    );
    assert_eq!(session.undo_labels(), ["Width"]);
}
#[test]
fn removed_sketch_cannot_be_selected_and_failed_validation_preserves_membership() {
    let mut session = selection_session(None);
    let pick = session.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    let mut picks = SelectionSet::default();
    picks.add(pick);
    session
        .run("Remove", |snapshot, _| {
            snapshot.sketches.remove(BROWSER_SKETCH)
        })
        .unwrap();
    assert!(session.browser_sketch_pick(BROWSER_SKETCH).is_err());
    assert!(
        session
            .apply_browser_sketch_click(Some(pick), Replace, false, &mut picks)
            .is_err()
    );
    assert_eq!(picks.items(), &[pick]);
    assert_eq!(
        session
            .apply_browser_sketch_click(None, Replace, false, &mut picks)
            .unwrap(),
        Some(0)
    );
}
#[test]
fn foreign_document_pick_cannot_replace_local_browser_selection() {
    let source = selection_session(None);
    let history = DocumentEditHistory::new(
        source.snapshot().clone(),
        RecipeIdAllocator::restore(Some(3), &[1, 2, 3]).unwrap(),
    )
    .unwrap();
    let recipient = DocumentSession::new(18, history, RecordingObserver::default());
    let local = recipient.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    let foreign = source.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    let mut picks = SelectionSet::default();
    picks.add(local);
    assert!(
        recipient
            .apply_browser_sketch_click(Some(foreign), Replace, false, &mut picks)
            .is_err()
    );
    assert_eq!(picks.items(), &[local]);
    let mut foreign_picks = SelectionSet::default();
    foreign_picks.add(foreign);
    assert!(
        recipient
            .apply_browser_sketch_click(Some(local), Toggle, false, &mut foreign_picks)
            .is_err()
    );
    assert_eq!(foreign_picks.items(), &[foreign]);
}
#[test]
fn tool_clicks_and_modified_empty_clicks_preserve_selection() {
    let session = selection_session(None);
    let pick = session.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    let mut picks = SelectionSet::default();
    picks.add(pick);
    for candidate in [None, Some(pick)] {
        assert_eq!(
            session
                .apply_browser_sketch_click(candidate, Toggle, true, &mut picks)
                .unwrap(),
            None
        );
        assert_eq!(picks.items(), &[pick]);
    }
    assert_eq!(
        session
            .apply_browser_sketch_click(None, Toggle, false, &mut picks)
            .unwrap(),
        None
    );
    assert_eq!(picks.items(), &[pick]);
}

#[test]
fn browser_identity_follows_existence_through_undo_and_redo() {
    let mut session = selection_session(None);
    let pick = session.browser_sketch_pick(BROWSER_SKETCH).unwrap();
    session
        .run("Remove", |snapshot, _| {
            snapshot.sketches.remove(BROWSER_SKETCH)
        })
        .unwrap();
    let mut picks = SelectionSet::default();
    assert!(
        session
            .apply_browser_sketch_click(Some(pick), Replace, false, &mut picks)
            .is_err()
    );
    session.undo().unwrap();
    assert_eq!(session.browser_sketch_pick(BROWSER_SKETCH).unwrap(), pick);
    assert_eq!(
        session
            .apply_browser_sketch_click(Some(pick), Replace, false, &mut picks)
            .unwrap(),
        Some(1)
    );
    session.redo().unwrap();
    assert!(
        session
            .apply_browser_sketch_click(Some(pick), Toggle, false, &mut picks)
            .is_err()
    );
    assert_eq!(picks.items(), &[pick]);
}

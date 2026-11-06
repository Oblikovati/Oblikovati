// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_deletion_selection::{assert_mixed_cascade, selection_session};
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};
const SKETCH: DocumentSketchId = DocumentSketchId(1);
pub(super) fn select_owned_mixed(session: &mut DocumentSession<RecordingObserver>) {
    for target in [
        SketchPickTarget::Entity(SketchEntityId(2)),
        SketchPickTarget::Dimension(SketchConstraintId(1)),
        SketchPickTarget::Dimension(SketchConstraintId(3)),
        SketchPickTarget::GeometricConstraint(SketchConstraintId(2)),
    ] {
        session
            .set_sketch_selection(SKETCH, Some(target), SelectionClickMode::Toggle, false)
            .unwrap();
    }
}
#[test]
fn owned_deletion_clears_only_adopted_picks_and_has_one_undo_step() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let report = session
        .delete_editor_selection(Some(SKETCH), false)
        .unwrap();
    assert!(!report.entities.is_empty());
    assert_mixed_cascade(&session);
    assert!(session.selected_editor_picks().is_empty());
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert!(session.selected_editor_picks().is_empty());
    session.redo().unwrap();
    assert_mixed_cascade(&session);
}
#[test]
fn owned_deletion_veto_keeps_the_selection_and_recipe() {
    let mut session = selection_session(Some(EVENT_TRANSACTION_COMMITTED));
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let picks = session.selected_editor_picks();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    let retained = session.selected_editor_picks();
    assert_eq!(
        retained
            .iter()
            .map(|pick| pick.target())
            .collect::<Vec<_>>(),
        picks.iter().map(|pick| pick.target()).collect::<Vec<_>>()
    );
}
#[test]
fn owned_deletion_respects_editor_tool_and_empty_guards() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    for (sketch, active) in [(None, false), (Some(SKETCH), true)] {
        assert!(
            session
                .delete_editor_selection(sketch, active)
                .unwrap()
                .entities
                .is_empty()
        );
        assert_eq!(session.selected_editor_picks().len(), 4);
    }
    session.clear_transient_selection();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .unwrap()
            .entities
            .is_empty()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
}

#[test]
fn owned_deletion_does_not_rebind_stale_picks_after_an_unrelated_abort() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    session.begin("Unrelated").unwrap();
    session.abort().unwrap();
    assert!(session.selected_editor_picks().is_empty());
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .is_err()
    );
    assert!(session.selected_editor_picks().is_empty());
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
}

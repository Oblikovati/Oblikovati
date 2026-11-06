// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    sketch_deletion_selection::selection_session, sketch_owned_deletion::select_owned_mixed,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCurveFormat, SketchEntityId},
};
fn format_pick_targets(
    session: &DocumentSession<RecordingObserver>,
) -> Vec<oblikovati_session::selection::SketchPickTarget> {
    session
        .selected_editor_picks()
        .iter()
        .map(|pick| pick.target())
        .collect()
}
const SKETCH: DocumentSketchId = DocumentSketchId(1);
#[test]
fn curve_conversion_retains_checked_picks_and_uses_one_metadata_history_step() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let targets = format_pick_targets(&session);
    let report = session
        .toggle_editor_curve_format(SKETCH, SketchCurveFormat::Construction)
        .unwrap();
    assert_eq!(report.applicable, [SketchEntityId(2)]);
    assert_eq!(format_pick_targets(&session), targets);
    assert_eq!(session.undo_labels(), ["Construction"]);
    let changed = session.snapshot().sketches.clone();
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert!(session.selected_editor_picks().is_empty());
    session.redo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&changed));
}
#[test]
fn curve_conversion_veto_restores_recipe_and_rebinds_only_verified_current_picks() {
    let mut session = selection_session(Some(EVENT_TRANSACTION_COMMITTED));
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let targets = format_pick_targets(&session);
    assert!(
        session
            .toggle_editor_curve_format(SKETCH, SketchCurveFormat::Construction)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(format_pick_targets(&session), targets);
    assert!(session.undo_labels().is_empty());
    assert!(!session.in_transaction());
}
#[test]
fn empty_format_selection_has_no_history_or_generation_change() {
    let mut session = selection_session(None);
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    assert!(
        session
            .toggle_editor_curve_format(SKETCH, SketchCurveFormat::Construction)
            .unwrap()
            .applicable
            .is_empty()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
}
#[test]
fn a_refused_format_command_cannot_rebind_picks_from_an_unrelated_abort() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    session.begin("Unrelated").unwrap();
    session.abort().unwrap();
    let before = session.snapshot().sketches.clone();
    assert!(
        session
            .toggle_editor_curve_format(SKETCH, SketchCurveFormat::Construction)
            .is_err()
    );
    assert!(session.selected_editor_picks().is_empty());
    assert!(session.snapshot().sketches.same_recipe(&before));
}

#[test]
fn inapplicable_centerline_selection_keeps_scope_and_history_unchanged() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let generation = session.generation();
    let picks = format_pick_targets(&session);
    let report = session
        .toggle_editor_curve_format(SKETCH, SketchCurveFormat::Centerline)
        .unwrap();
    assert!(report.applicable.is_empty());
    assert_eq!(session.generation(), generation);
    assert_eq!(format_pick_targets(&session), picks);
    assert!(session.undo_labels().is_empty());
}

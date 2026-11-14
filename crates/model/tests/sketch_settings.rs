// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use document_snapshot::empty_snapshot;
use oblikovati_api::types::*;
use oblikovati_model::{history::DocumentEditHistory, sketch_settings::default_sketch_settings};

#[test]
fn defaults_match_go_inference_and_grid_values() {
    let defaults = default_sketch_settings();
    assert!(defaults.infer_constraints && defaults.auto_apply_constraints);
    assert_eq!(defaults.constraint_priority, PRIORITY_HORIZONTAL_VERTICAL);
    assert_eq!(
        (defaults.x_snap_spacing, defaults.y_snap_spacing),
        (0.1, 0.1)
    );
    assert_eq!(defaults.snaps_per_minor_grid, 1);
    assert_eq!(defaults.minor_lines_per_major_grid_line, 10);
}
#[test]
fn defaults_match_go_creation_and_relax_values() {
    let defaults = default_sketch_settings();
    assert!(defaults.persist_inferred_constraints && defaults.display_constraints_on_creation);
    assert!(defaults.edit_dimensions_when_created);
    assert_eq!(
        defaults.over_constrained_behavior,
        OVER_CONSTRAINED_APPLY_DRIVEN
    );
    assert!(!defaults.enable_relax_mode);
    assert!(defaults.keep_dimensions_with_equation_in_relax_mode);
}
#[test]
fn read_defaults_keeps_record_absent_and_explicit_zero_values_remain_explicit() {
    let mut snapshot = empty_snapshot();
    assert_eq!(
        snapshot.effective_sketch_settings(),
        default_sketch_settings()
    );
    assert!(snapshot.sketch_settings.is_none());
    snapshot.sketch_settings = Some(SketchSettings::default());
    assert_eq!(
        snapshot.effective_sketch_settings(),
        SketchSettings::default()
    );
    let mut detached = snapshot.effective_sketch_settings();
    detached.infer_constraints = true;
    assert!(detached.infer_constraints);
    assert!(!snapshot.effective_sketch_settings().infer_constraints);
}
#[test]
fn refresh_publishes_settings_and_rejects_a_stale_noop_comparison() {
    let mut history = DocumentEditHistory::new(empty_snapshot(), Default::default()).unwrap();
    let initial = history.generation();
    history
        .prepare_refresh(|snapshot| {
            snapshot.sketch_settings = Some(default_sketch_settings());
            Ok::<(), String>(())
        })
        .unwrap()
        .publish();
    assert!(history.generation() > initial);
    let explicit = history.generation();
    history
        .prepare_refresh(|_| Ok::<(), String>(()))
        .unwrap()
        .publish();
    assert_eq!(history.generation(), explicit);
    assert!(history.snapshot().sketch_settings.is_some());
}

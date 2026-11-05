// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{types::*, wire::InferenceOptionsView};
use oblikovati_model::sketch_settings::{
    default_sketch_settings, inference_options_view, with_inference_options,
};
fn request(priority: &str) -> InferenceOptionsView {
    InferenceOptionsView {
        infer_enabled: false,
        constrain_enabled: true,
        priority: priority.into(),
    }
}
#[test]
fn all_frozen_priorities_round_trip_through_the_wire_view() {
    for priority in ["parallelPerpendicular", "horizontalVertical", "none"] {
        let candidate =
            with_inference_options(&default_sketch_settings(), &request(priority)).unwrap();
        assert_eq!(inference_options_view(&candidate), request(priority));
    }
}
#[test]
fn empty_priority_retains_the_current_enum_while_both_booleans_are_replaced() {
    let settings = with_inference_options(&default_sketch_settings(), &request("none")).unwrap();
    let candidate = with_inference_options(&settings, &InferenceOptionsView::default()).unwrap();
    assert_eq!(candidate.constraint_priority, PRIORITY_NONE);
    assert!(!candidate.infer_constraints && !candidate.auto_apply_constraints);
    assert_eq!(settings.constraint_priority, PRIORITY_NONE);
    assert!(settings.auto_apply_constraints);
}
#[test]
fn inference_only_edit_preserves_grid_display_and_relax_values() {
    let mut settings = default_sketch_settings();
    settings.x_snap_spacing = 0.75;
    settings.y_snap_spacing = 2.0;
    settings.enable_relax_mode = true;
    let mut candidate = with_inference_options(&settings, &request("none")).unwrap();
    candidate.infer_constraints = settings.infer_constraints;
    candidate.auto_apply_constraints = settings.auto_apply_constraints;
    candidate.constraint_priority = settings.constraint_priority;
    assert_eq!(candidate, settings);
}
#[test]
fn unknown_priority_spelling_is_an_atomic_typed_decline_with_context() {
    let settings = default_sketch_settings();
    for spelling in ["NONE", " none", "horizontal", "enum(?)", "50435"] {
        let error = with_inference_options(&settings, &request(spelling)).unwrap_err();
        assert_eq!(error.0, spelling);
        assert!(error.to_string().contains(spelling));
        assert!(error.to_string().contains("expected parallelPerpendicular"));
        assert!(std::error::Error::source(&error).is_none());
    }
    assert_eq!(settings, default_sketch_settings());
}
#[test]
fn unknown_stored_enum_uses_go_marker_and_empty_priority_does_not_repair_it() {
    let mut settings = default_sketch_settings();
    settings.constraint_priority = ConstraintInferencePriority(9876);
    assert_eq!(inference_options_view(&settings).priority, "enum(?)");
    let candidate = with_inference_options(&settings, &request("")).unwrap();
    assert_eq!(candidate.constraint_priority, settings.constraint_priority);
    let repaired = with_inference_options(&settings, &request("none")).unwrap();
    assert_eq!(inference_options_view(&repaired).priority, "none");
}

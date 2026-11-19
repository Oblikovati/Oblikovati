// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::FeatureId;
#[test]
fn selected_face_outline_is_green_and_browser_feature_outline_is_blue() {
    let mut commands = GuiCommandState::default();
    assert_eq!(feature_outline_color(&commands), EDGE_COLOR);
    commands.model_face_chosen = true;
    assert_eq!(feature_outline_color(&commands), EDGE_HOVER_COLOR);
    commands.selected_body_keys.push("body".into());
    assert_eq!(feature_outline_color(&commands), EDGE_COLOR);
}
#[test]
fn hover_uses_segment_distance_in_window_pixels() {
    let edge = [[[10, 20], [30, 20]], [[30, 20], [35, 25]]];
    assert!(edge_group_hovered([20., 27.], &edge));
    assert!(!edge_group_hovered([20., 27.], &[[[80, 20], [100, 20]]]));
    assert!(!edge_group_hovered([20., 29.], &edge));
    assert!(edge_group_hovered([36., 26.], &edge));
    assert!(!edge_group_hovered([f32::NAN, 20.], &edge));
    assert_eq!(segment_distance_squared([3., 4.], [[0, 0], [0, 0]]), 25.);
}
fn extrude(suppressed: bool) -> GuiBrowserFeature {
    GuiBrowserFeature {
        id: FeatureId(1),
        name: "Extrude1".into(),
        suppressed,
        extended: Some("New Solid, 20 mm".into()),
        kind: "Extrusion",
    }
}
#[test]
fn a_row_says_its_operation_and_depth_only_when_extended_names_are_on() {
    assert_eq!(feature_label(&extrude(false), false), "Extrude1");
    assert_eq!(
        feature_label(&extrude(false), true),
        "Extrude1 (New Solid, 20 mm)"
    );
    assert_eq!(
        feature_label(&extrude(true), false),
        "Extrude1 (suppressed)"
    );
}

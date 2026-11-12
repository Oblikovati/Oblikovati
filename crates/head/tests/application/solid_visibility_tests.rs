// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn capture_with_patch(color: [u8; 4], fill: bool) -> ViewportCapture {
    let mut rgba = vec![40_u8; 32 * 32 * 4];
    for y in 10..22 {
        for x in 10..22 {
            if fill || x == 10 || x == 21 || y == 10 || y == 21 {
                rgba[(y * 32 + x) * 4..(y * 32 + x) * 4 + 4].copy_from_slice(&color);
            }
        }
    }
    ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [32, 32],
        rgba,
    }
}
#[test]
fn visibility_accepts_solid_fill_in_plain_selected_and_dimmed_colors() {
    for color in [[0, 255, 0, 255], [63, 153, 255, 255], [22, 54, 90, 255]] {
        assert!(
            visible_solid_fill(&capture_with_patch(color, true), &[]),
            "expected visible fixture color {color:?}"
        );
    }
}
#[test]
fn a_blue_outline_or_gui_fill_does_not_prove_a_visible_solid() {
    assert!(!visible_solid_fill(
        &capture_with_patch([63, 153, 255, 255], false),
        &[]
    ));
    assert!(!visible_solid_fill(
        &capture_with_patch([63, 153, 255, 255], true),
        &[[8., 8., 24., 24.]]
    ));
    assert!(!visible_solid_fill(
        &capture_with_patch([40, 40, 40, 255], true),
        &[]
    ));
}

#[test]
fn black_appearance_requires_model_pixels_outside_all_ui_panels() {
    let black = capture_with_patch([19, 19, 19, 255], true);
    assert!(visible_appearance_fill(&black, &[], [19, 19, 19]));
    assert!(!visible_appearance_fill(
        &black,
        &[[8., 8., 24., 24.]],
        [19, 19, 19]
    ));
    assert!(!visible_appearance_fill(
        &capture_with_patch([63, 153, 255, 255], true),
        &[],
        [2, 2, 2]
    ));
}
#[test]
fn fixture_point_excludes_ui_and_requires_filled_green() {
    let green = capture_with_patch([0, 255, 0, 255], true);
    assert_eq!(visible_fixture_point(&green, &[]), Some([12, 12]));
    assert_eq!(visible_fixture_point(&green, &[[8., 8., 24., 24.]]), None);
    assert_eq!(
        visible_fixture_point(&capture_with_patch([0, 255, 0, 255], false), &[]),
        None
    );
    assert_eq!(
        visible_fixture_point(&capture_with_patch([63, 153, 255, 255], true), &[]),
        None
    );
}

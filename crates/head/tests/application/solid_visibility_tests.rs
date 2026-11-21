// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn preview_requires_orange_fill_outside_the_gui() {
    let orange = [189, 156, 64, 255];
    assert!(visible_preview_fill(&capture_with_patch(orange, true), &[]));
    assert!(!visible_preview_fill(
        &capture_with_patch(orange, false),
        &[]
    ));
    assert!(!visible_preview_fill(
        &capture_with_patch(orange, true),
        &[[8., 8., 24., 24.]]
    ));
    assert!(!visible_preview_fill(
        &capture_with_patch([210, 212, 214, 255], true),
        &[]
    ));
}
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
    for color in [[210, 212, 214, 255], [63, 153, 255, 255], [22, 54, 90, 255]] {
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
    assert!(!visible_solid_fill(
        &capture_with_patch([255, 255, 255, 255], true),
        &[]
    ));
}

#[test]
fn selected_face_requires_blue_fill_outside_the_gui() {
    let blue = capture_with_patch([46, 108, 177, 255], true);
    assert!(visible_selected_fill(&blue, &[]));
    assert!(!visible_selected_fill(&blue, &[[8., 8., 24., 24.]]));
    assert!(!visible_selected_fill(
        &capture_with_patch([46, 108, 177, 255], false),
        &[]
    ));
    assert!(!visible_selected_fill(
        &capture_with_patch([210, 212, 214, 255], true),
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
fn fixture_point_excludes_ui_and_requires_filled_neutral() {
    let neutral = capture_with_patch([210, 212, 214, 255], true);
    assert_eq!(visible_fixture_point(&neutral, &[]), Some([12, 12]));
    assert_eq!(visible_fixture_point(&neutral, &[[8., 8., 24., 24.]]), None);
    assert_eq!(
        visible_fixture_point(&capture_with_patch([210, 212, 214, 255], false), &[]),
        None
    );
    assert_eq!(
        visible_fixture_point(&capture_with_patch([63, 153, 255, 255], true), &[]),
        None
    );
}

#[test]
fn fixture_point_recognizes_a_lit_neutral_face_but_not_black() {
    let lit = capture_with_patch([174, 176, 178, 255], true);
    let black = capture_with_patch([19, 19, 19, 255], true);
    assert!(visible_fixture_point(&lit, &[]).is_some());
    assert_eq!(visible_fixture_point(&black, &[]), None);
}

#[test]
fn fixture_bounds_exclude_neutral_pixels_in_application_chrome() {
    let mut capture = capture_with_patch([210, 212, 214, 255], true);
    capture.rgba[(2 * 32 + 2) * 4..(2 * 32 + 2) * 4 + 4].copy_from_slice(&[210, 212, 214, 255]);
    assert_eq!(
        visible_fixture_bounds(&capture, &[[0., 0., 8., 8.]]),
        Some(([10, 10], [21, 21]))
    );
    assert_eq!(fixture_pixel_count(&capture, &[[0., 0., 8., 8.]]), 144);
}

#[test]
fn selected_face_fill_requires_teal_pixels_outside_windows() {
    let teal = capture_with_patch([0, 180, 170, 255], true);
    assert!(visible_selected_face_fill(&teal, &[]));
    assert!(!visible_selected_face_fill(&teal, &[[8., 8., 24., 24.]]));
    assert!(!visible_selected_face_fill(
        &capture_with_patch([210, 212, 214, 255], true),
        &[]
    ));
    assert!(!visible_selected_face_fill(
        &capture_with_patch([0, 180, 170, 255], false),
        &[]
    ));
}

// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn gradient_capture() -> ViewportCapture {
    let rgba = (0..8)
        .flat_map(|row| [row, row, row, 255].repeat(8))
        .collect();
    ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [8, 8],
        rgba,
    }
}

#[test]
fn navigation_overlay_is_excluded_but_browser_panel_is_detected() {
    let mut capture = gradient_capture();
    assert!(gradient_region_visible(&capture, [0, 8], [0, 8], &[]));
    capture.rgba[4..8].copy_from_slice(&[100, 100, 100, 255]);
    assert!(!gradient_region_visible(&capture, [0, 8], [0, 8], &[]));
    let controls = [[1., 0., 2., 1.]];
    assert!(gradient_region_visible(&capture, [0, 8], [0, 8], &controls));
    capture.rgba[8..12].copy_from_slice(&[20, 20, 20, 255]);
    assert!(!gradient_region_visible(
        &capture,
        [0, 8],
        [0, 8],
        &controls
    ));
}

#[test]
fn a_uniform_panel_or_fully_occluded_region_cannot_prove_viewport_space() {
    let mut capture = gradient_capture();
    assert!(!gradient_region_visible(
        &capture,
        [0, 8],
        [0, 8],
        &[[0., 0., 8., 8.]]
    ));
    capture.rgba.fill(20);
    assert!(!gradient_region_visible(&capture, [0, 8], [0, 8], &[]));
    assert!(!gradient_region_visible(&capture, [0, 8], [0, 0], &[]));
}

#[test]
fn navigation_rectangle_uses_exclusive_bottom_and_right_bounds() {
    let bounds = [1., 2., 3., 4.];
    assert!(inside_control(1, 2, bounds));
    assert!(!inside_control(3, 2, bounds));
    assert!(!inside_control(1, 4, bounds));
}

#[test]
fn an_open_menu_is_excluded_without_hiding_uncovered_geometry() {
    let mut capture = gradient_capture();
    for y in 2..6 {
        for x in 1..4 {
            let offset = (y * 8 + x) * 4;
            capture.rgba[offset..offset + 4].copy_from_slice(&[100, 100, 100, 255]);
        }
    }
    let menu = [[1., 2., 4., 6.]];
    assert!(!gradient_region_visible(&capture, [0, 8], [0, 8], &[]));
    assert!(gradient_region_visible(&capture, [0, 8], [0, 8], &menu));
    let geometry = (3 * 8 + 6) * 4;
    capture.rgba[geometry..geometry + 4].copy_from_slice(&[200, 200, 200, 255]);
    assert!(!gradient_region_visible(&capture, [0, 8], [0, 8], &menu));
}

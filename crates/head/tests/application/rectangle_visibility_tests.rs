// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn rectangle_capture(missing_edge: Option<usize>) -> ViewportCapture {
    let mut capture = ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [64, 64],
        rgba: vec![40; 64 * 64 * 4],
    };
    for coordinate in 10..=50 {
        for (edge, [x, y]) in [
            [coordinate, 10],
            [coordinate, 50],
            [10, coordinate],
            [50, coordinate],
        ]
        .into_iter()
        .enumerate()
        {
            if Some(edge) != missing_edge {
                let index = (y * 64 + x) * 4;
                capture.rgba[index..index + 4].copy_from_slice(&[160, 160, 160, 255]);
            }
        }
    }
    capture
}

#[test]
fn accepts_four_edges_and_rejects_each_missing_edge() {
    let corners = [[10, 10], [50, 50]];
    assert!(rectangle_edges_visible(&rectangle_capture(None), corners));
    for edge in 0..4 {
        assert!(!rectangle_edges_visible(
            &rectangle_capture(Some(edge)),
            corners
        ));
    }
}

#[test]
fn rejects_blank_noise_and_invalid_capture_bounds() {
    let mut capture = rectangle_capture(Some(0));
    capture.rgba.fill(40);
    capture.rgba[(5 * 64 + 5) * 4] = 255;
    assert!(!rectangle_edges_visible(&capture, [[10, 10], [50, 50]]));
    assert!(!rectangle_edges_visible(&capture, [[50, 50], [10, 10]]));
    assert!(!rectangle_edges_visible(&capture, [[10, 10], [80, 50]]));
}

#[test]
fn rejects_a_gap_in_an_otherwise_visible_edge() {
    let mut capture = rectangle_capture(None);
    for x in 24..=36 {
        let index = (10 * 64 + x) * 4;
        capture.rgba[index..index + 4].fill(40);
    }
    assert!(!rectangle_edges_visible(&capture, [[10, 10], [50, 50]]));
}

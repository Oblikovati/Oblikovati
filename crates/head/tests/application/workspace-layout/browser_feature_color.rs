// SPDX-License-Identifier: GPL-2.0-only
//! Count highlighted geometry without counting obscuring UI pixels (#4110).
use oblikovati_viewport::ViewportCapture;

const COLOR_REACH: i32 = 40;

pub(super) fn count(capture: &ViewportCapture, wanted: [u8; 3], windows: &[[f32; 4]]) -> usize {
    let width = capture.extent[0] as usize;
    if width == 0 {
        return 0;
    }
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .enumerate()
        .filter(|(index, pixel)| {
            let point = [(*index % width) as f32, (*index / width) as f32];
            outside_windows(point, windows)
                && (0..3).all(|channel| {
                    (pixel[channel] as i32 - wanted[channel] as i32).abs() < COLOR_REACH
                })
        })
        .count()
}

fn outside_windows([x, y]: [f32; 2], windows: &[[f32; 4]]) -> bool {
    !windows
        .iter()
        .any(|[left, top, right, bottom]| x >= *left && x < *right && y >= *top && y < *bottom)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn count_excludes_ui_but_keeps_the_visible_model_edge() {
        let capture = blue_points_capture();
        let blue = [64, 153, 255];
        assert_eq!(count(&capture, blue, &[]), 2);
        assert_eq!(count(&capture, blue, &[[0., 0., 8., 8.]]), 1);
        assert_eq!(count(&capture, blue, &[[0., 0., 40., 40.]]), 0);
    }

    fn blue_points_capture() -> ViewportCapture {
        let mut rgba = vec![0; 40 * 40 * 4];
        let blue = [64, 153, 255, 255];
        for [x, y] in [[2, 2], [20, 20]] {
            let offset = (y * 40 + x) * 4;
            rgba[offset..offset + 4].copy_from_slice(&blue);
        }
        ViewportCapture {
            view_id: 0,
            scene_generation: 0,
            camera_generation: 0,
            surface_generation: 0,
            overlay_generation: 0,
            extent: [40, 40],
            rgba,
        }
    }
}

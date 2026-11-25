// SPDX-License-Identifier: GPL-2.0-only
//! Visibility of the neutral fixture or its blue selection tint outside GUI windows.
use oblikovati_viewport::ViewportCapture;
const PATCH_SIDE: usize = 5;
const HUE_MARGIN: u8 = 20;
pub(super) fn visible_fixture_bounds(
    capture: &ViewportCapture,
    windows: &[[f32; 4]],
) -> Option<([usize; 2], [usize; 2])> {
    let width = capture.extent[0] as usize;
    let (mut low, mut high) = ([usize::MAX; 2], [0; 2]);
    for (index, pixel) in capture.rgba.as_chunks::<4>().0.iter().enumerate() {
        if neutral_fixture_hue(pixel) {
            let point = [index % width, index / width];
            if windows.iter().any(|window| point_in_window(point, *window)) {
                continue;
            }
            for axis in 0..2 {
                low[axis] = low[axis].min(point[axis]);
                high[axis] = high[axis].max(point[axis]);
            }
        }
    }
    (low[0] != usize::MAX).then_some((low, high))
}
pub(super) fn fixture_pixel_count(capture: &ViewportCapture, windows: &[[f32; 4]]) -> usize {
    let width = capture.extent[0] as usize;
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .enumerate()
        .filter(|(index, pixel)| {
            neutral_fixture_hue(*pixel)
                && !windows
                    .iter()
                    .any(|window| point_in_window([index % width, index / width], *window))
        })
        .count()
}
fn point_in_window(point: [usize; 2], window: [f32; 4]) -> bool {
    let [x, y] = point.map(|coordinate| coordinate as f32);
    x >= window[0] && x < window[2] && y >= window[1] && y < window[3]
}
pub(super) fn visible_solid_fill(capture: &ViewportCapture, windows: &[[f32; 4]]) -> bool {
    visible_fill(capture, windows, &fixture_hue)
}
pub(super) fn visible_selected_fill(capture: &ViewportCapture, windows: &[[f32; 4]]) -> bool {
    visible_fill(capture, windows, &blue_selection_hue)
}
pub(super) fn visible_selected_face_fill(capture: &ViewportCapture, windows: &[[f32; 4]]) -> bool {
    visible_fill(capture, windows, &|pixel| {
        pixel[1] > pixel[0].saturating_add(HUE_MARGIN)
            && pixel[2] > pixel[0].saturating_add(HUE_MARGIN)
    })
}
pub(super) fn visible_preview_fill(capture: &ViewportCapture, windows: &[[f32; 4]]) -> bool {
    visible_fill(capture, windows, &|pixel| {
        pixel[2] > pixel[0].saturating_add(HUE_MARGIN)
            && pixel[1] >= pixel[2]
            && pixel[1] - pixel[2] < 2 * HUE_MARGIN
    })
}
pub(super) fn visible_fixture_point(
    capture: &ViewportCapture,
    windows: &[[f32; 4]],
) -> Option<[i32; 2]> {
    let [width, height] = capture.extent.map(|side| side as usize);
    if width < PATCH_SIDE || height < PATCH_SIDE {
        return None;
    }
    for y in 0..=height - PATCH_SIDE {
        for x in 0..=width - PATCH_SIDE {
            if !windows.iter().any(|window| patch_overlaps(x, y, *window))
                && filled_patch(capture, x, y, &neutral_fixture_hue)
            {
                return Some([(x + PATCH_SIDE / 2) as i32, (y + PATCH_SIDE / 2) as i32]);
            }
        }
    }
    None
}
pub(super) fn visible_appearance_fill(
    capture: &ViewportCapture,
    windows: &[[f32; 4]],
    wanted: [u8; 3],
) -> bool {
    visible_fill(capture, windows, &|pixel| {
        (0..3).all(|channel| pixel[channel].abs_diff(wanted[channel]) <= 10)
    })
}
fn visible_fill(
    capture: &ViewportCapture,
    windows: &[[f32; 4]],
    shade: &impl Fn(&[u8]) -> bool,
) -> bool {
    let [width, height] = capture.extent.map(|side| side as usize);
    if width < PATCH_SIDE || height < PATCH_SIDE {
        return false;
    }
    (0..height - PATCH_SIDE + 1).any(|y| {
        (0..width - PATCH_SIDE + 1).any(|x| {
            !windows.iter().any(|window| patch_overlaps(x, y, *window))
                && filled_patch(capture, x, y, shade)
        })
    })
}
fn patch_overlaps(x: usize, y: usize, window: [f32; 4]) -> bool {
    (x as f32) < window[2]
        && ((x + PATCH_SIDE) as f32) > window[0]
        && (y as f32) < window[3]
        && ((y + PATCH_SIDE) as f32) > window[1]
}
fn filled_patch(
    capture: &ViewportCapture,
    x: usize,
    y: usize,
    shade: &impl Fn(&[u8]) -> bool,
) -> bool {
    let width = capture.extent[0] as usize;
    (y..y + PATCH_SIDE).all(|row| {
        (x..x + PATCH_SIDE).all(|column| {
            let offset = (row * width + column) * 4;
            let pixel = &capture.rgba[offset..offset + 4];
            shade(pixel)
        })
    })
}
pub(super) fn neutral_fixture_hue(pixel: &[u8]) -> bool {
    pixel[0] > 80
        && pixel[0] < 250
        && pixel[0] <= pixel[1]
        && pixel[1] <= pixel[2]
        && pixel[2] - pixel[0] >= 2
        && pixel[2] - pixel[0] < HUE_MARGIN
}
fn fixture_hue(pixel: &[u8]) -> bool {
    neutral_fixture_hue(pixel) || blue_selection_hue(pixel)
}
fn blue_selection_hue(pixel: &[u8]) -> bool {
    pixel[2] > pixel[0].saturating_add(HUE_MARGIN) && pixel[2] > pixel[1].saturating_add(HUE_MARGIN)
}
#[cfg(test)]
#[path = "solid_visibility_tests.rs"]
mod tests;

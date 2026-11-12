// SPDX-License-Identifier: GPL-2.0-only
//! Visibility of the green fixture or its blue selection tint outside GUI windows.
use oblikovati_viewport::ViewportCapture;
const PATCH_SIDE: usize = 5;
const HUE_MARGIN: u8 = 20;
pub(super) fn visible_solid_fill(capture: &ViewportCapture, windows: &[[f32; 4]]) -> bool {
    visible_fill(capture, windows, &fixture_hue)
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
                && filled_patch(capture, x, y, &green_fixture_hue)
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
fn green_fixture_hue(pixel: &[u8]) -> bool {
    pixel[1] > pixel[0].saturating_add(HUE_MARGIN) && pixel[1] > pixel[2].saturating_add(HUE_MARGIN)
}
fn fixture_hue(pixel: &[u8]) -> bool {
    let green = green_fixture_hue(pixel);
    let blue = pixel[2] > pixel[0].saturating_add(HUE_MARGIN)
        && pixel[2] > pixel[1].saturating_add(HUE_MARGIN);
    green || blue
}
#[cfg(test)]
#[path = "solid_visibility_tests.rs"]
mod tests;

// SPDX-License-Identifier: GPL-2.0-only
//! Check a rectangle's four expected edges in the viewport capture.
use super::ApplicationWorld;
use oblikovati_viewport::ViewportCapture;

const EDGE_SEARCH: i32 = 2;
const BACKGROUND_OFFSET: i32 = 6;
const MIN_CHANNEL_DIFFERENCE: u8 = 24;

pub(super) fn current_rectangle_visible(world: &mut ApplicationWorld) -> bool {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    rectangle_edges_visible(&capture, super::documents::RECTANGLE)
}

pub(super) fn rectangle_edges_visible(capture: &ViewportCapture, corners: [[i32; 2]; 2]) -> bool {
    let [[left, top], [right, bottom]] = corners;
    if left >= right || top >= bottom {
        return false;
    }
    [
        ([left, top], [right, top], [0, 1]),
        ([left, bottom], [right, bottom], [0, 1]),
        ([left, top], [left, bottom], [1, 0]),
        ([right, top], [right, bottom], [1, 0]),
    ]
    .into_iter()
    .all(|(start, end, normal)| edge_visible(capture, start, end, normal))
}

fn edge_visible(
    capture: &ViewportCapture,
    start: [i32; 2],
    end: [i32; 2],
    normal: [i32; 2],
) -> bool {
    (1..16).all(|fraction| {
        let point =
            std::array::from_fn(|axis| start[axis] + (end[axis] - start[axis]) * fraction / 16);
        sample_contrast(capture, point, normal)
    })
}

fn sample_contrast(capture: &ViewportCapture, point: [i32; 2], normal: [i32; 2]) -> bool {
    let background = offset_point(point, normal, BACKGROUND_OFFSET);
    let Some(background) = pixel(capture, background) else {
        return false;
    };
    (-EDGE_SEARCH..=EDGE_SEARCH).any(|offset| {
        pixel(capture, offset_point(point, normal, offset)).is_some_and(|sample| {
            (0..3).any(|channel| {
                sample[channel].abs_diff(background[channel]) >= MIN_CHANNEL_DIFFERENCE
            })
        })
    })
}

fn offset_point(point: [i32; 2], normal: [i32; 2], offset: i32) -> [i32; 2] {
    [point[0] + normal[0] * offset, point[1] + normal[1] * offset]
}

fn pixel(capture: &ViewportCapture, [x, y]: [i32; 2]) -> Option<&[u8]> {
    let [width, height] = capture.extent;
    let (x, y) = (u32::try_from(x).ok()?, u32::try_from(y).ok()?);
    if x >= width || y >= height {
        return None;
    }
    let index = ((y as usize * width as usize) + x as usize) * 4;
    capture.rgba.get(index..index + 4)
}

#[cfg(test)]
#[path = "rectangle_visibility_tests.rs"]
mod tests;

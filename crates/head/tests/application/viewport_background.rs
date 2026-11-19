// SPDX-License-Identifier: GPL-2.0-only
//! Verify empty viewport space without treating GUI window overlays as scene pixels.
use oblikovati_viewport::ViewportCapture;

pub(super) fn gradient_region_visible(
    capture: &ViewportCapture,
    x: [usize; 2],
    y: [usize; 2],
    controls: &[[f32; 4]],
) -> bool {
    let width = capture.extent[0] as usize;
    let rows: Option<Vec<_>> = (y[0]..y[1])
        .map(|row| background_row(&capture.rgba, width, x, row, controls))
        .collect();
    let Some(rows) = rows else { return false };
    // A uniform panel covering the whole region must not pass as a viewport gradient.
    rows.first()
        .is_some_and(|first| rows.iter().any(|row| row != first))
}

fn background_row(
    rgba: &[u8],
    width: usize,
    x: [usize; 2],
    y: usize,
    controls: &[[f32; 4]],
) -> Option<[u8; 4]> {
    let mut reference = None;
    for column in x[0]..x[1] {
        let occluded = controls
            .iter()
            .any(|bounds| inside_control(column, y, *bounds));
        if occluded {
            continue;
        }
        let offset = (y * width + column) * 4;
        let pixel: [u8; 4] = rgba.get(offset..offset + 4)?.try_into().ok()?;
        if reference.is_some_and(|expected| expected != pixel) {
            return None;
        }
        reference = Some(pixel);
    }
    reference
}

fn inside_control(x: usize, y: usize, bounds: [f32; 4]) -> bool {
    let [x, y] = [x as f32, y as f32];
    x >= bounds[0] && x < bounds[2] && y >= bounds[1] && y < bounds[3]
}

#[cfg(test)]
#[path = "viewport_background_tests.rs"]
mod tests;

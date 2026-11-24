// SPDX-License-Identifier: GPL-2.0-only
//! Verify empty viewport space without treating GUI window overlays as scene pixels.
use oblikovati_viewport::ViewportCapture;

pub(super) fn expect_empty_part(world: &mut super::ApplicationWorld) {
    let browser = world
        .state()
        .browser
        .expect("expected an empty Part browser");
    assert!(
        browser.bodies.is_empty() && browser.features.is_empty() && browser.sketches.is_empty(),
        "expected no bodies, features or sketches in the Part"
    );
    expect_empty_viewport(world);
}

pub(super) fn expect_empty_viewport(world: &mut super::ApplicationWorld) {
    let [left, top, right, bottom] = world.viewport_bounds().map(|edge| edge as usize);
    // Dock separators can bleed a few pixels into the viewport.
    let x_inset = (right - left) / 16;
    let y_inset = (bottom - top) / 16;
    world.expect_blank_region(
        [left + x_inset, right - x_inset],
        [top + y_inset, bottom - y_inset],
    );
}

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

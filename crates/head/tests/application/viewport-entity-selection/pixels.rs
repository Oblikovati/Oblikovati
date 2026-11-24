// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, HeadWindow, HeadWork, diagnostics};
use oblikovati_viewport::ViewportCapture;

pub(super) fn expect_highlighted_face(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_selected_face_fill(
        &capture, &windows
    ));
    let boundary = world.app().owner.selected_browser_outline();
    assert!(!boundary.is_empty(), "expected the selected face boundary");
    assert!(
        boundary.iter().any(|[first, last]| {
            let middle = std::array::from_fn(|axis| (first[axis] + last[axis]) * 0.5);
            let point = world.app().window.model_pixel(middle).unwrap();
            near_green_pixel(&capture, &windows, point)
        }),
        "expected a bright green pixel on the selected face boundary"
    );
}

pub(super) fn expect_neutral_face(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_fixture_bounds(&capture, &windows).is_some());
    assert!(!super::super::solid_visibility::visible_selected_face_fill(
        &capture, &windows
    ));
}

pub(in super::super) fn empty_viewport_point(world: &mut ApplicationWorld) -> [i32; 2] {
    let [left, top, right, bottom] = world.viewport_bounds().map(|side| side as i32);
    let windows = diagnostics::window_occlusions();
    let border = 24;
    let corners = [
        [left + border, top + border],
        [right - border, top + border],
        [left + border, bottom - border],
        [right - border, bottom - border],
    ];
    corners
        .into_iter()
        .find(|point| {
            if covered(&windows, *point) {
                return false;
            }
            let line = world.app().window.view_line(*point).unwrap();
            world
                .app()
                .owner
                .face_edges_on_line(line)
                .unwrap()
                .is_empty()
        })
        .expect("expected a clear viewport point with no solid along its view ray")
}

fn near_green_pixel(capture: &ViewportCapture, windows: &[[f32; 4]], [x, y]: [i32; 2]) -> bool {
    let [width, height] = capture.extent.map(|side| side as i32);
    (y - 3..=y + 3)
        .filter(|row| (0..height).contains(row))
        .any(|row| {
            (x - 3..=x + 3)
                .filter(|column| (0..width).contains(column))
                .any(|column| {
                    if covered(windows, [column, row]) {
                        return false;
                    }
                    let offset = ((row * width + column) * 4) as usize;
                    let [red, green, blue] = capture.rgba[offset..offset + 3].try_into().unwrap();
                    green > 120
                        && green > u8::saturating_add(red, 32)
                        && green > u8::saturating_add(blue, 20)
                })
        })
}

fn covered(windows: &[[f32; 4]], [x, y]: [i32; 2]) -> bool {
    windows.iter().any(|&[left, top, right, bottom]| {
        (left..right).contains(&(x as f32)) && (top..bottom).contains(&(y as f32))
    })
}

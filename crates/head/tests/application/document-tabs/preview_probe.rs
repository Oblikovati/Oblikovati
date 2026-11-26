// SPDX-License-Identifier: GPL-2.0-only
//! Reject blank tab preview picture interiors.

pub(in super::super) fn picture_has_detail(
    capture: &oblikovati_viewport::ViewportCapture,
    position: [f32; 2],
    size: [f32; 2],
    image_top: f32,
) -> bool {
    let Some([left, top, right, bottom]) = picture_probe_bounds(capture, position, size, image_top)
    else {
        return false;
    };
    let pixel = |x: usize, y: usize| {
        let offset = (y * capture.extent[0] as usize + x) * 4;
        capture.rgba.get(offset..offset + 3)
    };
    let Some(ground) = pixel(left, top) else {
        return false;
    };
    (left..right).step_by(3).any(|x| {
        (top..bottom)
            .step_by(3)
            .any(|y| pixel(x, y).is_some_and(|color| color != ground))
    })
}

fn picture_probe_bounds(
    capture: &oblikovati_viewport::ViewportCapture,
    position: [f32; 2],
    size: [f32; 2],
    image_top: f32,
) -> Option<[usize; 4]> {
    let left = position[0] as usize + 8;
    let top = image_top.max(position[1]) as usize;
    let right = ((position[0] + size[0]) as usize).saturating_sub(8);
    let bottom = ((position[1] + size[1]) as usize).saturating_sub(8);
    if left >= right
        || top >= bottom
        || right > capture.extent[0] as usize
        || bottom > capture.extent[1] as usize
    {
        return None;
    }
    Some([left, top, right, bottom])
}

#[test]
fn preview_detail_check_rejects_a_blank_interior() {
    let mut capture = oblikovati_viewport::ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [64, 64],
        rgba: vec![40; 64 * 64 * 4],
    };
    let position = [0., 0.];
    let size = [64., 64.];
    capture.rgba[0] = 255;
    assert!(!picture_has_detail(&capture, position, size, 20.));
    capture.rgba[(23 * 64 + 11) * 4] = 255;
    assert!(picture_has_detail(&capture, position, size, 20.));
    assert!(!picture_has_detail(&capture, position, [80., 64.], 20.));
    capture.rgba.clear();
    assert!(!picture_has_detail(&capture, position, size, 20.));
}

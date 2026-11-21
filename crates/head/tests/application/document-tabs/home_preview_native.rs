// SPDX-License-Identifier: GPL-2.0-only
//! Check Home preview capture through real native GUI frames.
use super::*;

#[test]
fn home_tab_capture_survives_hover_during_activation() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let home = world.expect_tab("Home").center().map(|value| value as i32);
    world.click(home);
    world.pump(Vec::new());
    world.save_capture("home-tab-activated");
    world.click_tab(part);
    world.pump(Vec::new());
    world.hover(home);
    world.pump(Vec::new());
    let (position, size) = window_geometry("##tab-preview").expect("expected Home preview");
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    world.save_capture("home-tab-clean-preview");
    assert!(
        preview_has_pixels(&image, position, size),
        "expected a Home workspace image"
    );
    world.expect_active(part);
    world.hover([900, 500]);
    assert!(window_geometry("##tab-preview").is_none());
    world.close();
}

#[test]
fn preview_pixel_oracle_rejects_blank_and_detects_upper_picture() {
    let mut capture = oblikovati_viewport::ViewportCapture {
        view_id: 0,
        scene_generation: 0,
        camera_generation: 0,
        surface_generation: 0,
        overlay_generation: 0,
        extent: [100, 100],
        rgba: vec![0; 100 * 100 * 4],
    };
    assert!(!preview_has_pixels(&capture, [0., 0.], [100., 100.]));
    capture.rgba[(38 * 100 + 11) * 4] = 255;
    assert!(preview_has_pixels(&capture, [0., 0.], [100., 100.]));
}

fn preview_has_pixels(
    capture: &oblikovati_viewport::ViewportCapture,
    position: [f32; 2],
    size: [f32; 2],
) -> bool {
    let left = position[0] as usize + 8;
    let top = (position[1] + size[1] * 0.35) as usize;
    let right = ((position[0] + size[0]) as usize).saturating_sub(8);
    let bottom = ((position[1] + size[1]) as usize).saturating_sub(8);
    if left >= right
        || top >= bottom
        || right > capture.extent[0] as usize
        || bottom > capture.extent[1] as usize
    {
        return false;
    }
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

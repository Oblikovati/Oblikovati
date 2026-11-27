// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Hole key tip and Cancel through the X11 input queue (#4093).
use super::*;

#[test]
fn x11_hole_key_tip_opens_and_cancel_closes_the_properties_panel() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let before = world.state().browser.unwrap().features.len();
    for key in [0xffe9, u32::from(b'm'), u32::from(b'h'), u32::from(b'o')] {
        super::super::x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    assert!(super::super::diagnostics::window_geometry("Hole").is_some());
    assert_hole_properties_fit_at_minimum_size(&mut world);
    let point = world.run_in_window("Hole", "Cancel").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("Hole").is_none());
    assert_eq!(world.state().browser.unwrap().features.len(), before);
    world.close();
}

fn assert_hole_properties_fit_at_minimum_size(world: &mut ApplicationWorld) {
    use super::super::{diagnostics, x11_input};
    x11_input::resize_application_window([640, 480]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [640, 480]);
    let viewport = world.viewport_bounds().map(|edge| edge as f32);
    let bounds = diagnostics::window_pixel_bounds("Hole").expect("visible Hole properties");
    assert!(
        bounds[0] >= viewport[0]
            && bounds[1] >= viewport[1]
            && bounds[2] <= viewport[2]
            && bounds[3] <= viewport[3],
        "properties {bounds:?}, viewport {viewport:?}"
    );
    world.save_capture("hole-at-minimum-size");
    let depth = world.run_in_window("Hole", "Depth");
    let through_all = world.run_in_window("Hole", "Throughall");
    assert!(depth.maximum[0] < through_all.minimum[0]);
    assert!(
        through_all.maximum[0] <= bounds[2] - 4.,
        "Depth hint extends past the panel"
    );
}

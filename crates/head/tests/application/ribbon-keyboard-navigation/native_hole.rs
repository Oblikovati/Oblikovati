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
    let point = world.run_in_window("Hole", "Cancel").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("Hole").is_none());
    assert_eq!(world.state().browser.unwrap().features.len(), before);
    world.close();
}

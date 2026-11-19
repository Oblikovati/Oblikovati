// SPDX-License-Identifier: GPL-2.0-only
//! Exercise document switching with native X11 keyboard input.
use super::*;

#[test]
fn native_control_tab_switches_both_directions() {
    let mut world = ApplicationWorld::new();
    adjacent_tabs(&mut world);
    left_hand_active(&mut world);
    super::super::x11_input::press_control_tab(false);
    world.pump(Vec::new());
    right_hand_active(&mut world);
    browser_and_viewport(&mut world);
    left_hand_open(&mut world);
    super::super::x11_input::press_control_tab(true);
    world.pump(Vec::new());
    left_active(&mut world);
    second_tab_remains(&mut world);
    assert_native_wrap(&mut world);
    world.close();
}

fn assert_native_wrap(world: &mut ApplicationWorld) {
    super::super::x11_input::press_control_tab(true);
    world.pump(Vec::new());
    assert!(window_geometry("##home-workspace").is_some());
    super::super::x11_input::press_control_tab(true);
    world.pump(Vec::new());
    right_hand_active(world);
    super::super::x11_input::press_control_tab(false);
    world.pump(Vec::new());
    assert!(window_geometry("##home-workspace").is_some());
}

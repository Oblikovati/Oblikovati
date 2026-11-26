// SPDX-License-Identifier: GPL-2.0-only
//! Exercise document switching through the native window's keyboard path.
use super::*;

pub(super) fn try_native_control_tab(world: &mut ApplicationWorld, reverse: bool) -> bool {
    use super::super::x11_input;

    if world.app().window.native.view.uses_wayland() {
        if std::env::var("OBK_BDD_WESTON_NESTED_X11").as_deref() != Ok("1") {
            return false;
        }
        x11_input::window_targets::press_weston_control_tab(reverse);
    } else {
        x11_input::press_control_tab(reverse);
    }
    world.pump(Vec::new());
    true
}

#[test]
fn native_control_tab_switches_both_directions() {
    let mut world = ApplicationWorld::new();
    adjacent_tabs(&mut world);
    left_hand_active(&mut world);
    assert!(try_native_control_tab(&mut world, false));
    right_hand_active(&mut world);
    browser_and_viewport(&mut world);
    left_hand_open(&mut world);
    assert!(try_native_control_tab(&mut world, true));
    left_active(&mut world);
    second_tab_remains(&mut world);
    assert_native_wrap(&mut world);
    world.close();
}

fn assert_native_wrap(world: &mut ApplicationWorld) {
    assert!(try_native_control_tab(world, true));
    assert!(window_geometry("##home-workspace").is_some());
    assert!(try_native_control_tab(world, true));
    right_hand_active(world);
    assert!(try_native_control_tab(world, false));
    assert!(window_geometry("##home-workspace").is_some());
}

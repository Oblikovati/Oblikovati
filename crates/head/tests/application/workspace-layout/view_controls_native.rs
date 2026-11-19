// SPDX-License-Identifier: GPL-2.0-only
//! Exercise independent workspace view controls through native input (#4134).
use super::*;

#[test]
fn view_menu_toggles_cube_navigation_and_status_independently() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    assert_controls(true, true, true);
    toggle(&mut world, "ViewCube");
    assert_controls(false, true, true);
    toggle(&mut world, "Navigation Bar");
    assert_controls(false, false, true);
    let with_status = world.viewport_bounds();
    toggle(&mut world, "Status Bar");
    assert_controls(false, false, false);
    let without_status = world.viewport_bounds();
    assert!(
        without_status[3] > with_status[3],
        "viewport {with_status:?} to {without_status:?}: expected freed status area"
    );
    toggle(&mut world, "ViewCube");
    assert_controls(true, false, false);
    toggle(&mut world, "Navigation Bar");
    toggle(&mut world, "Status Bar");
    assert_controls(true, true, true);
    world.close();
}

fn toggle(world: &mut ApplicationWorld, label: &str) {
    if !world.shows("ViewCube") {
        super::dockspace_restart::native_click_label(world, "View");
        super::dockspace_restart::native_click_label(world, "User Interface");
    }
    super::dockspace_restart::native_click_label(world, label);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn assert_controls(cube: bool, navigation: bool, status: bool) {
    for (window, visible) in [
        ("##orientation-control", cube),
        ("##navigation-bar", navigation),
        ("##status-bar", status),
    ] {
        assert_eq!(
            window_geometry(window).is_some(),
            visible,
            "{window} visibility"
        );
    }
}

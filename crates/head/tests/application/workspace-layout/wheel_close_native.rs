// SPDX-License-Identifier: GPL-2.0-only
//! Verify that Navigation Wheel Options releases its selected dock tab.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::{native_click_label, select_dock_tab};
use super::orientation_close_native::drag_title;
use super::*;

fn open_wheel(world: &mut ApplicationWorld) {
    let point = world.run_in_window("##navigation-bar", "W").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    super::super::x11_input::click([760, 480]);
    world.pump(Vec::new());
}

fn open_options(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry("##navigation-wheel").unwrap();
    let point = [position[0] + size[0] * 0.5, position[1] + size[1] * 0.5];
    super::super::x11_input::right_click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    native_click_label(world, "Options");
    assert!(window_geometry("Navigation Wheel Options").is_some());
}

fn dock_options(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, "Material Browser");
    drag_title(
        world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    open_wheel(world);
    open_options(world);
    let (position, size) = window_geometry("Material Browser").unwrap();
    drag_title(
        world,
        "Navigation Wheel Options",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Navigation Wheel Options"),
        Some(node)
    );
    node
}

fn close_and_reopen(world: &mut ApplicationWorld, node: u32, close: &str) {
    select_dock_tab(world, "Navigation Wheel Options");
    if close == "OK" {
        let row = world
            .run_in_window("Navigation Wheel Options", "ShowToolMessages")
            .clone();
        let point = [row.minimum[0] - 10., row.center()[1]];
        super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
        world.pump(Vec::new());
    }
    if close == "Escape" {
        super::super::x11_input::press_escape();
    } else {
        let point = world
            .run_in_window("Navigation Wheel Options", close)
            .center();
        super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        window_geometry("Navigation Wheel Options").is_none(),
        "{close}"
    );
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true),
        "{close}"
    );
    world.run_in_window("Material Browser", "Assigntopart");
    open_options(world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Navigation Wheel Options"),
        Some(node),
        "{close}"
    );
}

#[test]
fn wheel_options_content_close_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let node = dock_options(&mut world);
    for close in ["Cancel", "OK", "Escape"] {
        close_and_reopen(&mut world, node, close);
    }
    select_dock_tab(&mut world, "Navigation Wheel Options");
    world.save_capture("wheel-options-restored-to-shared-dock");
    world.close();
}

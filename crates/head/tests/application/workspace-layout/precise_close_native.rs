// SPDX-License-Identifier: GPL-2.0-only
//! Verify Precise Input close transitions in a shared dock.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::{native_click_label, select_dock_tab};
use super::orientation_close_native::drag_title;
use super::*;

fn open_precise(world: &mut ApplicationWorld) {
    let title = world.run("Create");
    let point = [title.maximum[0] - 2., title.center()[1]];
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    let point = world
        .run_in_window("##ribbon-additional-Create", "PreciseInput")
        .center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(window_geometry("Precise Input").is_some());
}

fn dock_precise(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, "Parameters");
    drag_title(
        world,
        "Parameters",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    native_click_label(world, "Line");
    open_precise(world);
    let (position, size) = window_geometry("Parameters").unwrap();
    drag_title(
        world,
        "Precise Input",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Parameters").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Precise Input"),
        Some(node)
    );
    node
}

#[test]
fn precise_menu_close_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let node = dock_precise(&mut world);
    select_dock_tab(&mut world, "Precise Input");
    let point = world.run_in_window("Precise Input", "...").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    native_click_label(&mut world, "Close");
    assert!(window_geometry("Precise Input").is_none());
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Parameters"),
        Some(true)
    );
    world.run_in_window("Parameters", "UserParameters");
    open_precise(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Precise Input"),
        Some(node)
    );
    select_dock_tab(&mut world, "Precise Input");
    world.save_capture("precise-restored-to-shared-dock");
    world.close();
}

#[test]
fn precise_target_loss_selects_next_tab_and_restores_same_dock() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let node = dock_precise(&mut world);
    select_dock_tab(&mut world, "Precise Input");
    super::super::x11_input::move_pointer([500, 500]);
    world.pump(Vec::new());
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Precise Input").is_none());
    assert!(world.state().precise_target.is_none());
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Parameters"),
        Some(true)
    );
    world.run_in_window("Parameters", "UserParameters");
    native_click_label(&mut world, "Line");
    assert!(world.state().precise_target.is_some());
    assert_eq!(
        super::super::diagnostics::window_dock_id("Precise Input"),
        Some(node)
    );
    select_dock_tab(&mut world, "Precise Input");
    world.save_capture("precise-target-restored-to-shared-dock");
    world.close();
}

// SPDX-License-Identifier: GPL-2.0-only
//! Verify Create Block closure in a shared dock while sketch editing remains active.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::select_dock_tab;
use super::orientation_close_native::drag_title;
use super::*;

fn open_block(world: &mut ApplicationWorld) {
    let title = world.run("Create");
    let point = [title.maximum[0] - 2., title.center()[1]];
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    let point = world
        .run_in_window("##ribbon-additional-Create", "CreateBlock")
        .center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(window_geometry("Create Block").is_some());
}

fn dock_block(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, "Parameters");
    drag_title(
        world,
        "Parameters",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    open_block(world);
    let (position, size) = window_geometry("Parameters").unwrap();
    drag_title(
        world,
        "Create Block",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Parameters").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Create Block"),
        Some(node)
    );
    node
}

#[test]
fn create_block_cancel_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let node = dock_block(&mut world);
    select_dock_tab(&mut world, "Create Block");
    let point = world.run_in_window("Create Block", "Cancel").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Create Block").is_none());
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Parameters"),
        Some(true)
    );
    world.run_in_window("Parameters", "UserParameters");
    open_block(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Create Block"),
        Some(node)
    );
    select_dock_tab(&mut world, "Create Block");
    world.save_capture("block-restored-to-shared-dock");
    world.close();
}

#[test]
fn create_block_completed_ok_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("Rectangle");
    for point in [[450, 350], [650, 550]] {
        super::super::x11_input::click(point);
        world.pump(Vec::new());
    }
    world.escape();
    let node = dock_block(&mut world);
    select_dock_tab(&mut world, "Create Block");
    let point = world.line_point(0, 0.5);
    super::super::x11_input::move_pointer(point);
    world.pump(Vec::new());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_selection.entities.len(),
        1,
        "point {point:?}, dialog {:?}, viewport {:?}",
        window_geometry("Create Block"),
        viewport(&mut world)
    );
    let ok = world.run_in_window("Create Block", "OK");
    assert_eq!(ok.alpha, u8::MAX);
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(ok.center()));
    super::external_close_native::assert_closed_with_visible_tab(
        &mut world,
        "Create Block",
        "Parameters",
    );
    assert_eq!(world.state().sketch_block_names, ["Block1"]);
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Parameters"),
        Some(true)
    );
    open_block(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Create Block"),
        Some(node)
    );
    world.save_capture("block-completed-restored-to-shared-dock");
    world.close();
}

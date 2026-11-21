// SPDX-License-Identifier: GPL-2.0-only
//! Move the same tool panel across the workspace docking edges.
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, viewport};
use cucumber::{then, when};

#[when("the user docks the panel to the right workspace edge")]
fn dock_right(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let area = viewport(world);
    let target = [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32];
    native_drag(world, start, target);
}

#[then("the panel is attached along the right edge")]
fn right_edge(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
    assert!((position[0] + size[0] - world.extent()[0] as f32).abs() <= 1.);
}

#[when("the user docks the panel to the top workspace edge")]
fn dock_top(world: &mut ApplicationWorld) {
    let area = world.docking_resize.unwrap().0;
    let target = [(world.extent()[0] / 2) as i32, area[1] as i32 + 15];
    drag_dock_tab(world, target);
}

#[then("the panel is attached below the ribbon")]
fn top_edge(world: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(PANEL).unwrap();
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
    assert!((position[1] - world.docking_resize.unwrap().0[1] as f32).abs() <= 1.);
}

#[then("the viewport occupies the workspace below the panel")]
fn viewport_below(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    assert!(viewport(world)[1] as f32 >= position[1] + size[1] - 1.);
}

#[when("the user docks the panel to the bottom workspace edge")]
fn dock_bottom(world: &mut ApplicationWorld) {
    let status = diagnostics::window_geometry("##status-bar").unwrap().0;
    let target = [(world.extent()[0] / 2) as i32, status[1] as i32 - 15];
    drag_dock_tab(world, target);
}

#[then("the panel is attached above the document tabs and status area")]
fn bottom_edge(_: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let (tabs, _) = diagnostics::window_geometry("##open-document-tabs").unwrap();
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
    assert!(position[1] + size[1] <= tabs[1] + 1.);
}

#[then("the viewport occupies the workspace above the panel")]
fn viewport_above(world: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(PANEL).unwrap();
    assert!(viewport(world)[3] as f32 <= position[1] + 1.);
    assert_eq!(world.state().document, world.previous_document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

fn drag_dock_tab(world: &mut ApplicationWorld, target: [i32; 2]) {
    let (min, max) = diagnostics::window_dock_tab_bounds(PANEL).unwrap();
    let start = [
        ((min[0] + max[0]) * 0.5) as i32,
        ((min[1] + max[1]) * 0.5) as i32,
    ];
    native_drag(world, start, target);
}

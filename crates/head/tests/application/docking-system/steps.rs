// SPDX-License-Identifier: GPL-2.0-only
//! Native shared-dockspace scenarios from the application benchmark (#4069).
use super::{ApplicationWorld, diagnostics, quick_access_icons, workspace_layout};
use cucumber::{given, then, when};
use workspace_layout::dockspace_native::{native_drag, resize_boundary, viewport};

#[path = "edges_scenario.rs"]
mod edges_scenario;
#[path = "float_scenario.rs"]
mod float_scenario;
#[path = "preview_scenario.rs"]
mod preview_scenario;
#[path = "restart_scenario.rs"]
mod restart_scenario;
#[path = "retention_scenario.rs"]
mod retention_scenario;
#[path = "tabs_scenario.rs"]
mod tabs_scenario;

const PANEL: &str = "Material Browser";

#[given("a dockable panel is docked beside the viewport")]
fn panel_docked(world: &mut ApplicationWorld) {
    let before = viewport(world);
    let icon = quick_access_icons::standard_icon_center(PANEL).expect("material browser icon");
    world.click(diagnostics::pointer_pixels(icon));
    let (position, size) = diagnostics::window_geometry(PANEL).expect("floating panel");
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        world,
        start,
        [before[2] as i32 - 15, ((before[1] + before[3]) / 2) as i32],
    );
    assert_ne!(diagnostics::window_dock_id(PANEL), Some(0));
    let area = viewport(world);
    let (_, size) = diagnostics::window_geometry(PANEL).expect("docked panel");
    world.docking_resize = Some((area, size[0]));
}

#[when("the user drags the panel boundary away from the viewport")]
fn grow_panel(world: &mut ApplicationWorld) {
    let (before, _) = world.docking_resize.unwrap();
    resize_boundary(world, before, -80);
}

#[then("the panel becomes wider")]
fn panel_grew(world: &mut ApplicationWorld) {
    let (_, original) = world.docking_resize.unwrap();
    let (_, size) = diagnostics::window_geometry(PANEL).unwrap();
    assert!(
        size[0] > original + 40.,
        "panel width {original} -> {}",
        size[0]
    );
}

#[then("the viewport becomes narrower by the corresponding amount")]
fn viewport_shrank(world: &mut ApplicationWorld) {
    let (before, width) = world.docking_resize.unwrap();
    let after = viewport(world);
    let (_, size) = diagnostics::window_geometry(PANEL).unwrap();
    let viewport_loss = (before[2] - after[2]) as f32;
    assert!(
        (viewport_loss - (size[0] - width)).abs() <= 2.,
        "viewport loss {viewport_loss}, panel growth {}",
        size[0] - width
    );
}

#[then("the panel contents and active part remain visible")]
fn content_remains(world: &mut ApplicationWorld) {
    world.run_in_window(PANEL, "Assigntopart");
    assert_eq!(world.state().document, world.previous_document);
    let area = viewport(world);
    assert!(area[0] < area[2] && area[1] < area[3]);
    world.expect_text("End of Part");
}

#[when("the user drags the boundary back")]
fn restore_boundary(world: &mut ApplicationWorld) {
    let current = viewport(world);
    let (before, _) = world.docking_resize.unwrap();
    resize_boundary(world, current, before[2] as i32 - current[2] as i32);
}

#[then("the panel and viewport return to their previous proportions")]
fn proportions_restored(world: &mut ApplicationWorld) {
    let (before, width) = world.docking_resize.unwrap();
    let current = viewport(world);
    let (_, size) = diagnostics::window_geometry(PANEL).unwrap();
    assert!(
        before[2].abs_diff(current[2]) <= 1,
        "viewport {before:?} -> {current:?}"
    );
    assert!(
        (width - size[0]).abs() <= 1.,
        "panel width {width} -> {}",
        size[0]
    );
}

#[given("a dockable panel is the active tab in a dock region")]
fn active_panel_in_dock(world: &mut ApplicationWorld) {
    panel_docked(world);
    assert_eq!(diagnostics::window_dock_tab_visible(PANEL), Some(true));
}

#[when("the user closes the panel")]
fn close_docked_panel(world: &mut ApplicationWorld) {
    let (min, max) = diagnostics::window_dock_tab_bounds(PANEL).expect("panel tab");
    let close = [max[0] - (max[1] - min[1]) * 0.5, (min[1] + max[1]) * 0.5];
    world.click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the panel is removed from the visible workspace")]
fn closed_panel_hidden(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(PANEL).is_none());
}

#[then("the remaining dock regions and viewport use the freed space")]
fn viewport_uses_freed_space(world: &mut ApplicationWorld) {
    let (docked, _) = world.docking_resize.expect("docked viewport");
    let current = viewport(world);
    assert!(current[2] > docked[2], "viewport {docked:?} -> {current:?}");
    world.expect_text("End of Part");
}

#[when("the user opens the interface visibility menu")]
fn open_visibility_menu(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}

#[when("the user selects the closed panel")]
fn select_closed_panel(world: &mut ApplicationWorld) {
    world.click_text(PANEL);
    world.pump(Vec::new());
}

#[then("the panel is restored to the workspace")]
fn restored_panel_docked(_: &mut ApplicationWorld) {
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
}

#[then("its contents are visible")]
fn restored_panel_contents(world: &mut ApplicationWorld) {
    world.run_in_window(PANEL, "Assigntopart");
}

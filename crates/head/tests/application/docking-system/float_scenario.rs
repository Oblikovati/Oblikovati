// SPDX-License-Identifier: GPL-2.0-only
//! Exercise floating and redocking through native pointer input.
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, viewport};
use cucumber::{then, when};

#[when("the user drags its title bar away from the dock targets")]
fn float_docked_panel(world: &mut ApplicationWorld) {
    let (minimum, maximum) = diagnostics::window_dock_tab_bounds(PANEL).unwrap();
    let start = [
        ((minimum[0] + maximum[0]) * 0.5) as i32,
        ((minimum[1] + maximum[1]) * 0.5) as i32,
    ];
    let area = viewport(world);
    let target = [
        ((area[0] + area[2]) / 2) as i32,
        ((area[1] + area[3]) / 2) as i32,
    ];
    native_drag(world, start, target);
}

#[then("the panel floats above the workspace")]
fn panel_is_floating(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_id(PANEL), Some(0));
    world.run_in_window(PANEL, "Assigntopart");
}

#[then("the viewport expands into the space released by the panel")]
fn viewport_expands(world: &mut ApplicationWorld) {
    let (docked, _) = world.docking_resize.expect("docked viewport");
    let floating = viewport(world);
    assert!(
        floating[2] > docked[2],
        "viewport {docked:?} -> {floating:?}"
    );
}

#[when("the user drags the floating panel to a workspace dock target")]
fn redock_floating_panel(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let area = viewport(world);
    let target = [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32];
    native_drag(world, start, target);
}

#[then("the panel docks at that target")]
fn panel_is_redocked(_: &mut ApplicationWorld) {
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
}

#[then("the viewport adjusts to the resulting workspace layout")]
fn viewport_adjusts(world: &mut ApplicationWorld) {
    let (before, _) = world.docking_resize.expect("original docked viewport");
    let after = viewport(world);
    assert_eq!(after, before, "viewport {before:?} -> {after:?}");
    world.run_in_window(PANEL, "Assigntopart");
    world.expect_text("End of Part");
}

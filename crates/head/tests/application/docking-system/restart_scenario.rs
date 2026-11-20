// SPDX-License-Identifier: GPL-2.0-only
//! Verify a saved dock group through an application restart.
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, panel_docked, viewport};
use cucumber::{given, then, when};
const APPEARANCE: &str = "Appearance Browser";

#[given("the user has changed the workspace panel arrangement")]
fn arrange_tabbed_panels(world: &mut ApplicationWorld) {
    super::open_docking_fixture(world);
    panel_docked(world);
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, APPEARANCE);
    let (source, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let (target, bounds) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [(source[0] + size[0] * 0.4) as i32, (source[1] + 10.) as i32];
    let end = [
        (target[0] + bounds[0] * 0.5) as i32,
        (target[1] + 10.) as i32,
    ];
    native_drag(world, start, end);
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(APPEARANCE), Some(node));
    world.docking_active_tab = diagnostics::window_dock_tab_visible(APPEARANCE);
}

#[given("the MCAD Software has saved that arrangement")]
fn workspace_is_saveable(world: &mut ApplicationWorld) {
    let bytes = world
        .app()
        .window
        .native
        .gui
        .as_mut()
        .unwrap()
        .save_workspace_layout()
        .unwrap();
    let store =
        crate::native::preferences::NativeWorkspaceLayout::new(world.preferences_directory());
    store.save(&bytes).unwrap();
    assert_eq!(store.load().unwrap(), Some(bytes));
}

#[when("the user closes and restarts the MCAD Software")]
fn restart_application(world: &mut ApplicationWorld) {
    world.reopen_application();
    super::open_docking_fixture(world);
    world.pump(Vec::new());
}

#[then("the saved panel positions, sizes and tab groups are restored")]
fn restored_group(world: &mut ApplicationWorld) {
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(APPEARANCE), Some(node));
    assert_eq!(
        diagnostics::window_dock_tab_visible(APPEARANCE),
        world.docking_active_tab
    );
    let (area, width) = world.docking_resize.unwrap();
    assert_eq!(viewport(world), area);
    assert_eq!(diagnostics::window_geometry(PANEL).unwrap().1[0], width);
}

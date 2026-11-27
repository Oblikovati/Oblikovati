// SPDX-License-Identifier: GPL-2.0-only
//! Verify the panel tab insertion target and the resulting selectable tabs.
use super::super::workspace_layout::dockspace_native::{move_drag, press_drag};
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, panel_docked, viewport};
use cucumber::{given, then, when};
const SOURCE: &str = "Appearance Browser";

#[given("two dockable panels are open in separate workspace regions")]
fn separate_panels(world: &mut ApplicationWorld) {
    super::open_docking_fixture(world);
    panel_docked(world);
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, SOURCE);
    let (position, size) = diagnostics::window_geometry(SOURCE).unwrap();
    let from = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let area = viewport(world);
    native_drag(world, from, [15, ((area[1] + area[3]) / 2) as i32]);
    let source = diagnostics::window_dock_id(SOURCE).unwrap();
    let target = diagnostics::window_dock_id(PANEL).unwrap();
    assert!(source != 0 && target != 0 && source != target);
}

#[when("the user drags one panel onto the center of the other panel's dock region")]
fn drag_to_panel(world: &mut ApplicationWorld) {
    let (min, max) = diagnostics::window_dock_tab_bounds(SOURCE).unwrap();
    let from =
        diagnostics::pointer_pixels(std::array::from_fn(|axis| (min[axis] + max[axis]) * 0.5));
    let area = viewport(world);
    let middle = [
        ((area[0] + area[2]) / 2) as i32,
        ((area[1] + area[3]) / 2) as i32,
    ];
    press_drag(world, from);
    move_drag(world, from, middle);

    // Detaching the source can change the destination width before the drop.
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let end = diagnostics::pointer_pixels(std::array::from_fn(|axis| {
        position[axis] + size[axis] * 0.5
    }));
    move_drag(world, middle, end);
    world.last_drag = Some([from, end]);
}

#[then("the destination region indicates a tab insertion target")]
fn tab_insertion_preview(world: &mut ApplicationWorld) {
    world.save_capture("tab-insertion-preview");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let (position, _) = diagnostics::window_geometry(PANEL).unwrap();
    let index =
        ((position[1] as usize + 50) * capture.extent[0] as usize + position[0] as usize + 50) * 4;
    let color = &capture.rgba[index..index + 3];
    assert!(color[2] > color[0] + 20, "tab preview color {color:?}");
}

#[then("both panels occupy the same dock region as tabs")]
fn panels_share_node(_: &mut ApplicationWorld) {
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(SOURCE), Some(node));
}

#[then("one panel is active while the other remains available")]
fn one_tab_active(_: &mut ApplicationWorld) {
    let first = diagnostics::window_dock_tab_visible(PANEL).unwrap();
    let second = diagnostics::window_dock_tab_visible(SOURCE).unwrap();
    assert_ne!(first, second);
}

#[when("the user selects the other panel tab")]
fn select_other_tab(world: &mut ApplicationWorld) {
    let other = if diagnostics::window_dock_tab_visible(PANEL) == Some(true) {
        SOURCE
    } else {
        PANEL
    };
    let (min, max) = diagnostics::window_dock_tab_bounds(other).unwrap();
    world.click(diagnostics::pointer_pixels([
        (min[0] + max[0]) * 0.5,
        (min[1] + max[1]) * 0.5,
    ]));
    world.pump(Vec::new());
    world.docking_selected_tab = Some(other.to_owned());
}

#[then("its contents replace the active panel contents in that region")]
fn other_tab_contents(world: &mut ApplicationWorld) {
    let chosen = world.docking_selected_tab.clone().unwrap();
    assert_eq!(diagnostics::window_dock_tab_visible(&chosen), Some(true));
    world.run_in_window(&chosen, "Assigntopart");
}

#[then("the viewport and active document remain available")]
fn viewport_and_document_remain(world: &mut ApplicationWorld) {
    let area = viewport(world);
    assert!(area[0] < area[2] && area[1] < area[3]);
    assert_eq!(world.state().document, world.previous_document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

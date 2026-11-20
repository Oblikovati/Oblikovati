// SPDX-License-Identifier: GPL-2.0-only
//! Verify the panel tab insertion target and the resulting selectable tabs.
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
    let from = [
        ((min[0] + max[0]) * 0.5) as i32,
        ((min[1] + max[1]) * 0.5) as i32,
    ];
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let end = [
        (position[0] + size[0] * 0.5) as i32,
        (position[1] + size[1] * 0.5) as i32,
    ];
    super::super::x11_input::move_pointer(from);
    world.pump(Vec::new());
    super::super::x11_input::left_button(from, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        let at = std::array::from_fn(|axis| from[axis] + (end[axis] - from[axis]) * step / 8);
        super::super::x11_input::move_pointer(at);
        world.pump(Vec::new());
    }
    world.last_drag = Some([from, end]);
}

#[then("the destination region indicates a tab insertion target")]
fn tab_insertion_preview(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let (position, _) = diagnostics::window_geometry(PANEL).unwrap();
    let index =
        ((position[1] as usize + 50) * capture.extent[0] as usize + position[0] as usize + 50) * 4;
    let color = &capture.rgba[index..index + 3];
    assert!(color[2] > color[0] + 20, "tab preview color {color:?}");
    world.save_capture("tab-insertion-preview");
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

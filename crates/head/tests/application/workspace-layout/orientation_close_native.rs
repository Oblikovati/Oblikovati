// SPDX-License-Identifier: GPL-2.0-only
//! Verify that orientation options hand a shared dock back to its other tab.
use super::dockspace_native::{native_drag, open_toolbar_tool, viewport};
use super::dockspace_restart::{native_click_label, select_dock_tab};
use super::*;
use oblikovati_gui::GuiOrientationLayout;

fn open_orientation(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry("##orientation-control").unwrap();
    let center = GuiOrientationLayout::for_side(size[0]).center;
    let point = [position[0] + center[0], position[1] + center[1]];
    super::super::x11_input::right_click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    native_click_label(world, "Options");
    assert!(window_geometry("Orientation Options").is_some());
}

fn dock_orientation(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, "Material Browser");
    drag_title(
        world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    open_orientation(world);
    let (position, size) = window_geometry("Material Browser").unwrap();
    drag_title(
        world,
        "Orientation Options",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Orientation Options"),
        Some(node)
    );
    node
}

pub(super) fn drag_title(world: &mut ApplicationWorld, title: &str, target: [i32; 2]) {
    let (position, size) = window_geometry(title).unwrap();
    let source = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(world, source, target);
}

#[test]
fn orientation_content_close_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let node = dock_orientation(&mut world);
    for close in ["Cancel", "OK", "Escape"] {
        close_and_reopen(&mut world, node, close);
    }
    select_dock_tab(&mut world, "Orientation Options");
    world.save_capture("orientation-restored-to-shared-dock");
    world.close();
}

fn close_and_reopen(world: &mut ApplicationWorld, node: u32, close: &str) {
    select_dock_tab(world, "Orientation Options");
    if close == "Escape" {
        super::super::x11_input::press_escape();
    } else {
        let point = world.run_in_window("Orientation Options", close).center();
        super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Orientation Options").is_none(), "{close}");
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true),
        "{close}"
    );
    world.run_in_window("Material Browser", "Assigntopart");
    open_orientation(world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Orientation Options"),
        Some(node),
        "{close}"
    );
}

#[test]
fn orientation_suspends_while_home_is_active_and_restores_the_shared_dock() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let document = world.active_id();
    let name = world.document_name(document);
    let node = dock_orientation(&mut world);
    select_dock_tab(&mut world, "Orientation Options");
    let home = world.expect_tab("Home").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(home));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("##home-workspace").is_some());
    assert!(window_geometry("Orientation Options").is_none());
    let part = world.expect_tab(&name).center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(part));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_eq!(
        super::super::diagnostics::window_dock_id("Orientation Options"),
        Some(node),
        "the suspended window must return to its original dock"
    );
    select_dock_tab(&mut world, "Orientation Options");
    world.run_in_window("Orientation Options", "Cancel");
    world.close();
}

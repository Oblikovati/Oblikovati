// SPDX-License-Identifier: GPL-2.0-only
//! Check an edge target before and after releasing a native pointer drag.
use super::super::workspace_layout::dockspace_native::{move_drag, press_drag, release_drag};
use super::{ApplicationWorld, PANEL, diagnostics, viewport};
use cucumber::{given, then, when};

#[given("a dockable panel is floating over the workspace")]
fn floating_panel(world: &mut ApplicationWorld) {
    super::open_docking_fixture(world);
    let area = viewport(world);
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, PANEL);
    assert_eq!(diagnostics::window_dock_id(PANEL), Some(0));
    world.docking_resize = Some((area, 0.));
}

#[when("the user drags the panel title bar toward the left workspace edge")]
fn drag_to_left_target(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let area = viewport(world);
    let end = [15, ((area[1] + area[3]) / 2) as i32];
    press_drag(world, start);
    move_drag(world, start, end);
    world.last_drag = Some([start, end]);
}

#[then("the left docking target is indicated before the panel is released")]
fn target_preview(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let area = world.docking_resize.unwrap().0;
    let index = ((area[1] + 40) * capture.extent[0] + 60) as usize * 4;
    let color = &capture.rgba[index..index + 3];
    assert!(color[2] > color[0] + 20, "docking preview color {color:?}");
    world.save_capture("left-docking-preview");
}

#[when("the user releases the panel over that target")]
fn release_target(world: &mut ApplicationWorld) {
    let end = world.last_drag.unwrap()[1];
    release_drag(world, end);
}

#[then("the panel docks along the left edge of the workspace")]
fn panel_on_left(_: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(PANEL).unwrap();
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
    assert!(position[0].abs() <= 1., "panel left edge: {position:?}");
}

#[then("the viewport resizes to occupy the remaining workspace")]
fn viewport_fits(world: &mut ApplicationWorld) {
    let before = world.docking_resize.unwrap().0;
    let after = viewport(world);
    assert!(
        after[0] > before[0] && after[0] < after[2],
        "viewport {before:?} -> {after:?}"
    );
    assert_eq!(world.state().document, world.previous_document);
}

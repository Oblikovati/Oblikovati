// SPDX-License-Identifier: GPL-2.0-only
//! Exercise width-compacted ribbon panels with the native window manager.
use super::*;

#[given("a part is active in the MCAD Software with the full 3D Model ribbon displayed")]
fn full_model_part(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
    world.expect_text("Extrude");
    world.expect_text("Hole");
}

#[given("the narrow window shows Modify as a compact panel button")]
fn narrow_modify(world: &mut ApplicationWorld) {
    narrow_window(world);
    world.expect_text("Modify");
    world.expect_no_text("Hole");
}

pub(super) fn narrow_window(world: &mut ApplicationWorld) {
    resize_window(world, 700);
}

pub(super) fn resize_window(world: &mut ApplicationWorld, width: i32) {
    super::super::x11_input::focus_window();
    let bounds = super::super::x11_input::application_bounds();
    let edge = [bounds[2] - 2, bounds[3] / 2];
    let target = [width, edge[1]];
    super::super::main_window_frame_native::drag_resize_handle(world, edge, target);
    world.pump(Vec::new());
    assert!(world.app().window.native.drawable_extent().unwrap()[0] < 800);
}

#[given("the temporary Modify panel is open")]
fn open_modify(world: &mut ApplicationWorld) {
    let label = world.run("Modify").center();
    let button = super::super::diagnostics::pointer_pixels([label[0], label[1] - 35.]);
    super::super::x11_input::click(button);
    world.pump(Vec::new());
    world.expect_text("Hole");
    assert!(world.run("Hole").minimum[1] > band_height());
}

#[when("the user maximizes the application window")]
fn maximize_window(world: &mut ApplicationWorld) {
    super::super::main_window_frame_native::press_maximize(world);
    super::super::main_window_frame_native::pump_until(world, true);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the Modify commands are displayed directly in the ribbon")]
fn modify_commands_in_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Hole");
    world.expect_text("Fillet");
    assert!(world.run("Hole").minimum[1] < band_height());
    assert_eq!(world.active_id(), world.previous_document.unwrap());
}

#[test]
#[ignore = "requires an X11 window manager"]
fn modal_save_prompt_blocks_native_maximize() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    super::super::x11_input::focus_window();
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    super::super::x11_input::click([width - 24, 17]);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text("Save changes to");
    super::super::main_window_frame_native::press_maximize(&mut world);
    world.pump(Vec::new());
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    world.expect_text("Save changes to");
    world.click_text("Cancel");
    world.close();
}

#[test]
#[ignore = "requires an X11 window manager"]
fn dragging_off_maximize_after_popup_dismissal_cancels_action() {
    let mut world = ApplicationWorld::new();
    full_model_part(&mut world);
    narrow_modify(&mut world);
    open_modify(&mut world);
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    let button = [width - 72, 16];
    super::super::x11_input::left_button(button, true);
    world.pump(Vec::new());
    let away = [width - 180, 16];
    super::super::x11_input::move_pointer(away);
    world.pump(Vec::new());
    super::super::x11_input::left_button(away, false);
    world.pump(Vec::new());
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    world.close();
}

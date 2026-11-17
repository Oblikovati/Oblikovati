// SPDX-License-Identifier: GPL-2.0-only
//! Native window-frame behavior driven by the shared application scenarios.
use super::ApplicationWorld;
use cucumber::{given, then, when};

#[when("the user invokes the desktop window Close shortcut")]
fn invoke_desktop_close(world: &mut ApplicationWorld) {
    world.application_close_requested = true;
    super::x11_input::press_alt_f4();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the main window remains open and the part retains its unsaved changes")]
fn unsaved_part_remains_open(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited part");
    assert!(world.document_is_open(document));
    assert!(world.state().document_storage[&document].unsaved_edits);
}

#[given("the MCAD Software is open in a restored window")]
fn restored_window(world: &mut ApplicationWorld) {
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    world.window_frame_before = Some(super::x11_input::application_bounds());
    super::x11_input::focus_window();
}
#[when("the user double-clicks an empty region of the application bar")]
#[when("the user double-clicks an empty region of the application bar again")]
fn double_click_bar(world: &mut ApplicationWorld) {
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    let point = [width - 390, 18];
    super::x11_input::click(point);
    super::x11_input::click(point);
    world.pump(Vec::new());
}
#[then("the main window is maximized")]
fn maximized(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, true);
}
#[then("the window controls show Restore")]
fn restore_tooltip(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}
#[then("the window controls show Maximize")]
fn maximize_tooltip(world: &mut ApplicationWorld) {
    control_tooltip(world, "Maximize");
}
fn control_tooltip(world: &mut ApplicationWorld, label: &str) {
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    super::x11_input::move_pointer([width - 94, 17]);
    world.pump(Vec::new());
    std::thread::sleep(std::time::Duration::from_millis(700));
    world.pump(Vec::new());
    world.expect_text(label);
}
#[then("the main window returns to its previous size and position")]
fn restored_bounds(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, false);
    assert_eq!(
        super::x11_input::application_bounds(),
        world.window_frame_before.unwrap()
    );
}

#[given("the main window of the MCAD Software is maximized")]
fn maximize_main_window(world: &mut ApplicationWorld) {
    super::x11_input::focus_window();
    super::main_window_frame_native::press_maximize(world);
    super::main_window_frame_native::pump_until(world, true);
    world.window_frame_before = Some(super::x11_input::application_bounds());
}
#[when("the user attempts to resize it from the left boundary")]
fn drag_maximized_edge(world: &mut ApplicationWorld) {
    let height = super::x11_input::application_bounds()[3];
    super::main_window_frame_native::drag_resize_handle(world, [1, height / 2], [40, height / 2]);
}
#[then("the window retains its maximized bounds")]
fn maximized_bounds_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        super::x11_input::application_bounds(),
        world.window_frame_before.unwrap()
    );
    assert!(world.app().window.native.view.window_maximized().unwrap());
}
#[then("Restore remains available in the application bar")]
fn restore_remains_available(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}

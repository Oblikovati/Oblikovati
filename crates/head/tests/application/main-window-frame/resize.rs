// SPDX-License-Identifier: GPL-2.0-only
//! Native resize scenarios compare pointer sprites and window-manager bounds.
use super::ApplicationWorld;
use cucumber::{given, then, when};

fn handle(name: &str) -> ([i32; 2], u16, &'static str) {
    match name {
        "left edge" => ([-1, 0], 70, "horizontal"),
        "right edge" => ([1, 0], 96, "horizontal"),
        "top edge" => ([0, -1], 138, "vertical"),
        "bottom edge" => ([0, 1], 16, "vertical"),
        "top left corner" => ([-1, -1], 134, "northwest-southeast"),
        "bottom right corner" => ([1, 1], 14, "northwest-southeast"),
        "top right corner" => ([1, -1], 136, "northeast-southwest"),
        "bottom left corner" => ([-1, 1], 12, "northeast-southwest"),
        _ => panic!("window handle {name:?}: expected an edge or corner"),
    }
}
fn coordinate(direction: i32, extent: i32) -> i32 {
    match direction {
        -1 => 1,
        1 => extent - 2,
        _ => extent / 2,
    }
}
fn handle_point(world: &ApplicationWorld) -> ([i32; 2], [i32; 2], u16) {
    let (direction, shape, _) = handle(world.window_frame_handle.as_deref().unwrap());
    let bounds = super::x11_input::application_bounds();
    (
        [
            coordinate(direction[0], bounds[2]),
            coordinate(direction[1], bounds[3]),
        ],
        direction,
        shape,
    )
}
#[given("the main window is restored with space to grow within the desktop work area")]
fn restored_with_space(world: &mut ApplicationWorld) {
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    world.window_frame_before = Some(super::x11_input::application_bounds());
    world.previous_document = Some(world.active_id());
    super::x11_input::focus_window();
}
#[when(regex = r"^the user moves the pointer onto the (.+) of the main window$")]
fn pointer_on_handle(world: &mut ApplicationWorld, name: String) {
    handle(&name);
    world.window_frame_handle = Some(name);
    let (point, _, _) = handle_point(world);
    super::x11_input::move_pointer(point);
    world.pump(Vec::new());
}
#[then(regex = r"^the pointer shows a (.+) resize cursor$")]
fn expected_cursor(world: &mut ApplicationWorld, direction: String) {
    let name = world.window_frame_handle.as_deref().unwrap();
    assert_eq!(direction, handle(name).2);
    let (point, _, shape) = handle_point(world);
    let expected = super::x11_input::reference_cursor(shape);
    super::x11_input::move_pointer(point);
    world.pump(Vec::new());
    assert_eq!(super::x11_input::cursor_sprite(), expected);
}
#[when("the user drags that resize handle outward and releases it")]
fn drag_handle(world: &mut ApplicationWorld) {
    let (point, direction, _) = handle_point(world);
    let target = [point[0] + direction[0] * 30, point[1] + direction[1] * 30];
    super::main_window_frame_native::drag_resize_handle(world, point, target);
}
#[then(regex = r"^the window grows along the (.+) axes of that handle$")]
fn grows_expected_axes(world: &mut ApplicationWorld, direction: String) {
    let name = world.window_frame_handle.as_deref().unwrap();
    let (axes, _, expected) = handle(name);
    assert_eq!(direction, expected);
    let before = world.window_frame_before.unwrap();
    let after = super::x11_input::application_bounds();
    for axis in 0..2 {
        super::main_window_frame_native::verify_resize_axis(before, after, axis, axes[axis]);
    }
}
#[then("the opposite boundary stays fixed")]
fn opposite_boundary(world: &mut ApplicationWorld) {
    let name = world.window_frame_handle.as_deref().unwrap();
    grows_expected_axes(world, handle(name).2.to_owned());
}
#[then("the viewport and visible ribbon fit within the resized window")]
fn resized_layout(world: &mut ApplicationWorld) {
    let bounds = super::x11_input::application_bounds();
    let (position, size) = super::window_geometry("##application-ribbon").unwrap();
    assert!(position[0] >= 0. && position[1] >= 0.);
    assert!(size[0] <= bounds[2] as f32 && size[1] < bounds[3] as f32);
    assert_eq!(world.active_id(), world.previous_document.unwrap());
}
#[then("all application window controls remain reachable")]
fn controls_within_frame(_: &mut ApplicationWorld) {
    let bounds = super::x11_input::application_bounds();
    let (position, size) = super::window_geometry("##application-window-controls").unwrap();
    assert!(position[0] >= 0. && position[1] >= 0.);
    assert!(position[0] + size[0] <= bounds[2] as f32);
    assert!(position[1] + size[1] <= bounds[3] as f32);
}

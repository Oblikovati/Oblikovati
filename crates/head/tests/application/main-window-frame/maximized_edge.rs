// SPDX-License-Identifier: GPL-2.0-only
//! Check compositor bounds after attempting to resize a maximized main window.
use super::super::{main_window_frame_native, x11_input};
use super::{ApplicationWorld, control_tooltip, open_frame_part, wayland};
use cucumber::{given, then, when};

#[given("the main window of the MCAD Software is maximized")]
fn maximize_main_window(world: &mut ApplicationWorld) {
    open_frame_part(world);
    if world.app().window.native.view.uses_wayland() {
        wayland::click_frame_control(world, 1);
    } else {
        x11_input::focus_window();
        main_window_frame_native::press_maximize(world);
    }
    main_window_frame_native::pump_until(world, true);
    world.window_frame_before = Some(maximized_frame_bounds(world));
}

#[when("the user attempts to resize it from the left boundary")]
fn drag_maximized_edge(world: &mut ApplicationWorld) {
    let [x, y, _, height] = world.window_frame_before.unwrap();
    if world.app().window.native.view.uses_wayland() {
        let middle = y + height / 2;
        wayland::drag_resize_handle(world, [x + 1, middle], [x + 40, middle]);
    } else {
        main_window_frame_native::drag_resize_handle(world, [1, height / 2], [40, height / 2]);
    }
}

#[then("the window retains its maximized bounds")]
fn maximized_bounds_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        maximized_frame_bounds(world),
        world.window_frame_before.unwrap()
    );
    assert!(world.app().window.native.view.window_maximized().unwrap());
}

fn maximized_frame_bounds(world: &mut ApplicationWorld) -> [i32; 4] {
    if world.app().window.native.view.uses_wayland() {
        wayland::compositor_frame_bounds().expect("expected maximized Weston bounds")
    } else {
        x11_input::application_bounds()
    }
}

#[then("Restore remains available in the application bar")]
fn restore_remains_available(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}

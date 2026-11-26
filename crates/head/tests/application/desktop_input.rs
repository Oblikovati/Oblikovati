// SPDX-License-Identifier: GPL-2.0-only
//! Send native input to the test application's Linux desktop.
use super::{ApplicationWorld, main_window_frame::wayland, x11_input};

pub(super) fn click(world: &mut ApplicationWorld, point: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        let target = compositor_point(world, point);
        wayland::click_compositor_point(world, target);
    } else {
        x11_input::focus_window();
        x11_input::move_pointer(point);
        world.pump(Vec::new());
        x11_input::click(point);
    }
    world.pump(Vec::new());
}

pub(super) fn right_click(world: &mut ApplicationWorld, point: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        let target = compositor_point(world, point);
        wayland::right_click_compositor_point(world, target);
    } else {
        x11_input::right_click(point);
    }
    world.pump(Vec::new());
}

pub(super) fn escape(world: &mut ApplicationWorld) {
    key(world, 0xff1b);
}

pub(super) fn drag(world: &mut ApplicationWorld, start: [i32; 2], end: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        let [x, y, _, _] = wayland::main_toplevel_bounds(world);
        let offset = |[px, py]: [i32; 2]| [x + px, y + py];
        wayland::drag_fullscreen_pointer(world, offset(start), &[offset(end)]);
    } else {
        x11_input::move_pointer(start);
        world.pump(Vec::new());
        x11_input::left_button(start, true);
        world.pump(Vec::new());
        x11_input::move_pointer(end);
        world.pump(Vec::new());
        x11_input::left_button(end, false);
        world.pump(Vec::new());
    }
}

fn compositor_point(world: &mut ApplicationWorld, [px, py]: [i32; 2]) -> [i32; 2] {
    let [x, y, _, _] = wayland::main_toplevel_bounds(world);
    [x + px, y + py]
}

pub(super) fn hover(world: &mut ApplicationWorld, point: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        let target = compositor_point(world, point);
        x11_input::window_targets::move_fullscreen_pointer(target);
    } else {
        x11_input::move_pointer(point);
    }
    world.pump(Vec::new());
}

pub(super) fn key(world: &mut ApplicationWorld, keysym: u32) {
    if world.app().window.native.view.uses_wayland() {
        x11_input::window_targets::press_weston_key(keysym);
    } else {
        x11_input::press_keysym(keysym);
    }
    world.pump(Vec::new());
}

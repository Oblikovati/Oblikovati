// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the visible Waybar task button on the private nested Labwc output.
use super::super::main_window_frame_native::{frame_control_center, pump_frame_controls};
use super::super::x11_input::window_targets::{
    click_fullscreen_output, fullscreen_left_button, labwc_output, move_fullscreen_pointer,
};
use super::ApplicationWorld;
use std::time::{Duration, Instant};
use x11rb::{
    connection::Connection,
    protocol::xproto::{ConnectionExt, ImageFormat, ImageOrder},
};

const ACTIVE: [u8; 3] = [0x28, 0x70, 0x44];
const MINIMIZED: [u8; 3] = [0x33, 0x55, 0x99];

pub(super) fn click_frame_button(world: &mut ApplicationWorld, index: usize) {
    wait_for_taskbar_color(world, ACTIVE);
    // The fixture anchors Oblikovati and its maximized work area at the output origin.
    let center = frame_control_center(index);
    move_fullscreen_pointer(center);
    world.pump(Vec::new());
    fullscreen_left_button(center, true);
    world.pump(Vec::new());
    fullscreen_left_button(center, false);
}

pub(super) fn wait_for_minimized(world: &mut ApplicationWorld) {
    wait_for_taskbar_color(world, MINIMIZED);
}

pub(super) fn activate(world: &mut ApplicationWorld) {
    click_fullscreen_output(taskbar_button_point().map(i32::from));
    wait_for_taskbar_color(world, ACTIVE);
    world.pump(Vec::new());
}

fn taskbar_button_point() -> [i16; 2] {
    let (connection, output) = labwc_output();
    let bounds = connection.get_geometry(output).unwrap().reply().unwrap();
    assert_eq!([bounds.x, bounds.y], [0, 0]);
    [5, (bounds.height - 18).try_into().unwrap()]
}

fn wait_for_taskbar_color(world: &mut ApplicationWorld, expected: [u8; 3]) {
    assert_eq!(std::env::var("OBK_BDD_LABWC_TASKBAR").as_deref(), Ok("1"));
    let point = taskbar_button_point();
    let deadline = Instant::now() + Duration::from_secs(5);
    loop {
        pump_frame_controls(world);
        let actual = taskbar_pixel(point);
        if actual == expected {
            return;
        }
        assert!(
            Instant::now() < deadline,
            "taskbar pixel at {point:?}: expected {expected:?}, found {actual:?}"
        );
        std::thread::sleep(Duration::from_millis(30));
    }
}

fn taskbar_pixel([x, y]: [i16; 2]) -> [u8; 3] {
    let (connection, output) = labwc_output();
    let image = connection
        .get_image(ImageFormat::Z_PIXMAP, output, x, y, 1, 1, u32::MAX)
        .unwrap()
        .reply()
        .unwrap();
    assert_eq!(image.depth, 24);
    let [blue, green, red, _] = image.data.try_into().expect("expected one Xvfb pixel");
    assert_eq!(connection.setup().image_byte_order, ImageOrder::LSB_FIRST);
    [red, green, blue]
}

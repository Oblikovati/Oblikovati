// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard and mouse checks for File menu dismissal.
use super::super::{diagnostics, x11_input};
use super::*;

fn settle(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

fn open_with_mouse(world: &mut ApplicationWorld) {
    let location = diagnostics::pointer_pixels(world.run("File").center());
    x11_input::click(location);
    settle(world);
    world.expect_text("Recent documents");
}

#[test]
fn native_escape_and_outside_click_preserve_the_active_part() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    open_with_mouse(&mut world);
    world.save_capture("file-menu-native-open-3894");
    x11_input::press_escape();
    settle(&mut world);
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    open_with_mouse(&mut world);
    x11_input::click(diagnostics::pointer_pixels([1000., 600.]));
    settle(&mut world);
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    world.close();
}

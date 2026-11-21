// SPDX-License-Identifier: GPL-2.0-only
//! Native mouse and keyboard input for body properties.
use super::super::{diagnostics, x11_input};
use super::*;

fn settle(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_body_properties_escape_retains_selection_and_highlight() {
    let mut world = ApplicationWorld::new();
    part_with_body(&mut world);
    let body = diagnostics::pointer_pixels(world.run(BODY).center());
    x11_input::right_click(body);
    settle(&mut world);
    let properties = diagnostics::pointer_pixels(world.run("Properties").center());
    x11_input::click(properties);
    settle(&mut world);
    world.expect_text(TITLE);
    world.save_capture("body-properties-native-open-3954");
    x11_input::press_escape();
    settle(&mut world);
    assert!(diagnostics::window_geometry(TITLE).is_none());
    row_selected(&mut world);
    highlighted_behind(&mut world);
    world.close();
}

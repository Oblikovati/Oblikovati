// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer checks for viewport orientation.
use super::super::{diagnostics, viewport_pan, x11_input};
use super::*;

fn native_press(world: &mut ApplicationWorld, local: [f32; 2], position: [f32; 2]) {
    let point = [position[0] + local[0], position[1] + local[1]];
    x11_input::click(diagnostics::pointer_pixels(point));
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_rotation_arrow_turns_and_restores_front_view() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    let part = world.active_id();
    world.observed_view_camera = Some(viewport_pan::camera(&mut world));
    let (position, parts) = layout();
    native_press(&mut world, parts.roll_upper, position);
    label_turned(&mut world);
    taller(&mut world);
    world.save_capture("orientation-native-rotated-3961");
    native_press(&mut world, parts.roll_lower, position);
    expect_control_shows(&world, "Front");
    assert!(label_is_upright(&world));
    assert_eq!(world.active_id(), part);
    world.close();
}

// SPDX-License-Identifier: GPL-2.0-only
//! Exact endpoint placement and retained geometry for a midpoint line.
use super::ApplicationWorld;
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[when(
    "the user select the sketch origin as the midpoint and a point 7.673 mm along positive sketch X as the endpoint"
)]
fn choose_origin_and_exact_endpoint(world: &mut ApplicationWorld) {
    super::click_sketch_origin(world);
    world.open_precise_input();
    world.precise_point([7.673, 0.]);
    world.close_precise_input();
}

#[then("a segment extends to both sides of the chosen midpoint")]
fn symmetric_segment_is_visible(world: &mut ApplicationWorld) {
    let (center, endpoints) = super::super::midpoint_positions(&world.edited_geometry());
    for axis in 0..2 {
        assert!((endpoints[0][axis] + endpoints[1][axis] - 2. * center[axis]).abs() < 1e-9);
    }
    let witness = world.line_point(0, 0.25);
    world.expect_segment_drawn(witness, witness);
}

#[then("the segment is retained")]
fn remember_retained_segment(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 1);
    let witness = world.line_point(0, 0.25);
    world.expect_segment_drawn(witness, witness);
    world.remember_edited_sketch(vec![witness]);
}

#[then("the segment is removed")]
fn segment_is_removed(world: &mut ApplicationWorld) {
    assert!(world.edited_geometry().lines().is_empty());
}

#[when("the user measure the restored segment")]
fn measure_restored_segment(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let point = world.line_point(0, 0.25);
    world.sketch_click(point);
}

#[then("Measure displays Curve length 15.346 mm")]
fn measure_exact_segment_length(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 15.346 mm");
}

#[cfg(target_os = "linux")]
#[test]
fn x11_precise_midpoint_endpoint_and_history() {
    let mut world = ApplicationWorld::new();
    world.start_sketch_on("YZ Plane");
    super::super::choose_midpoint_line(&mut world);
    let origin = world.sketch_pixel_of([0., 0.]);
    native_click(&mut world, origin);
    world.open_precise_input();
    native_precise_endpoint(&mut world);
    world.close_precise_input();
    assert_exact_midpoint_segment(&mut world);
    native_history_round_trip(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
fn native_click(world: &mut ApplicationWorld, point: [i32; 2]) {
    super::super::super::super::x11_input::click(point);
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
fn native_precise_endpoint(world: &mut ApplicationWorld) {
    let x = world.precise_field_position("X");
    native_click(world, x);
    super::super::super::super::x11_input::press_control_letter(b'a');
    for character in "7.673".chars() {
        super::super::super::super::x11_input::press_keysym(u32::from(character));
        world.pump(Vec::new());
    }
    let y = world.precise_field_position("Y");
    native_click(world, y);
    super::super::super::super::x11_input::press_control_letter(b'a');
    super::super::super::super::x11_input::type_digits("0");
    world.pump(Vec::new());
    super::super::super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
fn assert_exact_midpoint_segment(world: &mut ApplicationWorld) {
    let (center, ends) = super::super::midpoint_positions(&world.edited_geometry());
    assert_eq!(center, [0., 0.]);
    assert!(((ends[1][0] - ends[0][0]).abs() - 1.5346).abs() < 1e-8);
    assert!((ends[1][1] - ends[0][1]).abs() < 1e-8);
}

#[cfg(target_os = "linux")]
fn native_history_round_trip(world: &mut ApplicationWorld) {
    super::super::super::super::x11_input::press_escape();
    world.pump(Vec::new());
    super::super::super::super::x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert!(world.edited_geometry().lines().is_empty());
    super::super::super::super::x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
    assert_exact_midpoint_segment(world);
}

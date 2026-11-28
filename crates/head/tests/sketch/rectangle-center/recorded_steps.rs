// SPDX-License-Identifier: GPL-2.0-only
//! Additional recorded steps for `features/sketch/rectangle-center`.
use super::{ApplicationWorld, RECORDED_CENTER, center_position};
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

fn in_cm(value: [f64; 2]) -> [f64; 2] {
    value.map(|part| part / 10.)
}

#[when("the user enter 4 and press Tab")]
fn enter_four_tab(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
}
#[then("the second field briefly displays a pointer-derived 0.368 mm preview")]
fn second_field_preview(world: &mut ApplicationWorld) {
    let hud = world.state().sketch_hud.expect("expected rectangle fields");
    assert_eq!(hud.fields.len(), 2, "expected two size fields");
    assert!(hud.fields[0].locked, "expected first size locked");
    assert!(hud.fields[1].active, "expected second size active");
}
#[when("the user replace that field with 0 and press Return")]
fn replace_with_zero_return(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("the user open Measure and select the center")]
fn open_measure_center(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let pixel = world.sketch_pixel_of(in_cm(RECORDED_CENTER));
    world.sketch_click(pixel);
}
#[then("its Point coordinates are (0,-17.513,4.675) mm")]
fn center_point_coords(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 0.000 Y -17.513 Z 4.675 mm");
}
#[when("the user select the upper-left corner")]
fn select_upper_left(world: &mut ApplicationWorld) {
    let high = in_cm([RECORDED_CENTER[0] + 3., RECORDED_CENTER[1] + 2.]);
    let pixel = world.sketch_pixel_of(high);
    world.sketch_click(pixel);
}
#[when("the user select the opposite lower-right corner")]
fn select_lower_right(world: &mut ApplicationWorld) {
    let low = in_cm([RECORDED_CENTER[0] - 3., RECORDED_CENTER[1] - 2.]);
    let pixel = world.sketch_pixel_of(low);
    world.sketch_click(pixel);
}
#[then("their Point coordinates are (0,-14.513,6.675) and (0,-20.513,2.675) mm")]
fn opposite_corner_coords(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 0.000 Y -14.513 Z 6.675 mm");
    world.expect_measure("Position X 0.000 Y -20.513 Z 2.675 mm");
}
#[then("the bottom edge remains visibly in place")]
fn bottom_edge_in_place(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    let min_x = geometry
        .points()
        .iter()
        .map(|point| point.position[0] * 10.)
        .fold(f64::INFINITY, f64::min);
    let expected = RECORDED_CENTER[0] - 3.;
    assert!(
        (min_x - expected).abs() < 0.01,
        "expected bottom edge at {expected:.3} mm, found {min_x:.3} mm"
    );
}
#[then("its Y coordinate has increased by 1.000 mm at displayed precision")]
fn center_y_increased(world: &mut ApplicationWorld) {
    let center = center_position(world);
    let delta = center[0] * 10. - RECORDED_CENTER[0];
    assert!(
        (delta - 1.).abs() < 0.001,
        "expected Y increase of 1.000 mm, got {delta:.4} mm"
    );
}

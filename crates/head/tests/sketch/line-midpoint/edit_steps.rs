// SPDX-License-Identifier: GPL-2.0-only
//! Verify that an accepted length edit preserves the midpoint.
use super::ApplicationWorld;
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user change the length dimension to 12 mm and accept")]
fn accept_length_twelve(world: &mut ApplicationWorld) {
    world.open_dimension_label("10");
    world.type_text("12");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[when("the user measure both endpoints")]
fn measure_both_endpoints(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    for fraction in [0., 1.] {
        let point = world.line_point(0, fraction);
        world.sketch_click(point);
    }
}

#[then("their distance is 12.000 mm")]
fn distance_twelve(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
    world.expect_measure("Selection 2: Point");
    world.expect_measure("Minimum distance 12.000 mm");
}

#[then("their global coordinates are (0.000, 5.196, 3.000) and (0.000, -5.196, -3.000) mm")]
fn coordinates_twelve(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 0.000 Y 5.196 Z 3.000 mm");
    world.expect_measure("Position X 0.000 Y -5.196 Z -3.000 mm");
}

#[then("their displayed coordinate differences are (0.000, 10.392, 6.000) mm")]
fn differences_twelve(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum delta X 0.000 Y 10.392 Z 6.000 mm");
}

#[when("the user close Measure and undo the accepted edit")]
fn close_measure_and_undo(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
    world.undo();
}

#[then("the visible length and extent return to 10 mm")]
#[then("the recorded 10 mm baseline is restored")]
fn baseline_length(world: &mut ApplicationWorld) {
    expect_length_and_extent(world, "10", 1.0);
}

#[then("the visible length and extent return to 12 mm")]
fn edited_length(world: &mut ApplicationWorld) {
    expect_length_and_extent(world, "12", 1.2);
}

fn expect_length_and_extent(world: &mut ApplicationWorld, label: &str, length_cm: f64) {
    world.expect_annotation(label);
    let (center, endpoints) = super::super::midpoint_positions(&world.edited_geometry());
    let actual = (endpoints[1][0] - endpoints[0][0]).hypot(endpoints[1][1] - endpoints[0][1]);
    assert!(
        (actual - length_cm).abs() < 1e-8,
        "expected {length_cm} cm, got {actual}"
    );
    for axis in 0..2 {
        assert!(center[axis].abs() < 1e-8, "midpoint moved: {center:?}");
        assert!((endpoints[0][axis] + endpoints[1][axis]).abs() < 1e-8);
    }
}

#[then("the 30.00-degree annotation remains")]
fn angle_remains(world: &mut ApplicationWorld) {
    world.expect_annotation("30.00");
}

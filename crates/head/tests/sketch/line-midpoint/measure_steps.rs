// SPDX-License-Identifier: GPL-2.0-only
//! Measure the numeric midpoint line through the sketch viewport.
use super::ApplicationWorld;
use cucumber::{then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user open Measure and select the segment away from its midpoint")]
fn measure_segment(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let point = world.line_point(0, 0.25);
    world.sketch_click(point);
}

#[then("the curve length is 10.000 mm")]
fn curve_is_ten_millimetres(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 10.000 mm");
}

#[when("the user click empty canvas to clear the selection")]
#[when("the user click empty canvas to clear the endpoint pair")]
fn clear_measure_selection(world: &mut ApplicationWorld) {
    world.sketch_click([1100, 650]);
    world.expect_measure("Select geometry to measure");
}

#[when("the user select both endpoints")]
fn select_endpoints(world: &mut ApplicationWorld) {
    for fraction in [0., 1.] {
        let point = world.line_point(0, fraction);
        world.sketch_click(point);
    }
}

#[then("their distance is 10.000 mm")]
fn endpoints_are_ten_millimetres_apart(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
    world.expect_measure("Selection 2: Point");
    world.expect_measure("Minimum distance 10.000 mm");
}

#[then("their coordinates are (0.000, 4.330, 2.500) and (0.000, -4.330, -2.500) mm")]
fn endpoint_coordinates(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 0.000 Y 4.330 Z 2.500 mm");
    world.expect_measure("Position X 0.000 Y -4.330 Z -2.500 mm");
}

#[then("their displayed coordinate differences are (0.000, 8.660, 5.000) mm")]
fn endpoint_differences(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum delta X 0.000 Y 8.660 Z 5.000 mm");
}

#[then("their average equals the chosen origin within displayed precision")]
#[then("their average remains the origin within displayed precision")]
fn displayed_midpoint_is_origin(world: &mut ApplicationWorld) {
    let panel = world.state().measure.expect("expected Measure rows");
    let positions: Vec<_> = panel
        .rows
        .iter()
        .filter_map(|row| parse_position(row))
        .collect();
    assert_eq!(positions.len(), 2, "expected two displayed point positions");
    for row in panel.rows.iter().filter(|row| row.starts_with("Position ")) {
        world.expect_measure(row);
    }
    for (first, second) in positions[0].iter().zip(positions[1]) {
        assert!(((first + second) / 2.).abs() <= 0.0005);
    }
}

fn parse_position(row: &str) -> Option<[f64; 3]> {
    let fields: Vec<_> = row.split_whitespace().collect();
    if fields.len() != 8
        || fields[0] != "Position"
        || fields[1] != "X"
        || fields[3] != "Y"
        || fields[5] != "Z"
        || fields[7] != "mm"
    {
        return None;
    }
    Some([
        fields[2].parse().ok()?,
        fields[4].parse().ok()?,
        fields[6].parse().ok()?,
    ])
}

#[when("the user select the origin")]
fn select_origin_in_measure(world: &mut ApplicationWorld) {
    let origin = world.sketch_pixel_of([0., 0.]);
    world.sketch_click(origin);
}

#[then("Measure reports Point (0.000, 0.000, 0.000) mm")]
fn measure_origin(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
    world.expect_measure("Position X 0.000 Y 0.000 Z 0.000 mm");
}

#[test]
fn displayed_position_parser_checks_shape_and_units() {
    assert_eq!(
        parse_position("Position X 0.000 Y -4.330 Z 2.500 mm"),
        Some([0., -4.33, 2.5])
    );
    assert_eq!(parse_position("Position X 0.000 Y -4.330 Z 2.500 cm"), None);
    assert_eq!(parse_position("Position X zero Y -4.330 Z 2.500 mm"), None);
}

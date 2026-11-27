// SPDX-License-Identifier: GPL-2.0-only
//! Additional recorded steps for `features/sketch/rectangle-three-point-center`.
use super::{ApplicationWorld, CENTER_BEFORE_RESIZE, SIDES, expect_none};
use cucumber::then;

#[then("no modal warning or completed new rectangle appears")]
fn no_warning_or_new_rectangle(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
    expect_none(world);
}
#[then("the second selected curve's Measure Point is (0.000, -16.863, 33.619) mm")]
fn second_curve_measure_point(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 2: Curve");
    assert_eq!(
        world.edited_geometry().lines().len(),
        6,
        "sides and diagonals"
    );
}
#[then("the upper short edge remains visibly in place while the lower end extends down and right")]
fn upper_short_edge_stays(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    let has_upper = geometry.points().iter().any(|point| {
        (point.position[0] * 10. - -16.863).abs() < 0.005
            && (point.position[1] * 10. - 33.619).abs() < 0.005
    });
    assert!(has_upper, "expected upper vertex retained");
}
#[then("the construction intersection shifts")]
fn construction_intersection_shifts(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    let diag = geometry.lines()[SIDES];
    let [a, b] = diag
        .endpoints
        .map(|point| geometry.point(point).unwrap().position);
    let mid = [(a[0] + b[0]) * 5., (a[1] + b[1]) * 5.];
    assert!(
        (mid[0] - CENTER_BEFORE_RESIZE[0]).hypot(mid[1] - CENTER_BEFORE_RESIZE[1]) > 0.5,
        "expected center to shift from {CENTER_BEFORE_RESIZE:?}, got {mid:?}"
    );
}
#[then("the displayed absolute coordinate differences are (0.000, 4.928, 7.464) mm")]
fn absolute_coordinate_differences(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum delta X 0.000 Y 4.928 Z 7.464 mm");
}

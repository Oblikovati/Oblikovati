// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{ApplicationWorld, diagnostics};
use super::choose_ellipse;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const RECORDED_CENTER: [f64; 2] = [27.574, -69.211];
const RECORDED_MAJOR: f64 = 7.5793;
const RECORDED_MINOR: f64 = 3.8063;
const ROTATED_AXIS: [f64; 2] = [0.6, 0.8];

#[given("the rotated ellipse is restored by Redo")]
fn restored_rotated_ellipse(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    choose_ellipse(world);
    world.open_precise_input();
    for point in recorded_control_points() {
        world.precise_point(point);
    }
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.redo();
}
fn recorded_control_points() -> [[f64; 2]; 3] {
    let [x, y] = RECORDED_CENTER;
    let [u, v] = ROTATED_AXIS;
    [
        RECORDED_CENTER,
        [x + RECORDED_MAJOR * u, y + RECORDED_MAJOR * v],
        [x - RECORDED_MINOR * v, y + RECORDED_MINOR * u],
    ]
}
#[when("I open Measure, move its panel aside and select the curve")]
fn measure_rotated_ellipse(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let (position, _) = diagnostics::window_geometry("Measure").expect("expected Measure");
    world.drag(
        [(position[0] + 60.) as i32, (position[1] + 10.) as i32],
        [440, 230],
    );
    let geometry = world.edited_geometry();
    let ellipse = *geometry
        .ellipses()
        .last()
        .expect("expected the restored ellipse");
    let center = geometry.point(ellipse.center()).unwrap().position;
    let point = ellipse.evaluate(center, 0.7).unwrap().position;
    let pixel = world.sketch_pixel_of(point);
    world.sketch_click(pixel);
}
#[then("Selection 1 is identified as Ellipse")]
fn ellipse_selection(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Ellipse");
}
#[then("major and minor radii are 7.579 and 3.806 millimeters")]
fn recorded_radii(world: &mut ApplicationWorld) {
    world.expect_measure("Major radius 7.579 mm");
    world.expect_measure("Minor radius 3.806 mm");
}
#[then("major and minor diameters are 15.159 and 7.613 millimeters")]
fn recorded_diameters(world: &mut ApplicationWorld) {
    world.expect_measure("Major diameter 15.159 mm");
    world.expect_measure("Minor diameter 7.613 mm");
}
#[then("length is 36.758 millimeters")]
fn recorded_perimeter(world: &mut ApplicationWorld) {
    world.expect_measure("Length 36.758 mm");
}
#[then("global center is X 0.000, Y 27.574, Z -69.211 millimeters")]
fn recorded_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 27.574 Z -69.211 mm");
}

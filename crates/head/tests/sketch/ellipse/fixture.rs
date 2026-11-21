// SPDX-License-Identifier: GPL-2.0-only
//! Open a writable copy of the published YZ sketch and project its recorded picks.
use super::{ApplicationWorld, ellipse_active};
use cucumber::given;
use oblikovati_model::{quantity::Dimension, units::UnitsOfMeasure};

#[given(
    "a copy of `fixtures/cad/ellipse/empty-yz.opd` is open with Sketch1 being edited in RIGHT view"
)]
fn open_empty_yz_fixture(world: &mut ApplicationWorld) {
    let source = super::super::super::scenario_root().join("fixtures/cad/ellipse/empty-yz.opd");
    let path = world.scratch_folder().join("empty-yz.opd");
    std::fs::copy(&source, &path).unwrap_or_else(|error| {
        panic!("fixture {source:?}: expected a readable native Part: {error}")
    });
    world.reopen_through_dialog(&path);
    let sketch = world.run("Sketch1").center().map(|axis| axis as i32);
    world.double_click(sketch);
    world.pump(Vec::new());
    world.pump(Vec::new());
    super::super::super::viewport_orientation::expect_control_shows(world, "Right");
    assert!(world.edited_geometry().ellipses().is_empty());
    world.expect_ready();
    world.remember_sketch_baseline();
}

#[given(
    regex = r"^the fixture center is \((-?[\d.]+), (-?[\d.]+)\) mm, first-axis point is \((-?[\d.]+), (-?[\d.]+)\) mm and point on the ellipse is \((-?[\d.]+), (-?[\d.]+)\) mm in sketch coordinates$"
)]
fn record_sketch_points(
    world: &mut ApplicationWorld,
    center_x: f64,
    center_y: f64,
    axis_x: f64,
    axis_y: f64,
    perimeter_x: f64,
    perimeter_y: f64,
) {
    let units = UnitsOfMeasure::default();
    let points = [
        [center_x, center_y],
        [axis_x, axis_y],
        [perimeter_x, perimeter_y],
    ];
    world.ellipse_fixture_points = Some(
        points.map(|point| point.map(|value| units.from_preferred(value, Dimension::Length).value)),
    );
}

#[given("the user starts Ellipse")]
fn start_ellipse(world: &mut ApplicationWorld) {
    ellipse_active(world);
}

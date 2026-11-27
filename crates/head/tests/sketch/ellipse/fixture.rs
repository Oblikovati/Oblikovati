// SPDX-License-Identifier: GPL-2.0-only
//! Open a writable copy of the published YZ sketch and project its recorded picks.
use super::super::super::diagnostics;
use super::{ApplicationWorld, ellipse_active};
use cucumber::given;
use oblikovati_model::{quantity::Dimension, units::UnitsOfMeasure};

#[given(
    regex = r"^a copy of `fixtures/cad/ellipse/(empty-yz|circle-midpoint-line|axis-initial|axis-major-8|axis-both-8-4|axis-both-8-4-measure|rotated-measured)\.opd` is open with Sketch1 being edited in RIGHT view$"
)]
fn open_ellipse_yz_fixture(world: &mut ApplicationWorld, fixture: String) {
    let path = copy_ellipse_fixture(world, &fixture);
    world.reopen_through_dialog(&path);
    if fixture == "axis-both-8-4" {
        // Preserve the fixture's scale so both annotations stay within the sketch viewport.
        orient_fixture_to_isometric(world);
    }
    edit_fixture_sketch(world);
    let expected = usize::from(!matches!(
        fixture.as_str(),
        "empty-yz" | "circle-midpoint-line"
    ));
    assert_eq!(world.edited_geometry().ellipses().len(), expected);
    world.expect_ready();
    world.remember_sketch_baseline();
}

fn copy_ellipse_fixture(world: &mut ApplicationWorld, fixture: &str) -> std::path::PathBuf {
    let name = format!("{fixture}.opd");
    let source = super::super::super::scenario_root()
        .join("fixtures/cad/ellipse")
        .join(&name);
    let path = world.scratch_folder().join(name);
    std::fs::copy(&source, &path).unwrap_or_else(|error| {
        panic!("fixture {source:?}: expected a readable native Part: {error}")
    });
    path
}

fn edit_fixture_sketch(world: &mut ApplicationWorld) {
    let sketch = world.run("Sketch1").center().map(|axis| axis as i32);
    world.double_click(sketch);
    world.pump(Vec::new());
    world.pump(Vec::new());
    super::super::super::viewport_orientation::expect_control_shows(world, "Right");
}

fn orient_fixture_to_isometric(world: &mut ApplicationWorld) {
    use oblikovati_api::types::ISO_TOP_RIGHT_VIEW_ORIENTATION;
    use oblikovati_automation::PartViewDisplayAccess;

    let mut display = crate::part_view_display::WindowPartViewDisplay(&mut world.app().window);
    let current = display.camera().unwrap();
    let isometric = display
        .orient(&current, ISO_TOP_RIGHT_VIEW_ORIENTATION)
        .unwrap();
    display.apply(&isometric).unwrap();
    world.pump(Vec::new());
}

/// Set the oblique Home view before opening a sketch, for example before a Finish Sketch test.
pub(super) fn show_home_view(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry("##orientation-control").unwrap();
    let home = oblikovati_gui::GuiOrientationLayout::for_side(size[0]).home;
    world.click([
        (position[0] + home[0]) as i32,
        (position[1] + home[1]) as i32,
    ]);
    world.pump(Vec::new());
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
    let points = [
        [center_x, center_y],
        [axis_x, axis_y],
        [perimeter_x, perimeter_y],
    ];
    world.ellipse_fixture_points = Some(points.map(millimetre_point));
}

#[given(
    regex = r"^the fixture center is the free sketch coordinate \((-?[\d.]+), (-?[\d.]+)\) mm$"
)]
fn record_alias_center(world: &mut ApplicationWorld, x: f64, y: f64) {
    world.ellipse_alias_center = Some(millimetre_point([x, y]));
}

fn millimetre_point(point: [f64; 2]) -> [f64; 2] {
    let units = UnitsOfMeasure::default();
    point.map(|value| units.from_preferred(value, Dimension::Length).value)
}

#[given("the user starts Ellipse")]
fn start_ellipse(world: &mut ApplicationWorld) {
    ellipse_active(world);
}

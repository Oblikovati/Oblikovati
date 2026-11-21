// SPDX-License-Identifier: GPL-2.0-only
//! Edit the saved ellipse's major radius and verify the separate history transactions.
use super::{
    ApplicationWorld,
    dimension_lifecycle::{assert_rendered_ellipse, place_ellipse_radius},
};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const INITIAL_RADIUS: f64 = 0.63832549;

#[given("the recorded ellipse fixture has been created in an editable sketch")]
fn initial_ellipse(world: &mut ApplicationWorld) {
    expect_major_radius(world, INITIAL_RADIUS);
    world.expect_ready();
}

#[when("General Dimension selects its curve and places the horizontal axis extent")]
fn place_major(world: &mut ApplicationWorld) {
    place_ellipse_radius(world, true);
}

#[then("Edit Dimension displays 6.3832549 mm selected")]
fn initial_editor(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("6.3832549"));
    assert!(world.window_text("Edit Dimension").contains("6.3832549"));
}

#[when("the 8 key and Return accept the new value and Escape exits the command")]
fn accept_major_edit(world: &mut ApplicationWorld) {
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
    world.expect_ready();
}

#[then("annotation 8 and a wider ellipse remain")]
#[then("the wider ellipse and annotation 8 return")]
fn widened_ellipse(world: &mut ApplicationWorld) {
    expect_major_radius(world, 0.8);
    world.expect_annotation("8");
}

#[given(
    "the user places the major-radius dimension at 6.3832549 mm and edits it to 8 mm in this session"
)]
fn edit_major_in_session(world: &mut ApplicationWorld) {
    initial_ellipse(world);
    place_major(world);
    initial_editor(world);
    accept_major_edit(world);
    widened_ellipse(world);
}

#[when("Measure selects the ellipse")]
fn select_ellipse_for_measure(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = ellipse_axis_end_pixel(world);
    world.sketch_click(at);
    world.expect_measure("Selection 1: Ellipse");
}

fn ellipse_axis_end_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let geometry = world.edited_geometry();
    let ellipse = geometry.ellipses()[0];
    let center = geometry.point(ellipse.center()).unwrap().position;
    let point = ellipse.evaluate(center, 0.).unwrap().position;
    world.sketch_pixel_of(point)
}

#[given("Measure has closed after the accepted major-radius value of 8")]
fn measure_closed(world: &mut ApplicationWorld) {
    select_ellipse_for_measure(world);
    close_measure(world);
}

#[when("Done closes Measure")]
fn close_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.expect_ready();
}

#[given("the recorded fixture ellipse has driving radii of 8 and 4 mm")]
fn dimensioned_ellipse(world: &mut ApplicationWorld) {
    expect_major_radius(world, 0.8);
    let minor = world.edited_geometry().ellipses()[0].radii()[1];
    assert!(
        (minor - 0.4).abs() < 1e-8,
        "minor radius {minor}: expected 0.4 cm"
    );
    world.expect_annotation("8");
    world.expect_annotation("4");
}

#[when("General Dimension is exited and Measure selects the ellipse curve")]
fn exit_dimension_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.expect_ready();
    select_ellipse_for_measure(world);
}

#[then("major and minor radii are 8.000 and 4.000 mm")]
fn both_radii(world: &mut ApplicationWorld) {
    world.expect_measure("Major radius 8.000 mm");
    world.expect_measure("Minor radius 4.000 mm");
}

#[then("major and minor diameters are 16.000 and 8.000 mm")]
fn both_diameters(world: &mut ApplicationWorld) {
    world.expect_measure("Major diameter 16.000 mm");
    world.expect_measure("Minor diameter 8.000 mm");
}

#[then("length is 38.754 mm")]
fn dimensioned_ellipse_length(world: &mut ApplicationWorld) {
    world.expect_measure("Length 38.754 mm");
}

#[then("global center is 0.000, -7.787, 19.076 mm")]
fn dimensioned_ellipse_global_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y -7.787 Z 19.076 mm");
}

#[then("the retained ellipse remains visible")]
fn dimensioned_ellipse_remains_visible(world: &mut ApplicationWorld) {
    dimensioned_ellipse(world);
    assert_rendered_ellipse(world);
}

#[when("Ctrl+Z is pressed")]
fn undo_radius_edit(world: &mut ApplicationWorld) {
    world.undo();
}

#[when("Ctrl+Y is pressed")]
fn redo_radius_edit(world: &mut ApplicationWorld) {
    world.redo();
}

#[then("major radius is 8.000 mm and major diameter is 16.000 mm")]
fn major_measurement(world: &mut ApplicationWorld) {
    world.expect_measure("Major radius 8.000 mm");
    world.expect_measure("Major diameter 16.000 mm");
}

#[then("minor radius is 2.979 mm and minor diameter is 5.958 mm")]
fn measured_initial_minor(world: &mut ApplicationWorld) {
    world.expect_measure("Minor radius 2.979 mm");
    world.expect_measure("Minor diameter 5.958 mm");
}

#[then("length is 36.320 mm and global center is 0.000, -6.213, 15.221 mm")]
fn measured_initial_center(world: &mut ApplicationWorld) {
    world.expect_measure("Length 36.320 mm");
    world.expect_measure("Center X 0.000 Y -6.213 Z 15.221 mm");
}

#[then("the ellipse narrows and the retained dimension annotation becomes 6.383")]
fn original_radius_returns(world: &mut ApplicationWorld) {
    expect_major_radius(world, INITIAL_RADIUS);
    world.expect_annotation("6.383");
}

#[then("the major-radius dimension remains")]
fn major_dimension_remains(world: &mut ApplicationWorld) {
    let labels = crate::HeadSketchWork::sketch_dimension_annotations(&world.app().owner).unwrap();
    assert_eq!(labels.labels.len(), 1);
    widened_ellipse(world);
}

fn expect_major_radius(world: &mut ApplicationWorld, expected: f64) {
    let geometry = world.edited_geometry();
    let [ellipse] = geometry.ellipses() else {
        panic!("expected one edited ellipse");
    };
    let actual = ellipse.radii()[0];
    assert!(
        (actual - expected).abs() < 1e-8,
        "major radius {actual}: expected {expected}"
    );
}

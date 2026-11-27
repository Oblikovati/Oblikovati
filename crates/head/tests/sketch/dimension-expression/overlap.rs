// SPDX-License-Identifier: GPL-2.0-only
//! Interacting with overlapping driving and reference dimension annotations.
use super::*;

#[given("the line has source 30, dependent fx:60 and driven length (67.082)")]
fn overlapping_dimensions(world: &mut ApplicationWorld) {
    line_fixture::create_overlapping(world);
    world.key(ViewportPhysicalKey::D, false);
    click_component_line(world);
    world.type_text("d63*2");
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
    world.open_dimension_label("20");
    world.type_text("30");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:60");
    world.expect_annotation("(67.082)");
}
fn click_component_line(world: &mut ApplicationWorld) {
    let point = world.sketch_pixel_of([5., 2.5]);
    world.sketch_click(point);
}
#[when("the user click the overlapping annotation while General Dimension is active")]
fn click_overlapping_annotation(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    world.click(world.exact_run_center("(67.082)"));
}
#[then("Driven Dimension d78 opens showing 67.082 mm")]
fn reference_viewer(world: &mut ApplicationWorld) {
    let shown = world.window_text("Driven Dimension");
    assert!(
        shown.contains("d78") && shown.contains("67.082mm"),
        "{shown:?}"
    );
}
#[when("the user press Escape and select the component dimension line away from the overlap")]
fn select_separate_line(world: &mut ApplicationWorld) {
    world.escape();
    click_component_line(world);
}
#[then("Edit Dimension d64 opens showing d63 * 2 ul")]
fn component_editor(world: &mut ApplicationWorld) {
    let shown = world.window_text("Edit Dimension");
    assert!(
        shown.contains("d64") && shown.contains("d63*2ul"),
        "{shown:?}"
    );
}

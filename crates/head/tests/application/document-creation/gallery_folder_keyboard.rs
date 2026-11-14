// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard highlight and folder activation in the template chooser.
use super::*;

#[given("the details pane identifies the selected Standard (mm) part template")]
fn metric_part_selected(world: &mut ApplicationWorld) {
    world.click_text(METRIC_PART);
    expect_selected_template(world, METRIC_PART);
    world.expect_text("Length: millimeters (mm)");
}
#[when("the user clicks Metric in the folder tree and presses Up")]
fn highlight_english(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.key(ViewportPhysicalKey::UpArrow, false);
}
#[then("English is highlighted in the tree")]
fn english_highlighted(world: &mut ApplicationWorld) {
    assert!(world.row_accented_beside("English", "Metric"));
}
#[then("the metric gallery and Standard (mm) details remain visible")]
fn metric_gallery_retained(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_SHEET_METAL);
    expect_selected_template(world, METRIC_PART);
    world.expect_text("Length: millimeters (mm)");
}
#[when("the user clicks the highlighted English folder")]
fn activate_english(world: &mut ApplicationWorld) {
    world.click_text("English");
}
#[then("the details pane still identifies Standard (mm) with millimeter units")]
fn metric_details_retained(world: &mut ApplicationWorld) {
    expect_selected_template(world, METRIC_PART);
    world.expect_text("Length: millimeters (mm)");
}

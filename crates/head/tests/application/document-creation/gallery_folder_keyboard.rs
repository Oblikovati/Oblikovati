// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard highlight and folder activation in the template chooser.
use super::*;

#[given("the details pane identifies the selected Standard (mm) part template")]
fn metric_part_selected(_: &mut ApplicationWorld) {
    super::captured_catalog::require_named_catalog("selected Standard (mm) part template");
}

#[when("the user clicks the Metric folder in the folder tree")]
fn click_metric_folder(world: &mut ApplicationWorld) {
    world.click_text("Metric");
}
#[when("the user presses Up")]
fn press_folder_up(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::UpArrow, false);
}
#[when("the user clicks Metric in the folder tree and presses Up")]
fn highlight_english(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.key(ViewportPhysicalKey::UpArrow, false);
}
#[then("English is highlighted in the tree")]
#[then("the English folder is highlighted in the tree")]
fn english_highlighted(world: &mut ApplicationWorld) {
    assert!(world.row_accented_beside("English", "Metric"));
}
#[then("the metric gallery and Standard (mm) details remain visible")]
fn metric_gallery_retained(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_SHEET_METAL);
    expect_selected_template(world, METRIC_PART);
    world.expect_text("Length: millimeters (mm)");
}
#[then("the metric template gallery remains visible")]
fn metric_gallery_visible(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_SHEET_METAL);
    world.expect_text(METRIC_PART);
}
#[then("the details pane still identifies Standard (mm)")]
fn captured_standard_details(_: &mut ApplicationWorld) {
    super::captured_catalog::require_named_catalog("selected Standard (mm) part template");
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

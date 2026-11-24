// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard highlight and folder activation in the template chooser.
use super::*;

#[given("the details pane identifies the selected Standard (mm) part template")]
fn metric_part_selected(world: &mut ApplicationWorld) {
    select_metric(world);
    expect_part_details(world);
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
    metric_gallery(world);
    expect_part_details(world);
}
#[then("the metric template gallery remains visible")]
fn metric_gallery_visible(world: &mut ApplicationWorld) {
    metric_gallery(world);
}
#[then("the details pane still identifies Standard (mm)")]
fn captured_standard_details(world: &mut ApplicationWorld) {
    expect_part_details(world);
}
#[when("the user clicks the highlighted English folder")]
fn activate_english(world: &mut ApplicationWorld) {
    world.click_text("English");
}
#[then("the details pane still identifies Standard (mm) with millimeter units")]
fn metric_details_retained(world: &mut ApplicationWorld) {
    expect_part_details(world);
}

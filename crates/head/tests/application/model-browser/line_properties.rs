// SPDX-License-Identifier: GPL-2.0-only
//! Geometry Properties and the color dialog behind it.
use super::find::{expect_control, open_list};
use super::*;

const TITLE: &str = "Geometry Properties";
fn open_properties(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
    choose_menu_row(world, "Properties");
    settle(world);
    world.expect_text(TITLE);
}
fn cancel(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    settle(world);
}
fn apply_enabled(world: &ApplicationWorld) -> bool {
    world.run_is_enabled("Apply")
}
fn choose_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    settle(world);
}
fn expect_field_and_apply(world: &ApplicationWorld, field: &str, value: &str, enabled: bool) {
    expect_control(world, field, value);
    assert_eq!(apply_enabled(world), enabled, "Apply");
}
#[given("the sketch line type is Default")]
#[given("its line color is Default")]
fn sketch_default(_: &mut ApplicationWorld) {}
#[when("I open Properties from the sketch row context menu")]
#[given("its geometry properties dialog is open")]
#[given("its geometry properties dialog is open with Scale unchecked")]
fn open_dialog(world: &mut ApplicationWorld) {
    open_properties(world);
}
#[then("the geometry properties dialog shows line color, line type and line weight controls")]
fn controls(world: &mut ApplicationWorld) {
    expect_rows(world, &["Line Color", "Line Type", "Line Weight", "Scale"]);
}
#[then("the line type field displays Default")]
fn type_default(world: &mut ApplicationWorld) {
    expect_control(world, "Line Type", "Default");
}
#[then("Apply is disabled")]
fn apply_off(world: &mut ApplicationWorld) {
    assert!(!apply_enabled(world), "expected Apply disabled");
}
#[then("Apply is enabled")]
fn apply_on(world: &mut ApplicationWorld) {
    assert!(apply_enabled(world), "expected Apply enabled");
}
#[when("I choose Dashed as the line type")]
fn choose_dashed(world: &mut ApplicationWorld) {
    open_list(world, "Line Type");
    choose_row(world, "Dashed");
}
#[then("the line type field displays Dashed and Apply is enabled")]
fn type_dashed(world: &mut ApplicationWorld) {
    expect_field_and_apply(world, "Line Type", "Dashed", true);
}
#[when("I click Cancel and reopen the sketch properties")]
fn cancel_reopen(world: &mut ApplicationWorld) {
    cancel(world);
    world.expect_no_text(TITLE);
    open_properties(world);
}
#[then("the properties dialog closes")]
#[then("the geometry properties dialog closes")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text(TITLE);
}
// Color
#[when("I open the Line Color list")]
fn open_colors(world: &mut ApplicationWorld) {
    open_list(world, "Line Color");
}
#[then("the list offers Default, Black, Red, Yellow, Green, Cyan, Blue, Magenta and White")]
fn color_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Default", "Black", "Red", "Yellow", "Green", "Cyan", "Blue", "Magenta", "White",
        ],
    );
}
#[then("it offers a Select action")]
fn select_row(world: &mut ApplicationWorld) {
    world.expect_text("Select");
}
#[when("I choose Red")]
fn choose_red(world: &mut ApplicationWorld) {
    choose_row(world, "Red");
}
#[then("the Line Color field displays Red and Apply is enabled")]
fn color_red(world: &mut ApplicationWorld) {
    expect_field_and_apply(world, "Line Color", "Red", true);
}
#[then("the Line Color field displays Default and Apply is disabled")]
#[then("the geometry properties dialog shows Default line color")]
fn color_default(world: &mut ApplicationWorld) {
    expect_field_and_apply(world, "Line Color", "Default", false);
}
#[when("I choose Select from the Line Color list")]
fn choose_select(world: &mut ApplicationWorld) {
    open_list(world, "Line Color");
    choose_row(world, "Select");
    settle(world);
}
#[then("a color dialog opens with basic color swatches and custom color slots")]
fn color_dialog(world: &mut ApplicationWorld) {
    expect_rows(world, &["Basic colors", "Custom colors"]);
}
#[when("I choose Define Custom Colors")]
fn define(world: &mut ApplicationWorld) {
    choose_row(world, "Define Custom Colors");
}
#[then("the dialog expands to show a color field, brightness control and color preview")]
fn expanded(world: &mut ApplicationWorld) {
    world.expect_text("Preview");
    let (_, size) = diagnostics::window_geometry("Color").expect("expected the color dialog");
    assert!(size[0] > 500., "color dialog width {}", size[0]);
}
#[then("it shows hue, saturation, luminance and red, green and blue fields")]
fn number_fields(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Hue", "Saturation", "Luminance", "Red", "Green", "Blue"],
    );
}
#[then("it offers Add to Custom Colors")]
fn add_custom(world: &mut ApplicationWorld) {
    world.expect_text("Add to Custom Colors");
}
#[when("I cancel the color dialog")]
fn cancel_colors(world: &mut ApplicationWorld) {
    cancel(world);
}
#[when("I cancel the geometry properties dialog")]
fn cancel_properties(world: &mut ApplicationWorld) {
    cancel(world);
}
#[then("the dialog closes without applying a color change")]
fn closed_unapplied(world: &mut ApplicationWorld) {
    world.expect_no_text(TITLE);
    let browser = world.state().browser.expect("expected a browser");
    let sketch = browser.sketches.first().expect("expected the sketch");
    assert_eq!(sketch.line_style.color, None, "line color");
}
// Weight
#[when("I open the Line Weight list")]
fn open_weights(world: &mut ApplicationWorld) {
    open_list(world, "Line Weight");
}
#[then("the list shows Default and numeric millimeter weights with line samples")]
fn weights(world: &mut ApplicationWorld) {
    expect_rows(world, &["Default", "0.13 mm", "0.5 mm", "1 mm"]);
}
#[then("0.25 mm is available")]
fn quarter(world: &mut ApplicationWorld) {
    world.expect_text("0.25 mm");
}
#[when("I choose 0.25 mm")]
fn choose_quarter(world: &mut ApplicationWorld) {
    choose_row(world, "0.25 mm");
}
#[then("the Line Weight field displays 0.25 mm and Apply is enabled")]
fn weight_quarter(world: &mut ApplicationWorld) {
    expect_field_and_apply(world, "Line Weight", "0.25 mm", true);
}
#[then("the Line Weight field displays Default and Apply is disabled")]
fn weight_default(world: &mut ApplicationWorld) {
    expect_field_and_apply(world, "Line Weight", "Default", false);
}
// Scale
fn scale_field_enabled(world: &ApplicationWorld) -> bool {
    world.run_is_enabled("1.00")
}
#[then("the scale value is shown as 1.00 and its field is disabled")]
#[then("the scale field is disabled and still shows 1.00")]
fn scale_off(world: &mut ApplicationWorld) {
    world.expect_text("1.00");
    assert!(!scale_field_enabled(world), "expected the field disabled");
}
#[when("I check Scale")]
#[when("I clear Scale")]
fn toggle_scale(world: &mut ApplicationWorld) {
    choose_row(world, "Scale");
}
#[then("the scale field is enabled and shows 1.00")]
fn scale_on(world: &mut ApplicationWorld) {
    world.expect_text("1.00");
    assert!(scale_field_enabled(world), "expected the field enabled");
}

// SPDX-License-Identifier: GPL-2.0-only
//! Two-point rectangle dimension editing and zero-size input scenarios.
use super::*;

// rectangle-two-point-12
#[given("the rectangle height editor is open with 6 mm selected")]
fn editor_open(world: &mut ApplicationWorld) {
    typed_rectangle(world, "6", "4");
    world.open_dimension_label("6");
    assert!(world.text_is_selected("6mm"), "expected the value selected");
}
#[when("I enter 8 and press Escape")]
#[when("the user enter 8 and press Escape")]
fn enter_and_escape(world: &mut ApplicationWorld) {
    world.type_text("8");
    world.escape();
    world.pump(Vec::new());
}
#[when("I reopen the height annotation")]
#[when("the user reopen the height annotation")]
fn reopen_height(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
}
#[then("the height editor still contains 6 mm")]
fn still_six(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("6mm"), "expected 6 mm in the editor {text:?}");
}
// rectangle-two-point-13
#[given("the numeric rectangle retains height 6 and width 4 annotations")]
fn retained_six_four(world: &mut ApplicationWorld) {
    fresh_command(world);
    typed_six_four(world);
    world.escape();
    world.pump(Vec::new());
    six_four(world);
}
#[given(
    "the recorded Right-view sketch contains the overlapping ellipse and a rectangle annotated 6 mm high by 4 mm wide, with its height edit active"
)]
fn recorded_height_edit(world: &mut ApplicationWorld) {
    retained_six_four(world);
    super::super::super::viewport_orientation::expect_control_shows(world, "Right");
    overlaps_ellipse(world);
    world.open_dimension_label("6");
    assert!(world.text_is_selected("6mm"));
}
#[when("I accept height 8 mm and exit General Dimension")]
#[when("the user accept height 8 mm and exit General Dimension")]
fn accept_eight(world: &mut ApplicationWorld) {
    let [min, max] = rectangle_bounds(world);
    world.observed_view = Some([min[1], max[1]]);
    world.open_dimension_label("6");
    accept_open_height(world);
}
#[when("the user accepts height 8 mm and exits General Dimension")]
fn accept_open_height(world: &mut ApplicationWorld) {
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
#[then("the height annotation shows 8 and the width annotation remains 4")]
fn height_eight_width_four(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_annotation("4");
    assert_rectangle_edge_lengths(&world.edited_geometry(), [0.4, 0.8]);
}
#[then("the rectangle extends downward while its top edge stays in place in the recorded view")]
fn height_extends_down(world: &mut ApplicationWorld) {
    let before = world
        .observed_view
        .expect("expected previous rectangle bounds");
    let [min, max] = rectangle_bounds(world);
    assert!(
        (min[1] - before[0]).abs() <= 1,
        "top moved: {before:?} to {min:?}..{max:?}"
    );
    assert!(
        max[1] > before[1],
        "bottom did not move down: {before:?} to {min:?}..{max:?}"
    );
}
#[when("I open Measure and select the left edge")]
#[when("the user open Measure and select the left edge")]
#[when("the user opens Measure and selects the left edge")]
fn measure_left(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let count = sides(world);
    let left = (0..count)
        .map(|index| {
            (
                index,
                world.line_point(index, 0.),
                world.line_point(index, 1.),
            )
        })
        .filter(|(_, from, to)| from[0] == to[0])
        .min_by_key(|(_, from, _)| from[0])
        .map(|(index, _, _)| index)
        .expect("expected an upright side");
    let at = world.line_point(left, 0.3);
    world.sketch_click(at);
}
#[then("Measure reports the selected left edge length as 8.000 mm")]
fn left_eight(world: &mut ApplicationWorld) {
    world.expect_measure("Length 8.000 mm");
}
#[when("I close Measure and undo the height edit")]
#[when("the user close Measure and undo the height edit")]
#[when("the user closes Measure and undoes the height edit")]
fn close_and_undo(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
    world.undo();
}
#[then("the visible height annotation returns to 6")]
fn height_six(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    assert_rectangle_edge_lengths(&world.edited_geometry(), [0.4, 0.6]);
}
#[when("I redo the height edit")]
#[when("the user redo the height edit")]
#[when("the user redoes the height edit")]
fn redo_height(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the visible height annotation returns to 8")]
fn height_eight(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    assert_rectangle_edge_lengths(&world.edited_geometry(), [0.4, 0.8]);
}
#[then("width annotation 4 remains")]
#[then("the width annotation remains 4")]
fn width_four(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
}
// rectangle-two-point-14
#[given("the recorded sketch is Ready with the 4-by-8 rectangle retained")]
fn retained_four_eight(world: &mut ApplicationWorld) {
    typed_rectangle(world, "8", "4");
    assert_typed_rectangle_lengths(&world.edited_geometry(), [0.4, 0.8]);
}
#[when("I enter 4, press Tab, enter 0 and press Return")]
#[when("the user enter 4, press Tab, enter 0 and press Return")]
fn zero_second(world: &mut ApplicationWorld) {
    type_sizes(world, "4", "0");
}
#[then("Select first corner returns without a warning or a visible new rectangle")]
fn back_no_new(world: &mut ApplicationWorld) {
    world.expect_status("Select first corner");
    world.expect_no_text("Warning");
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
}
#[then("Ready returns and the prior rectangle remains")]
fn prior_remains(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
}

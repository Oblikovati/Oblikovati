// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/rectangle-three-point`: oblique rectangles by pointer and by typed values.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const FIRST: [i32; 2] = [800, 520];
const SECOND: [i32; 2] = [980, 440];
const THIRD: [i32; 2] = [1030, 560];
const SIDES: usize = 4;
/// Recorded lengths of the pointer rectangle's adjacent edges, in millimetres.
const RECORDED_LENGTH: f64 = 10.912;
const RECORDED_WIDTH: f64 = 5.296;

fn open_tool(world: &mut ApplicationWorld) {
    world.click_text("3-Pt Rect.");
    world.pump(Vec::new());
}
fn sides(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().lines().len()
}
fn pointer_rectangle(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(FIRST);
    world.sketch_click(SECOND);
    world.sketch_click(THIRD);
}
fn length_and_angle(world: &mut ApplicationWorld, length: &str, angle: &str) {
    world.type_text(length);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(angle);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn accept_value(world: &mut ApplicationWorld, value: &str) {
    world.type_text(value);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn numeric_rectangle(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    open_tool(world);
    world.sketch_click(FIRST);
    length_and_angle(world, "6", "30");
    accept_value(world, "4");
    world.escape();
    world.pump(Vec::new());
    assert_eq!(sides(world), SIDES, "sides");
}
fn select_adjacent(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let first = world.line_point(0, 0.3);
    world.sketch_click(first);
    let second = world.line_point(1, 0.3);
    world.sketch_click(second);
}
// rectangle-three-point-01
#[when(
    "I select Three Point Rectangle, choose two corners defining an oblique first edge, then choose its width"
)]
fn three_points(world: &mut ApplicationWorld) {
    pointer_rectangle(world);
    world.hover([1200, 760]);
}
#[then("four connected sides form a rectangle with the chosen first-edge direction")]
fn formed(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), SIDES, "sides");
    world.expect_segment_drawn(FIRST, SECOND);
}
// rectangle-three-point-06
#[when("I start Three Point Rectangle and select the first corner")]
fn tool_and_first(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(FIRST);
}
#[then("the prompt is Select second corner")]
fn asks_second(world: &mut ApplicationWorld) {
    world.expect_status("Select second corner");
}
#[then("Ready returns with no new rectangle")]
fn ready_none(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), 0, "sides");
}
// rectangle-three-point-07
#[given("an editable sketch")]
fn editable(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[when("I select distinct first and second corners through Three Point Rectangle")]
fn first_and_second(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(FIRST);
    world.sketch_click(SECOND);
}
#[then("the prompt is Select third corner")]
fn asks_third(world: &mut ApplicationWorld) {
    world.expect_status("Select third corner");
}
#[then("Ready returns without retained edges from the partial rectangle")]
fn ready_no_edges(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), 0, "sides");
}
// rectangle-three-point-08
#[given("Three Point Rectangle has accepted the first corner at a chosen location")]
fn accepted_first(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    tool_and_first(world);
}
#[when("I click the same location again")]
fn click_again(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
}
#[then("the command remains at Select second corner with zero length and no modal error")]
fn zero_length(world: &mut ApplicationWorld) {
    asks_second(world);
    let text = world.window_text("##sketch-hud");
    assert!(
        text.contains("0.000mm"),
        "expected a zero length in {text:?}"
    );
    world.expect_no_text("Warning");
}
#[when(
    "I select a distinct second corner defining an oblique first edge and a third point with nonzero width"
)]
fn second_and_third(world: &mut ApplicationWorld) {
    world.sketch_click(SECOND);
    world.sketch_click(THIRD);
}
#[then("the oblique rectangle is committed and the prompt returns to Select first corner")]
fn committed_back(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), SIDES, "sides");
    world.expect_status("Select first corner");
}
#[when("I press Escape and Ctrl+Z")]
fn escape_and_undo(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.undo();
}
#[then("the rectangle disappears")]
fn gone(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), 0, "sides");
}
#[then("the rectangle returns")]
fn back(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), SIDES, "sides");
}
// rectangle-three-point-09
#[given("the recorded oblique rectangle has been restored")]
fn restored(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    open_tool(world);
    world.open_precise_input();
    let turn = 25_f64.to_radians();
    let [cos, sin] = [turn.cos(), turn.sin()];
    let second = [RECORDED_LENGTH * cos, RECORDED_LENGTH * sin];
    world.precise_point([0., -40.]);
    world.precise_point([second[0], -40. + second[1]]);
    world.precise_point([
        second[0] - RECORDED_WIDTH * sin,
        -40. + second[1] + RECORDED_WIDTH * cos,
    ]);
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.redo();
}
#[when("I open Measure and select the two adjacent edges away from midpoint snaps")]
#[when("I open Measure and select the two adjacent edges")]
fn measure_adjacent(world: &mut ApplicationWorld) {
    select_adjacent(world);
}
#[then("both selections are Curves")]
fn both_curves(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Selection 2: Curve");
}
#[then("their lengths are 10.912 mm and 5.296 mm for this fixture")]
fn recorded_lengths(world: &mut ApplicationWorld) {
    world.expect_measure("Length 10.912 mm");
    world.expect_measure("Length 5.296 mm");
}
#[then("the angle is 90.00 degrees and minimum distance is 0.000 mm")]
#[then("their angle is 90.00 degrees and minimum distance is 0.000 mm")]
fn square_and_touching(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 90.00 deg");
    world.expect_measure("Minimum distance 0.000 mm");
}
// rectangle-three-point-10 and rectangle-three-point-11
#[when("I choose Three Point Rectangle and select the fixture first corner")]
fn choose_and_first(world: &mut ApplicationWorld) {
    tool_and_first(world);
}
#[when("I enter 6, press Tab, enter 3 then 0 and press Return")]
fn typed_first_edge(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("3");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("the command requests Select third corner with a distance field")]
fn third_with_field(world: &mut ApplicationWorld) {
    asks_third(world);
    world.hover(THIRD);
    world.expect_hud_shows_length();
}
#[given("the numeric first edge has been accepted")]
fn first_edge_accepted(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    tool_and_first(world);
    typed_first_edge(world);
    world.hover(THIRD);
}
#[when("I enter 0 in the third-corner distance field and press Return")]
fn zero_width(world: &mut ApplicationWorld) {
    accept_value(world, "0");
}
#[then("the expression remains red and Select third corner remains active")]
fn zero_red(world: &mut ApplicationWorld) {
    world.expect_hud_value_invalid("0");
    asks_third(world);
}
#[then("no modal warning or completed rectangle appears")]
fn nothing_completed(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
    assert_eq!(sides(world), 0, "sides");
}
#[when("I press Ctrl+A, enter 4 and press Return")]
fn select_all_four(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::A, true);
    accept_value(world, "4");
}
#[then("an oblique rectangle appears with annotations 6, 4 and 30.00 degrees")]
fn annotated(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), SIDES, "sides");
    for value in ["6", "4", "30.00"] {
        world.expect_annotation(value);
    }
}
// rectangle-three-point-12
#[given("the numeric rectangle from the recovery workflow is retained")]
fn numeric_retained(world: &mut ApplicationWorld) {
    numeric_rectangle(world);
}
#[then("both selections are Curves with lengths 6.000 and 4.000 mm")]
fn six_and_four(world: &mut ApplicationWorld) {
    both_curves(world);
    world.expect_measure("Length 6.000 mm");
    world.expect_measure("Length 4.000 mm");
}
#[then("maximum distance is 7.211 mm")]
fn max_diagonal(world: &mut ApplicationWorld) {
    world.expect_measure("Maximum distance 7.211 mm");
}
// rectangle-three-point-13 to rectangle-three-point-15
#[given("the retained numeric rectangle has a driving width dimension equal to 4 mm")]
fn retained_width(world: &mut ApplicationWorld) {
    numeric_rectangle(world);
}
#[when("I open its width annotation, enter 5 and press Escape")]
fn width_escape(world: &mut ApplicationWorld) {
    world.open_dimension_label("4");
    world.type_text("5");
    world.escape();
    world.pump(Vec::new());
}
#[then("Edit Dimension still contains 4 mm")]
fn still_four(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "expected 4 mm in the editor {text:?}");
}
#[given("the width dimension editor contains 4 mm")]
fn width_editor(world: &mut ApplicationWorld) {
    numeric_rectangle(world);
    world.open_dimension_label("4");
}
#[when("I enter 5, press Return and exit General Dimension")]
fn accept_five(world: &mut ApplicationWorld) {
    accept_value(world, "5");
    world.escape();
    world.pump(Vec::new());
}
#[then("the rectangle widens and its length 6 and direction 30 annotations remain")]
fn widened(world: &mut ApplicationWorld) {
    for value in ["5", "6", "30.00"] {
        world.expect_annotation(value);
    }
}
#[when("I measure the width edge and adjacent length edge")]
fn measure_width_and_length(world: &mut ApplicationWorld) {
    select_adjacent(world);
}
#[then("both selections are Curves with lengths 5.000 and 6.000 mm")]
fn five_and_six(world: &mut ApplicationWorld) {
    both_curves(world);
    world.expect_measure("Length 5.000 mm");
    world.expect_measure("Length 6.000 mm");
}
#[then("maximum distance is 7.810 mm")]
fn max_after_widening(world: &mut ApplicationWorld) {
    world.expect_measure("Maximum distance 7.810 mm");
}
#[given("the accepted width 5 rectangle is retained and Measure is closed")]
fn accepted_width(world: &mut ApplicationWorld) {
    numeric_rectangle(world);
    world.open_dimension_label("4");
    accept_five(world);
}
#[then("the narrower shape and width annotation 4 return")]
fn narrower(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    world.expect_no_annotation("5");
}
#[then("width annotation 5 and the wider shape return")]
fn wider(world: &mut ApplicationWorld) {
    world.expect_annotation("5");
    world.expect_no_annotation("4");
}
#[then("length 6 and direction 30 annotations remain")]
fn other_annotations(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_annotation("30.00");
}

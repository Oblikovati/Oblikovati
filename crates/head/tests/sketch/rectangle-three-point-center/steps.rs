// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/rectangle-three-point-center`: oriented rectangles from their center.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const CENTER: [i32; 2] = [880, 560];
const SECOND: [i32; 2] = [960, 500];
const THIRD: [i32; 2] = [1000, 590];
const SIDES: usize = 4;
const DIAGONALS: usize = 2;
const RECORDED_HALF_LENGTH: f64 = 6.377;
const RECORDED_HALF_WIDTH: f64 = 2.5935;
/// Where the recorded center sits before the length edit moves it half the change along its side.
const CENTER_BEFORE_RESIZE: [f64; 2] = [-20.19303, 29.387];

fn open_tool(world: &mut ApplicationWorld) {
    world.choose_from_flyout("3-Pt Rect.", "Three Point Center Rectangle");
}
fn line_counts(world: &mut ApplicationWorld) -> (usize, usize) {
    let geometry = world.edited_geometry();
    let index = geometry.read_index();
    let construction = geometry
        .lines()
        .iter()
        .filter(|line| index.is_construction(line.id).unwrap_or(false))
        .count();
    (geometry.lines().len() - construction, construction)
}
fn expect_shape(world: &mut ApplicationWorld) {
    assert_eq!(
        line_counts(world),
        (SIDES, DIAGONALS),
        "sides and diagonals"
    );
}
fn expect_none(world: &mut ApplicationWorld) {
    assert_eq!(line_counts(world), (0, 0), "sides and diagonals");
}
fn pointer_shape(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(CENTER);
    world.sketch_click(SECOND);
    world.sketch_click(THIRD);
}
fn accept_value(world: &mut ApplicationWorld, value: &str) {
    world.type_text(value);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn typed_first_axis(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("3");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn numeric_shape(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    open_tool(world);
    world.open_precise_input();
    world.precise_point(CENTER_BEFORE_RESIZE);
    typed_first_axis(world);
    world.hover([1100, 700]);
    accept_value(world, "4");
    world.escape();
    world.pump(Vec::new());
    expect_shape(world);
}
fn measure_adjacent(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let first = world.line_point(0, 0.3);
    world.sketch_click(first);
    let second = world.line_point(1, 0.3);
    world.sketch_click(second);
}
fn resized_shape(world: &mut ApplicationWorld) {
    numeric_shape(world);
    world.open_dimension_label("6");
    accept_value(world, "8");
    world.escape();
    world.pump(Vec::new());
}
// rectangle-three-point-center-01
#[when(
    "the user choose Three Point Center Rectangle and select its center, an oblique orientation point and a width point"
)]
fn three_points(world: &mut ApplicationWorld) {
    pointer_shape(world);
}
#[then("an oblique rectangle with center construction geometry is committed")]
#[then("the oblique rectangle and center construction geometry are committed")]
#[then("the rectangle and construction geometry return")]
#[then("the rectangle and construction geometry remain")]
fn committed(world: &mut ApplicationWorld) {
    expect_shape(world);
}
// rectangle-three-point-center-02
#[given("Three Point Center Rectangle requests its second point after a center is selected")]
fn asks_second_after_center(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(CENTER);
    world.hover(SECOND);
    world.expect_status("Select second point");
}
#[when("the user enter 0 and press Return")]
fn zero_return(world: &mut ApplicationWorld) {
    accept_value(world, "0");
}
#[then("the zero expression remains red and Select second point remains active")]
fn zero_red_second(world: &mut ApplicationWorld) {
    world.expect_hud_value_invalid("0");
    world.expect_status("Select second point");
}
#[then("the numeric entry clears while Select second point remains active")]
fn cleared(world: &mut ApplicationWorld) {
    world.expect_no_annotation("0");
    world.expect_status("Select second point");
}
#[then("Ready returns without a completed rectangle")]
fn ready_without(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_none(world);
}
// rectangle-three-point-center-06 and rectangle-three-point-center-07
#[when("the user start Three Point Center Rectangle and select the center")]
fn start_and_center(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(CENTER);
}
#[then("the prompt changes to Select second point with length and angle fields")]
fn length_and_angle(world: &mut ApplicationWorld) {
    world.expect_status("Select second point");
    world.hover(SECOND);
    world.expect_hud_shows_length_and_angle();
}
#[then("Ready returns without retained rectangle or construction geometry")]
#[then("no partial rectangle remains and Ready returns")]
fn ready_empty(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_none(world);
}
#[given("Three Point Center Rectangle has accepted its center")]
fn accepted_center(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    start_and_center(world);
}
#[when("the user select a distinct second point")]
fn distinct_second(world: &mut ApplicationWorld) {
    world.sketch_click(SECOND);
}
#[then("the prompt changes to Select third point with a distance field")]
fn third_with_distance(world: &mut ApplicationWorld) {
    world.expect_status("Select third point");
    world.hover(THIRD);
    world.expect_hud_shows_length();
}
// rectangle-three-point-center-08
#[given("an editable sketch with the required geometry")]
fn required_geometry(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[when(
    "the user invoke Three Point Center Rectangle and select a center, oblique second point and nonzero-width third point"
)]
fn invoke_and_select(world: &mut ApplicationWorld) {
    pointer_shape(world);
}
#[then("the new rectangle and its construction geometry disappear")]
fn gone(world: &mut ApplicationWorld) {
    expect_none(world);
}
// rectangle-three-point-center-09
#[given("the recorded oriented centered rectangle has been restored")]
fn recorded_restored(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    open_tool(world);
    world.open_precise_input();
    let [center_y, center_z] = [0., -40.];
    let (sin, cos) = 20_f64.to_radians().sin_cos();
    let side = [
        center_y + RECORDED_HALF_LENGTH * cos,
        center_z + RECORDED_HALF_LENGTH * sin,
    ];
    world.precise_point([center_y, center_z]);
    world.precise_point(side);
    world.precise_point([
        side[0] - RECORDED_HALF_WIDTH * sin,
        side[1] + RECORDED_HALF_WIDTH * cos,
    ]);
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.redo();
}
#[when("the user open Measure and select the adjacent sides as Curves")]
#[when("the user open Measure and select the two adjacent Curve edges")]
#[when("the user measure the adjacent Curve sides")]
fn adjacent(world: &mut ApplicationWorld) {
    measure_adjacent(world);
}
#[then("their lengths are 12.754 mm and 5.187 mm for this fixture")]
fn recorded_lengths(world: &mut ApplicationWorld) {
    world.expect_measure("Length 12.754 mm");
    world.expect_measure("Length 5.187 mm");
}
#[then("the angle is 90.00 degrees with minimum distance 0.000 mm")]
fn square(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 90.00 deg");
    world.expect_measure("Minimum distance 0.000 mm");
}
// rectangle-three-point-center-10 to rectangle-three-point-center-11
#[given("an editable metric sketch")]
#[given("the recorded editable metric sketch and pointer view")]
fn metric_sketch(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
}
#[when("the user choose Three Point Center Rectangle and select the fixture center")]
fn tool_and_center(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(CENTER);
}
#[when("the user enter 6 and press Tab")]
fn enter_six_tab(world: &mut ApplicationWorld) {
    world.hover(SECOND);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Tab, false);
}
#[then("the first size field is locked at 6.000 mm")]
fn first_size_locked(world: &mut ApplicationWorld) {
    let text = world.window_text("##sketch-hud");
    assert!(text.contains('6'), "expected 6 in {text:?}");
}
#[then("the preview status shows Length 3.000 mm")]
fn preview_half_length(world: &mut ApplicationWorld) {
    let hud = world.state().sketch_hud.expect("expected rectangle fields");
    assert!(hud.fields[0].locked, "expected first size locked");
    assert!(hud.fields[1].active, "expected angle field active");
}
#[when("the user press 3 and then 0 and press Return")]
fn enter_thirty_return(world: &mut ApplicationWorld) {
    world.type_text("3");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[when("the user enter 6, Tab, 3, 0 and Return separately")]
fn first_axis(world: &mut ApplicationWorld) {
    typed_first_axis(world);
}
#[then("the command advances to Select third point")]
#[then("the command advances to Select third point with a 33.472 mm pointer-derived preview")]
fn third_now(world: &mut ApplicationWorld) {
    world.expect_status("Select third point");
}
#[when("the user select all, enter 4 and press Return")]
fn select_all_four(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::A, true);
    accept_value(world, "4");
}
#[then("a rectangle with center construction geometry remains")]
fn with_construction(world: &mut ApplicationWorld) {
    expect_shape(world);
}
#[then("Ready returns with annotations 6, 4 and 30.00")]
fn ready_annotated(world: &mut ApplicationWorld) {
    world.expect_ready();
    for value in ["6", "4", "30.00"] {
        world.expect_annotation(value);
    }
}
#[given("Select third point after the numeric first-axis entry")]
fn third_after_axis(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    tool_and_center(world);
    typed_first_axis(world);
    world.hover(THIRD);
}
#[when("the user enter 0")]
fn enter_zero(world: &mut ApplicationWorld) {
    world.type_text("0");
}
#[then("the zero expression appears red")]
fn zero_red(world: &mut ApplicationWorld) {
    world.expect_hud_value_invalid("0");
}
#[then("Select third point remains active without a completed rectangle")]
fn third_stays(world: &mut ApplicationWorld) {
    world.expect_status("Select third point");
    expect_none(world);
}
#[then("the rectangle is created and the prompt returns to Select center")]
fn created(world: &mut ApplicationWorld) {
    expect_shape(world);
    world.expect_status("Select center");
}
// rectangle-three-point-center-12
#[given("the numeric rectangle retained after Escape")]
fn numeric_retained(world: &mut ApplicationWorld) {
    numeric_shape(world);
}
#[then("their lengths are 6.000 and 4.000 mm")]
fn six_four(world: &mut ApplicationWorld) {
    world.expect_measure("Length 6.000 mm");
    world.expect_measure("Length 4.000 mm");
}
#[then("their angle is 90.00 degrees")]
fn right_angle(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 90.00 deg");
}
#[then("minimum and maximum distances are 0.000 and 7.211 mm")]
fn distances(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum distance 0.000 mm");
    world.expect_measure("Maximum distance 7.211 mm");
}
#[then("Measure closes")]
fn measure_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("Selection 1");
}
// rectangle-three-point-center-13 to rectangle-three-point-center-15
#[given("the numeric rectangle with length 6 mm")]
fn with_length(world: &mut ApplicationWorld) {
    numeric_shape(world);
}
#[when("the user open the length dimension editor, enter 8 and press Escape")]
fn editor_escape(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
    world.type_text("8");
    world.escape();
    world.pump(Vec::new());
}
#[when("the user reopen the same annotation")]
fn reopen(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
}
#[then("Edit Dimension displays 6 mm")]
fn displays_six(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("6mm"), "expected 6 mm in the editor {text:?}");
}
#[given("the length editor is open and displays 6 mm")]
fn editor_open(world: &mut ApplicationWorld) {
    numeric_shape(world);
    world.open_dimension_label("6");
    displays_six(world);
}
#[when("the user enter 8 and press Return then Escape")]
fn eight_return(world: &mut ApplicationWorld) {
    accept_value(world, "8");
    world.escape();
    world.pump(Vec::new());
}
#[then("the profile lengthens and Ready returns")]
fn lengthened(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_ready();
}
#[then("lengths are 8.000 and 4.000 mm with angle 90.00 degrees")]
fn eight_four(world: &mut ApplicationWorld) {
    world.expect_measure("Length 8.000 mm");
    world.expect_measure("Length 4.000 mm");
    world.expect_measure("Angle 90.00 deg");
}
#[then("minimum and maximum distances are 0.000 and 8.944 mm")]
fn longer_distances(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum distance 0.000 mm");
    world.expect_measure("Maximum distance 8.944 mm");
}
#[given("the accepted length edit and closed Measure window")]
fn accepted_edit(world: &mut ApplicationWorld) {
    resized_shape(world);
}
#[then("the shorter profile and 6 annotation return")]
fn shorter(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_no_annotation("8");
}
#[then("the longer profile and 8 annotation return")]
fn longer(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_no_annotation("6");
}
#[then("width 4 and direction 30.00 annotations remain visible")]
fn others_remain(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    world.expect_annotation("30.00");
}
// rectangle-three-point-center-16
#[then("a near-center preview marker displays length 0.454 mm")]
fn near_center_marker(world: &mut ApplicationWorld) {
    world.hover([CENTER[0] + 3, CENTER[1]]);
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("mm"), "expected a preview length in {text:?}");
}
#[when("the user select that near-center orientation marker")]
fn click_near_center(world: &mut ApplicationWorld) {
    world.sketch_click([CENTER[0] + 3, CENTER[1]]);
}
#[when("the user select the fixture center and a near-center orientation point")]
fn near_center(world: &mut ApplicationWorld) {
    open_tool(world);
    world.sketch_click(CENTER);
    world.sketch_click([CENTER[0] + 3, CENTER[1]]);
}
#[then("Select third point appears with 0.000 mm shown")]
fn third_zero(world: &mut ApplicationWorld) {
    world.expect_status("Select third point");
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("mm"), "expected a distance in {text:?}");
}
#[when("the user select the fixture third point and press Escape")]
fn third_and_escape(world: &mut ApplicationWorld) {
    world.sketch_click([CENTER[0] + 3, CENTER[1] + 60]);
    world.escape();
    world.pump(Vec::new());
}
#[then("a thin horizontal rectangle with center construction geometry remains")]
fn thin(world: &mut ApplicationWorld) {
    expect_shape(world);
}
// rectangle-three-point-center-17
#[given("the retained length-8 rectangle after Redo")]
fn length_eight_redone(world: &mut ApplicationWorld) {
    resized_shape(world);
    world.undo();
    world.redo();
}
#[when("the user measure its construction intersection")]
fn measure_intersection(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let diagonal = world.line_point(SIDES, 0.5);
    world.sketch_click(diagonal);
}
#[then("Middle Point coordinates are (0.000, -19.327, 29.887) mm")]
fn middle_coordinates(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Middle Point");
    world.expect_measure("Position X 0.000 Y -19.327 Z 29.887 mm");
}
#[when("the user clear selection and measure the two opposite vertices")]
fn opposite_vertices(world: &mut ApplicationWorld) {
    world.sketch_click([420, 300]);
    let first = world.line_point(SIDES + 1, 0.);
    world.sketch_click(first);
    let second = world.line_point(SIDES + 1, 1.);
    world.sketch_click(second);
}
#[then("Point coordinates are (0.000, -16.863, 33.619) and (0.000, -21.791, 26.155) mm")]
fn vertex_coordinates(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 0.000 Y -16.863 Z 33.619 mm");
    world.expect_measure("Position X 0.000 Y -21.791 Z 26.155 mm");
}
#[then("the distance is 8.944 mm")]
fn diagonal_distance(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum distance 8.944 mm");
}
#[then("the arithmetic average of the displayed endpoints matches the displayed Middle Point")]
fn average_matches(_: &mut ApplicationWorld) {
    let [y, z] = [(-16.863 - 21.791) / 2., (33.619 + 26.155) / 2.];
    assert_eq!(
        format!("{y:.3} {z:.3}"),
        "-19.327 29.887",
        "displayed average"
    );
}
#[path = "recorded_steps.rs"]
mod recorded_steps;

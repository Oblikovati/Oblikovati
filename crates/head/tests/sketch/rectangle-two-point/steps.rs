// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/rectangle-two-point`: pointer and typed rectangles and their dimensions.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const FIRST: [i32; 2] = [820, 520];
const OPPOSITE: [i32; 2] = [1040, 660];
const EMPTY: [i32; 2] = [420, 300];
const RECTANGLE_SIDES: usize = 4;
const POINTER: [i32; 2] = [1100, 700];
const ELLIPSE_CENTER: [i32; 2] = [915, 500];
const ELLIPSE_AXIS: [i32; 2] = [1015, 500];
const ELLIPSE_MINOR: [i32; 2] = [915, 440];

fn select_rectangle(world: &mut ApplicationWorld) {
    world.click_text("Rectangle");
    world.pump(Vec::new());
}
fn sides(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().lines().len()
}
fn type_sizes(world: &mut ApplicationWorld, first: &str, second: &str) {
    world.type_text(first);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(second);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn typed_rectangle(world: &mut ApplicationWorld, first: &str, second: &str) {
    world.start_sketch_on("YZ Plane");
    select_rectangle(world);
    world.sketch_click(FIRST);
    type_sizes(world, first, second);
    world.escape();
    world.pump(Vec::new());
}
/// A side's pick clear of its ends: the one running up the screen, or across it.
fn side_pixel(world: &mut ApplicationWorld, upright: bool) -> [i32; 2] {
    let count = sides(world);
    (0..count)
        .map(|index| {
            let from = world.line_point(index, 0.);
            let to = world.line_point(index, 1.);
            (index, from, to)
        })
        .find(|(_, from, to)| (from[0] == to[0]) == upright)
        .map(|(index, _, _)| world.line_point(index, 0.3))
        .expect("expected a side")
}
fn measure_side(world: &mut ApplicationWorld, upright: bool) {
    let at = side_pixel(world, upright);
    world.sketch_click(at);
}
// rectangle-two-point-01
#[given("an editable planar sketch")]
fn planar_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[when("I select Two Point Rectangle and click diagonally opposite corners")]
fn two_corners(world: &mut ApplicationWorld) {
    select_rectangle(world);
    world.sketch_click(FIRST);
    world.sketch_click(OPPOSITE);
    world.hover(POINTER);
}
fn expect_rectangle_drawn(world: &mut ApplicationWorld) {
    let corners = [
        FIRST,
        [OPPOSITE[0], FIRST[1]],
        OPPOSITE,
        [FIRST[0], OPPOSITE[1]],
    ];
    for index in 0..corners.len() {
        world.expect_segment_drawn(corners[index], corners[(index + 1) % corners.len()]);
    }
}
#[then("four connected sides form an axis-aligned closed rectangle")]
fn closed_rectangle(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
    expect_rectangle_drawn(world);
}
#[then("the rectangle remains and the sketch is Ready")]
fn remains_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
}
// rectangle-two-point-06
#[when("I choose Rectangle Two Point and click the fixture first corner")]
#[when("I choose Two Point Rectangle and the fixture first corner")]
#[when("I start Two Point Rectangle and click the fixture first corner")]
#[when("I start Two Point Rectangle and select a first corner")]
fn first_corner(world: &mut ApplicationWorld) {
    select_rectangle(world);
    world.sketch_click(FIRST);
}
#[then("the prompt requests the opposite corner")]
fn asks_opposite(world: &mut ApplicationWorld) {
    world.expect_status("Select opposite corner");
}
#[then("the MCAD Software returns to Ready without a new rectangle")]
fn ready_without(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), 0, "sides");
}
// rectangle-two-point-07
#[when("I invoke Two Point Rectangle")]
fn invoke_rectangle(world: &mut ApplicationWorld) {
    select_rectangle(world);
}
#[when("I click the same fixture position for both corners")]
fn same_corner(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.sketch_click(FIRST);
}
#[then("Select opposite corner remains active")]
fn still_asks_opposite(world: &mut ApplicationWorld) {
    world.expect_status("Select opposite corner");
}
#[then("both dynamic size fields display 0,000 mm")]
fn both_zero(world: &mut ApplicationWorld) {
    // The recording's decimal comma is its session's locale; these fields read the document's.
    let text = world.window_text("##sketch-hud");
    assert_eq!(text.matches("0.000mm").count(), 2, "fields {text:?}");
}
#[then("no error dialog or rectangle appears")]
fn nothing_appears(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
    assert_eq!(sides(world), 0, "sides");
}
// rectangle-two-point-08
#[given("an editable sketch in RIGHT view with the required fixture")]
#[given("the recorded fixture sketch is Ready")]
fn right_view_sketch(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
}
#[when("I press 1, 0, Tab, 6, and Enter separately")]
fn typed_ten_six(world: &mut ApplicationWorld) {
    world.type_text("1");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("I press 1 and then 0")]
fn type_ten(world: &mut ApplicationWorld) {
    world.type_text("1");
    world.type_text("0");
}
#[when("I press Tab")]
fn type_tab(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
}
#[when("I press 6 and then Enter")]
fn type_six_and_enter(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("a rectangle with size annotations 10 and 6 is committed")]
#[then("the rectangle and both annotations return")]
fn ten_six(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
    world.expect_annotation("10");
    world.expect_annotation("6");
}
#[then("the command returns to Select first corner")]
fn asks_first(world: &mut ApplicationWorld) {
    world.expect_status("Select first corner");
}
#[when("I press Escape and then Ctrl+Z")]
fn escape_and_undo(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.undo();
}
#[then("the rectangle and its two size annotations disappear")]
fn rectangle_gone(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), 0, "sides");
    world.expect_no_annotation("10");
    world.expect_no_annotation("6");
}
// rectangle-two-point-09
#[given("the recorded numeric rectangle in RIGHT view has been restored")]
fn restored_rectangle(world: &mut ApplicationWorld) {
    typed_rectangle(world, "10", "6");
    world.undo();
    world.redo();
}
#[when("I open Measure and select its vertical edge away from endpoints")]
fn measure_vertical(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    measure_side(world, true);
}
#[then("Selection 1 is a Curve with Length 10.000 mm")]
fn ten_long(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 10.000 mm");
}
#[when("I click empty canvas and select its horizontal edge away from endpoints")]
fn then_horizontal(world: &mut ApplicationWorld) {
    world.sketch_click(EMPTY);
    measure_side(world, false);
}
#[then("Selection 1 is a Curve with Length 6.000 mm")]
fn six_long(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 6.000 mm");
    let text = world.window_text("Measure");
    assert!(
        !text.contains("Selection2"),
        "expected one selection in {text:?}"
    );
}
#[then("Measure closes and the rectangle remains")]
fn measure_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Selection 1");
    assert!(
        sides(world) >= RECTANGLE_SIDES,
        "expected the rectangle's sides"
    );
}
// rectangle-two-point-10
#[given("the recorded fixture sketch is Ready with 22 dimensions needed")]
fn fixture_twenty_two(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.seed_dimensions_needed(22);
    world.observed_counts = vec![sides(world)];
    world.expect_ready();
    world.expect_dimensions_needed(22);
}
#[when("I enter 0 and press Tab")]
fn lock_zero_first(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Tab, false);
}
#[then("the first size is locked at 0.000 mm and the second field is focused")]
fn zero_locked(world: &mut ApplicationWorld) {
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("0.000mm"), "expected zero in {text:?}");
    assert!(
        world.text_is_selected("0.000mm"),
        "expected the second field selected"
    );
}
#[when("I enter 4 and press Return")]
fn complete_zero_first(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("the immediate status temporarily shows 24 dimensions needed")]
fn transient_twenty_four(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(24);
}
#[then("Ready returns with 22 dimensions needed and no new rectangle visible")]
fn fixture_unchanged(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(22);
    assert_eq!(sides(world), world.observed_counts[0]);
}
#[when("I enter 0, press Tab, enter 4 and press Return")]
fn zero_first(world: &mut ApplicationWorld) {
    type_sizes(world, "0", "4");
}
#[then("the command returns to Select first corner without a warning dialog")]
fn back_quietly(world: &mut ApplicationWorld) {
    world.expect_status("Select first corner");
    world.expect_no_text("Warning");
}
#[then("Ready returns and no new rectangle is visible")]
fn ready_none(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), 0, "sides");
}
// rectangle-two-point-11
#[given("a fresh Two Point Rectangle command in the recorded sketch view")]
fn fresh_command(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.choose_from_flyout("Circle", "Ellipse");
    for point in [ELLIPSE_CENTER, ELLIPSE_AXIS, ELLIPSE_MINOR] {
        world.sketch_click(point);
    }
    world.escape();
    select_rectangle(world);
}
#[given("the existing ellipse from the recorded fixture is visible")]
fn existing_ellipse(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().ellipses().len(), 1);
    world.expect_segment_drawn(ELLIPSE_AXIS, ELLIPSE_AXIS);
    world.expect_segment_drawn(ELLIPSE_MINOR, ELLIPSE_MINOR);
}
#[when("I select the fixture first corner, enter 6, press Tab, enter 4 and press Return")]
fn typed_six_four(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.hover([900, 440]);
    type_sizes(world, "6", "4");
}
#[then("the rectangle shows height 6 and width 4")]
fn six_four(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_annotation("4");
}
fn rectangle_bounds(world: &mut ApplicationWorld) -> [[i32; 2]; 2] {
    let points: Vec<_> = (0..sides(world))
        .flat_map(|line| [world.line_point(line, 0.), world.line_point(line, 1.)])
        .collect();
    let min = std::array::from_fn(|axis| points.iter().map(|p| p[axis]).min().unwrap());
    let max = std::array::from_fn(|axis| points.iter().map(|p| p[axis]).max().unwrap());
    [min, max]
}
#[then("the rectangle extends upward and rightward in the recorded view")]
fn extends_up_and_right(world: &mut ApplicationWorld) {
    let [min, max] = rectangle_bounds(world);
    assert!(min[1] < FIRST[1] && max[0] > FIRST[0]);
    assert!((min[0] - FIRST[0]).abs() <= 1 && (max[1] - FIRST[1]).abs() <= 1);
}
#[then("it overlaps part of the existing ellipse")]
fn overlaps_ellipse(world: &mut ApplicationWorld) {
    let [min, max] = rectangle_bounds(world);
    let nearest = std::array::from_fn(|axis| ELLIPSE_CENTER[axis].clamp(min[axis], max[axis]));
    let farthest = std::array::from_fn(|axis| {
        if (min[axis] - ELLIPSE_CENTER[axis]).abs() > (max[axis] - ELLIPSE_CENTER[axis]).abs() {
            min[axis]
        } else {
            max[axis]
        }
    });
    assert!(
        ellipse_distance(nearest) < 1. && ellipse_distance(farthest) > 1.,
        "rectangle {min:?}..{max:?} misses ellipse"
    );
}
fn ellipse_distance(point: [i32; 2]) -> f64 {
    let x = (point[0] - ELLIPSE_CENTER[0]) as f64 / (ELLIPSE_AXIS[0] - ELLIPSE_CENTER[0]) as f64;
    let y = (point[1] - ELLIPSE_CENTER[1]) as f64 / (ELLIPSE_CENTER[1] - ELLIPSE_MINOR[1]) as f64;
    x * x + y * y
}
#[then("the rectangle remains and Ready returns")]
fn remains(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
}
// rectangle-two-point-12
#[given("the rectangle height editor is open with 6 mm selected")]
fn editor_open(world: &mut ApplicationWorld) {
    typed_rectangle(world, "6", "4");
    world.open_dimension_label("6");
    assert!(world.text_is_selected("6mm"), "expected the value selected");
}
#[when("I enter 8 and press Escape")]
fn enter_and_escape(world: &mut ApplicationWorld) {
    world.type_text("8");
    world.escape();
    world.pump(Vec::new());
}
#[when("I reopen the height annotation")]
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
    typed_rectangle(world, "6", "4");
    six_four(world);
}
#[when("I accept height 8 mm and exit General Dimension")]
fn accept_eight(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
#[when("I open Measure and select the left edge")]
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
fn close_and_undo(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
    world.undo();
}
#[then("the visible height annotation returns to 6")]
fn height_six(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
}
#[when("I redo the height edit")]
fn redo_height(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the visible height annotation returns to 8")]
fn height_eight(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
}
#[then("width annotation 4 remains")]
#[then("the width annotation remains 4")]
fn width_four(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
}
// rectangle-two-point-14
#[given("the recorded sketch is Ready with the 4-by-8 rectangle retained")]
fn retained_four_eight(world: &mut ApplicationWorld) {
    typed_rectangle(world, "4", "8");
}
#[when("I enter 4, press Tab, enter 0 and press Return")]
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

#[test]
fn native_dimension_editor_retains_arithmetic_through_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    typed_rectangle(&mut world, "10", "6");
    world.open_dimension_label("10");
    world.type_text("2*6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_annotation("12");
    world.open_dimension_label("12");
    assert!(world.window_text("Edit Dimension").contains("2*6"));
    world.escape();
    world.undo();
    world.expect_annotation("10");
    world.redo();
    world.expect_annotation("12");
    world.open_dimension_label("12");
    assert!(world.window_text("Edit Dimension").contains("2*6"));
    world.close();
}

#[test]
fn native_dimension_editor_keeps_rejected_input_for_correction() {
    let mut world = ApplicationWorld::new();
    typed_rectangle(&mut world, "10", "6");
    world.open_dimension_label("10");
    world.type_text("2*");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert!(world.window_text("Edit Dimension").contains("2*"));
    world.expect_annotation("10");
    assert!(world.status_shows("Command failed"));
    world.type_text("2*6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_annotation("12");
    world.close();
}

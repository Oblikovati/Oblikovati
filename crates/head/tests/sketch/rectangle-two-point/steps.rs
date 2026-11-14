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
pub(super) fn typed_rectangle(world: &mut ApplicationWorld, first: &str, second: &str) {
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        world.start_sketch_on("YZ Plane");
    }
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
#[when("the user select Two Point Rectangle and click diagonally opposite corners")]
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
    let geometry = world.edited_geometry();
    assert!(
        topology::is_axis_aligned_closed(&geometry),
        "expected one closed axis-aligned four-sided loop"
    );
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
#[when("the user choose Two Point Rectangle and the fixture first corner")]
#[when("I start Two Point Rectangle and click the fixture first corner")]
#[when("the user start Two Point Rectangle and click the fixture first corner")]
#[when("I start Two Point Rectangle and select a first corner")]
#[when("the user choose Rectangle Two Point and click the fixture first corner")]
#[when("the user start Two Point Rectangle and select a first corner")]
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
#[when("the user invoke Two Point Rectangle")]
fn invoke_rectangle(world: &mut ApplicationWorld) {
    select_rectangle(world);
}
#[when("I click the same fixture position for both corners")]
#[when("the user click the same fixture position for both corners")]
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
#[when("the user press 1, 0, Tab, 6, and Enter separately")]
fn typed_ten_six(world: &mut ApplicationWorld) {
    world.type_text("1");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("I press 1 and then 0")]
#[when("the user press 1 and then 0")]
fn type_ten(world: &mut ApplicationWorld) {
    world.type_text("1");
    world.type_text("0");
}
#[when("I press Tab")]
#[when("the user press Tab")]
fn type_tab(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
}
#[when("I press 6 and then Enter")]
#[when("the user press 6 and then Enter")]
fn type_six_and_enter(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("a rectangle with size annotations 10 and 6 is committed")]
#[then("the rectangle and both annotations return")]
fn ten_six(world: &mut ApplicationWorld) {
    assert_eq!(sides(world), RECTANGLE_SIDES, "sides");
    assert_typed_rectangle_lengths(&world.edited_geometry(), [0.6, 1.0]);
    world.expect_annotation("10");
    world.expect_annotation("6");
}
fn assert_typed_rectangle_lengths(geometry: &oblikovati_model::sketch::Sketch2D, sizes: [f64; 2]) {
    assert!(topology::is_axis_aligned_closed(geometry));
    assert_rectangle_edge_lengths(geometry, sizes);
}
fn assert_rectangle_edge_lengths(geometry: &oblikovati_model::sketch::Sketch2D, sizes: [f64; 2]) {
    let mut lengths: Vec<f64> = geometry
        .lines()
        .iter()
        .map(|line| {
            let [a, b] = line
                .endpoints
                .map(|id| geometry.point(id).unwrap().position);
            (a[0] - b[0]).hypot(a[1] - b[1])
        })
        .collect();
    lengths.sort_by(f64::total_cmp);
    assert_eq!(
        lengths.len(),
        RECTANGLE_SIDES,
        "expected four rectangle edges"
    );
    // The default sketch working unit is one centimetre.
    for (actual, expected) in lengths
        .into_iter()
        .zip([sizes[0], sizes[0], sizes[1], sizes[1]])
    {
        assert!(
            (actual - expected).abs() < 1e-6,
            "edge length {actual}: expected {expected} cm"
        );
    }
}
#[then("the command returns to Select first corner")]
fn asks_first(world: &mut ApplicationWorld) {
    world.expect_status("Select first corner");
}
#[when("I press Escape and then Ctrl+Z")]
#[when("the user press Escape and then Ctrl+Z")]
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
#[when("the user open Measure and select its vertical edge away from endpoints")]
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
#[when("the user click empty canvas and select its horizontal edge away from endpoints")]
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
#[when("the user enter 0 and press Tab")]
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
#[when("the user enter 4 and press Return")]
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
#[when("the user enter 0, press Tab, enter 4 and press Return")]
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
#[when("the user select the fixture first corner, enter 6, press Tab, enter 4 and press Return")]
fn typed_six_four(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.hover([900, 440]);
    type_sizes(world, "6", "4");
}
#[then("the rectangle shows height 6 and width 4")]
fn six_four(world: &mut ApplicationWorld) {
    assert_typed_rectangle_lengths(&world.edited_geometry(), [0.4, 0.6]);
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
#[path = "dimension_edits.rs"]
mod dimension_edits;
#[cfg(test)]
#[path = "dimension_tests.rs"]
mod dimension_tests;
#[path = "topology.rs"]
mod topology;

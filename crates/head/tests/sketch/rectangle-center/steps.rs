// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/rectangle-center`: a rectangle from its center, with construction diagonals.
use super::super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
#[path = "shape.rs"]
mod shape;

const TOOL: &str = "Center Rect.";
const SIDES: usize = 4;
const DIAGONALS: usize = 2;
const CENTER: [i32; 2] = [860, 560];
const CORNER: [i32; 2] = [1000, 660];
/// Y then Z, in millimetres, on the YZ plane: where the recording measured its center.
const RECORDED_CENTER: [f64; 2] = [-17.513, 4.675];

fn select_tool(world: &mut ApplicationWorld) {
    world.click_text(TOOL);
    world.pump(Vec::new());
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
    shape::expect_centered_rectangle(&world.edited_geometry());
}
fn expect_no_shape(world: &mut ApplicationWorld) {
    assert_eq!(line_counts(world), (0, 0), "sides and diagonals");
}
fn type_sizes(world: &mut ApplicationWorld, first: &str, second: &str) {
    world.type_text(first);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(second);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
/// The recorded rectangle: an exact center, then typed sizes, with the command left.
fn recorded_rectangle(world: &mut ApplicationWorld, first: &str, second: &str) {
    world.start_sketch_on("YZ Plane");
    select_tool(world);
    world.open_precise_input();
    world.precise_point(RECORDED_CENTER);
    type_sizes(world, first, second);
    world.escape();
    world.pump(Vec::new());
}
fn pointer_rectangle(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(CENTER);
    world.sketch_click(CORNER);
}
/// The sides' shared corners and the center, from the sketch rather than from a recording.
fn center_position(world: &mut ApplicationWorld) -> [f64; 2] {
    let geometry = world.edited_geometry();
    let index = geometry.read_index();
    let diagonal = geometry
        .lines()
        .iter()
        .find(|line| index.is_construction(line.id).unwrap_or(false))
        .copied()
        .expect("expected a diagonal");
    let [from, to] = diagonal
        .endpoints
        .map(|point| geometry.point(point).unwrap().position);
    [(from[0] + to[0]) * 0.5, (from[1] + to[1]) * 0.5]
}
fn measure_position(world: &mut ApplicationWorld, at: [f64; 2]) -> String {
    world.key(ViewportPhysicalKey::M, false);
    let pixel = world.sketch_pixel_of(at);
    world.sketch_click(pixel);
    let rows = world.window_text("Measure");
    world.click_text("Done");
    world.pump(Vec::new());
    rows
}
// rectangle-center-01
#[when("I select Two Point Center Rectangle, choose a center, then a corner outside the input HUD")]
#[when(
    "the user select Two Point Center Rectangle, choose a center, then a corner outside the input HUD"
)]
fn center_and_corner(world: &mut ApplicationWorld) {
    pointer_rectangle(world);
}
#[then(
    "the rectangle is centered on the first point and includes the observed diagonal construction geometry"
)]
fn centered(world: &mut ApplicationWorld) {
    expect_shape(world);
    let center = center_position(world);
    assert_eq!(world.sketch_pixel_of(center), CENTER, "center pixel");
}
#[then("Ready returns with the rectangle and construction diagonals retained")]
fn ready_retained(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_shape(world);
}
// rectangle-center-02
#[given("Two Point Center Rectangle is active in an editable sketch")]
fn tool_active(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    select_tool(world);
}
#[given("a center has been selected and the command requests a corner")]
#[given("the command has accepted the center at a chosen location")]
fn center_selected(world: &mut ApplicationWorld) {
    if world.state().dimensions_needed.is_none() {
        tool_active(world);
    }
    world.sketch_click(CENTER);
    world.expect_status("Select corner");
}
#[when("I click a dynamic input field")]
#[when("the user click a dynamic input field")]
fn click_field(world: &mut ApplicationWorld) {
    world.hover(CORNER);
    let (position, size) = diagnostics::window_geometry("##sketch-hud").expect("expected fields");
    world.click([
        (position[0] + size[0] * 0.25) as i32,
        (position[1] + size[1] * 0.5) as i32,
    ]);
}
#[then("no rectangle is committed and corner selection remains active")]
fn nothing_committed(world: &mut ApplicationWorld) {
    expect_no_shape(world);
    world.expect_status("Select corner");
}
#[when("I click a distinct corner outside the input fields")]
#[when("I click a distinct corner outside the dynamic input HUD")]
#[when("the user click a distinct corner outside the input fields")]
#[when("the user click a distinct corner outside the dynamic input HUD")]
fn distinct_corner(world: &mut ApplicationWorld) {
    world.sketch_click([CORNER[0], CORNER[1] + 90]);
}
#[then("a rectangle with construction diagonals is committed")]
#[then("the rectangle and diagonal construction geometry are committed")]
fn shape_committed(world: &mut ApplicationWorld) {
    expect_shape(world);
}
#[then("the command requests another center")]
#[then("the prompt returns to Select center")]
#[then("the command returns to Select center")]
fn asks_center(world: &mut ApplicationWorld) {
    world.expect_status("Select center");
}
// rectangle-center-06
#[when("I select Two Point Center Rectangle and click its center")]
#[when("the user select Two Point Center Rectangle and click its center")]
fn tool_and_center(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(CENTER);
}
#[then("the prompt changes to Select corner and two size fields appear")]
fn two_fields(world: &mut ApplicationWorld) {
    world.expect_status("Select corner");
    let text = world.window_text("##sketch-hud");
    assert_eq!(text.matches("mm").count(), 2, "fields {text:?}");
}
#[then("Ready returns without a new rectangle or construction diagonals")]
fn ready_empty(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_no_shape(world);
}
// rectangle-center-07
#[when("I click the identical location as the corner")]
#[when("the user click the identical location as the corner")]
fn identical(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("Select corner remains and both size fields show zero")]
fn both_zero(world: &mut ApplicationWorld) {
    world.expect_status("Select corner");
    let text = world.window_text("##sketch-hud");
    assert_eq!(text.matches("0.000mm").count(), 2, "fields {text:?}");
}
#[then("no error dialog appears")]
fn no_dialog(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
}
// rectangle-center-08
#[given("a mouse-created centered rectangle")]
fn mouse_rectangle(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    pointer_rectangle(world);
    expect_shape(world);
}
#[then("its sides and construction diagonals disappear")]
fn shape_gone(world: &mut ApplicationWorld) {
    expect_no_shape(world);
}
#[then("the sides and construction diagonals return")]
fn shape_back(world: &mut ApplicationWorld) {
    expect_shape(world);
}
// rectangle-center-09
#[given("Ready with a mouse-created rectangle")]
fn ready_with_rectangle(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    select_tool(world);
    world.sketch_click([600, 700]);
    world.sketch_click([520, 760]);
    world.escape();
    world.pump(Vec::new());
}
#[when("I start Two Point Center Rectangle and select center at a chosen location")]
#[when("the user start Two Point Center Rectangle and select center at a chosen location")]
#[when("I choose Two Point Center Rectangle and select the fixture center")]
#[when("I start a fresh Two Point Center Rectangle and select the fixture center")]
#[when("I start Two Point Center Rectangle and select the fixture center")]
#[when("the user start Two Point Center Rectangle and select the fixture center")]
fn start_and_center(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(CENTER);
}
#[when("I press 8, Tab, 4, and Enter separately")]
#[when("the user press 8, Tab, 4, and Enter separately")]
fn typed_eight_four(world: &mut ApplicationWorld) {
    type_sizes(world, "8", "4");
}
#[then("a rectangle with size annotations 8 and 4 is committed")]
fn eight_four(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_annotation("4");
    assert_eq!(
        line_counts(world),
        (2 * SIDES, 2 * DIAGONALS),
        "two rectangles"
    );
    shape::expect_latest_sizes(&world.edited_geometry(), 0.4, 0.8);
}
#[then("Ready returns and the committed rectangle remains")]
fn ready_committed(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(
        line_counts(world),
        (2 * SIDES, 2 * DIAGONALS),
        "two rectangles"
    );
}
// rectangle-center-10
#[given("the numeric centered rectangle")]
fn numeric_rectangle(world: &mut ApplicationWorld) {
    recorded_rectangle(world, "8", "4");
    world.remember_edited_sketch(Vec::new());
}
#[when("I open Measure and select its adjacent vertical and horizontal sides as Curves")]
#[when("the user open Measure and select its adjacent vertical and horizontal sides as Curves")]
fn select_adjacent(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let upright = world.line_point(0, 0.3);
    world.sketch_click(upright);
    let across = world.line_point(1, 0.3);
    world.sketch_click(across);
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Selection 2: Curve");
}
#[then("their lengths are 8.000 mm and 4.000 mm")]
fn lengths(world: &mut ApplicationWorld) {
    world.expect_measure("Length 8.000 mm");
    world.expect_measure("Length 4.000 mm");
}
#[then("the included angle is 90.00 degrees and minimum distance is 0.000 mm")]
fn angle_and_distance(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 90.00 deg");
    world.expect_measure("Minimum distance 0.000 mm");
}
#[then("Measure closes without changing the rectangle")]
fn measure_closed(world: &mut ApplicationWorld) {
    assert!(
        diagnostics::window_geometry("Measure").is_none(),
        "expected Measure to close"
    );
    expect_shape(world);
    world.expect_edited_sketch_retained();
}
// rectangle-center-11 to rectangle-center-13
#[then("the command requests Select center without a warning or visible new rectangle")]
fn asks_center_quietly(world: &mut ApplicationWorld) {
    world.expect_status("Select center");
    world.expect_no_text("Warning");
    let expected = world.observed_counts.first().copied().unwrap_or(0);
    assert_eq!(line_counts(world).0, expected, "sides");
}
#[then("Ready returns with no visible new sides or diagonals")]
fn ready_no_new(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_no_shape(world);
}
#[given("the previous sketch geometry is retained at Ready")]
fn previous_geometry(world: &mut ApplicationWorld) {
    ready_with_rectangle(world);
    world.observed_counts = vec![SIDES];
    let witnesses = (0..SIDES)
        .map(|index| world.line_point(index, 0.3))
        .collect();
    world.remember_edited_sketch(witnesses);
}
#[then("Ready returns with the previous geometry retained")]
fn previous_retained(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_edited_sketch_retained();
}
#[when("I enter 6, press Tab, enter 4 and press Return")]
#[when("the user enter 6, press Tab, enter 4 and press Return")]
fn six_four_typed(world: &mut ApplicationWorld) {
    type_sizes(world, "6", "4");
}
#[then("the rectangle shows vertical 6 and horizontal 4 with construction diagonals")]
fn six_four(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_annotation("4");
    expect_shape(world);
    shape::expect_axis_sizes(&world.edited_geometry(), [0.6, 0.4]);
}
#[then("Ready returns")]
fn ready_returns(world: &mut ApplicationWorld) {
    world.expect_ready();
}
// rectangle-center-14
#[given("the recorded 6-by-4 centered rectangle is retained")]
#[given("Measure is closed and the numeric rectangle is retained")]
fn recorded_six_four(world: &mut ApplicationWorld) {
    recorded_rectangle(world, "6", "4");
}
#[when("I measure the center and both opposite corners")]
fn measure_three(world: &mut ApplicationWorld) {
    let in_cm = |value: [f64; 2]| value.map(|part| part / 10.);
    let center = in_cm(RECORDED_CENTER);
    let high = in_cm([RECORDED_CENTER[0] + 3., RECORDED_CENTER[1] + 2.]);
    let low = in_cm([RECORDED_CENTER[0] - 3., RECORDED_CENTER[1] - 2.]);
    let mut rows = measure_position(world, center);
    rows.push_str(&measure_position(world, high));
    rows.push_str(&measure_position(world, low));
    world.key(ViewportPhysicalKey::M, false);
    for corner in [high, low] {
        let pixel = world.sketch_pixel_of(corner);
        world.sketch_click(pixel);
    }
    rows.push_str(&world.window_text("Measure"));
    world.observed_rows = Some(rows);
}
#[then("their Point coordinates are (0,-17.513,4.675), (0,-14.513,6.675) and (0,-20.513,2.675) mm")]
fn point_rows(world: &mut ApplicationWorld) {
    let rows = world.observed_rows.clone().expect("expected measured rows");
    for row in [
        "PositionX0.000Y-17.513Z4.675mm",
        "PositionX0.000Y-14.513Z6.675mm",
        "PositionX0.000Y-20.513Z2.675mm",
    ] {
        assert!(rows.contains(row), "expected {row} in {rows:?}");
    }
}
#[then("the corner distance is 7.211 mm with displayed component differences 0, 6 and 4 mm")]
fn corner_distance(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum distance 7.211 mm");
    world.expect_measure("Minimum delta X 0.000 Y 6.000 Z 4.000 mm");
}
#[then("the opposite-corner arithmetic midpoint equals the measured center at displayed precision")]
fn midpoint_is_center(world: &mut ApplicationWorld) {
    let center = center_position(world).map(|value| (value * 10.).abs());
    assert!(
        (center[0] - RECORDED_CENTER[0].abs()).abs() < 5e-4,
        "center {center:?}"
    );
    assert!(
        (center[1] - RECORDED_CENTER[1].abs()).abs() < 5e-4,
        "center {center:?}"
    );
}
// rectangle-center-15
#[then("its sides, construction diagonals and size dimensions disappear")]
fn numeric_gone(world: &mut ApplicationWorld) {
    expect_no_shape(world);
    world.expect_no_annotation("6");
    world.expect_no_annotation("4");
}
#[then("the rectangle and its annotations return")]
fn numeric_back(world: &mut ApplicationWorld) {
    expect_shape(world);
    world.expect_annotation("6");
    world.expect_annotation("4");
}
// rectangle-center-16
#[given("the retained centered rectangle has a driving height dimension equal to 6 mm")]
fn retained_height(world: &mut ApplicationWorld) {
    recorded_rectangle(world, "6", "4");
}
#[when("I open its height annotation, enter 8 and press Escape")]
#[when("the user open its height annotation, enter 8 and press Escape")]
fn height_and_escape(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
    world.type_text("8");
    world.escape();
    world.pump(Vec::new());
}
// rectangle-center-17
#[given("the height editor is open with 6 mm selected")]
fn height_editor(world: &mut ApplicationWorld) {
    recorded_rectangle(world, "6", "4");
    let center = center_position(world);
    world.observed_view = Some(world.sketch_pixel_of(center));
    world.open_dimension_label("6");
    assert!(world.text_is_selected("6mm"), "expected the value selected");
}
#[given("the earlier measured center is (0,-17.513,4.675) mm")]
fn earlier_center(world: &mut ApplicationWorld) {
    let center = center_position(world);
    let shown = format!("{:.3} {:.3}", center[0] * 10., center[1] * 10.);
    assert_eq!(shown, "-17.513 4.675", "center");
}
#[when("I enter 8, press Return and exit General Dimension")]
fn accept_eight(world: &mut ApplicationWorld) {
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
#[then("the top edge and diagonal intersection move upward visibly")]
fn moved_up(world: &mut ApplicationWorld) {
    let before = world.observed_view.expect("expected the earlier center");
    let center = center_position(world);
    let now = world.sketch_pixel_of(center);
    assert!(now[1] < before[1], "center pixel {now:?} from {before:?}");
}
#[when("I open Measure and select the center point")]
fn measure_center(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let center = center_position(world);
    let at = world.sketch_pixel_of(center);
    world.sketch_click(at);
}
#[then("Measure reports center Point (0,-16.513,4.675) mm")]
fn moved_center(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
    world.expect_measure("Position X 0.000 Y -16.513 Z 4.675 mm");
}
#[when("I clear the selection and select the vertical edge")]
fn select_vertical(world: &mut ApplicationWorld) {
    world.sketch_click([420, 300]);
    let at = world.line_point(0, 0.3);
    world.sketch_click(at);
}
#[then("the selected vertical Curve length is 8.000 mm")]
fn vertical_eight(world: &mut ApplicationWorld) {
    world.expect_measure("Length 8.000 mm");
}
#[then("width annotation 4 and construction diagonals remain visible")]
fn width_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    expect_shape(world);
}
// rectangle-center-18
#[given("Measure is closed after accepting height 8")]
fn accepted_height(world: &mut ApplicationWorld) {
    recorded_rectangle(world, "6", "4");
    world.open_dimension_label("6");
    accept_eight(world);
}
#[then("height annotation 6 and the shorter shape return")]
fn shorter(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_no_annotation("8");
}
#[then("height annotation 8 and the taller shape return")]
fn taller(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_no_annotation("6");
}
#[then("the center-marker location changes with the shape while width annotation 4 remains")]
fn marker_moves(world: &mut ApplicationWorld) {
    let center = center_position(world);
    let shown = format!("{:.3}", center[0] * 10.);
    assert_eq!(shown, "-16.513", "center along the height");
    world.expect_annotation("4");
}

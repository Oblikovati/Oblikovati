// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/arc-center-point`: arcs from a center, typed radius and sweep, and their edits.
use super::super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchArc2D;
use oblikovati_viewport::ViewportPhysicalKey;
#[path = "start_angle.rs"]
mod start_angle;

const TOOL: &str = "Center Arc";
const CENTER: [i32; 2] = [860, 560];
const START: [i32; 2] = [960, 560];
const BELOW: [i32; 2] = [860, 660];
const ABOVE: [i32; 2] = [860, 460];
const EMPTY: [i32; 2] = [420, 300];
/// On the arc from CENTER through START and BELOW: a quarter turn, its middle at 45 degrees.
const ON_THE_SHORT_ARC: [i32; 2] = [931, 631];
/// Recorded center of the pointer-created arc, in millimetres along model Y and Z.
const RECORDED_POINTER_CENTER: [f64; 2] = [-7.023, -83.568];
const RECORDED_RADIUS: f64 = 4.759;
const RECORDED_LENGTH: f64 = 7.037;
const RECORDED_NUMERIC_CENTER: [f64; 2] = [4.914, 17.623];

fn select_tool(world: &mut ApplicationWorld) {
    world.click_text(TOOL);
    world.pump(Vec::new());
}
fn arc_count(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().arcs().len()
}
fn expect_no_arc(world: &mut ApplicationWorld) {
    assert_eq!(arc_count(world), 0, "arcs");
}
fn newest_arc(world: &mut ApplicationWorld) -> SketchArc2D {
    *world
        .edited_geometry()
        .arcs()
        .last()
        .expect("expected an arc")
}
/// A point on the newest arc this fraction along its turn, away from where a pick reads a point.
pub(super) fn arc_fraction_pixel(world: &mut ApplicationWorld, fraction: f64) -> [i32; 2] {
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    let position = |point| geometry.point(point).unwrap().position;
    let (center, start, end) = (position(arc.center), position(arc.start), position(arc.end));
    let from = (start[1] - center[1]).atan2(start[0] - center[0]);
    let to = (end[1] - center[1]).atan2(end[0] - center[0]);
    let turn = if arc.counter_clockwise {
        (to - from).rem_euclid(std::f64::consts::TAU)
    } else {
        -(from - to).rem_euclid(std::f64::consts::TAU)
    };
    let radius = (start[0] - center[0]).hypot(start[1] - center[1]);
    let angle = from + turn * fraction;
    world.sketch_pixel_of([
        center[0] + radius * angle.cos(),
        center[1] + radius * angle.sin(),
    ])
}
fn measure_arc(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = arc_fraction_pixel(world, 0.25);
    world.sketch_click(at);
}
fn close_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
}
/// Radius and sweep typed, the rest by pointer, as the recording's numeric arc was made.
fn numeric_arc(world: &mut ApplicationWorld, radius: &str, sweep: &str) {
    world.type_text(radius);
    world.key(ViewportPhysicalKey::Tab, false);
    world.sketch_click(START);
    world.type_text(sweep);
    world.key(ViewportPhysicalKey::Tab, false);
    world.sketch_click(ABOVE);
}
fn created_numeric_arc(world: &mut ApplicationWorld, radius: &str) {
    world.start_sketch_on("YZ Plane");
    select_tool(world);
    world.open_precise_input();
    world.precise_point(RECORDED_NUMERIC_CENTER);
    numeric_arc(world, radius, "90");
    world.escape();
    world.pump(Vec::new());
}
fn accept_value(world: &mut ApplicationWorld, value: &str) {
    world.type_text(value);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
// arc-center-point-01
#[when("I select Center Point Arc, choose its center, start point, and end direction")]
fn pointer_arc(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(CENTER);
    world.sketch_click(START);
    world.sketch_click(BELOW);
}
#[then("an arc is created from the selected center, start and end direction")]
fn arc_created(world: &mut ApplicationWorld) {
    assert_eq!(arc_count(world), 1, "arcs");
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    let center = geometry.point(arc.center).unwrap().position;
    assert_eq!(world.sketch_pixel_of(center), CENTER, "center pixel");
}
#[then("the arc remains and the sketch returns to Ready")]
fn arc_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(arc_count(world), 1, "arcs");
}
// arc-center-point-06 and arc-center-point-07
#[given("the observed sketch fixture is Ready")]
fn fixture_ready(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.expect_ready();
}
#[when("I choose Center Point Arc and select a center")]
fn tool_and_center(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(CENTER);
}
#[then("the prompt requests the start point")]
fn asks_start(world: &mut ApplicationWorld) {
    world.expect_status("Select start point");
}
#[then("the MCAD Software returns to Ready without a new arc")]
fn ready_without(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_no_arc(world);
}
#[when("I invoke Center Point Arc")]
fn invoke(world: &mut ApplicationWorld) {
    select_tool(world);
}
#[when("I select a center and a start point")]
fn center_and_start(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
    world.sketch_click(START);
}
#[then("the command requests the end point")]
#[then("the prompt requests the end point")]
fn asks_end(world: &mut ApplicationWorld) {
    world.expect_status("Select end point");
}
// arc-center-point-08
#[when("I select the recorded center, a start to its right, and an end direction below it")]
fn center_right_below(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
    world.sketch_click(START);
    world.sketch_click(BELOW);
    world.hover([1200, 760]);
}
#[then("the short arc appears below and right of the center")]
fn short_arc(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(ON_THE_SHORT_ARC, ON_THE_SHORT_ARC);
}
#[then("the arc remains")]
#[then("the arc returns")]
fn arc_there(world: &mut ApplicationWorld) {
    assert_eq!(arc_count(world), 1, "arcs");
    world.expect_segment_drawn(ON_THE_SHORT_ARC, ON_THE_SHORT_ARC);
}
#[then("the arc disappears")]
fn arc_gone(world: &mut ApplicationWorld) {
    expect_no_arc(world);
    world.expect_segment_absent(ON_THE_SHORT_ARC, ON_THE_SHORT_ARC);
}
// arc-center-point-09
#[given("the recorded pointer-created center-point arc has been restored by Redo")]
fn recorded_pointer_arc(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    select_tool(world);
    world.open_precise_input();
    let turn = RECORDED_LENGTH / RECORDED_RADIUS;
    let [cy, cz] = RECORDED_POINTER_CENTER;
    world.precise_point([cy, cz]);
    world.precise_point([cy + RECORDED_RADIUS, cz]);
    world.precise_point([
        cy + RECORDED_RADIUS * turn.cos(),
        cz + RECORDED_RADIUS * turn.sin(),
    ]);
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.redo();
}
#[when("I open Measure and select its middle point")]
fn select_middle(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = arc_fraction_pixel(world, 0.5);
    world.sketch_click(at);
}
#[then("the selection type is Middle Point")]
fn is_middle(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Middle Point");
}
#[when("I clear the selection and select the curve near its start")]
fn clear_and_select(world: &mut ApplicationWorld) {
    world.sketch_click(EMPTY);
    let at = arc_fraction_pixel(world, 0.25);
    world.sketch_click(at);
}
#[then("the selection type is Curve")]
fn is_curve(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
}
#[then("the radius is 4.759 mm and diameter is 9.518 mm")]
fn recorded_radius(world: &mut ApplicationWorld) {
    world.expect_measure("Radius 4.759 mm");
    world.expect_measure("Diameter 9.518 mm");
}
// The captured length and radius imply 84.72 degrees, even at their rounding limits.
// Keep the recorded 81.72-degree assertion until the source observation is reviewed.
#[then("the length is 7.037 mm and sweep is 81.72 degrees")]
fn recorded_length(world: &mut ApplicationWorld) {
    world.expect_measure("Length 7.037 mm");
    world.expect_measure("Sweep 81.72 deg");
}
#[then("its center is X 0.000 mm, Y -7.023 mm, Z -83.568 mm")]
fn recorded_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y -7.023 Z -83.568 mm");
}
// arc-center-point-10
#[given("Center Point Arc requests its start point after center selection")]
fn asks_start_after_center(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    select_tool(world);
    world.sketch_click(CENTER);
    world.expect_status("Select start point");
}
#[when("I click the same requested center position")]
fn same_center(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("the prompt remains Select start point with radius zero")]
fn start_with_zero(world: &mut ApplicationWorld) {
    world.expect_status("Select start point");
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("0.000mm"), "expected radius zero in {text:?}");
}
#[when("I press 4 in this captured state")]
fn press_four(world: &mut ApplicationWorld) {
    world.type_text("4");
}
#[then("command alias input appears instead of an edited radius")]
fn alias_appears(world: &mut ApplicationWorld) {
    let alias = world.window_text("##command-alias");
    assert!(alias.contains('4'), "expected the typed key in {alias:?}");
    let fields = world.window_text("##sketch-hud");
    assert!(
        fields.contains("0.000mm"),
        "expected the radius unedited in {fields:?}"
    );
}
#[then("the command exits without committing a new arc")]
fn exits_without(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_no_arc(world);
    assert!(
        diagnostics::window_geometry("##command-alias").is_none(),
        "expected the command box to close"
    );
}
// arc-center-point-11
#[given("a fresh Center Point Arc command with a selected center")]
fn fresh_with_center(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    select_tool(world);
    world.open_precise_input();
    world.precise_point(RECORDED_NUMERIC_CENTER);
    world.hover(START);
}
#[when("I enter 4 for radius")]
fn type_radius(world: &mut ApplicationWorld) {
    world.type_text("4");
}
#[then("the radius is locked at 4 mm and focus moves to the start-angle field")]
fn radius_locked_and_angle_active(world: &mut ApplicationWorld) {
    let hud = world.state().sketch_hud.expect("expected arc fields");
    assert_eq!(hud.fields.len(), 2);
    assert!(hud.fields[0].locked && hud.fields[0].text.contains("4.000"));
    assert!(hud.fields[1].active && hud.fields[1].text.contains("deg"));
}
#[when("I click the recorded start location")]
fn click_recorded_start(world: &mut ApplicationWorld) {
    world.sketch_click(START);
}
#[when("I press 9 and then 0 for the sweep")]
fn type_sweep(world: &mut ApplicationWorld) {
    world.type_text("9");
    world.type_text("0");
}
#[then("the sweep is locked at 90 degrees")]
fn sweep_locked(world: &mut ApplicationWorld) {
    let hud = world.state().sketch_hud.expect("expected sweep field");
    assert_eq!(hud.fields.len(), 1);
    assert!(hud.fields[0].locked && hud.fields[0].text.contains("90.00"));
}
#[when("I click the recorded endpoint location")]
fn click_recorded_end(world: &mut ApplicationWorld) {
    world.sketch_click(ABOVE);
}
#[when("I enter 4 mm for radius and press Tab")]
fn radius_four(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Tab, false);
}
#[when("I select the start direction")]
fn start_direction(world: &mut ApplicationWorld) {
    world.sketch_click(START);
}
#[when("I enter 90 degrees, press Tab and select the endpoint")]
fn sweep_ninety(world: &mut ApplicationWorld) {
    world.type_text("90");
    world.key(ViewportPhysicalKey::Tab, false);
    world.sketch_click(ABOVE);
}
#[when("I press Escape and measure the arc")]
fn escape_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    measure_arc(world);
}
#[then("Measure reports radius 4.000 mm, sweep 90.00 degrees and length 6.283 mm")]
fn typed_rows(world: &mut ApplicationWorld) {
    world.expect_measure("Radius 4.000 mm");
    world.expect_measure("Sweep 90.00 deg");
    world.expect_measure("Length 6.283 mm");
}
#[then("the resulting arc is oriented toward the upper right")]
fn upper_right(world: &mut ApplicationWorld) {
    let middle = arc_fraction_pixel(world, 0.5);
    let arc = newest_arc(world);
    let center = world.edited_geometry().point(arc.center).unwrap().position;
    let center = world.sketch_pixel_of(center);
    assert!(
        middle[0] > center[0] && middle[1] < center[1],
        "middle {middle:?}, center {center:?}"
    );
}
// arc-center-point-12 and arc-center-point-13
#[given("the radius dimension editor is open with 4 mm selected")]
fn radius_editor(world: &mut ApplicationWorld) {
    created_numeric_arc(world, "4");
    world.open_dimension_label("4");
    assert!(world.text_is_selected("4mm"), "expected the value selected");
}
#[when("I replace its expression with 5 and press Escape")]
fn replace_and_escape(world: &mut ApplicationWorld) {
    world.type_text("5");
    world.escape();
    world.pump(Vec::new());
}
#[when("I reopen the radius annotation")]
fn reopen_radius(world: &mut ApplicationWorld) {
    world.open_dimension_label("4");
}
#[then("the editor still contains 4 mm")]
fn still_four(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "expected 4 mm in the editor {text:?}");
}
#[given("the created arc has radius 4 mm and sweep 90 degrees")]
fn created_arc(world: &mut ApplicationWorld) {
    created_numeric_arc(world, "4");
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    world.observed_rows = Some(format!(
        "{:?}",
        geometry.point(arc.center).unwrap().position
    ));
}
#[given("its measured center is (0, 4.914, 17.623) mm")]
fn measure_numeric_center(world: &mut ApplicationWorld) {
    measure_arc(world);
    world.expect_measure("Center X 0.000 Y 4.914 Z 17.623 mm");
    close_measure(world);
}
#[when("I open the radius dimension and change it to 5 mm")]
fn change_radius(world: &mut ApplicationWorld) {
    world.open_dimension_label("4");
    accept_value(world, "5");
}
#[when("I exit dimension editing and measure the arc")]
fn exit_and_measure(world: &mut ApplicationWorld) {
    measure_arc(world);
}
#[then("Measure reports radius 5.000 mm, length 7.854 mm and the same center")]
fn five_rows(world: &mut ApplicationWorld) {
    world.expect_measure("Radius 5.000 mm");
    world.expect_measure("Length 7.854 mm");
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    let center = format!("{:?}", geometry.point(arc.center).unwrap().position);
    assert_eq!(
        world.observed_rows.as_deref(),
        Some(center.as_str()),
        "center"
    );
}
#[when("I close Measure and undo")]
fn close_and_undo(world: &mut ApplicationWorld) {
    close_measure(world);
    world.undo();
}
#[then("the displayed radius returns to 4 mm")]
fn radius_four_back(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
}
#[then("the displayed radius returns to 5 mm")]
fn radius_five_back(world: &mut ApplicationWorld) {
    world.expect_annotation("5");
}
// arc-center-point-14
#[given("the recorded arc has radius 5 mm, sweep 90 degrees and center (0, 4.914, 17.623) mm")]
fn recorded_numeric_arc(world: &mut ApplicationWorld) {
    created_numeric_arc(world, "5");
}
#[when("I open the sweep dimension")]
fn open_sweep(world: &mut ApplicationWorld) {
    world.open_dimension_label("90.00");
}
#[when("I press 1, then 2, then 0")]
fn type_sweep_digits(world: &mut ApplicationWorld) {
    for digit in ["1", "2", "0"] {
        world.type_text(digit);
    }
}
#[when("I open the sweep dimension and enter 120 degrees")]
fn enter_sweep(world: &mut ApplicationWorld) {
    world.open_dimension_label("90.00");
    world.type_text("120");
}
#[when("I accept the edit, exit dimension editing and measure the arc")]
fn accept_and_measure(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
    measure_arc(world);
}
#[then("Measure reports sweep 120 degrees, length 10.472 mm and radius 5 mm")]
fn sweep_rows(world: &mut ApplicationWorld) {
    world.expect_measure("Sweep 120.00 deg");
    world.expect_measure("Length 10.472 mm");
    world.expect_measure("Radius 5.000 mm");
}
#[then("its center is (0, 4.914, 17.623) mm")]
#[then("its center remains (0, 4.914, 17.623) mm")]
fn numeric_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 4.914 Z 17.623 mm");
}
#[then("the arc extends toward the upper left in the recorded view")]
fn arc_extends_upper_left(world: &mut ApplicationWorld) {
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    let center = geometry.point(arc.center).unwrap().position;
    let end = geometry.point(arc.end).unwrap().position;
    let center = world.sketch_pixel_of(center);
    let end = world.sketch_pixel_of(end);
    assert!(
        end[0] < center[0] && end[1] < center[1],
        "arc end {end:?}, center {center:?}"
    );
}
#[then("the displayed sweep returns to 90 degrees")]
fn sweep_ninety_back(world: &mut ApplicationWorld) {
    world.expect_annotation("90.00");
}
#[then("the displayed sweep returns to 120 degrees")]
fn sweep_back(world: &mut ApplicationWorld) {
    world.expect_annotation("120.00");
}

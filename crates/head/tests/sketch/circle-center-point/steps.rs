// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/circle-center-point`: pointer and typed circles, and their diameter dimension.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const LINE: [[i32; 2]; 2] = [[420, 720], [600, 620]];
const CENTER: [i32; 2] = [860, 520];
const THROUGH: [i32; 2] = [940, 520];
/// Recorded in millimetres, as sketch x then sketch y on the YZ plane.
const RECORDED_CENTER: [f64; 2] = [6.553, -12.44];
const RECORDED_UPPER_CENTER: [f64; 2] = [27.574, -85.283];
/// Rounds to the recorded radius 4.187, diameter 8.375 and circumference 26.310.
const RECORDED_UPPER_RADIUS: f64 = 4.1873;
/// A pixel's rounding either way when reading back where a circle passes.
const PIXEL_SLACK: f64 = 1.5;
const RADIUS_SLACK_CM: f64 = 1e-6;

fn select_circle(world: &mut ApplicationWorld) {
    world.click_text("Circle");
    world.pump(Vec::new());
}
/// On the circle's circumference, along sketch +x from its center.
fn circle_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let geometry = world.edited_geometry();
    let circle = geometry
        .circles()
        .last()
        .copied()
        .expect("expected a circle");
    let center = geometry.point(circle.center).unwrap().position;
    world.sketch_pixel_of([center[0] + circle.radius, center[1]])
}
fn measure_circle(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = circle_pixel(world);
    world.sketch_click(at);
}
fn close_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
}
fn expect_radius_cm(world: &mut ApplicationWorld, radius: f64) {
    let geometry = world.edited_geometry();
    let circle = geometry
        .circles()
        .last()
        .copied()
        .expect("expected a circle");
    assert!(
        (circle.radius - radius).abs() < RADIUS_SLACK_CM,
        "expected radius {radius} cm, found {}",
        circle.radius
    );
}
fn sketch_with_line(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.select_line_tool();
    for corner in LINE {
        world.sketch_click(corner);
    }
    world.escape();
    world.pump(Vec::new());
}
fn typed_recorded_circle(world: &mut ApplicationWorld) {
    world.open_precise_input();
    world.precise_point(RECORDED_CENTER);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn retained_six_millimetre_circle(world: &mut ApplicationWorld) {
    sketch_with_line(world);
    select_circle(world);
    typed_recorded_circle(world);
    world.escape();
    world.pump(Vec::new());
    world.expect_annotation("6");
}
fn expect_center_row(world: &ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 6.553 Z -12.440 mm");
}
// circle-center-point-10
#[given("an editable metric sketch with a midpoint line")]
fn metric_sketch(world: &mut ApplicationWorld) {
    sketch_with_line(world);
}
#[when("I invoke Center Point Circle using Ctrl+Shift+C")]
fn invoke_by_shortcut(world: &mut ApplicationWorld) {
    world.control_shift_key(ViewportPhysicalKey::C);
    world.pump(Vec::new());
    world.expect_status("Select center of circle");
}
#[when(
    "I select the recorded free center at (0.000, 6.553, -12.440) mm and enter 6 followed by Return"
)]
fn typed_center_and_diameter(world: &mut ApplicationWorld) {
    typed_recorded_circle(world);
}
#[then("the command returns to center selection with a 6 mm circle dimension retained")]
fn back_to_center(world: &mut ApplicationWorld) {
    world.expect_status("Select center of circle");
    world.expect_annotation("6");
}
#[when("I exit and measure the circle")]
fn exit_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    measure_circle(world);
}
#[then("its diameter is 6.000 mm and radius is 3.000 mm")]
fn six_millimetres(world: &mut ApplicationWorld) {
    world.expect_measure("Diameter 6.000 mm");
    world.expect_measure("Radius 3.000 mm");
}
#[then("its global center is (0.000, 6.553, -12.440) mm within displayed precision")]
fn global_center(world: &mut ApplicationWorld) {
    expect_center_row(world);
}
// circle-center-point-11
#[given("the recorded circle has a driving diameter dimension equal to 6 mm")]
#[given("the retained circle has a driving diameter dimension equal to 6 mm")]
#[given("an editable sketch contains the retained 6 mm circle and midpoint line")]
fn retained_circle(world: &mut ApplicationWorld) {
    retained_six_millimetre_circle(world);
}
#[given("its displayed center is (0.000, 6.553, -12.440) mm")]
fn displayed_center(world: &mut ApplicationWorld) {
    measure_circle(world);
    expect_center_row(world);
    close_measure(world);
}
#[when("I open the diameter dimension and accept the value 8 mm")]
fn accept_eight(world: &mut ApplicationWorld) {
    world.open_dimension_label("6");
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("I exit General Dimension and measure the circle")]
fn exit_dimension_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    measure_circle(world);
}
#[then("diameter is 8.000 mm and radius is 4.000 mm")]
fn eight_millimetres(world: &mut ApplicationWorld) {
    world.expect_measure("Diameter 8.000 mm");
    world.expect_measure("Radius 4.000 mm");
}
#[then("its measured center remains (0.000, 6.553, -12.440) mm")]
fn center_remains(world: &mut ApplicationWorld) {
    expect_center_row(world);
}
#[when("I close Measure and undo the edit")]
fn close_and_undo(world: &mut ApplicationWorld) {
    close_measure(world);
    world.undo();
}
#[then("the visible dimension and extent return to 6 mm")]
fn back_to_six(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    expect_radius_cm(world, 0.3);
}
#[when("I redo the edit")]
fn redo_edit(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the visible dimension and extent return to 8 mm")]
fn back_to_eight(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    expect_radius_cm(world, 0.4);
}
#[when("I undo once more")]
fn undo_once_more(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("the 6 mm baseline is restored with the surrounding midpoint line retained")]
fn baseline_restored(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    expect_radius_cm(world, 0.3);
    assert_eq!(
        world.edited_geometry().lines().len(),
        1,
        "the midpoint line"
    );
}
// circle-center-point-01
#[when("I select Center Point Circle, click a center, then a distinct circumference point")]
fn pointer_circle(world: &mut ApplicationWorld) {
    select_circle(world);
    world.sketch_click(CENTER);
    world.sketch_click(THROUGH);
}
#[then("one circle has the selected center and passes through the circumference point")]
fn circle_through(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.circles().len(), 1, "circles");
    let circle = geometry.circles()[0];
    let center = geometry.point(circle.center).unwrap().position;
    let center_pixel = world.sketch_pixel_of(center);
    let edge = world.sketch_point_of(THROUGH);
    let reach = (edge[0] - center[0]).hypot(edge[1] - center[1]);
    assert_eq!(center_pixel, CENTER, "center pixel");
    assert!(
        (reach - circle.radius).abs() < RADIUS_SLACK_CM,
        "radius {}",
        circle.radius
    );
    let across = f64::from(THROUGH[0] - CENTER[0]);
    let pixels = world.sketch_pixel_of([center[0] + circle.radius, center[1]]);
    let shown = f64::from(pixels[0] - CENTER[0]).hypot(f64::from(pixels[1] - CENTER[1]));
    assert!(
        (shown - across).abs() < PIXEL_SLACK,
        "radius {shown} pixels"
    );
}
#[then("the command returns to Select center of circle")]
fn asks_for_center(world: &mut ApplicationWorld) {
    world.expect_status("Select center of circle");
}
// circle-center-point-06
#[when("I select Center Point Circle from the dropdown")]
fn from_dropdown(world: &mut ApplicationWorld) {
    let caption = world.run("Circle").clone();
    let arrow = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "v"
                && run.minimum[0] > caption.maximum[0]
                && (run.center()[1] - caption.center()[1]).abs() < 4.
        })
        .expect("expected the Circle dropdown")
        .center()
        .map(|value| value as i32);
    world.click(arrow);
    world.pump(Vec::new());
    world.click_text("Center Point Circle");
    world.pump(Vec::new());
}
#[when("I select a center")]
fn select_center(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("the prompt requests a point on the circle")]
fn asks_for_point(world: &mut ApplicationWorld) {
    world.expect_status("Select point on circle");
}
#[then("the MCAD Software returns to Ready without a new circle")]
fn ready_without_circle(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert!(
        world.edited_geometry().circles().is_empty(),
        "expected no circle"
    );
}
// circle-center-point-07
#[when("I invoke Center Point Circle")]
fn invoke_circle(world: &mut ApplicationWorld) {
    select_circle(world);
}
#[when("I select a center and a distinct circumference point")]
fn center_and_edge(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
    world.sketch_click(THROUGH);
    world.hover([1100, 700]);
}
#[then("a circle appears and the command requests another center")]
fn circle_and_next(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().circles().len(), 1, "circles");
    world.expect_segment_drawn([CENTER[0], CENTER[1] - 80], [CENTER[0], CENTER[1] - 80]);
    world.expect_status("Select center of circle");
}
#[then("the circle is retained")]
#[then("the circle returns")]
fn circle_retained(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().circles().len(), 1, "circles");
    world.expect_segment_drawn([CENTER[0], CENTER[1] - 80], [CENTER[0], CENTER[1] - 80]);
}
#[when("I undo")]
fn undo(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("the circle is removed")]
fn circle_removed(world: &mut ApplicationWorld) {
    assert!(
        world.edited_geometry().circles().is_empty(),
        "expected no circle"
    );
    world.expect_segment_absent([CENTER[0], CENTER[1] - 80], [CENTER[0], CENTER[1] - 80]);
}
// circle-center-point-08
#[given("the upper-right center-point circle has been restored by redo")]
fn restored_upper_circle(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    select_circle(world);
    world.open_precise_input();
    world.precise_point(RECORDED_UPPER_CENTER);
    world.precise_point([
        RECORDED_UPPER_CENTER[0] + RECORDED_UPPER_RADIUS,
        RECORDED_UPPER_CENTER[1],
    ]);
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.redo();
}
#[when("I open Measure and select its circumference")]
fn measure_circumference(world: &mut ApplicationWorld) {
    measure_circle(world);
}
#[then("Measure displays diameter 8.375 mm and radius 4.187 mm")]
fn upper_size(world: &mut ApplicationWorld) {
    world.expect_measure("Diameter 8.375 mm");
    world.expect_measure("Radius 4.187 mm");
}
#[then("length 26.310 mm and angle 360.00 degrees")]
fn upper_length(world: &mut ApplicationWorld) {
    world.expect_measure("Length 26.310 mm");
    world.expect_measure("Angle 360.00 deg");
}
#[then("center X 0.000 mm, Y 27.574 mm, Z -85.283 mm")]
fn upper_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 27.574 Z -85.283 mm");
}
// circle-center-point-09
#[when("I click the same position for its circumference")]
fn same_position(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("Select point on circle remains active")]
fn still_asking(world: &mut ApplicationWorld) {
    world.expect_status("Select point on circle");
}
#[then("the displayed radius is 0.000 mm")]
fn zero_shown(world: &mut ApplicationWorld) {
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("0.000mm"), "expected a zero size in {text:?}");
}
#[then("no error dialog or new circle appears")]
fn nothing_appears(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
    assert!(
        world.edited_geometry().circles().is_empty(),
        "expected no circle"
    );
}
#[then("the MCAD Software returns to Ready")]
fn back_to_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
}
// circle-center-point-12
#[when("Center Point Circle is invoked and a free center is selected")]
fn invoked_with_center(world: &mut ApplicationWorld) {
    select_circle(world);
    world.sketch_click(CENTER);
    world.hover(THROUGH);
}
#[when("the 0 key is pressed followed by Return")]
fn zero_and_return(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("the size field remains red with 0 and Select point on circle remains active")]
fn zero_red(world: &mut ApplicationWorld) {
    world.expect_hud_value_invalid("0");
    world.expect_status("Select point on circle");
}
#[then("no modal error dialog appears")]
fn no_modal(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
}
#[when("Escape is pressed once")]
#[when("Escape is pressed again")]
fn escape_pressed(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("the numeric entry clears and a mouse-driven circle preview remains active")]
fn entry_cleared(world: &mut ApplicationWorld) {
    world.expect_no_annotation("0");
    world.expect_hud_shows_length();
    world.expect_status("Select point on circle");
}
#[then("Ready appears and the preview disappears")]
fn ready_and_gone(world: &mut ApplicationWorld) {
    world.expect_ready();
    let hud = world.window_text("##sketch-hud");
    assert!(hud.is_empty(), "expected no fields, found {hud:?}");
}
#[then("the retained circle and midpoint line remain visible")]
fn retained_remain(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.circles().len(), 1, "circles");
    assert_eq!(geometry.lines().len(), 1, "lines");
}
// circle-center-point-13
fn click_annotation(world: &mut ApplicationWorld, value: &str) {
    let at = world.exact_run_center(value);
    world.hover(at);
    world.click(at);
    world.pump(Vec::new());
}
#[when(
    "General Dimension opens the diameter dimension and the selected expression is replaced with 9"
)]
fn dimension_tool_edit(world: &mut ApplicationWorld) {
    world.click_text("Dimension");
    world.pump(Vec::new());
    click_annotation(world, "6");
    world.type_text("9");
}
#[when("Escape cancels the editor")]
fn escape_cancels_editor(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("General Dimension remains active and the circle keeps its visible size")]
fn dimension_stays(world: &mut ApplicationWorld) {
    assert!(
        world.state().dimension_tool_active,
        "expected the command still up"
    );
    world.expect_annotation("6");
    expect_radius_cm(world, 0.3);
}
#[when("the diameter annotation is clicked again")]
fn clicked_again(world: &mut ApplicationWorld) {
    click_annotation(world, "6");
}
#[then("Edit Dimension displays the retained 6 mm expression")]
fn retained_expression(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("6mm"), "expected 6 mm in the editor {text:?}");
}
#[when("Escape closes the editor and Escape exits General Dimension")]
fn two_escapes(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
#[then("Ready appears")]
fn ready_appears(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert!(
        !world.state().dimension_tool_active,
        "expected the command to end"
    );
}

// SPDX-License-Identifier: GPL-2.0-only
//! The recorded YZ fixtures, built exactly: Precise Input places the ends, typed fields the rest.
//! Sketch x is model Y (up on screen) and sketch y is model Z (left on screen).
use super::super::super::ApplicationWorld;
use crate::HeadWindow;
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportPhysicalKey;

/// The reference, along model Z so it reads horizontal on screen: start and length.
const REFERENCE_START: [f64; 2] = [-18.27, -15.];
const REFERENCE_LENGTH: &str = "13.544";
/// The numeric line's start; its minimum separation from the reference is the recorded 9.664.
const NUMERIC_START: [f64; 2] = [-10., -20.];
const SOURCE_PIXELS: [[i32; 2]; 2] = [[560, 340], [760, 340]];
const FIRST_DRAG: [i32; 2] = [900, 460];
const LATER_DRAG: [i32; 2] = [980, 300];
/// Each corner clear of the numeric fields, which sit just below and right of the last click.
const CHAIN: [[i32; 2]; 3] = [[420, 260], [560, 230], [620, 320]];
const SEPARATE: [[i32; 2]; 2] = [[900, 280], [1100, 280]];
/// Away from the chain's edges, so no parallel relation takes one of its four directions.
const OBLIQUE: [[i32; 2]; 2] = [[1000, 560], [1180, 620]];
const TRIANGLE_START: [f64; 2] = [-40., 40.];
const CLEAR_START: [i32; 2] = [620, 480];
/// Where the pointer rests while values are typed; clear of every fixture.
const TYPING_POINTER: [i32; 2] = [1000, 300];

fn enter_length_and_angle(world: &mut ApplicationWorld, length: &str, angle: &str) {
    world.type_text(length);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(angle);
    world.key(ViewportPhysicalKey::Enter, false);
}
/// A line with typed length and angle from an exact start; the command is left at Ready.
fn exact_typed_line(world: &mut ApplicationWorld, start: [f64; 2], length: &str, angle: &str) {
    world.select_line_tool();
    world.precise_point(start);
    world.hover(TYPING_POINTER);
    enter_length_and_angle(world, length, angle);
    world.escape();
    world.pump(Vec::new());
}
pub(super) fn measure_two_curves(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let numeric = world.line_point(1, 0.3);
    world.sketch_click(numeric);
    let reference = world.line_point(0, 0.3);
    world.sketch_click(reference);
}
fn close_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
}
/// Save, close the tab, open it again through the dialog and edit its sketch from the browser.
pub(super) fn reopen_and_edit(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved part");
    world.close_active_tab();
    world.pump(Vec::new());
    world.reopen_through_dialog(&path);
    world.pump(Vec::new());
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected a sketch row in the browser")
        .center()
        .map(|value| value as i32);
    world.double_click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn finish_and_save(world: &mut ApplicationWorld) {
    world.finish_sketch();
    let part = world.active_id();
    world.saved_path = Some(world.save_part_to_file(part, "Part6.opd"));
}
// line-17
#[given("a horizontal source segment in an editable YZ sketch")]
fn horizontal_source(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.select_line_tool();
    for corner in SOURCE_PIXELS {
        world.sketch_click(corner);
    }
    world.escape();
    world.pump(Vec::new());
    world.observed_counts = vec![world.dimensions_needed()];
}
#[given("Line is active at its endpoint with Precise Input open")]
fn line_at_endpoint(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.open_precise_input();
}
fn drag_from_source_end(world: &mut ApplicationWorld, to: [i32; 2]) {
    world.drag(SOURCE_PIXELS[1], to);
    world.last_drag = Some([SOURCE_PIXELS[1], to]);
}
#[when("I drag from the endpoint and release")]
fn drag_with_toolbar(world: &mut ApplicationWorld) {
    drag_from_source_end(world, FIRST_DRAG);
}
#[then("the pending preview remains straight")]
fn preview_straight(world: &mut ApplicationWorld) {
    let [from, to] = world.last_drag.expect("expected an earlier drag");
    world.expect_segment_drawn(from, to);
    assert!(
        world.edited_geometry().arcs().is_empty(),
        "expected no accepted arc"
    );
}
#[when("I open the Line context menu and press Escape to dismiss it")]
fn menu_and_escape(world: &mut ApplicationWorld) {
    let [_, end] = world.last_drag.expect("expected an earlier drag");
    world.right_click(end);
    world.pump(Vec::new());
    world.expect_text("Cancel (ESC)");
    world.escape();
    world.pump(Vec::new());
    world.expect_no_text("Cancel (ESC)");
    world.expect_status("Select end of line");
}
#[when("I press Escape again to end Line")]
fn escape_again_ends(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.expect_ready();
}
#[when("I restart Line and close Precise Input through its toolbar options")]
fn restart_and_close(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.close_precise_input();
}
#[when("I acquire the source endpoint and drag from it, then release")]
fn acquire_and_drag(world: &mut ApplicationWorld) {
    drag_from_source_end(world, LATER_DRAG);
}
#[then("only the accepted horizontal source remains")]
fn only_source(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_segment_drawn(SOURCE_PIXELS[0], SOURCE_PIXELS[1]);
    world.expect_segment_absent(SOURCE_PIXELS[1], LATER_DRAG);
    assert_eq!(world.edited_geometry().lines().len(), 1, "accepted lines");
    let needed = world.dimensions_needed();
    assert_eq!(world.observed_counts, [needed], "dimensions needed");
}
// line-18
#[given("the recorded YZ sketch fixture contains an isolated horizontal reference curve")]
fn isolated_reference(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.open_precise_input();
    exact_typed_line(world, REFERENCE_START, REFERENCE_LENGTH, "90");
}
/// With Precise Input up the recorded start is exact; otherwise any clear pixel will do.
#[when("I start Line and choose a start point")]
fn start_point(world: &mut ApplicationWorld) {
    world.select_line_tool();
    if super::super::super::diagnostics::window_geometry("Precise Input").is_some() {
        world.precise_point(NUMERIC_START);
    } else {
        world.sketch_click(CLEAR_START);
    }
    world.hover(TYPING_POINTER);
}
#[when("I enter length 10 and angle 30 in separate dynamic fields and accept")]
fn enter_numeric(world: &mut ApplicationWorld) {
    enter_length_and_angle(world, "10", "30");
}
#[when("I exit Line and open Measure")]
fn exit_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::M, false);
}
#[when("I select the new curve and the horizontal reference")]
fn select_both(world: &mut ApplicationWorld) {
    let numeric = world.line_point(1, 0.3);
    world.sketch_click(numeric);
    let reference = world.line_point(0, 0.3);
    world.sketch_click(reference);
}
#[then("Measure reports the new curve length as 10.000 mm")]
fn new_curve_length(world: &mut ApplicationWorld) {
    world.expect_measure("Length 10.000 mm");
}
#[then("the reference length is 13.544 mm")]
fn reference_length(world: &mut ApplicationWorld) {
    world.expect_measure("Length 13.544 mm");
}
#[then("the angle between them is 60.00 degrees")]
fn angle_sixty(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 60.00 deg");
}
#[then("the minimum and maximum distances are 9.664 mm and 21.681 mm")]
fn distances(world: &mut ApplicationWorld) {
    world.expect_measure("Minimum distance 9.664 mm");
    world.expect_measure("Maximum distance 21.681 mm");
    world.observed_rows = Some(world.window_text("Measure"));
}
#[when("I close Measure, undo and redo creation")]
fn undo_redo_creation(world: &mut ApplicationWorld) {
    close_measure(world);
    world.undo();
    world.expect_no_annotation("30.00");
    world.redo();
}
#[when("I open Measure and select both curves again")]
fn measure_again(world: &mut ApplicationWorld) {
    measure_two_curves(world);
}
#[then("the lengths, angle and distances have the same displayed values")]
fn same_rows(world: &mut ApplicationWorld) {
    let before = world.observed_rows.clone().expect("expected earlier rows");
    assert_eq!(world.window_text("Measure"), before, "Measure rows");
}
// line-19
#[given("the recorded YZ sketch fixture contains the numeric line and horizontal reference")]
fn numeric_and_reference(world: &mut ApplicationWorld) {
    isolated_reference(world);
    exact_typed_line(world, NUMERIC_START, "10", "30");
}
#[when("the saved sketch is edited and the two curves are selected in Measure")]
fn saved_sketch_measured(world: &mut ApplicationWorld) {
    measure_two_curves(world);
}
#[when("Finish Sketch settles and the part document is saved, closed, and reopened")]
fn saved_closed_reopened(world: &mut ApplicationWorld) {
    finish_and_save(world);
    reopen_and_edit(world);
}
#[then("the numeric curve measures 10.000 mm and the reference measures 13.544 mm")]
fn lengths_measured(world: &mut ApplicationWorld) {
    world.expect_measure("Length 10.000 mm");
    world.expect_measure("Length 13.544 mm");
}
#[then("their angle is 60.00 degrees and minimum distance is 9.664 mm")]
fn angle_and_minimum(world: &mut ApplicationWorld) {
    world.expect_measure("Angle 60.00 deg");
    world.expect_measure("Minimum distance 9.664 mm");
}
#[then("the maximum distance is 21.681 mm with minimum delta X 0.000 Y 8.270 Z 5.000 mm")]
fn maximum_and_delta(world: &mut ApplicationWorld) {
    world.expect_measure("Maximum distance 21.681 mm");
    world.expect_measure("Minimum delta X 0.000 Y 8.270 Z 5.000 mm");
}
// line-25
/// Two typed legs and Close: only its first vertex stays free.
fn closed_triangle(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.precise_point(TRIANGLE_START);
    world.hover(TYPING_POINTER);
    for (length, angle) in [("4", "0"), ("4.0565", "90")] {
        enter_length_and_angle(world, length, angle);
    }
    world.right_click([1200, 700]);
    world.pump(Vec::new());
    world.click_text("Close");
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
}
fn pointer_chain(world: &mut ApplicationWorld, corners: &[[i32; 2]]) {
    world.select_line_tool();
    for corner in corners {
        world.sketch_click(*corner);
    }
    world.escape();
    world.pump(Vec::new());
}
#[given("the retained Line groups after Ready-state Redo show 20 dimensions needed")]
fn retained_groups(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    pointer_chain(world, &SOURCE_PIXELS);
    world.expect_dimensions_needed(3);
    pointer_chain(world, &CHAIN);
    world.expect_dimensions_needed(9);
    world.open_precise_input();
    exact_typed_line(world, NUMERIC_START, "10", "30");
    world.expect_dimensions_needed(11);
    closed_triangle(world);
    world.expect_dimensions_needed(13);
    pointer_chain(world, &SEPARATE);
    world.expect_dimensions_needed(16);
    pointer_chain(world, &OBLIQUE);
    world.undo();
    world.redo();
    world.expect_dimensions_needed(20);
}
#[when("I finish the sketch and save the part document")]
fn finish_sketch_and_save(world: &mut ApplicationWorld) {
    finish_and_save(world);
}
#[when("close its document tab then reopen the part document and edit the saved sketch")]
fn close_reopen_edit(world: &mut ApplicationWorld) {
    reopen_and_edit(world);
}
#[then("the upper-left two-edge chain and original source are visible")]
fn chain_and_source(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(CHAIN[0], CHAIN[1]);
    world.expect_segment_drawn(CHAIN[1], CHAIN[2]);
    world.expect_segment_drawn(SOURCE_PIXELS[0], SOURCE_PIXELS[1]);
}
#[then(
    "the 10 and 30.00 annotations, triangle, separate horizontal segment and oblique segment remain"
)]
fn the_rest_remains(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    world.expect_annotation("30.00");
    world.expect_segment_drawn(SEPARATE[0], SEPARATE[1]);
    world.expect_segment_drawn(OBLIQUE[0], OBLIQUE[1]);
    assert_eq!(world.edited_geometry().lines().len(), 9, "retained lines");
}
#[then("Ready shows 20 dimensions needed in RIGHT view")]
fn right_view(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(20);
    let window = &world.app().window;
    let origin = HeadWindow::sketch_pixel(window, SketchPlane::yz(), [0., 0.]).unwrap();
    let model_y = HeadWindow::sketch_pixel(window, SketchPlane::yz(), [1., 0.]).unwrap();
    let model_z = HeadWindow::sketch_pixel(window, SketchPlane::yz(), [0., 1.]).unwrap();
    assert!(
        model_y[1] < origin[1] && model_y[0] == origin[0],
        "model Y up: {model_y:?} from {origin:?}"
    );
    assert!(
        model_z[0] < origin[0] && model_z[1] == origin[1],
        "model Z left: {model_z:?} from {origin:?}"
    );
}

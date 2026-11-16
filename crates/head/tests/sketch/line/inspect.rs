// SPDX-License-Identifier: GPL-2.0-only
//! Measure, Region Properties and dimension edits against Line results.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const START: [i32; 2] = [620, 480];
const POINTER: [i32; 2] = [900, 300];
/// Clear of a short segment's ends and middle, whose picks are points of their own.
const ON_THE_CURVE: f64 = 0.3;

fn line_with_values(world: &mut ApplicationWorld, length: &str, angle: &str) {
    world.select_line_tool();
    world.sketch_click(START);
    world.hover(POINTER);
    world.type_text(length);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(angle);
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
}
fn press_m_and_select_last_line(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = world.last_line_point(ON_THE_CURVE);
    world.sketch_click(at);
}
// line-04 and line-08
#[when("I open Measure and select the segment")]
#[when("the user open Measure and select the segment")]
#[when("I measure the visible upper segment")]
fn measure_upper(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
}
#[then("Measure reports its length as 10.000 mm")]
fn reports_ten(world: &mut ApplicationWorld) {
    world.expect_measure("Length 10.000 mm");
}
#[then("Measure identifies a Curve of length 10.000 mm")]
fn identifies_ten(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 10.000 mm");
}
// line-11
#[given("a length-four segment remains after exiting Line")]
fn retained_four(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    line_with_values(world, "4", "30");
}
#[when("I press M and select the segment")]
fn press_m(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
}
#[then("Measure identifies Curve with Length 4.000 mm")]
fn identifies_four(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 4.000 mm");
}
#[when("I click Done")]
#[when("the user click Done")]
fn click_done(world: &mut ApplicationWorld) {
    world.click_text("Done");
}
#[then("the panel closes and the segment remains")]
fn panel_closed(world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry("Measure").is_none(),
        "expected Measure to close"
    );
    let [from, to] = world.last_line_pixels();
    world.expect_segment_drawn(from, to);
}
// line-12, line-13 and line-14
fn line_with_driving_length(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    line_with_values(world, "4", "30");
}
#[given("a line has a driving length dimension storing 4 mm")]
fn stores_four(world: &mut ApplicationWorld) {
    line_with_driving_length(world);
    world.open_dimension_label("4");
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "expected 4 mm in the editor {text:?}");
    world.escape();
}
#[when("I open the length dimension and type 5 then press Escape")]
fn type_five_escape(world: &mut ApplicationWorld) {
    world.open_dimension_label("4");
    world.type_text("5");
    world.escape();
}
#[then("the editor closes and the shorter segment remains")]
fn editor_closed(world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry("Edit Dimension").is_none(),
        "expected the editor to close"
    );
    world.expect_annotation("4");
}
#[when("I reopen the annotation")]
#[when("the user reopen the annotation")]
fn reopen_annotation(world: &mut ApplicationWorld) {
    let value = world
        .opened_value
        .clone()
        .expect("expected an opened dimension");
    world.open_dimension_label(&value);
}
#[then("the stored expression is 4 mm")]
fn expression_four(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "expected 4 mm in the editor {text:?}");
    world.escape();
}
#[given("the driving length dimension is reopened with 4 mm selected")]
fn reopened_selected(world: &mut ApplicationWorld) {
    line_with_driving_length(world);
    world.open_dimension_label("4");
    assert!(world.text_is_selected("4mm"), "expected the value selected");
}
#[when("I type 5 and press Enter then Escape")]
fn type_five_enter(world: &mut ApplicationWorld) {
    world.type_text("5");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
}
#[then("the segment lengthens with annotation 5 and retained annotation 30.00")]
fn lengthened(world: &mut ApplicationWorld) {
    world.expect_annotation("5");
    world.expect_annotation("30.00");
}
#[then("Selection 1 is Curve with Length 5.000 mm")]
fn selection_five(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 5.000 mm");
}
#[then("the measurement panel closes")]
fn measurement_closed(_: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry("Measure").is_none(),
        "expected Measure to close"
    );
}
#[given("the length dimension was changed to 5 mm and Measure is closed")]
fn measure_closed_five(world: &mut ApplicationWorld) {
    reopened_selected(world);
    type_five_enter(world);
    press_m_and_select_last_line(world);
    world.click_text("Done");
    world.expect_annotation("5");
}
#[when("I press Ctrl+Z")]
#[when("the user press Ctrl+Z")]
fn ctrl_z(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("annotation 4 and the shorter segment return")]
fn four_back(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    world.expect_no_annotation("5");
}
#[then("annotation 5 and the longer segment return")]
fn five_back(world: &mut ApplicationWorld) {
    world.expect_annotation("5");
    world.expect_no_annotation("4");
}
// line-20: two typed edges so the closed triangle has the recorded area exactly.
#[when("I leave Line and select the triangle interior in Region Properties")]
fn select_triangle(world: &mut ApplicationWorld) {
    world.escape();
    world.right_click([1200, 600]);
    world.pump(Vec::new());
    world.click_text("Region Properties");
    world.pump(Vec::new());
    let at = world.chain_interior();
    world.sketch_click(at);
}
#[then("Sketch Loop1 is accepted and Calculate reports area 8.113 square millimetres")]
fn loop_area(world: &mut ApplicationWorld) {
    world.expect_text("Sketch Loop1");
    world.click_text("Calculate");
    let text = world.window_text("Region Properties");
    assert!(
        text.contains("Area8.113mm"),
        "expected the area in {text:?}"
    );
}

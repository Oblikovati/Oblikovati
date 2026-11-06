// SPDX-License-Identifier: GPL-2.0-only
//! Measure, Region Properties and dimension edits against Line results.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const START: [i32; 2] = [620, 480];
const POINTER: [i32; 2] = [900, 300];
/// A recorded name is the 35th dimension of its session, so earlier ones are made first.
const DIMENSIONS_BEFORE_D34: usize = 34;

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
    let at = world.last_line_point(0.25);
    world.sketch_click(at);
}
fn expect_measure(world: &ApplicationWorld, row: &str) {
    let text = world.window_text("Measure");
    assert!(
        text.contains(&row.replace(' ', "")),
        "expected {row:?} in Measure {text:?}"
    );
}
// line-04 and line-08
#[then("the later selected Curve measures 10.000 mm")]
fn later_curve(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
    expect_measure(world, "Length 10.000 mm");
}
#[when("I measure the visible upper segment as recorded in steps 34 and 35")]
fn measure_upper(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
}
#[then("Measure identifies a Curve of length 10.000 mm")]
fn identifies_ten(world: &mut ApplicationWorld) {
    expect_measure(world, "Selection 1: Curve");
    expect_measure(world, "Length 10.000 mm");
}
#[then(
    "the upper endpoint is outside the frame and no independent numeric angle measurement is claimed"
)]
fn upper_outside(world: &mut ApplicationWorld) {
    // Only one curve is selected, so the panel has no angle row to claim.
    let text = world.window_text("Measure");
    assert!(!text.contains("Angle"), "expected no angle in {text:?}");
}
// line-11
#[given("the recorded length-four segment is retained after Escape")]
fn retained_four(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(51);
    line_with_values(world, "4", "30");
}
#[when("I press M and select the segment at (314,784)")]
fn press_m(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
}
#[then("Measure identifies Curve with Length 4.000 mm")]
fn identifies_four(world: &mut ApplicationWorld) {
    expect_measure(world, "Selection 1: Curve");
    expect_measure(world, "Length 4.000 mm");
}
#[when("I click Done")]
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
#[then("numeric angle measurement and saved persistence are not inferred")]
fn not_inferred(_: &mut ApplicationWorld) {}
// line-12, line-13 and line-14
fn recorded_d34(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_named_dimensions(DIMENSIONS_BEFORE_D34);
    world.seed_dimensions_needed(51);
    line_with_values(world, "4", "30");
    world.expect_dimensions_needed(53);
}
fn open_length_editor(world: &mut ApplicationWorld, value: &str) {
    let at = world.exact_run_center(value);
    world.double_click(at);
    world.pump(Vec::new());
}
#[given("the recorded line length d34 stores 4 mm")]
fn stores_four(world: &mut ApplicationWorld) {
    recorded_d34(world);
    open_length_editor(world, "4");
    world.expect_text("d34");
    world.escape();
}
#[when("I open d34 and type 5 then press Escape")]
fn type_five_escape(world: &mut ApplicationWorld) {
    open_length_editor(world, "4");
    world.expect_text("d34");
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
fn reopen_annotation(world: &mut ApplicationWorld) {
    open_length_editor(world, "4");
}
#[then("the stored expression is 4 mm")]
fn expression_four(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "expected 4 mm in the editor {text:?}");
    world.escape();
}
#[given("d34 is reopened with 4 mm selected")]
fn reopened_selected(world: &mut ApplicationWorld) {
    recorded_d34(world);
    open_length_editor(world, "4");
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
#[then("Ready shows 53 dimensions needed")]
fn ready_53(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(53);
}
#[when("I open Measure and select the segment at (301,763)")]
fn open_measure(world: &mut ApplicationWorld) {
    press_m_and_select_last_line(world);
}
#[then("Selection 1 is Curve with Length 5.000 mm")]
fn selection_five(world: &mut ApplicationWorld) {
    expect_measure(world, "Selection 1: Curve");
    expect_measure(world, "Length 5.000 mm");
}
#[then("the measurement panel closes")]
fn measurement_closed(_: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry("Measure").is_none(),
        "expected Measure to close"
    );
}
#[given("Measure has closed after the recorded length-five edit")]
fn measure_closed_five(world: &mut ApplicationWorld) {
    reopened_selected(world);
    type_five_enter(world);
    press_m_and_select_last_line(world);
    world.click_text("Done");
    world.expect_annotation("5");
}
#[when("I press Ctrl+Z")]
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
#[then("no post-history remeasurement is inferred")]
fn no_remeasure(_: &mut ApplicationWorld) {}
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

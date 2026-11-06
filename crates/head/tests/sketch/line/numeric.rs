// SPDX-License-Identifier: GPL-2.0-only
//! Typed length and angle: rejected values, retained annotations and their history.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const START: [i32; 2] = [620, 480];
const POINTER: [i32; 2] = [900, 300];

fn start_numeric_line(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.sketch_click(START);
    world.hover(POINTER);
}
fn enter_length_and_angle(world: &mut ApplicationWorld, length: &str, angle: &str) {
    world.type_text(length);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(angle);
}
/// The whole recorded sequence: typed values accepted, then the command left.
fn accept_numeric_line(world: &mut ApplicationWorld, length: &str, angle: &str) {
    start_numeric_line(world);
    enter_length_and_angle(world, length, angle);
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
}
// line-09 and line-10
#[given("the recorded Part1 Sketch1 baseline has 51 dimensions needed")]
fn baseline_51(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(51);
}
#[when("I start Line and choose start (350,850)")]
fn start_line_recorded(world: &mut ApplicationWorld) {
    start_numeric_line(world);
}
#[when("type 0 in its length HUD and press Enter")]
fn type_zero(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the length expression remains red")]
fn length_red(world: &mut ApplicationWorld) {
    world.expect_hud_value_invalid("0");
}
#[then("the endpoint-selection prompt remains without a modal dialog")]
fn prompt_remains(world: &mut ApplicationWorld) {
    world.expect_status("Select end of line");
    assert!(
        super::super::super::diagnostics::window_geometry("Warning").is_none(),
        "expected no modal dialog"
    );
}
#[then("no accepted new segment is visible")]
fn no_accepted(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(51);
}
#[given("the recorded Line length input contains rejected zero")]
fn contains_zero(world: &mut ApplicationWorld) {
    baseline_51(world);
    start_numeric_line(world);
    type_zero(world);
    world.expect_hud_value_invalid("0");
}
#[when("I press Ctrl+A and type 4")]
fn select_all_four(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("4");
}
#[when("press Tab then type 3 and 0 separately and press Enter")]
fn tab_three_zero(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("3");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the segment is accepted and Line continues endpoint selection")]
fn accepted_continues(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(53);
    world.expect_status("Select end of line");
}
#[then("Ready appears with annotations 4 and 30.00")]
fn ready_annotations(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_annotation("4");
    world.expect_annotation("30.00");
}
#[then("53 dimensions needed is displayed")]
fn shows_53(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(53);
}
// line-04 and line-08
#[given("the recorded Sketch3 baseline is Ready with 220 dimensions needed")]
fn baseline_220(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(220);
    world.expect_ready();
}
#[given("the length-10 and angle-30 segment has been accepted and Line exited")]
fn numeric_accepted(world: &mut ApplicationWorld) {
    baseline_220(world);
    accept_numeric_line(world, "10", "30");
    world.expect_dimensions_needed(222);
}
#[when("I undo its creation in step 32")]
fn undo_creation(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("its numeric annotations disappear and the count returns to 220")]
fn annotations_gone(world: &mut ApplicationWorld) {
    world.expect_no_annotation("10");
    world.expect_no_annotation("30.00");
    world.expect_dimensions_needed(220);
}
#[when("I redo creation in step 33")]
fn redo_creation(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the numeric annotations return and the count becomes 222")]
fn annotations_back(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    world.expect_annotation("30.00");
    world.expect_dimensions_needed(222);
}
#[when("I select Line and choose the start point in step 23")]
fn select_line_start(world: &mut ApplicationWorld) {
    start_numeric_line(world);
}
#[when("I press Tab to lock length and select angle")]
fn tab_lock_length(world: &mut ApplicationWorld) {
    enter_length_and_angle(world, "10", "30");
}
#[when("I press Return and then Escape")]
fn return_then_escape(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
}
#[then("the retained annotations show 10 and 30.00 degrees")]
fn retained_ten_thirty(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    world.expect_annotation("30.00");
}
#[then("the sketch is Ready with 222 dimensions needed")]
fn ready_222(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(222);
}
#[when("I dismiss the transient selector and undo then redo the creation")]
fn undo_then_redo(world: &mut ApplicationWorld) {
    world.undo();
    world.observed_counts.push(world.status_dimensions_needed());
    world.observed_annotations = Some(world.annotation_shown("10"));
    world.redo();
    world.observed_counts.push(world.status_dimensions_needed());
}
#[then("the count changes to 220 and back to 222")]
fn count_there_and_back(world: &mut ApplicationWorld) {
    assert_eq!(
        world.observed_counts,
        [220, 222],
        "counts after undo and redo"
    );
}
#[then("the numeric annotations disappear and return")]
fn annotations_there_and_back(world: &mut ApplicationWorld) {
    assert_eq!(
        world.observed_annotations,
        Some(false),
        "annotations after undo"
    );
    world.expect_annotation("10");
}

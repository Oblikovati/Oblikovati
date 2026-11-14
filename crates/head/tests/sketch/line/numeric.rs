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
// line-09 and line-10
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
    assert!(
        world.edited_geometry().lines().is_empty(),
        "expected no line"
    );
}
#[given("Line is active and its length input contains a rejected zero")]
fn contains_zero(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
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
    assert_eq!(
        world.edited_geometry().lines().len(),
        1,
        "accepted segments"
    );
    world.expect_status("Select end of line");
}
#[then("Ready appears with annotations 4 and 30.00")]
fn ready_annotations(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_annotation("4");
    world.expect_annotation("30.00");
}
// line-04 and line-08
#[given("the recorded numeric-input sketch fixture is Ready")]
fn numeric_fixture(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.expect_ready();
}
#[given("the recorded length-10 and angle-30 segment has been accepted and Line exited")]
fn numeric_accepted(world: &mut ApplicationWorld) {
    numeric_fixture(world);
    start_numeric_line(world);
    enter_ten(world);
    tab_lock_length(world);
    enter_thirty(world);
    return_then_escape(world);
    world.expect_annotation("30.00");
}
#[then("its numeric annotations disappear")]
fn annotations_gone(world: &mut ApplicationWorld) {
    world.expect_no_annotation("10");
    world.expect_no_annotation("30.00");
}
#[then("the numeric annotations return")]
fn annotations_back(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    world.expect_annotation("30.00");
}
#[then("the sketch is Ready")]
fn is_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
}
#[when("I undo its creation")]
fn undo_creation(world: &mut ApplicationWorld) {
    world.undo();
}
#[when("I redo creation")]
fn redo_creation(world: &mut ApplicationWorld) {
    world.redo();
}
#[when("I enter 10 in the length field")]
fn enter_ten(world: &mut ApplicationWorld) {
    world.type_text("10");
}
#[when("I press Tab to lock length and select angle")]
fn tab_lock_length(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
}
#[when("I enter 30 in the angle field")]
fn enter_thirty(world: &mut ApplicationWorld) {
    world.type_text("30");
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
#[when("I dismiss the transient selector and undo then redo the creation")]
fn undo_then_redo(world: &mut ApplicationWorld) {
    world.undo();
    world.observed_annotations = Some(world.annotation_shown("10"));
    world.redo();
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

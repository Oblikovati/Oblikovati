// SPDX-License-Identifier: GPL-2.0-only
//! Pointer scenarios: chains, double-clicks, drags, the active menu and history.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};

/// A start with room on every side, clear of the origin axes.
const START: [i32; 2] = [620, 300];
const END: [i32; 2] = [840, 420];
/// The next segment leaves `END`; not horizontal or vertical, so nothing is inferred.
const NEXT: [i32; 2] = [1010, 350];

fn hud_is_open(world: &ApplicationWorld) -> bool {
    !world.window_text("##sketch-hud").is_empty()
}
// line-03 and line-06
#[given("Sketch3 is Ready with 215 dimensions needed in step 6")]
#[given("editable Sketch3 is Ready with 215 dimensions needed as recorded in step 6")]
fn ready_with_215(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(215);
    world.expect_ready();
    world.expect_dimensions_needed(215);
}
#[when("I select Line and choose the start in step 9")]
fn line_and_start(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.sketch_click(START);
    world.hover(END);
    assert!(hud_is_open(world), "expected the fields of the pending end");
}
#[when("press Escape before an endpoint")]
#[when("I press Escape before accepting an endpoint")]
fn escape_before_end(world: &mut ApplicationWorld) {
    world.escape();
}
#[then("the HUD closes and Ready returns with 215 dimensions needed")]
fn hud_closed_215(world: &mut ApplicationWorld) {
    assert!(!hud_is_open(world), "expected the fields to close");
    world.expect_ready();
    world.expect_dimensions_needed(215);
}
#[then("no new segment is visible at the canceled location")]
#[then("no line is visible at the cancelled location")]
fn no_segment(world: &mut ApplicationWorld) {
    world.expect_segment_absent(START, END);
}
#[when("I select ordinary Line through the variants menu")]
fn line_from_variants(world: &mut ApplicationWorld) {
    world.open_line_variants();
    world.click_text("Line");
}
#[when("I choose the first point recorded in step 9")]
fn first_point(world: &mut ApplicationWorld) {
    world.sketch_click(START);
    world.hover(END);
}
#[then("the HUD closes and the sketch returns to Ready")]
fn hud_closed_ready(world: &mut ApplicationWorld) {
    assert!(!hud_is_open(world), "expected the fields to close");
    world.expect_ready();
}
#[then("the status still reports 215 dimensions needed")]
fn still_215(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(215);
}
// line-05
#[given("the Line start and endpoint inputs in steps 3-4")]
fn start_and_end_inputs(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click(START);
}
#[when("the endpoint is accepted")]
fn endpoint_accepted(world: &mut ApplicationWorld) {
    world.sketch_click(END);
}
#[then("Line offers the next connected segment from that endpoint")]
fn offers_next(world: &mut ApplicationWorld) {
    world.expect_status("Select end of line");
    world.hover(NEXT);
    world.expect_segment_drawn(END, NEXT);
}
#[when("I press Escape in step 5")]
fn escape_step_five(world: &mut ApplicationWorld) {
    world.escape();
}
#[then("Ready returns and the accepted segment remains")]
fn ready_segment_remains(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_segment_drawn(START, END);
    world.expect_segment_absent(END, NEXT);
}
// line-22
#[given("Line has a newly chosen start point")]
fn newly_chosen_start(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(12);
    world.select_line_tool();
    world.sketch_click(START);
}
#[when("I double-click the intended endpoint as one uninterrupted gesture")]
fn double_click_end(world: &mut ApplicationWorld) {
    world.double_click(END);
}
#[then("the straight segment is accepted and Line requests a new start")]
fn accepted_new_start(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(START, END);
    world.expect_status("Select start of line");
    world.expect_dimensions_needed(16);
}
#[when("I exit Line then undo once")]
fn exit_and_undo(world: &mut ApplicationWorld) {
    world.escape();
    world.undo();
}
#[then("only that last segment disappears and dimensions needed changes from 16 to 12")]
fn last_gone(world: &mut ApplicationWorld) {
    world.expect_segment_absent(START, END);
    world.expect_dimensions_needed(12);
}
#[when("I redo")]
fn redo(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the segment returns and dimensions needed is 16")]
fn segment_back(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(START, END);
    world.expect_dimensions_needed(16);
}
// line-23 and line-24
/// A second segment off a free first: 20 with the seeds, one direction inferred horizontal.
const HORIZONTAL_END: [i32; 2] = [1040, 420];
#[given("the two connected segments accepted in steps 177 through 180")]
#[given("two accepted connected segments and the active pending preview in step 185")]
fn two_connected(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(15);
    world.select_line_tool();
    world.sketch_click(START);
    world.sketch_click(END);
    world.sketch_click(HORIZONTAL_END);
    world.expect_dimensions_needed(20);
    world.hover(NEXT);
}
#[given("Line is active with its dynamic length field selected")]
fn length_field_selected(world: &mut ApplicationWorld) {
    assert!(hud_is_open(world), "expected the fields of the pending end");
}
#[when("I press Ctrl+Z and inspect a observations")]
fn ctrl_z_active(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("both accepted segments remain visible and Line remains active")]
fn both_remain(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(START, END);
    world.expect_segment_drawn(END, HORIZONTAL_END);
    world.expect_status("Select end of line");
}
#[when("I move the pointer to display a nonzero pending segment and press Ctrl+Z again")]
fn move_and_ctrl_z(world: &mut ApplicationWorld) {
    world.hover([900, 300]);
    world.undo();
}
#[then("the accepted segments and pending preview remain visible")]
fn accepted_and_preview(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(START, END);
    world.expect_segment_drawn(HORIZONTAL_END, [900, 300]);
}
#[when("I terminate Line through marking-menu OK and press Ctrl+Z at Ready")]
fn terminate_and_undo(world: &mut ApplicationWorld) {
    choose_ok(world);
    world.undo();
}
fn choose_ok(world: &mut ApplicationWorld) {
    world.right_click(NEXT);
    world.pump(Vec::new());
    world.click_text("OK");
}
#[then("the second accepted segment disappears and dimensions needed is 19")]
fn second_gone(world: &mut ApplicationWorld) {
    world.expect_segment_absent(END, HORIZONTAL_END);
    world.expect_segment_drawn(START, END);
    world.expect_dimensions_needed(19);
}
#[when("I press Ctrl+Y")]
fn ctrl_y(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("the second segment returns and dimensions needed is 20")]
fn second_back(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(END, HORIZONTAL_END);
    world.expect_dimensions_needed(20);
}
#[then("no cause is asserted for the different active-command result")]
fn no_cause(_: &mut ApplicationWorld) {}
#[when("I choose OK in the marking menu")]
fn ok_in_menu(world: &mut ApplicationWorld) {
    choose_ok(world);
}
#[then("Line ends and Ready shows 20 dimensions needed")]
fn ended_20(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(20);
}
#[then("the two accepted segments remain")]
fn two_remain(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(START, END);
    world.expect_segment_drawn(END, HORIZONTAL_END);
}
#[then("the pending third preview is discarded")]
fn third_gone(world: &mut ApplicationWorld) {
    world.expect_segment_absent(HORIZONTAL_END, NEXT);
}
// line-21
#[given("the separate one-segment chain is accepted and the next segment is pending")]
fn one_segment_pending(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click(START);
    world.sketch_click(END);
    world.hover(NEXT);
}
#[when("I open the active menu")]
fn open_active_menu(world: &mut ApplicationWorld) {
    world.right_click(NEXT);
    world.pump(Vec::new());
}
#[then("Restart is enabled and Close is disabled")]
fn restart_enabled(world: &mut ApplicationWorld) {
    assert!(
        world.run_is_enabled("Restart"),
        "expected an enabled Restart"
    );
    assert!(!world.run_is_enabled("Close"), "expected a disabled Close");
}
#[when("I choose Restart")]
fn choose_restart(world: &mut ApplicationWorld) {
    world.click_text("Restart");
}
#[then("the pending preview disappears and the accepted segment remains")]
fn preview_gone(world: &mut ApplicationWorld) {
    world.expect_segment_absent(END, NEXT);
    world.expect_segment_drawn(START, END);
}
#[then("Line requests a new start point")]
fn new_start(world: &mut ApplicationWorld) {
    world.expect_status("Select start of line");
}
// line-20, up to the loop: the Region Properties half is in `regions.rs`
#[given("two connected straight segments have been accepted with Line still active")]
fn two_with_line_active(world: &mut ApplicationWorld) {
    use oblikovati_viewport::ViewportPhysicalKey::{Enter, Tab};
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click(START);
    world.hover(NEXT);
    // Legs of 4 and 4.0565 at a right angle: half their product is the recorded 8.113.
    for (length, angle) in [("4", "0"), ("4.0565", "90")] {
        world.type_text(length);
        world.key(Tab, false);
        world.type_text(angle);
        world.key(Enter, false);
    }
    world.hover(NEXT);
}
#[when("I open the active context menu and choose enabled Close")]
fn choose_close(world: &mut ApplicationWorld) {
    world.right_click(NEXT);
    world.pump(Vec::new());
    assert!(world.run_is_enabled("Close"), "expected an enabled Close");
    world.click_text("Close");
}
#[then("a third edge connects the current end to the first point")]
fn third_edge(world: &mut ApplicationWorld) {
    let [from, to] = world.last_line_pixels();
    world.expect_segment_drawn(from, to);
    world.expect_dimensions_needed(2);
}
#[then("Line returns to start-point selection")]
fn back_to_start(world: &mut ApplicationWorld) {
    world.expect_status("Select start of line");
}
// line-15 and line-16: a drag never accepts, a click does.
const SOURCE: [[i32; 2]; 2] = [[620, 280], [860, 280]];
const DRAG_END: [i32; 2] = [960, 380];
const CLICK_END: [i32; 2] = [980, 450];
const LATER_DRAG: [i32; 2] = [860, 560];
#[given("the recorded source line ends at (1550,190) with Line active")]
fn source_with_line_active(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click(SOURCE[0]);
    world.sketch_click(SOURCE[1]);
}
#[when("I drag from (1550,190) to (1590,260) and release")]
fn drag_from_source(world: &mut ApplicationWorld) {
    world.drag(SOURCE[1], DRAG_END);
}
#[then("the observations does not establish an arc preview")]
fn no_arc_preview(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(SOURCE[1], DRAG_END);
}
#[when("I click (1590,300)")]
fn click_after_drag(world: &mut ApplicationWorld) {
    world.sketch_click(CLICK_END);
}
#[then("a connected straight descending segment appears")]
fn connected_descending(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(SOURCE[1], CLICK_END);
}
#[when("I drag from (1590,300) to (1530,350) and release")]
fn second_drag(world: &mut ApplicationWorld) {
    world.drag(CLICK_END, LATER_DRAG);
}
#[then("the pending preview is straight with a length and angle HUD")]
fn straight_with_hud(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(CLICK_END, LATER_DRAG);
    world.expect_hud_shows_length_and_angle();
}
#[then("Ready appears with the two earlier straight segments retained")]
fn two_earlier_retained(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_segment_drawn(SOURCE[0], SOURCE[1]);
    world.expect_segment_drawn(SOURCE[1], CLICK_END);
}
#[then("the pending preview disappears")]
fn pending_disappears(world: &mut ApplicationWorld) {
    world.expect_segment_absent(CLICK_END, LATER_DRAG);
}
#[then("no successful tangent-arc transition is inferred")]
fn no_tangent_transition(_: &mut ApplicationWorld) {}
#[given("the retained source visibly ends near (1590,295)")]
fn retained_source(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click(SOURCE[0]);
    world.sketch_click(SOURCE[1]);
    world.sketch_click(CLICK_END);
    world.escape();
    world.seed_dimensions_needed(58);
}
#[when("I restart Line and drag from that endpoint to (1510,370) and release")]
fn restart_and_drag(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.drag(CLICK_END, LATER_DRAG);
}
#[then("a straight pending preview and length-angle HUD appear")]
fn preview_and_hud(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(CLICK_END, LATER_DRAG);
    world.expect_hud_shows_length_and_angle();
}
#[when("I right-click at (1510,370)")]
fn right_click_pending(world: &mut ApplicationWorld) {
    world.right_click(LATER_DRAG);
    world.pump(Vec::new());
}
#[then("the active Line menu offers Cancel (ESC) and OK")]
fn cancel_and_ok(world: &mut ApplicationWorld) {
    world.expect_text("Cancel (ESC)");
    assert!(world.run("OK").text == "OK", "expected an OK row");
}
#[then("Line and AutoProject are checked and Close is disabled")]
fn line_autoproject_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("AutoProject"), "expected AutoProject checked");
    assert!(world.menu_row_checked("Line"), "expected Line checked");
    assert!(!world.run_is_enabled("Close"), "expected a disabled Close");
}
#[when("I click Cancel (ESC)")]
fn click_cancel(world: &mut ApplicationWorld) {
    world.click_text("Cancel (ESC)");
}
#[then("Ready returns with 58 dimensions needed")]
fn ready_58(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(58);
}
#[then("the pending preview is removed with the two earlier segments retained")]
fn removed_retained(world: &mut ApplicationWorld) {
    world.expect_segment_absent(CLICK_END, LATER_DRAG);
    world.expect_segment_drawn(SOURCE[0], SOURCE[1]);
    world.expect_segment_drawn(SOURCE[1], CLICK_END);
}

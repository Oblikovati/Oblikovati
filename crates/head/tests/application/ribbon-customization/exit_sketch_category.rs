// SPDX-License-Identifier: GPL-2.0-only
//! Navigate the published sketch-exit customization category.
use super::{
    ApplicationWorld, click_in_customize, diagnostics, empty_tools_customization,
    list_keyboard_steps::{
        capture_plain_source_rows, expect_source_selection, expect_target_selection,
    },
    list_steps::source_rows,
    row_option_checked,
    transaction_steps::{expect_target_pending, target_row_runs, target_rows},
};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const SOURCE_ORDER: [&str; 3] = ["<Separator>", "CancelSketch", "FinishSketch"];
const TARGET_ORDER: [&str; 3] = ["CancelSketch", "<Separator>", "FinishSketch"];

#[given("ribbon customization shows an empty Part Tools target")]
#[given("ribbon customization is open in the MCAD Software for an empty Part Tools custom panel")]
fn open_empty_tools_draft(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
}

#[given("the source category is \"Exit 2D Sketch Tab\"")]
#[given("the available command category is Exit 2D Sketch Tab")]
fn select_exit_sketch_filter(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
    world.click_text("Exit 2D Sketch Tab");
    world.run_in_window("Customize", "Exit2DSketchTab");
}

#[given("the source list shows the separator, \"Cancel Sketch\", and \"Finish Sketch\"")]
#[given("the source list shows Separator, Cancel Sketch, and Finish Sketch in that order")]
fn exit_sketch_source_order(world: &mut ApplicationWorld) {
    let rows: Vec<_> = source_rows(world)
        .into_iter()
        .map(|run| run.text.as_str())
        .collect();
    assert_eq!(rows, SOURCE_ORDER);
}

#[when("the user clicks Cancel Sketch in the source list")]
fn select_cancel_in_source(world: &mut ApplicationWorld) {
    capture_plain_source_rows(world);
    let row = source_rows(world)
        .into_iter()
        .find(|run| run.text == "CancelSketch")
        .unwrap();
    world.click(diagnostics::pointer_pixels(row.center()));
}

#[when("the user selects \"Cancel Sketch\" and presses Page Down")]
fn page_down_from_cancel(world: &mut ApplicationWorld) {
    select_cancel_in_source(world);
    world.key(ViewportPhysicalKey::PageDown, false);
}

#[then("\"Finish Sketch\" is highlighted")]
fn finish_highlighted(world: &mut ApplicationWorld) {
    exit_row_highlighted(world, "FinishSketch");
}

#[then("the separator is highlighted")]
fn separator_highlighted(world: &mut ApplicationWorld) {
    exit_row_highlighted(world, "<Separator>");
}

fn exit_row_highlighted(world: &mut ApplicationWorld, expected: &str) {
    if target_rows(world) == TARGET_ORDER {
        expect_target_selection(world, expected);
    } else {
        world.hover([1250, 780]);
        assert!(world.row_accented_beside(expected, "Cancel Sketch"));
    }
}

#[then("all three source entries are highlighted")]
fn all_exit_commands_highlighted(world: &mut ApplicationWorld) {
    exit_sketch_source_order(world);
    expect_source_selection(world, None, &SOURCE_ORDER);
}

#[when("the user clicks the transfer-to-target button")]
fn transfer_selected_rows(world: &mut ApplicationWorld) {
    click_in_customize(world, ">");
}

#[then("the target contains Cancel Sketch, Separator, and Finish Sketch in that order")]
#[then("the staged rows remain in their original order")]
fn exit_sketch_target_order(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), TARGET_ORDER);
}

#[given("the target contains \"Cancel Sketch\", a separator, and \"Finish Sketch\" in that order")]
fn stage_exit_commands(world: &mut ApplicationWorld) {
    select_exit_sketch_filter(world);
    select_cancel_in_source(world);
    world.key(ViewportPhysicalKey::A, true);
    transfer_selected_rows(world);
    exit_sketch_target_order(world);
}

#[when("the user selects \"Cancel Sketch\" in the target list")]
fn select_cancel_in_target(world: &mut ApplicationWorld) {
    let row = target_row_runs(world)
        .into_iter()
        .find(|run| run.text == "CancelSketch")
        .expect("Cancel Sketch in the target list");
    world.click(diagnostics::pointer_pixels(row.center()));
    expect_target_selection(world, "CancelSketch");
}

#[then("the source list still contains Separator")]
fn reusable_separator_remains(world: &mut ApplicationWorld) {
    let rows = source_rows(world);
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].text, "<Separator>");
}

#[then("the two commands have Large and Text unchecked")]
fn staged_exit_command_options(world: &mut ApplicationWorld) {
    for command in ["CancelSketch", "FinishSketch"] {
        for field in ["Large", "Text"] {
            assert!(!row_option_checked(world, command, field));
        }
    }
}

#[then("Part Tools is marked as having pending changes")]
fn part_tools_has_staged_changes(world: &mut ApplicationWorld) {
    expect_target_pending(world, true);
}

// SPDX-License-Identifier: GPL-2.0-only
//! Check selection and display options across changes to the visible draft.
use super::transaction_steps::{add_command, all_commands, choose_target, target_is_empty};
use super::*;

#[given("the Ribbon customization dialog shows All Commands and the Part 3D Model target")]
#[given("the Ribbon customization dialog shows All Commands and an empty Part 3D Model target")]
fn model_source(world: &mut ApplicationWorld) {
    open_panel_menu(world);
    choose_customize(world);
    model_target(world);
    all_commands(world);
    target_is_empty(world);
}

#[given("the Part Tools custom panel list is initially empty")]
fn tools_target_is_initially_empty(world: &mut ApplicationWorld) {
    choose_target(world, "Tools");
    target_is_empty(world);
    choose_target(world, "3D Model");
}

#[then("\"2D Sketches\" remains selected in the available command list")]
#[then("2D Sketches is highlighted in the available commands list")]
fn sketches_remain_selected(world: &mut ApplicationWorld) {
    world.hover([800, 900]);
    assert!(world.row_accented_beside("2D Sketches", "<Separator>"));
    assert_eq!(world.run_in_window("Customize", ">").alpha, u8::MAX);
}

#[when("the user returns the command source to All Commands")]
fn return_to_all_commands(world: &mut ApplicationWorld) {
    click_in_customize(world, "SketchTab");
    world.click_text("All Commands");
    world.pump(Vec::new());
}

#[then("a \"<Separator>\" row appears in the target list")]
fn staged_separator(world: &mut ApplicationWorld) {
    assert!(transaction_steps::target_rows(world).contains(&"<Separator>"));
}

#[then("\"2D Sketches\" is not added to the target list")]
fn sketches_not_staged(world: &mut ApplicationWorld) {
    assert!(!transaction_steps::target_rows(world).contains(&"2DSketches"));
}

#[given("the ribbon customization dialog has an empty custom panel for \"Part | Tools\"")]
fn open_new_tools_draft(world: &mut ApplicationWorld) {
    world.add_part();
    empty_tools_customization(world);
}

#[when("the user opens ribbon customization with All Commands selected")]
fn open_empty_tools(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
}

#[given("the Part Tools custom panel list is empty")]
fn tools_before_open(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    close_customization_button(world);
}

#[given("its Large and Text options are checked")]
#[when("the user checks its Large and Text options")]
#[when("the user checks Large and Text for that command")]
fn check_sketches_display_options(world: &mut ApplicationWorld) {
    for option in ["Large", "Text"] {
        assert!(!row_option_checked(world, "2DSketches", option));
        toggle_row_option(world, "2DSketches", option);
    }
}

#[then("its Large and Text options are checked")]
fn both_sketches_display_options(world: &mut ApplicationWorld) {
    for option in ["Large", "Text"] {
        assert!(row_option_checked(world, "2DSketches", option));
    }
}

#[when("the user selects the 2D Sketches target row and clicks the remove command button")]
fn remove_staged_sketches(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    click_in_customize(world, "2DSketches");
    transfer_to_source(world);
}

#[then("2D Sketches appears in the target list")]
#[then("2D Sketches remains staged on Part Tools")]
#[then("\"2D Sketches\" remains in the target list")]
fn target_has_sketches(world: &mut ApplicationWorld) {
    sketches_in_target(world);
}

#[given("ribbon customization shows All Commands and empty Part Tools and Part Manage targets")]
fn tools_and_manage_empty(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    choose_target(world, "Manage");
    target_is_empty(world);
    choose_target(world, "Tools");
}

#[when("the user stages 2D Sketches on Part Tools")]
#[when("the user stages 2D Sketches on Part Manage")]
fn stage_sketches(world: &mut ApplicationWorld) {
    add_command(world, "2DSketches");
}

#[when("the user switches the target to Part Manage")]
fn switch_to_manage(world: &mut ApplicationWorld) {
    choose_target(world, "Manage");
}

#[when("the user switches the target back to Part Tools")]
fn switch_to_tools(world: &mut ApplicationWorld) {
    choose_target(world, "Tools");
}

#[then("the Part Manage row has Large and Text unchecked")]
fn manage_options_clear(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    for option in ["Large", "Text"] {
        assert!(!row_option_checked(world, "2DSketches", option));
    }
}

#[then("the Part Tools target shows a pending-change marker")]
#[then("the Part Manage target shows a pending-change marker")]
fn draft_remains_pending(world: &mut ApplicationWorld) {
    transaction_steps::expect_target_pending(world, true);
}

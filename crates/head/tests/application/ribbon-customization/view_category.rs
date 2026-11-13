// SPDX-License-Identifier: GPL-2.0-only
//! Returning a staged command to its View source category.
use super::*;

#[given("the ribbon customization dialog has an empty Part Tools target")]
fn empty_target(world: &mut ApplicationWorld) {
    open_tools_customization(world);
    empty_tools_target(world);
}

#[given("the command source is All Commands")]
fn all_commands(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("AllCommands"));
}

#[when("the user double-clicks 2D Sketches in the source list")]
#[when("the user double-clicks 2D Sketches in the target")]
fn double_click_command(world: &mut ApplicationWorld) {
    double_click_sketches(world);
}

#[then("2D Sketches appears in the target and is absent from the visible source list")]
#[then("2D Sketches remains in the target and is absent from the visible View Tab source list")]
fn command_only_in_target(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    assert_eq!(
        world.window_text("Customize").matches("2DSketches").count(),
        1
    );
}

#[when("the user selects View Tab as the command source")]
fn select_view_source(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
    world.click_text("View Tab");
    world.pump(Vec::new());
}

#[then("the target is empty")]
#[then("2D Sketches appears in the View Tab source list")]
fn command_only_in_source(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("ViewTab"));
    empty_tools_target(world);
}

#[then("the target remains marked as changed")]
fn target_remains_changed(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}

#[when("the user cancels customization")]
fn cancel(world: &mut ApplicationWorld) {
    click_in_customize(world, "Cancel");
}

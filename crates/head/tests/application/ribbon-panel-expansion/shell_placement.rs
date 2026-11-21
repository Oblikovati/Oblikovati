// SPDX-License-Identifier: GPL-2.0-only
//! Shell placement follows the same action across ribbon areas (#4106).
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
const EXPANSION: &str = "##ribbon-additional-PartModify";

#[given("\"Shell\" appears in the main area of the 3D Model \"Modify\" panel")]
#[given("Shell is in the Modify main panel")]
#[then("\"Shell\" appears again in the main area of the \"Modify\" panel")]
#[then("Shell appears in the main Modify panel")]
fn shell_in_main(world: &mut ApplicationWorld) {
    world.expect_text("Shell");
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
}
#[when("the user right-clicks \"Shell\" and selects \"Move to Expanded Panel\"")]
#[when("the user moves Shell to the expanded panel using its context menu")]
fn move_shell_to_expanded(world: &mut ApplicationWorld) {
    let shell = diagnostics::pointer_pixels(world.run("Shell").center());
    world.right_click(shell);
    world.click_text("Move to Expanded Panel");
    world.pump(Vec::new());
}
#[then("\"Shell\" is no longer visible in the main area")]
#[then("Shell is absent from the Modify main panel")]
fn shell_not_in_main(world: &mut ApplicationWorld) {
    world.expect_no_text("Shell");
}
#[when("the user opens the \"Modify\" panel expansion")]
#[when("the user opens the Modify expanded panel")]
fn open_modify(world: &mut ApplicationWorld) {
    super::click_expansion_arrow(world, "Modify");
    world.pump(Vec::new());
}
#[then("\"Shell\" appears in the expanded area")]
#[then("Shell is available in the expanded panel")]
fn shell_in_expanded(world: &mut ApplicationWorld) {
    world.run_in_window(EXPANSION, "Shell");
}
#[when("the user right-clicks \"Shell\" in the expanded area")]
fn right_click_shell_expanded(world: &mut ApplicationWorld) {
    let shell = diagnostics::pointer_pixels(world.run_in_window(EXPANSION, "Shell").center());
    world.right_click(shell);
}
fn restore_shell(world: &mut ApplicationWorld) {
    world.click_text("Move to Main Panel");
    world.pump(Vec::new());
}
#[when("the user moves Shell back to the main panel using its context menu")]
fn restore_shell_from_expanded(world: &mut ApplicationWorld) {
    right_click_shell_expanded(world);
    restore_shell(world);
}
#[when("the user selects Tools and then returns to 3D Model")]
fn switch_tabs(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.click_text("3D Model");
}
